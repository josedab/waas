package eventmesh

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterMeshRoutes registers mesh networking routes
func (h *Handler) RegisterMeshRoutes(r *gin.RouterGroup) {
	mesh := r.Group("/eventmesh/mesh")
	{
		mesh.POST("/join", h.JoinMesh)
		mesh.DELETE("/nodes/:id", h.RemoveNode)
		mesh.GET("/topology", h.GetTopology)
		mesh.POST("/nodes/:id/heartbeat", h.Heartbeat)
		mesh.POST("/route-event", h.RouteMeshEvent)
		mesh.POST("/detect-failures", h.DetectFailures)
		mesh.POST("/resolve-split-brain", h.ResolveSplitBrain)
		mesh.GET("/replication", h.GetReplicationState)
		mesh.GET("/failovers", h.GetFailoverEvents)
	}
}

// MeshHandler wraps a MeshManager for HTTP access
type MeshHandler struct {
	manager *MeshManager
}

// NewMeshHandler creates a new mesh handler
func NewMeshHandler(manager *MeshManager) *MeshHandler {
	return &MeshHandler{manager: manager}
}

// RegisterMeshHandlerRoutes registers mesh routes using a dedicated handler
func (mh *MeshHandler) RegisterRoutes(r *gin.RouterGroup) {
	mesh := r.Group("/eventmesh/mesh")
	{
		mesh.POST("/join", mh.JoinMesh)
		mesh.DELETE("/nodes/:id", mh.RemoveNode)
		mesh.GET("/topology", mh.GetTopology)
		mesh.POST("/nodes/:id/heartbeat", mh.Heartbeat)
		mesh.POST("/route-event", mh.RouteMeshEvent)
		mesh.POST("/detect-failures", mh.DetectFailures)
		mesh.POST("/resolve-split-brain", mh.ResolveSplitBrain)
		mesh.GET("/replication", mh.GetReplicationState)
		mesh.GET("/failovers", mh.GetFailoverEvents)
	}
}

func (mh *MeshHandler) JoinMesh(c *gin.Context) {
	var req JoinMeshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node, err := mh.manager.JoinMesh(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, node)
}

func (mh *MeshHandler) RemoveNode(c *gin.Context) {
	nodeID := c.Param("id")
	if err := mh.manager.RemoveNode(c.Request.Context(), nodeID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (mh *MeshHandler) GetTopology(c *gin.Context) {
	topo := mh.manager.GetTopology(c.Request.Context())
	c.JSON(http.StatusOK, topo)
}

func (mh *MeshHandler) Heartbeat(c *gin.Context) {
	nodeID := c.Param("id")
	if err := mh.manager.Heartbeat(c.Request.Context(), nodeID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"acknowledged": true})
}

func (mh *MeshHandler) RouteMeshEvent(c *gin.Context) {
	var event MeshEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node, err := mh.manager.RouteToMesh(c.Request.Context(), &event)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"routed_to": node, "event": event})
}

func (mh *MeshHandler) DetectFailures(c *gin.Context) {
	failovers := mh.manager.DetectFailures(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"failovers": failovers, "count": len(failovers)})
}

type SplitBrainRequest struct {
	PartitionA []string `json:"partition_a" binding:"required"`
	PartitionB []string `json:"partition_b" binding:"required"`
}

func (mh *MeshHandler) ResolveSplitBrain(c *gin.Context) {
	var req SplitBrainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resolution, err := mh.manager.ResolveSplitBrain(c.Request.Context(), req.PartitionA, req.PartitionB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resolution)
}

func (mh *MeshHandler) GetReplicationState(c *gin.Context) {
	states := mh.manager.GetReplicationState(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"replication": states})
}

func (mh *MeshHandler) GetFailoverEvents(c *gin.Context) {
	events := mh.manager.GetFailoverEvents(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"failovers": events})
}

// Stub handlers on the main Handler for backwards compatibility
func (h *Handler) JoinMesh(c *gin.Context)          { c.JSON(http.StatusNotImplemented, gin.H{"error": "mesh manager not configured"}) }
func (h *Handler) RemoveNode(c *gin.Context)         { c.JSON(http.StatusNotImplemented, gin.H{"error": "mesh manager not configured"}) }
func (h *Handler) GetTopology(c *gin.Context)        { c.JSON(http.StatusNotImplemented, gin.H{"error": "mesh manager not configured"}) }
func (h *Handler) Heartbeat(c *gin.Context)          { c.JSON(http.StatusNotImplemented, gin.H{"error": "mesh manager not configured"}) }
func (h *Handler) RouteMeshEvent(c *gin.Context)     { c.JSON(http.StatusNotImplemented, gin.H{"error": "mesh manager not configured"}) }
func (h *Handler) DetectFailures(c *gin.Context)     { c.JSON(http.StatusNotImplemented, gin.H{"error": "mesh manager not configured"}) }
func (h *Handler) ResolveSplitBrain(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"error": "mesh manager not configured"}) }
func (h *Handler) GetReplicationState(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"error": "mesh manager not configured"}) }
func (h *Handler) GetFailoverEvents(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"error": "mesh manager not configured"}) }
