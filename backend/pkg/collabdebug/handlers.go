package collabdebug

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	pkgerrors "github.com/josedab/waas/pkg/errors"
)

// Handler provides HTTP handlers for collaborative debugging
type Handler struct {
	service *Service
}

// NewHandler creates a new collaborative debugging handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers collaborative debugging routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	cd := router.Group("/collab-debug")
	{
		// Sessions
		cd.POST("/sessions", h.CreateSession)
		cd.GET("/sessions", h.ListSessions)
		cd.GET("/sessions/:id", h.GetSession)
		cd.POST("/sessions/:id/close", h.CloseSession)

		// Participants
		cd.POST("/sessions/:id/join", h.JoinSession)
		cd.POST("/sessions/:id/leave", h.LeaveSession)
		cd.POST("/sessions/:id/presence", h.UpdatePresence)
		cd.GET("/sessions/:id/participants", h.GetParticipants)

		// Annotations
		cd.POST("/sessions/:id/annotations", h.CreateAnnotation)
		cd.GET("/sessions/:id/annotations", h.GetAnnotations)
		cd.POST("/annotations/:id/resolve", h.ResolveAnnotation)

		// Shared state
		cd.POST("/sessions/:id/state", h.UpdateSharedState)

		// Activities & recordings
		cd.GET("/sessions/:id/activities", h.GetActivities)
		cd.GET("/sessions/:id/recording", h.GetRecording)

		// Summary
		cd.GET("/summary", h.GetSummary)
	}
}

// @Summary Create a collaborative debugging session
// @Tags CollabDebug
// @Accept json
// @Produce json
// @Param body body CreateSessionRequest true "Session configuration"
// @Success 201 {object} DebugSession
// @Router /collab-debug/sessions [post]
func (h *Handler) CreateSession(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	session, err := h.service.CreateSession(c.Request.Context(), tenantID, tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, session)
}

// @Summary List collaborative debugging sessions
// @Tags CollabDebug
// @Produce json
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /collab-debug/sessions [get]
func (h *Handler) ListSessions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	sessions, total, err := h.service.ListSessions(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions, "total": total})
}

// @Summary Get a collaborative debugging session
// @Tags CollabDebug
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} DebugSession
// @Router /collab-debug/sessions/{id} [get]
func (h *Handler) GetSession(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sessionID := c.Param("id")

	session, err := h.service.GetSession(c.Request.Context(), tenantID, sessionID)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "session")
		return
	}

	c.JSON(http.StatusOK, session)
}

// @Summary Close a collaborative debugging session
// @Tags CollabDebug
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} map[string]interface{}
// @Router /collab-debug/sessions/{id}/close [post]
func (h *Handler) CloseSession(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sessionID := c.Param("id")

	if err := h.service.CloseSession(c.Request.Context(), tenantID, sessionID); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session closed"})
}

// @Summary Join a collaborative debugging session
// @Tags CollabDebug
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Param body body JoinSessionRequest true "Join configuration"
// @Success 200 {object} Participant
// @Router /collab-debug/sessions/{id}/join [post]
func (h *Handler) JoinSession(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sessionID := c.Param("id")
	var req JoinSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	participant, err := h.service.JoinSession(c.Request.Context(), tenantID, sessionID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, participant)
}

// @Summary Leave a collaborative debugging session
// @Tags CollabDebug
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} map[string]interface{}
// @Router /collab-debug/sessions/{id}/leave [post]
func (h *Handler) LeaveSession(c *gin.Context) {
	sessionID := c.Param("id")
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	if err := h.service.LeaveSession(c.Request.Context(), sessionID, req.UserID); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "left session"})
}

// @Summary Update participant presence
// @Tags CollabDebug
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Param body body UpdateCursorRequest true "Cursor update"
// @Success 200 {object} map[string]interface{}
// @Router /collab-debug/sessions/{id}/presence [post]
func (h *Handler) UpdatePresence(c *gin.Context) {
	sessionID := c.Param("id")
	var req UpdateCursorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	if err := h.service.UpdatePresence(c.Request.Context(), sessionID, &req); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "presence updated"})
}

// @Summary Get session participants
// @Tags CollabDebug
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} map[string]interface{}
// @Router /collab-debug/sessions/{id}/participants [get]
func (h *Handler) GetParticipants(c *gin.Context) {
	sessionID := c.Param("id")

	participants, err := h.service.GetOnlineParticipants(c.Request.Context(), sessionID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"participants": participants})
}

// @Summary Create an annotation
// @Tags CollabDebug
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Param body body CreateAnnotationRequest true "Annotation content"
// @Success 201 {object} Annotation
// @Router /collab-debug/sessions/{id}/annotations [post]
func (h *Handler) CreateAnnotation(c *gin.Context) {
	sessionID := c.Param("id")
	var req CreateAnnotationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	annotation, err := h.service.CreateAnnotation(c.Request.Context(), sessionID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, annotation)
}

// @Summary Get annotations for a session
// @Tags CollabDebug
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} map[string]interface{}
// @Router /collab-debug/sessions/{id}/annotations [get]
func (h *Handler) GetAnnotations(c *gin.Context) {
	sessionID := c.Param("id")

	annotations, err := h.service.GetAnnotations(c.Request.Context(), sessionID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"annotations": annotations})
}

// @Summary Resolve an annotation
// @Tags CollabDebug
// @Produce json
// @Param id path string true "Annotation ID"
// @Success 200 {object} map[string]interface{}
// @Router /collab-debug/annotations/{id}/resolve [post]
func (h *Handler) ResolveAnnotation(c *gin.Context) {
	annotationID := c.Param("id")

	if err := h.service.ResolveAnnotation(c.Request.Context(), annotationID); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "annotation resolved"})
}

// @Summary Update shared state
// @Tags CollabDebug
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Param body body SharedState true "Shared state"
// @Success 200 {object} map[string]interface{}
// @Router /collab-debug/sessions/{id}/state [post]
func (h *Handler) UpdateSharedState(c *gin.Context) {
	sessionID := c.Param("id")
	var state SharedState
	if err := c.ShouldBindJSON(&state); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	state.SessionID = sessionID
	if err := h.service.UpdateSharedState(c.Request.Context(), &state); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "shared state updated"})
}

// @Summary Get session activities
// @Tags CollabDebug
// @Produce json
// @Param id path string true "Session ID"
// @Param limit query int false "Limit" default(100)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /collab-debug/sessions/{id}/activities [get]
func (h *Handler) GetActivities(c *gin.Context) {
	sessionID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	activities, err := h.service.repo.GetActivities(c.Request.Context(), sessionID, limit, offset)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"activities": activities})
}

// @Summary Get session recording
// @Tags CollabDebug
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} SessionRecording
// @Router /collab-debug/sessions/{id}/recording [get]
func (h *Handler) GetRecording(c *gin.Context) {
	sessionID := c.Param("id")

	recording, err := h.service.GetRecording(c.Request.Context(), sessionID)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "recording")
		return
	}

	c.JSON(http.StatusOK, recording)
}

// @Summary Get collaborative debugging summary
// @Tags CollabDebug
// @Produce json
// @Success 200 {object} SessionSummary
// @Router /collab-debug/summary [get]
func (h *Handler) GetSummary(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	summary, err := h.service.GetSessionSummary(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, summary)
}
