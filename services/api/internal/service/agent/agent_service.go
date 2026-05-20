// Package agentsvc implements the AI Agent application service.
package agentsvc

import (
	"context"
	"strings"
	"time"

	agentdom "github.com/Paca-AI/api/internal/domain/agent"
	"github.com/Paca-AI/api/internal/events"
	"github.com/Paca-AI/api/internal/platform/messaging"
	"github.com/google/uuid"
)

// projectMemberWriter is the minimal interface this service needs to insert
// an agent as a project member and remove it on deletion.
type projectMemberWriter interface {
	AddAgentMember(ctx context.Context, memberID, projectID, agentID, roleID uuid.UUID) error
	RemoveAgentMember(ctx context.Context, projectID, agentID uuid.UUID) error
}

// Service is the concrete AI Agent service.
type Service struct {
	repo      agentdom.Repository
	projRepo  projectMemberWriter
	publisher *messaging.Publisher
}

// New returns a configured agent service.
func New(repo agentdom.Repository, projRepo projectMemberWriter, publisher *messaging.Publisher) *Service {
	return &Service{repo: repo, projRepo: projRepo, publisher: publisher}
}

// -------------------------------------------------------------------------
// Agents
// -------------------------------------------------------------------------

func (s *Service) ListAgents(ctx context.Context, projectID uuid.UUID) ([]*agentdom.Agent, error) {
	return s.repo.ListAgents(ctx, projectID)
}

