package graphql

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/josedab/waas/pkg/utils"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Server provides the GraphQL HTTP and WebSocket server
type Server struct {
	resolver *Resolver
	upgrader websocket.Upgrader
}

// NewServer creates a new GraphQL server
func NewServer(resolver *Resolver) *Server {
	return &Server{
		resolver: resolver,
		upgrader: websocket.Upgrader{
			CheckOrigin:     utils.CheckWebSocketOrigin(),
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// GraphQLRequest represents an incoming GraphQL request
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL response
type GraphQLResponse struct {
	Data   interface{}    `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message    string                 `json:"message"`
	Locations  []Location             `json:"locations,omitempty"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// Location represents a GraphQL error location
type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// RegisterRoutes registers GraphQL routes on the Gin router
func (s *Server) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/graphql", s.HandleQuery)
	router.GET("/graphql", s.HandleWebSocket)
	router.GET("/graphql/playground", s.HandlePlayground)
}

// HandleQuery handles GraphQL queries and mutations over HTTP
func (s *Server) HandleQuery(c *gin.Context) {
	var req GraphQLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, GraphQLResponse{
			Errors: []GraphQLError{{Message: "Invalid request body: " + err.Error()}},
		})
		return
	}

	// Get tenant ID from context (set by auth middleware)
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, GraphQLResponse{
			Errors: []GraphQLError{{Message: "Unauthorized"}},
		})
		return
	}

	// Execute the query using the QueryExecutor
	executor := NewQueryExecutor(s.resolver)
	result := executor.Execute(c.Request.Context(), tenantID, req)
	c.JSON(http.StatusOK, result)
}

// HandleWebSocket handles GraphQL subscriptions over WebSocket
func (s *Server) HandleWebSocket(c *gin.Context) {
	// Upgrade to WebSocket
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Handle WebSocket messages
	s.handleWebSocketConnection(conn, c)
}

// WebSocket message types (graphql-ws protocol)
const (
	GQLConnectionInit      = "connection_init"
	GQLConnectionAck       = "connection_ack"
	GQLConnectionError     = "connection_error"
	GQLConnectionTerminate = "connection_terminate"
	GQLStart               = "start"
	GQLData                = "data"
	GQLError               = "error"
	GQLComplete            = "complete"
	GQLStop                = "stop"
)

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	ID      string                 `json:"id,omitempty"`
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

func (s *Server) handleWebSocketConnection(conn *websocket.Conn, c *gin.Context) {
	var tenantID string
	var subMu sync.RWMutex
	subscriptions := make(map[string]*Subscriber)

	defer func() {
		subMu.RLock()
		subs := make([]*Subscriber, 0, len(subscriptions))
		for _, sub := range subscriptions {
			subs = append(subs, sub)
		}
		subMu.RUnlock()
		for _, sub := range subs {
			s.resolver.subscriptions.Unsubscribe(sub)
		}
	}()

	for {
		var msg WebSocketMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}

		switch msg.Type {
		case GQLConnectionInit:
			// Authenticate from payload
			if payload, ok := msg.Payload["authToken"].(string); ok {
				tenantID = s.authenticateToken(payload)
			}

			if tenantID == "" {
				conn.WriteJSON(WebSocketMessage{
					Type:    GQLConnectionError,
					Payload: map[string]interface{}{"message": "Unauthorized"},
				})
				return
			}

			conn.WriteJSON(WebSocketMessage{Type: GQLConnectionAck})

		case GQLStart:
			if tenantID == "" {
				conn.WriteJSON(WebSocketMessage{
					ID:      msg.ID,
					Type:    GQLError,
					Payload: map[string]interface{}{"message": "Not authenticated"},
				})
				continue
			}

			// Parse subscription query
			query, _ := msg.Payload["query"].(string)
			variables, _ := msg.Payload["variables"].(map[string]interface{})

			channel, filter := s.parseSubscription(query, variables)
			if channel == "" {
				conn.WriteJSON(WebSocketMessage{
					ID:      msg.ID,
					Type:    GQLError,
					Payload: map[string]interface{}{"message": "Invalid subscription"},
				})
				continue
			}

			// Create subscription
			sub := s.resolver.subscriptions.Subscribe(c.Request.Context(), tenantID, channel, filter)
			subMu.Lock()
			subscriptions[msg.ID] = sub
			subMu.Unlock()

			// Forward messages
			go func(id string, sub *Subscriber) {
				for {
					select {
					case data := <-sub.Messages:
						var payload map[string]interface{}
						json.Unmarshal(data, &payload)
						conn.WriteJSON(WebSocketMessage{
							ID:      id,
							Type:    GQLData,
							Payload: map[string]interface{}{"data": payload},
						})
					case <-sub.Done:
						return
					}
				}
			}(msg.ID, sub)

		case GQLStop:
			subMu.Lock()
			if sub, ok := subscriptions[msg.ID]; ok {
				s.resolver.subscriptions.Unsubscribe(sub)
				delete(subscriptions, msg.ID)
			}
			subMu.Unlock()
			conn.WriteJSON(WebSocketMessage{ID: msg.ID, Type: GQLComplete})

		case GQLConnectionTerminate:
			return
		}
	}
}

func (s *Server) authenticateToken(token string) string {
	// Decode base64 token (simplified - in production, verify JWT)
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return ""
	}
	return string(decoded)
}

func (s *Server) parseSubscription(query string, variables map[string]interface{}) (string, map[string]interface{}) {
	// Simple subscription parsing - in production use a proper GraphQL parser
	switch {
	case containsSubscription(query, "deliveryUpdated"):
		filter := make(map[string]interface{})
		if endpointID, ok := variables["endpointId"]; ok {
			filter["endpointId"] = endpointID
		}
		return ChannelDeliveryUpdated, filter
	case containsSubscription(query, "anomalyDetected"):
		return ChannelAnomalyDetected, nil
	case containsSubscription(query, "metricsUpdated"):
		return ChannelMetricsUpdated, nil
	default:
		return "", nil
	}
}

func containsSubscription(query, name string) bool {
	return len(query) > 0 && (query == name ||
		contains(query, "subscription") && contains(query, name))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// HandlePlayground serves the GraphQL Playground UI
func (s *Server) HandlePlayground(c *gin.Context) {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>WAAS GraphQL Playground</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
  <link rel="shortcut icon" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/favicon.png" />
  <script src="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>
<body>
  <div id="root">
    <style>
      body { background-color: rgb(23, 42, 58); font-family: 'Open Sans', sans-serif; height: 90vh; }
      #root { height: 100%%; width: 100%%; overflow: hidden; }
      .loading { font-size: 32px; font-weight: 200; color: rgba(255, 255, 255, .6); margin-left: 20px; }
      img { width: 78px; height: 78px; }
      .title { font-weight: 400; }
    </style>
    <img src='https://cdn.jsdelivr.net/npm/graphql-playground-react/build/logo.png' alt='Loading'>
    <div class="loading">Loading <span class="title">WAAS GraphQL Playground</span></div>
  </div>
  <script>window.addEventListener('load', function (event) {
    GraphQLPlayground.init(document.getElementById('root'), {
      endpoint: '%s',
      settings: { 'editor.theme': 'dark' }
    })
  })</script>
</body>
</html>`, c.Request.URL.Path[:len(c.Request.URL.Path)-11]) // Remove "/playground"

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}
