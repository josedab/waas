package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
)

// CollaborationHandler handles team collaboration endpoints
type CollaborationHandler struct {
	repo   repository.CollaborationRepository
	logger *utils.Logger
}

// NewCollaborationHandler creates a new collaboration handler
func NewCollaborationHandler(repo repository.CollaborationRepository, logger *utils.Logger) *CollaborationHandler {
	return &CollaborationHandler{
		repo:   repo,
		logger: logger,
	}
}

// CreateTeam creates a new team workspace
// @Summary Create a team
// @Description Creates a new team workspace for collaboration
// @Tags collaboration
// @Accept json
// @Produce json
// @Param request body models.CreateTeamRequest true "Team creation request"
// @Success 201 {object} models.Team
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /teams [post]
func (h *CollaborationHandler) CreateTeam(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	var req models.CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	team := &models.Team{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Settings:    req.Settings,
	}

	if err := h.repo.CreateTeam(c.Request.Context(), team); err != nil {
		h.logger.Error("Failed to create team", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create team"})
		return
	}

	// Add activity
	h.recordActivity(c, team.ID, nil, "team_created", "team", &team.ID, team.Name, nil)

	c.JSON(http.StatusCreated, team)
}

// GetTeams returns all teams for the tenant
// @Summary List teams
// @Description Returns all teams for the current tenant
// @Tags collaboration
// @Produce json
// @Success 200 {array} models.Team
// @Failure 401 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /teams [get]
func (h *CollaborationHandler) GetTeams(c *gin.Context) {
	tenantID, ok := RequireTenantID(c)
	if !ok {
		return
	}

	teams, err := h.repo.GetTeamsByTenant(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to get teams", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get teams"})
		return
	}

	c.JSON(http.StatusOK, teams)
}

// GetTeam returns a specific team
// @Summary Get team details
// @Description Returns details for a specific team
// @Tags collaboration
// @Produce json
// @Param team_id path string true "Team ID"
// @Success 200 {object} models.Team
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /teams/{team_id} [get]
func (h *CollaborationHandler) GetTeam(c *gin.Context) {
	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team_id"})
		return
	}

	team, err := h.repo.GetTeam(c.Request.Context(), teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return
	}

	c.JSON(http.StatusOK, team)
}

// InviteMember invites a member to the team
// @Summary Invite team member
// @Description Invites a new member to the team
// @Tags collaboration
// @Accept json
// @Produce json
// @Param team_id path string true "Team ID"
// @Param request body models.InviteMemberRequest true "Invite request"
// @Success 201 {object} models.CollabTeamMember
// @Failure 400 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /teams/{team_id}/members [post]
func (h *CollaborationHandler) InviteMember(c *gin.Context) {
	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team_id"})
		return
	}

	var req models.InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if member already exists
	existing, _ := h.repo.GetTeamMemberByEmail(c.Request.Context(), teamID, req.Email)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "member already exists"})
		return
	}

	member := &models.CollabTeamMember{
		TeamID: teamID,
		UserID: uuid.New(), // Would be resolved from user lookup
		Email:  req.Email,
		Role:   req.Role,
	}

	if err := h.repo.AddTeamMember(c.Request.Context(), member); err != nil {
		h.logger.Error("Failed to add team member", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add member"})
		return
	}

	h.recordActivity(c, teamID, nil, "member_invited", "member", &member.ID, req.Email, map[string]interface{}{"role": req.Role})

	c.JSON(http.StatusCreated, member)
}

// GetTeamMembers returns all members of a team
// @Summary List team members
// @Description Returns all members of a team
// @Tags collaboration
// @Produce json
// @Param team_id path string true "Team ID"
// @Success 200 {array} models.CollabTeamMember
// @Security ApiKeyAuth
// @Router /teams/{team_id}/members [get]
func (h *CollaborationHandler) GetTeamMembers(c *gin.Context) {
	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team_id"})
		return
	}

	members, err := h.repo.GetTeamMembers(c.Request.Context(), teamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get members"})
		return
	}

	c.JSON(http.StatusOK, members)
}

// CreateChangeRequest creates a change request for a configuration
// @Summary Create change request
// @Description Creates a PR-style change request for a shared configuration
// @Tags collaboration
// @Accept json
// @Produce json
// @Param team_id path string true "Team ID"
// @Param request body models.CreateChangeRequestRequest true "Change request"
// @Success 201 {object} models.ChangeRequest
// @Failure 400 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /teams/{team_id}/change-requests [post]
func (h *CollaborationHandler) CreateChangeRequest(c *gin.Context) {
	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team_id"})
		return
	}

	var req models.CreateChangeRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	configID, err := uuid.Parse(req.ConfigID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config_id"})
		return
	}

	// Get current config version
	config, err := h.repo.GetSharedConfig(c.Request.Context(), configID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "configuration not found"})
		return
	}

	cr := &models.ChangeRequest{
		TeamID:          teamID,
		ConfigID:        configID,
		Title:           req.Title,
		Description:     req.Description,
		ProposedChanges: req.ProposedChanges,
		BaseVersion:     config.Version,
		CreatedBy:       uuid.New(), // Would come from auth context
	}

	if err := h.repo.CreateChangeRequest(c.Request.Context(), cr); err != nil {
		h.logger.Error("Failed to create change request", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create change request"})
		return
	}

	h.recordActivity(c, teamID, nil, "change_request_created", "change_request", &cr.ID, req.Title, nil)

	c.JSON(http.StatusCreated, cr)
}

