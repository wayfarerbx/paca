package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/Paca-AI/api/internal/apierr"
	agentdom "github.com/Paca-AI/api/internal/domain/agent"
	agentsvc "github.com/Paca-AI/api/internal/service/agent"
	"github.com/Paca-AI/api/internal/transport/http/dto"
	"github.com/Paca-AI/api/internal/transport/http/middleware"
	"github.com/Paca-AI/api/internal/transport/http/presenter"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AgentHandler handles AI agent management endpoints.
type AgentHandler struct {
	svc        agentdom.Service
	aiAgentURL string
	httpClient *http.Client
}

// NewAgentHandler returns an AgentHandler wired to the agent service.
func NewAgentHandler(svc agentdom.Service, aiAgentURL string) *AgentHandler {
	return &AgentHandler{
		svc:        svc,
		aiAgentURL: aiAgentURL,
		httpClient: &http.Client{},
	}
}

// --- Agents -----------------------------------------------------------------

// ListAgents handles GET /projects/:projectId/agents.
func (h *AgentHandler) ListAgents(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	agents, err := h.svc.ListAgents(c.Request.Context(), projectID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.AgentResponse, 0, len(agents))
	for _, a := range agents {
		resp = append(resp, dto.AgentFromEntity(a))
	}
	presenter.OK(c, gin.H{"items": resp})
}

// GetAgent handles GET /projects/:projectId/agents/:agentId.
func (h *AgentHandler) GetAgent(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	agentID, err := parseParamUUID(c, "agentId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	a, err := h.svc.GetAgent(c.Request.Context(), projectID, agentID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.AgentFromEntity(a))
}

// CreateAgent handles POST /projects/:projectId/agents.
func (h *AgentHandler) CreateAgent(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var req dto.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, err)
		return
	}
	claims := middleware.ClaimsFrom(c)
	callerID, _ := uuid.Parse(claims.Subject)

	a, err := h.svc.CreateAgent(c.Request.Context(), projectID, agentdom.CreateAgentInput{
		Name:              req.Name,
		Handle:            req.Handle,
		LLMProvider:       req.LLMProvider,
		LLMModel:          req.LLMModel,
		LLMAPIKey:         req.LLMAPIKey,
		LLMBaseURL:        req.LLMBaseURL,
		SystemPrompt:      req.SystemPrompt,
		CanCloneRepos:     req.CanCloneRepos,
		CanCreatePRs:      req.CanCreatePRs,
		MaxIterations:     req.MaxIterations,
		TimeoutMinutes:    req.TimeoutMinutes,
		GitCommitterName:  req.GitCommitterName,
		GitCommitterEmail: req.GitCommitterEmail,
		ProjectRoleID:     req.ProjectRoleID,
		CreatedBy:         &callerID,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, dto.AgentFromEntity(a))
}

