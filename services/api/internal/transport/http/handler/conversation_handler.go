package handler

import (
	agentdom "github.com/Paca-AI/api/internal/domain/agent"
	"github.com/Paca-AI/api/internal/transport/http/dto"
	"github.com/Paca-AI/api/internal/transport/http/middleware"
	"github.com/Paca-AI/api/internal/transport/http/presenter"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ConversationHandler handles agent conversation endpoints.
type ConversationHandler struct {
	svc agentdom.Service
}

// NewConversationHandler returns a ConversationHandler wired to the agent service.
func NewConversationHandler(svc agentdom.Service) *ConversationHandler {
	return &ConversationHandler{svc: svc}
}

// ListConversations handles GET /projects/:projectId/conversations.
func (h *ConversationHandler) ListConversations(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	offset, limit := parseOffsetLimit(c)
	filter := agentdom.ListConversationsFilter{
		ProjectID: &projectID,
		Limit:     limit,
		Offset:    offset,
	}
	if agentIDStr := c.Query("agent_id"); agentIDStr != "" {
		if id, err := uuid.Parse(agentIDStr); err == nil {
			filter.AgentID = &id
		}
	}
	if statusStr := c.Query("status"); statusStr != "" {
		filter.Status = &statusStr
	}

	convs, total, err := h.svc.ListConversations(c.Request.Context(), filter)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.AgentConversationResponse, 0, len(convs))
	for _, conv := range convs {
		resp = append(resp, dto.ConversationFromEntity(conv))
	}
	presenter.OK(c, gin.H{"items": resp, "total": total})
}

// GetConversation handles GET /projects/:projectId/conversations/:conversationId.
func (h *ConversationHandler) GetConversation(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	convID, err := parseParamUUID(c, "conversationId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	conv, err := h.svc.GetConversation(c.Request.Context(), projectID, convID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.ConversationFromEntity(conv))
}

// ListConversationEvents handles GET /projects/:projectId/conversations/:conversationId/events.
func (h *ConversationHandler) ListConversationEvents(c *gin.Context) {
	convID, err := parseParamUUID(c, "conversationId")
	if err != nil {
		presenter.Error(c, err)
		return
	}

	offset, limit := parseOffsetLimit(c)
	events, total, err := h.svc.ListConversationEvents(c.Request.Context(), convID, offset, limit)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.AgentConversationEventResponse, 0, len(events))
	for _, e := range events {
		resp = append(resp, dto.ConversationEventFromEntity(e))
	}
	presenter.OK(c, gin.H{"items": resp, "total": total})
}

// PauseConversation handles POST /projects/:projectId/conversations/:conversationId/pause.
func (h *ConversationHandler) PauseConversation(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	convID, err := parseParamUUID(c, "conversationId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.PauseConversation(c.Request.Context(), projectID, convID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, gin.H{"message": "conversation paused"})
}

// ResumeConversation handles POST /projects/:projectId/conversations/:conversationId/resume.
func (h *ConversationHandler) ResumeConversation(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	convID, err := parseParamUUID(c, "conversationId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.ResumeConversation(c.Request.Context(), projectID, convID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, gin.H{"message": "conversation resumed"})
}

// StopConversation handles POST /projects/:projectId/conversations/:conversationId/stop.
func (h *ConversationHandler) StopConversation(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	convID, err := parseParamUUID(c, "conversationId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.StopConversation(c.Request.Context(), projectID, convID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, gin.H{"message": "conversation stopped"})
}

// SendConversationMessage handles POST /projects/:projectId/conversations/:conversationId/messages.
func (h *ConversationHandler) SendConversationMessage(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	convID, err := parseParamUUID(c, "conversationId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var req dto.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, err)
		return
	}
	claims := middleware.ClaimsFrom(c)
	memberID, _ := uuid.Parse(claims.Subject)

	if err := h.svc.SendConversationMessage(c.Request.Context(), projectID, convID, req.Message, memberID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, gin.H{"message": "message sent"})
}