// GetChangeRequests returns all change requests for a team
// @Summary List change requests
// @Description Returns all change requests for a team
// @Tags collaboration
// @Produce json
// @Param team_id path string true "Team ID"
// @Param status query string false "Filter by status"
// @Success 200 {array} models.ChangeRequest
// @Security ApiKeyAuth
// @Router /teams/{team_id}/change-requests [get]
func (h *CollaborationHandler) GetChangeRequests(c *gin.Context) {
	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team_id"})
		return
	}

	status := c.Query("status")
	crs, err := h.repo.GetTeamChangeRequests(c.Request.Context(), teamID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get change requests"})
		return
	}

	c.JSON(http.StatusOK, crs)
}

// ReviewChangeRequest submits a review for a change request
// @Summary Review change request
// @Description Submits a review (approve/reject/comment) for a change request
// @Tags collaboration
// @Accept json
// @Produce json
// @Param team_id path string true "Team ID"
// @Param cr_id path string true "Change Request ID"
// @Param request body models.ReviewChangeRequestRequest true "Review"
// @Success 200 {object} models.ChangeRequestReview
// @Failure 400 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /teams/{team_id}/change-requests/{cr_id}/reviews [post]
func (h *CollaborationHandler) ReviewChangeRequest(c *gin.Context) {
	crID, err := uuid.Parse(c.Param("cr_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid change request id"})
		return
	}

	teamID, _ := uuid.Parse(c.Param("team_id"))

	var req models.ReviewChangeRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	review := &models.ChangeRequestReview{
		ChangeRequestID: crID,
		ReviewerID:      uuid.New(), // Would come from auth context
		Status:          req.Status,
		Comments:        req.Comments,
	}

	if err := h.repo.AddReview(c.Request.Context(), review); err != nil {
		h.logger.Error("Failed to add review", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add review"})
		return
	}

	// Update CR status if approved
	if req.Status == models.ReviewApproved {
		cr, _ := h.repo.GetChangeRequest(c.Request.Context(), crID)
		if cr != nil {
			cr.Status = models.ChangeRequestApproved
			h.repo.UpdateChangeRequest(c.Request.Context(), cr)
		}
	}

	h.recordActivity(c, teamID, nil, "change_request_reviewed", "change_request", &crID, req.Status, nil)

	c.JSON(http.StatusOK, review)
}

// MergeChangeRequest merges an approved change request
// @Summary Merge change request
// @Description Merges an approved change request into the configuration
// @Tags collaboration
// @Produce json
// @Param team_id path string true "Team ID"
// @Param cr_id path string true "Change Request ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /teams/{team_id}/change-requests/{cr_id}/merge [post]
func (h *CollaborationHandler) MergeChangeRequest(c *gin.Context) {
	crID, err := uuid.Parse(c.Param("cr_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid change request id"})
		return
	}

	teamID, _ := uuid.Parse(c.Param("team_id"))

	cr, err := h.repo.GetChangeRequest(c.Request.Context(), crID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "change request not found"})
		return
	}

	if cr.Status != models.ChangeRequestApproved {
		c.JSON(http.StatusBadRequest, gin.H{"error": "change request must be approved before merging"})
		return
	}

	// Get and update the config
	config, err := h.repo.GetSharedConfig(c.Request.Context(), cr.ConfigID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "configuration not found"})
		return
	}

	// Save current version before update
	version := &models.ConfigVersion{
		ConfigID:      config.ID,
		Version:       config.Version,
		ConfigData:    config.ConfigData,
		ChangeSummary: cr.Title,
		ChangedBy:     cr.CreatedBy,
	}
	h.repo.SaveConfigVersion(c.Request.Context(), version)

	// Apply changes
	for k, v := range cr.ProposedChanges {
		config.ConfigData[k] = v
	}

	if err := h.repo.UpdateSharedConfig(c.Request.Context(), config); err != nil {
		h.logger.Error("Failed to update config", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to merge changes"})
		return
	}

	// Mark CR as merged
	cr.Status = models.ChangeRequestMerged
	h.repo.UpdateChangeRequest(c.Request.Context(), cr)

	h.recordActivity(c, teamID, nil, "change_request_merged", "change_request", &crID, cr.Title, nil)

	c.JSON(http.StatusOK, gin.H{"message": "change request merged successfully"})
}

// GetActivityFeed returns the activity feed for a team
// @Summary Get activity feed
// @Description Returns the activity feed for a team
// @Tags collaboration
// @Produce json
// @Param team_id path string true "Team ID"
// @Param limit query int false "Limit results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} models.ActivityFeedItem
// @Security ApiKeyAuth
// @Router /teams/{team_id}/activity [get]
func (h *CollaborationHandler) GetActivityFeed(c *gin.Context) {
	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team_id"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	activities, err := h.repo.GetTeamActivity(c.Request.Context(), teamID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get activity"})
		return
	}

	c.JSON(http.StatusOK, activities)
}

// Helper to record activity
func (h *CollaborationHandler) recordActivity(c *gin.Context, teamID uuid.UUID, actorID *uuid.UUID, actionType, resourceType string, resourceID *uuid.UUID, resourceName string, details map[string]interface{}) {
	activity := &models.ActivityFeedItem{
		TeamID:       teamID,
		ActorID:      actorID,
		ActionType:   actionType,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Details:      details,
	}
	if err := h.repo.AddActivity(c.Request.Context(), activity); err != nil {
		h.logger.Error("Failed to record activity", map[string]interface{}{"error": err.Error()})
	}
}