// UpdateAgent handles PATCH /projects/:projectId/agents/:agentId.
func (h *AgentHandler) UpdateAgent(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	agentID, err := parseParamUUID(c, "agentId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var req dto.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, err)
		return
	}
	a, err := h.svc.UpdateAgent(c.Request.Context(), projectID, agentID, agentdom.UpdateAgentInput{
		Name:              req.Name,
		Handle:            req.Handle,
		LLMProvider:       req.LLMProvider,
		LLMModel:          req.LLMModel,
		LLMAPIKey:         req.LLMAPIKey,
		LLMBaseURL:        req.LLMBaseURL,
		SystemPrompt:      req.SystemPrompt,
		CanCloneRepos:     req.CanCloneRepos,
		CanCreatePRs:      req.CanCreatePRs,
		MaxIterations:     req.MaxIterations,
		TimeoutMinutes:    req.TimeoutMinutes,
		GitCommitterName:  req.GitCommitterName,
		GitCommitterEmail: req.GitCommitterEmail,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.AgentFromEntity(a))
}

// DeleteAgent handles DELETE /projects/:projectId/agents/:agentId.
func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	agentID, err := parseParamUUID(c, "agentId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.DeleteAgent(c.Request.Context(), projectID, agentID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, gin.H{"message": "agent deleted"})
}

// --- MCP Servers ------------------------------------------------------------

// ListMCPServers handles GET /projects/:projectId/agents/:agentId/mcp-servers.
func (h *AgentHandler) ListMCPServers(c *gin.Context) {
	agentID, err := parseParamUUID(c, "agentId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	servers, err := h.svc.ListMCPServers(c.Request.Context(), agentID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.AgentMCPServerResponse, 0, len(servers))
	for _, s := range servers {
		resp = append(resp, dto.MCPServerFromEntity(s))
	}
	presenter.OK(c, gin.H{"items": resp})
}

// AddMCPServer handles POST /projects/:projectId/agents/:agentId/mcp-servers.
func (h *AgentHandler) AddMCPServer(c *gin.Context) {
	agentID, err := parseParamUUID(c, "agentId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var req dto.AddMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, err)
		return
	}
	srv, err := h.svc.AddMCPServer(c.Request.Context(), agentID, agentdom.AddMCPServerInput{
		ServerName: req.ServerName,
		Transport:  req.Transport,
		Command:    req.Command,
		Args:       req.Args,
		URL:        req.URL,
		Env:        req.Env,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, dto.MCPServerFromEntity(srv))
}

// UpdateMCPServer handles PATCH /projects/:projectId/agents/:agentId/mcp-servers/:serverId.
func (h *AgentHandler) UpdateMCPServer(c *gin.Context) {
	agentID, err := parseParamUUID(c, "agentId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	serverID, err := parseParamUUID(c, "serverId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var req dto.UpdateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, err)
		return
	}
	srv, err := h.svc.UpdateMCPServer(c.Request.Context(), agentID, serverID, agentdom.UpdateMCPServerInput{
		Command:   req.Command,
		Args:      req.Args,
		URL:       req.URL,
		Env:       req.Env,
		IsEnabled: req.IsEnabled,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.MCPServerFromEntity(srv))
}

// DeleteMCPServer handles DELETE /projects/:projectId/agents/:agentId/mcp-servers/:serverId.
func (h *AgentHandler) DeleteMCPServer(c *gin.Context) {
	agentID, err := parseParamUUID(c, "agentId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	serverID, err := parseParamUUID(c, "serverId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.DeleteMCPServer(c.Request.Context(), agentID, serverID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, gin.H{"message": "mcp server deleted"})
}

// --- Skills -----------------------------------------------------------------

// ListSkills handles GET /projects/:projectId/agents/:agentId/skills.
func (h *AgentHandler) ListSkills(c *gin.Context) {
	agentID, err := parseParamUUID(c, "agentId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	skills, err := h.svc.ListSkills(c.Request.Context(), agentID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.AgentSkillResponse, 0, len(skills))
	for _, s := range skills {
		resp = append(resp, dto.SkillFromEntity(s))
	}
	presenter.OK(c, gin.H{"items": resp})
}

// AddSkill handles POST /projects/:projectId/agents/:agentId/skills.
func (h *AgentHandler) AddSkill(c *gin.Context) {
	agentID, err := parseParamUUID(c, "agentId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var req dto.AddSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, err)
		return
	}
	skill, err := h.svc.AddSkill(c.Request.Context(), agentID, agentdom.AddSkillInput{
		SkillName:    req.SkillName,
		SkillSource:  req.SkillSource,
		SkillContent: req.SkillContent,
		SourceURL:    req.SourceURL,
		Triggers:     req.Triggers,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, dto.SkillFromEntity(skill))
}

// UpdateSkill handles PATCH /projects/:projectId/agents/:agentId/skills/:skillId.
func (h *AgentHandler) UpdateSkill(c *gin.Context) {
	agentID, err := parseParamUUID(c, "agentId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	skillID, err := parseParamUUID(c, "skillId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var req dto.UpdateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, err)
		return
	}
	skill, err := h.svc.UpdateSkill(c.Request.Context(), agentID, skillID, agentdom.UpdateSkillInput{
		SkillContent: req.SkillContent,
		Triggers:     req.Triggers,
		IsEnabled:    req.IsEnabled,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.SkillFromEntity(skill))
}

// DeleteSkill handles DELETE /projects/:projectId/agents/:agentId/skills/:skillId.
func (h *AgentHandler) DeleteSkill(c *gin.Context) {
	agentID, err := parseParamUUID(c, "agentId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	skillID, err := parseParamUUID(c, "skillId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.DeleteSkill(c.Request.Context(), agentID, skillID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, gin.H{"message": "skill deleted"})
}

// --- Chat Sessions ----------------------------------------------------------

// ListChatSessions handles GET /projects/:projectId/agents/:agentId/chat-sessions.
func (h *AgentHandler) ListChatSessions(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	agentID, err := parseParamUUID(c, "agentId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	claims := middleware.ClaimsFrom(c)
	memberID, _ := uuid.Parse(claims.Subject)

	sessions, err := h.svc.ListChatSessions(c.Request.Context(), projectID, agentID, memberID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.AgentChatSessionResponse, 0, len(sessions))
	for _, s := range sessions {
		resp = append(resp, dto.ChatSessionFromEntity(s))
	}
	presenter.OK(c, gin.H{"items": resp})
}

// StartChatSession handles POST /projects/:projectId/agents/:agentId/chat-sessions.
func (h *AgentHandler) StartChatSession(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	agentID, err := parseParamUUID(c, "agentId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var req dto.StartChatSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, err)
		return
	}
	claims := middleware.ClaimsFrom(c)
	memberID, _ := uuid.Parse(claims.Subject)

	session, conv, err := h.svc.StartChatSession(c.Request.Context(), projectID, agentID, memberID, req.Message)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, gin.H{
		"session":      dto.ChatSessionFromEntity(session),
		"conversation": dto.ConversationFromEntity(conv),
	})
}

// SendChatMessage handles POST /projects/:projectId/agents/:agentId/chat-sessions/:sessionId/messages.
func (h *AgentHandler) SendChatMessage(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	sessionID, err := parseParamUUID(c, "sessionId")
	if err != nil {
		presenter.Error(c, err)
		return
	}
	var req dto.SendChatMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.Error(c, err)
		return
	}
	claims := middleware.ClaimsFrom(c)
	memberID, _ := uuid.Parse(claims.Subject)

	conv, err := h.svc.SendChatMessage(c.Request.Context(), projectID, sessionID, memberID, req.Message)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, gin.H{"conversation": dto.ConversationFromEntity(conv)})
}

// --- helpers ----------------------------------------------------------------

func parseParamUUID(c *gin.Context, param string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		return uuid.Nil, apierr.New(apierr.CodeBadRequest, fmt.Sprintf("invalid %s", param))
	}
	return id, nil
}

func parseOffsetLimit(c *gin.Context) (offset, limit int) {
	offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", "50"))
	if offset < 0 {
		offset = 0
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	return offset, limit
}

// --- Skill templates --------------------------------------------------------

// ListSkillTemplates handles GET /agents/skill-templates.
// Returns the hardcoded built-in skill template catalog.
func (h *AgentHandler) ListSkillTemplates(c *gin.Context) {
	templates := agentsvc.ListSkillTemplates()
	resp := make([]dto.SkillTemplateResponse, 0, len(templates))
	for _, t := range templates {
		resp = append(resp, dto.SkillTemplateFromEntity(t))
	}
	presenter.OK(c, resp)
}

// --- LLM models proxy -------------------------------------------------------

// GetLLMModels handles GET /agents/llm-models.
// It proxies the request to the ai-agent service and returns the verified
// LLM models grouped by provider.
func (h *AgentHandler) GetLLMModels(c *gin.Context) {
	if h.aiAgentURL == "" {
		presenter.Error(c, apierr.New(apierr.CodeInternalError, "ai-agent service URL not configured"))
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, h.aiAgentURL+"/llm/models", nil)
	if err != nil {
		presenter.Error(c, apierr.New(apierr.CodeInternalError, "failed to create request"))
		return
	}
	resp, err := h.httpClient.Do(req)
	if err != nil {
		presenter.Error(c, apierr.New(apierr.CodeInternalError, "failed to reach ai-agent service"))
		return
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		presenter.Error(c, apierr.New(apierr.CodeInternalError, "failed to read ai-agent response"))
		return
	}

	if resp.StatusCode != http.StatusOK {
		presenter.Error(c, apierr.New(apierr.CodeInternalError, "ai-agent service returned an error"))
		return
	}

	c.Data(http.StatusOK, "application/json", body)
}
