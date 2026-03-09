package collabdebug

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WebSocketMessage represents a real-time debugging message.
type WebSocketMessage struct {
	Type      string      `json:"type"` // cursor_move, annotation, state_sync, chat, highlight
	SessionID string      `json:"session_id"`
	UserID    string      `json:"user_id"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// LiveCursor represents a participant's cursor position.
type LiveCursor struct {
	UserID     string `json:"user_id"`
	DisplayName string `json:"display_name"`
	Section    string `json:"section"` // payload, headers, response, timeline
	Line       int    `json:"line,omitempty"`
	Column     int    `json:"column,omitempty"`
}

// DebugBreakpoint represents a breakpoint in the delivery pipeline.
type DebugBreakpoint struct {
	ID         string `json:"id"`
	SessionID  string `json:"session_id"`
	Step       string `json:"step"` // received, validated, transformed, queued, delivering
	Condition  string `json:"condition,omitempty"`
	IsActive   bool   `json:"is_active"`
	CreatedBy  string `json:"created_by"`
}

// SessionReplayState stores the full state for a debug session replay.
type SessionReplayState struct {
	SessionID  string                 `json:"session_id"`
	Cursors    []LiveCursor           `json:"cursors"`
	Breakpoints []DebugBreakpoint     `json:"breakpoints"`
	Annotations []Annotation          `json:"annotations"`
	ChatLog    []ChatMessage          `json:"chat_log"`
	ViewState  map[string]interface{} `json:"view_state"`
	RecordedAt time.Time              `json:"recorded_at"`
}

// ChatMessage represents a chat message in a debug session.
type ChatMessage struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// AddBreakpoint adds a debug breakpoint to a session.
func (s *Service) AddBreakpoint(ctx context.Context, tenantID, sessionID string, step, condition, createdBy string) (*DebugBreakpoint, error) {
	bp := &DebugBreakpoint{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Step:      step,
		Condition: condition,
		IsActive:  true,
		CreatedBy: createdBy,
	}
	return bp, nil
}

// GetSessionReplayState returns the full state for replaying a debug session.
func (s *Service) GetSessionReplayState(ctx context.Context, tenantID, sessionID string) (*SessionReplayState, error) {
	session, err := s.GetSession(ctx, tenantID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	annotations, _ := s.GetAnnotations(ctx, sessionID)

	state := &SessionReplayState{
		SessionID:   sessionID,
		Annotations: annotations,
		RecordedAt:  time.Now(),
	}

	_ = session
	return state, nil
}

// RegisterWebSocketRoutes registers WebSocket-based collaborative debugging routes.
func (h *Handler) RegisterWebSocketRoutes(router *gin.RouterGroup) {
	collab := router.Group("/collab-debug")
	{
		collab.POST("/sessions/:id/breakpoints", h.AddBreakpoint)
		collab.GET("/sessions/:id/replay-state", h.GetSessionReplayState)
		collab.POST("/sessions/:id/chat", h.PostChatMessage)
	}
}

func (h *Handler) AddBreakpoint(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessionID := c.Param("id")
	var req struct {
		Step      string `json:"step" binding:"required"`
		Condition string `json:"condition,omitempty"`
		CreatedBy string `json:"created_by" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	bp, err := h.service.AddBreakpoint(c.Request.Context(), tenantID, sessionID, req.Step, req.Condition, req.CreatedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, bp)
}

func (h *Handler) GetSessionReplayState(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessionID := c.Param("id")
	state, err := h.service.GetSessionReplayState(c.Request.Context(), tenantID, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, state)
}

func (h *Handler) PostChatMessage(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var msg ChatMessage
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	msg.ID = uuid.New().String()
	msg.Timestamp = time.Now()

	c.JSON(http.StatusCreated, msg)
}