func (s *Service) GetAgent(ctx context.Context, projectID, agentID uuid.UUID) (*agentdom.Agent, error) {
	a, err := s.repo.FindAgentByID(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if a.ProjectID != projectID {
		return nil, agentdom.ErrAgentNotFound
	}
	return a, nil
}

func (s *Service) CreateAgent(ctx context.Context, projectID uuid.UUID, in agentdom.CreateAgentInput) (*agentdom.Agent, error) {
	handle := strings.TrimSpace(in.Handle)
	if handle == "" {
		return nil, agentdom.ErrAgentHandleInvalid
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, agentdom.ErrAgentNameInvalid
	}

	// Check handle uniqueness
	if existing, err := s.repo.FindAgentByHandle(ctx, projectID, handle); err == nil && existing != nil {
		return nil, agentdom.ErrAgentHandleTaken
	}

	now := time.Now()
	a := &agentdom.Agent{
		ID:              uuid.New(),
		ProjectID:       projectID,
		Name:            name,
		Handle:          handle,
		LLMProvider:     in.LLMProvider,
		LLMModel:        in.LLMModel,
		LLMAPIKeySecret: in.LLMAPIKey, // stored directly; encryption handled at transport layer
		LLMBaseURL:      in.LLMBaseURL,
		SystemPrompt:    in.SystemPrompt,
		CanCloneRepos:   in.CanCloneRepos,
		CanCreatePRs:    in.CanCreatePRs,
		MaxIterations:   in.MaxIterations,
		TimeoutMinutes:  in.TimeoutMinutes,
		CreatedBy:       in.CreatedBy,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if a.MaxIterations == 0 {
		a.MaxIterations = 50
	}
	if a.TimeoutMinutes == 0 {
		a.TimeoutMinutes = 30
	}

	if err := s.repo.CreateAgent(ctx, a); err != nil {
		return nil, err
	}

	// Add the agent as a project member
	memberID := uuid.New()
	if err := s.projRepo.AddAgentMember(ctx, memberID, projectID, a.ID, in.ProjectRoleID); err != nil {
		// Non-fatal: agent was created, membership might already exist
		_ = err
	} else {
		a.MemberID = &memberID
	}

	return a, nil
}

func (s *Service) UpdateAgent(ctx context.Context, projectID, agentID uuid.UUID, in agentdom.UpdateAgentInput) (*agentdom.Agent, error) {
	a, err := s.GetAgent(ctx, projectID, agentID)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		a.Name = strings.TrimSpace(*in.Name)
	}
	if in.Handle != nil {
		h := strings.TrimSpace(*in.Handle)
		if h != a.Handle {
			if existing, err := s.repo.FindAgentByHandle(ctx, projectID, h); err == nil && existing != nil {
				return nil, agentdom.ErrAgentHandleTaken
			}
			a.Handle = h
		}
	}
	if in.LLMProvider != nil {
		a.LLMProvider = *in.LLMProvider
	}
	if in.LLMModel != nil {
		a.LLMModel = *in.LLMModel
	}
	if in.LLMAPIKey != nil {
		a.LLMAPIKeySecret = *in.LLMAPIKey
	}
	if in.LLMBaseURL != nil {
		a.LLMBaseURL = in.LLMBaseURL
	}
	if in.SystemPrompt != nil {
		a.SystemPrompt = *in.SystemPrompt
	}
	if in.CanCloneRepos != nil {
		a.CanCloneRepos = *in.CanCloneRepos
	}
	if in.CanCreatePRs != nil {
		a.CanCreatePRs = *in.CanCreatePRs
	}
	if in.MaxIterations != nil {
		a.MaxIterations = *in.MaxIterations
	}
	if in.TimeoutMinutes != nil {
		a.TimeoutMinutes = *in.TimeoutMinutes
	}
	a.UpdatedAt = time.Now()

	if err := s.repo.UpdateAgent(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Service) DeleteAgent(ctx context.Context, projectID, agentID uuid.UUID) error {
	a, err := s.GetAgent(ctx, projectID, agentID)
	if err != nil {
		return err
	}
	if err := s.repo.SoftDeleteAgent(ctx, a.ID); err != nil {
		return err
	}
	// Remove from project_members
	_ = s.projRepo.RemoveAgentMember(ctx, projectID, agentID)
	return nil
}

// -------------------------------------------------------------------------
// MCP Servers
// -------------------------------------------------------------------------

func (s *Service) ListMCPServers(ctx context.Context, agentID uuid.UUID) ([]*agentdom.AgentMCPServer, error) {
	return s.repo.ListMCPServers(ctx, agentID)
}

func (s *Service) AddMCPServer(ctx context.Context, agentID uuid.UUID, in agentdom.AddMCPServerInput) (*agentdom.AgentMCPServer, error) {
	now := time.Now()
	srv := &agentdom.AgentMCPServer{
		ID:         uuid.New(),
		AgentID:    agentID,
		ServerName: strings.TrimSpace(in.ServerName),
		Transport:  in.Transport,
		Command:    in.Command,
		Args:       in.Args,
		URL:        in.URL,
		Env:        in.Env,
		IsEnabled:  true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if srv.Args == nil {
		srv.Args = []string{}
	}
	if srv.Env == nil {
		srv.Env = map[string]string{}
	}
	if err := s.repo.CreateMCPServer(ctx, srv); err != nil {
		return nil, err
	}
	return srv, nil
}

func (s *Service) UpdateMCPServer(ctx context.Context, agentID, serverID uuid.UUID, in agentdom.UpdateMCPServerInput) (*agentdom.AgentMCPServer, error) {
	srv, err := s.repo.FindMCPServerByID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if srv.AgentID != agentID {
		return nil, agentdom.ErrMCPServerNotFound
	}
	if in.Command != nil {
		srv.Command = in.Command
	}
	if in.Args != nil {
		srv.Args = in.Args
	}
	if in.URL != nil {
		srv.URL = in.URL
	}
	if in.Env != nil {
		srv.Env = in.Env
	}
	if in.IsEnabled != nil {
		srv.IsEnabled = *in.IsEnabled
	}
	srv.UpdatedAt = time.Now()
	if err := s.repo.UpdateMCPServer(ctx, srv); err != nil {
		return nil, err
	}
	return srv, nil
}

func (s *Service) DeleteMCPServer(ctx context.Context, agentID, serverID uuid.UUID) error {
	srv, err := s.repo.FindMCPServerByID(ctx, serverID)
	if err != nil {
		return err
	}
	if srv.AgentID != agentID {
		return agentdom.ErrMCPServerNotFound
	}
	return s.repo.DeleteMCPServer(ctx, serverID)
}

// -------------------------------------------------------------------------
// Skills
// -------------------------------------------------------------------------

func (s *Service) ListSkills(ctx context.Context, agentID uuid.UUID) ([]*agentdom.AgentSkill, error) {
	return s.repo.ListSkills(ctx, agentID)
}

func (s *Service) AddSkill(ctx context.Context, agentID uuid.UUID, in agentdom.AddSkillInput) (*agentdom.AgentSkill, error) {
	now := time.Now()
	skill := &agentdom.AgentSkill{
		ID:           uuid.New(),
		AgentID:      agentID,
		SkillName:    strings.TrimSpace(in.SkillName),
		SkillSource:  in.SkillSource,
		SkillContent: in.SkillContent,
		SourceURL:    in.SourceURL,
		Triggers:     in.Triggers,
		IsEnabled:    true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if skill.Triggers == nil {
		skill.Triggers = []string{}
	}
	if err := s.repo.CreateSkill(ctx, skill); err != nil {
		return nil, err
	}
	return skill, nil
}

func (s *Service) UpdateSkill(ctx context.Context, agentID, skillID uuid.UUID, in agentdom.UpdateSkillInput) (*agentdom.AgentSkill, error) {
	skill, err := s.repo.FindSkillByID(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if skill.AgentID != agentID {
		return nil, agentdom.ErrSkillNotFound
	}
	if in.SkillContent != nil {
		skill.SkillContent = *in.SkillContent
	}
	if in.Triggers != nil {
		skill.Triggers = in.Triggers
	}
	if in.IsEnabled != nil {
		skill.IsEnabled = *in.IsEnabled
	}
	skill.UpdatedAt = time.Now()
	if err := s.repo.UpdateSkill(ctx, skill); err != nil {
		return nil, err
	}
	return skill, nil
}

func (s *Service) DeleteSkill(ctx context.Context, agentID, skillID uuid.UUID) error {
	skill, err := s.repo.FindSkillByID(ctx, skillID)
	if err != nil {
		return err
	}
	if skill.AgentID != agentID {
		return agentdom.ErrSkillNotFound
	}
	return s.repo.DeleteSkill(ctx, skillID)
}

// -------------------------------------------------------------------------
// Conversations
// -------------------------------------------------------------------------

func (s *Service) ListConversations(ctx context.Context, in agentdom.ListConversationsFilter) ([]*agentdom.AgentConversation, int64, error) {
	return s.repo.ListConversations(ctx, in)
}

func (s *Service) GetConversation(ctx context.Context, projectID, conversationID uuid.UUID) (*agentdom.AgentConversation, error) {
	c, err := s.repo.FindConversationByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if c.ProjectID != projectID {
		return nil, agentdom.ErrConversationNotFound
	}
	return c, nil
}

func (s *Service) ListConversationEvents(ctx context.Context, conversationID uuid.UUID, offset, limit int) ([]*agentdom.AgentConversationEvent, int64, error) {
	return s.repo.ListConversationEvents(ctx, conversationID, offset, limit)
}

func (s *Service) PauseConversation(ctx context.Context, projectID, conversationID uuid.UUID) error {
	c, err := s.GetConversation(ctx, projectID, conversationID)
	if err != nil {
		return err
	}
	if c.Status != "running" {
		return agentdom.ErrConversationNotRunning
	}
	if err := s.repo.UpdateConversationStatus(ctx, conversationID, "paused"); err != nil {
		return err
	}
	return s.publishTrigger(ctx, events.TopicAgentPause, map[string]any{
		"conversation_id": conversationID.String(),
		"project_id":      projectID.String(),
	})
}

func (s *Service) ResumeConversation(ctx context.Context, projectID, conversationID uuid.UUID) error {
	c, err := s.GetConversation(ctx, projectID, conversationID)
	if err != nil {
		return err
	}
	if c.Status != "paused" {
		return agentdom.ErrConversationNotRunning
	}
	if err := s.repo.UpdateConversationStatus(ctx, conversationID, "running"); err != nil {
		return err
	}
	return s.publishTrigger(ctx, events.TopicAgentResume, map[string]any{
		"conversation_id": conversationID.String(),
		"project_id":      projectID.String(),
	})
}

func (s *Service) StopConversation(ctx context.Context, projectID, conversationID uuid.UUID) error {
	c, err := s.GetConversation(ctx, projectID, conversationID)
	if err != nil {
		return err
	}
	if c.Status == "finished" || c.Status == "stopped" || c.Status == "failed" {
		return agentdom.ErrConversationAlreadyStopped
	}
	if err := s.repo.UpdateConversationStatus(ctx, conversationID, "stopped"); err != nil {
		return err
	}
	return s.publishTrigger(ctx, events.TopicAgentStop, map[string]any{
		"conversation_id": conversationID.String(),
		"project_id":      projectID.String(),
	})
}

func (s *Service) SendConversationMessage(ctx context.Context, projectID, conversationID uuid.UUID, message string, memberID uuid.UUID) error {
	c, err := s.GetConversation(ctx, projectID, conversationID)
	if err != nil {
		return err
	}
	if c.Status != "running" && c.Status != "paused" {
		return agentdom.ErrConversationNotRunning
	}
	return s.publishTrigger(ctx, events.TopicAgentChatMessage, map[string]any{
		"conversation_id": conversationID.String(),
		"project_id":      projectID.String(),
		"message":         message,
		"member_id":       memberID.String(),
	})
}

// -------------------------------------------------------------------------
// Chat Sessions
// -------------------------------------------------------------------------

func (s *Service) ListChatSessions(ctx context.Context, projectID, agentID, memberID uuid.UUID) ([]*agentdom.AgentChatSession, error) {
	return s.repo.ListChatSessions(ctx, agentID, memberID)
}

func (s *Service) StartChatSession(ctx context.Context, projectID, agentID, memberID uuid.UUID, message string) (*agentdom.AgentChatSession, *agentdom.AgentConversation, error) {
	now := time.Now()

	session := &agentdom.AgentChatSession{
		ID:            uuid.New(),
		AgentID:       agentID,
		ProjectID:     projectID,
		MemberID:      memberID,
		LastMessageAt: &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.repo.CreateChatSession(ctx, session); err != nil {
		return nil, nil, err
	}

	conv, err := s.createConversation(ctx, projectID, agentID, memberID, agentdom.AgentConversation{
		TriggerType:   "chat_message",
		ChatSessionID: &session.ID,
	})
	if err != nil {
		return nil, nil, err
	}

	if err := s.publishChatTrigger(ctx, agentID, conv.ID, session.ID, projectID, memberID, message); err != nil {
		return nil, nil, err
	}

	return session, conv, nil
}

func (s *Service) SendChatMessage(ctx context.Context, projectID, sessionID, memberID uuid.UUID, message string) (*agentdom.AgentConversation, error) {
	session, err := s.repo.FindChatSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.ProjectID != projectID {
		return nil, agentdom.ErrChatSessionNotFound
	}

	conv, err := s.createConversation(ctx, projectID, session.AgentID, memberID, agentdom.AgentConversation{
		TriggerType:   "chat_message",
		ChatSessionID: &sessionID,
	})
	if err != nil {
		return nil, err
	}

	if err := s.publishChatTrigger(ctx, session.AgentID, conv.ID, sessionID, projectID, memberID, message); err != nil {
		return nil, err
	}

	// Update last_message_at
	now := time.Now()
	session.LastMessageAt = &now
	_ = s.repo.UpdateChatSession(ctx, session)

	return conv, nil
}

func (s *Service) ListChatMessages(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*agentdom.AgentConversationEvent, int64, error) {
	// TODO: We'd need to aggregate events from all conversations in this session.
	// For now, return events from the most recent conversation with this session_id.
	filter := agentdom.ListConversationsFilter{
		Limit:  1,
		Offset: 0,
	}
	_ = sessionID
	convs, _, err := s.repo.ListConversations(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if len(convs) == 0 {
		return []*agentdom.AgentConversationEvent{}, 0, nil
	}
	return s.repo.ListConversationEvents(ctx, convs[0].ID, offset, limit)
}

// -------------------------------------------------------------------------
// Internal helpers
// -------------------------------------------------------------------------

func (s *Service) createConversation(ctx context.Context, projectID, agentID, memberID uuid.UUID, template agentdom.AgentConversation) (*agentdom.AgentConversation, error) {
	now := time.Now()
	conv := &agentdom.AgentConversation{
		ID:                  uuid.New(),
		AgentID:             agentID,
		ProjectID:           projectID,
		TriggerType:         template.TriggerType,
		TaskID:              template.TaskID,
		CommentID:           template.CommentID,
		ChatSessionID:       template.ChatSessionID,
		TriggeredByMemberID: memberID,
		Status:              "queued",
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := s.repo.CreateConversation(ctx, conv); err != nil {
		return nil, err
	}
	return conv, nil
}

// TriggerTaskAssigned creates a conversation and publishes the trigger event
// when a task is assigned to an agent member.
func (s *Service) TriggerTaskAssigned(ctx context.Context, projectID, agentID, taskID, triggeredByMemberID uuid.UUID) (*agentdom.AgentConversation, error) {
	conv, err := s.createConversation(ctx, projectID, agentID, triggeredByMemberID, agentdom.AgentConversation{
		TriggerType: "task_assigned",
		TaskID:      &taskID,
	})
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"conversation_id": conv.ID.String(),
		"project_id":      projectID.String(),
		"agent_id":        agentID.String(),
		"task_id":         taskID.String(),
		"actor_member_id": triggeredByMemberID.String(),
		"trigger_type":    "task_assigned",
	}
	_ = s.publishTrigger(ctx, events.TopicAgentTaskAssigned, payload)
	return conv, nil
}

// TriggerCommentMention creates a conversation and publishes a comment-mention trigger.
func (s *Service) TriggerCommentMention(ctx context.Context, projectID, agentID, taskID, commentID, triggeredByMemberID uuid.UUID) (*agentdom.AgentConversation, error) {
	conv, err := s.createConversation(ctx, projectID, agentID, triggeredByMemberID, agentdom.AgentConversation{
		TriggerType: "comment_mention",
		TaskID:      &taskID,
		CommentID:   &commentID,
	})
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"conversation_id": conv.ID.String(),
		"project_id":      projectID.String(),
		"agent_id":        agentID.String(),
		"task_id":         taskID.String(),
		"comment_id":      commentID.String(),
		"actor_member_id": triggeredByMemberID.String(),
		"trigger_type":    "comment_mention",
	}
	_ = s.publishTrigger(ctx, events.TopicAgentCommentMention, payload)
	return conv, nil
}

func (s *Service) publishTrigger(ctx context.Context, topic string, payload map[string]any) error {
	if s.publisher == nil {
		return nil
	}
	// Write flat fields so services/ai-agent can read them without JSON decoding.
	// The trigger_type is embedded in the payload; the stream entry type field
	// mirrors it for routing convenience.
	payload["type"] = topic
	return s.publisher.AppendFlat(ctx, events.StreamAgentTriggers, payload)
}

func (s *Service) publishChatTrigger(ctx context.Context, agentID, convID, sessionID, projectID, memberID uuid.UUID, message string) error {
	payload := map[string]any{
		"conversation_id": convID.String(),
		"project_id":      projectID.String(),
		"agent_id":        agentID.String(),
		"chat_session_id": sessionID.String(),
		"actor_member_id": memberID.String(),
		"trigger_type":    "chat_message",
		"message":         message,
	}
	return s.publishTrigger(ctx, events.TopicAgentChatMessage, payload)
}
