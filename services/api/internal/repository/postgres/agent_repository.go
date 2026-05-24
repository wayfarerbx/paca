package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	agentdom "github.com/Paca-AI/api/internal/domain/agent"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// -------------------------------------------------------------------------
// GORM record types
// -------------------------------------------------------------------------

type agentRecord struct {
	ID              string `gorm:"primarykey;type:uuid"`
	ProjectID       string `gorm:"type:uuid;not null;column:project_id"`
	Name            string
	Handle          string
	AvatarURL       *string `gorm:"column:avatar_url"`
	LLMProvider     string  `gorm:"column:llm_provider"`
	LLMModel        string  `gorm:"column:llm_model"`
	LLMAPIKeySecret string  `gorm:"column:llm_api_key_secret"`
	LLMBaseURL      *string `gorm:"column:llm_base_url"`
	SystemPrompt    string  `gorm:"column:system_prompt"`
	CanCloneRepos   bool    `gorm:"column:can_clone_repos"`
	CanCreatePRs    bool    `gorm:"column:can_create_prs"`
	MaxIterations   int     `gorm:"column:max_iterations"`
	TimeoutMinutes  int     `gorm:"column:timeout_minutes"`
	CreatedBy       *string `gorm:"type:uuid;column:created_by"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (agentRecord) TableName() string { return "agents" }

// agentReadRow is the result of the SELECT query.
type agentReadRow struct {
	ID              string
	ProjectID       string
	Name            string
	Handle          string
	AvatarURL       *string
	LLMProvider     string
	LLMModel        string
	LLMAPIKeySecret string
	LLMBaseURL      *string
	SystemPrompt    string
	CanCloneRepos   bool
	CanCreatePRs    bool
	MaxIterations   int
	TimeoutMinutes  int
	CreatedBy       *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt
	MemberID        *string // populated when joining with project_members
}

type agentMCPServerRecord struct {
	ID         string `gorm:"primarykey;type:uuid"`
	AgentID    string `gorm:"type:uuid;not null;column:agent_id"`
	ServerName string `gorm:"column:server_name"`
	Transport  string
	Command    *string
	Args       []byte `gorm:"type:jsonb"`
	URL        *string
	Env        []byte `gorm:"type:jsonb"`
	IsEnabled  bool   `gorm:"column:is_enabled"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (agentMCPServerRecord) TableName() string { return "agent_mcp_servers" }

type agentSkillRecord struct {
	ID           string  `gorm:"primarykey;type:uuid"`
	AgentID      string  `gorm:"type:uuid;not null;column:agent_id"`
	SkillName    string  `gorm:"column:skill_name"`
	SkillSource  string  `gorm:"column:skill_source"`
	SkillContent string  `gorm:"column:skill_content"`
	SourceURL    *string `gorm:"column:source_url"`
	Triggers     []byte  `gorm:"type:jsonb"`
	IsEnabled    bool    `gorm:"column:is_enabled"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (agentSkillRecord) TableName() string { return "agent_skills" }

type agentConversationRecord struct {
	ID                  string  `gorm:"primarykey;type:uuid"`
	AgentID             string  `gorm:"type:uuid;not null;column:agent_id"`
	ProjectID           string  `gorm:"type:uuid;not null;column:project_id"`
	TriggerType         string  `gorm:"column:trigger_type"`
	TaskID              *string `gorm:"type:uuid;column:task_id"`
	CommentID           *string `gorm:"type:uuid;column:comment_id"`
	ChatSessionID       *string `gorm:"type:uuid;column:chat_session_id"`
	TriggeredByMemberID string  `gorm:"type:uuid;not null;column:triggered_by_member_id"`
	Status              string
	ContainerID         *string    `gorm:"column:container_id"`
	HostPort            *int       `gorm:"column:host_port"`
	IterationCount      int        `gorm:"column:iteration_count"`
	ErrorMessage        *string    `gorm:"column:error_message"`
	RepoPluginID        *string    `gorm:"type:uuid;column:repo_plugin_id"`
	RepoCloneURL        *string    `gorm:"column:repo_clone_url"`
	BranchName          *string    `gorm:"column:branch_name"`
	PRUrl               *string    `gorm:"column:pr_url"`
	PersistenceDir      *string    `gorm:"column:persistence_dir"`
	StartedAt           *time.Time `gorm:"column:started_at"`
	FinishedAt          *time.Time `gorm:"column:finished_at"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (agentConversationRecord) TableName() string { return "agent_conversations" }

type agentConversationEventRecord struct {
	ID             string `gorm:"primarykey;type:uuid"`
	ConversationID string `gorm:"type:uuid;not null;column:conversation_id"`
	EventIndex     int    `gorm:"column:event_index"`
	EventType      string `gorm:"column:event_type"`
	EventSource    string `gorm:"column:event_source"`
	Payload        []byte `gorm:"type:jsonb"`
	CreatedAt      time.Time
}

func (agentConversationEventRecord) TableName() string { return "agent_conversation_events" }

type agentChatSessionRecord struct {
	ID            string `gorm:"primarykey;type:uuid"`
	AgentID       string `gorm:"type:uuid;not null;column:agent_id"`
	ProjectID     string `gorm:"type:uuid;not null;column:project_id"`
	MemberID      string `gorm:"type:uuid;not null;column:member_id"`
	Title         *string
	LastMessageAt *time.Time `gorm:"column:last_message_at"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (agentChatSessionRecord) TableName() string { return "agent_chat_sessions" }

// -------------------------------------------------------------------------
// Repository
// -------------------------------------------------------------------------

// AgentRepository is the GORM implementation of agentdom.Repository.
type AgentRepository struct {
	db *gorm.DB
}

// NewAgentRepository returns a new AgentRepository.
func NewAgentRepository(db *gorm.DB) *AgentRepository {
	return &AgentRepository{db: db}
}

// -------------------------------------------------------------------------
// Agent Types
// -------------------------------------------------------------------------

// -------------------------------------------------------------------------
// Agents
// -------------------------------------------------------------------------

// ListAgents returns all agents belonging to the given project.
func (r *AgentRepository) ListAgents(ctx context.Context, projectID uuid.UUID) ([]*agentdom.Agent, error) {
	rows, err := r.db.WithContext(ctx).
		Raw(`SELECT a.*,
                    pm.id AS member_id
             FROM agents a
             LEFT JOIN project_members pm ON pm.agent_id = a.id AND pm.deleted_at IS NULL
             WHERE a.project_id = ? AND a.deleted_at IS NULL`, projectID.String()).
		Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []*agentdom.Agent
	for rows.Next() {
		var row agentReadRow
		if err := r.db.ScanRows(rows, &row); err != nil {
			return nil, err
		}
		result = append(result, agentFromReadRow(row))
	}
	return result, nil
}

// FindAgentByID returns a single agent with its MCP servers and skills.
func (r *AgentRepository) FindAgentByID(ctx context.Context, id uuid.UUID) (*agentdom.Agent, error) {
	var row agentReadRow
	err := r.db.WithContext(ctx).
		Raw(`SELECT a.*,
                    pm.id AS member_id
             FROM agents a
             LEFT JOIN project_members pm ON pm.agent_id = a.id AND pm.deleted_at IS NULL
             WHERE a.id = ? AND a.deleted_at IS NULL`, id.String()).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == "" {
		return nil, agentdom.ErrAgentNotFound
	}
	agent := agentFromReadRow(row)
	// Load MCP servers and skills
	mcpServers, err := r.ListMCPServers(ctx, id)
	if err != nil {
		return nil, err
	}
	skills, err := r.ListSkills(ctx, id)
	if err != nil {
		return nil, err
	}
	agent.MCPServers = mcpServers
	agent.Skills = skills
	return agent, nil
}

// FindAgentByHandle returns an agent by its unique handle within a project.
func (r *AgentRepository) FindAgentByHandle(ctx context.Context, projectID uuid.UUID, handle string) (*agentdom.Agent, error) {
	var row agentReadRow
	err := r.db.WithContext(ctx).
		Raw(`SELECT a.*,
                    pm.id AS member_id
             FROM agents a
             LEFT JOIN project_members pm ON pm.agent_id = a.id AND pm.deleted_at IS NULL
             WHERE a.project_id = ? AND a.handle = ? AND a.deleted_at IS NULL`,
			projectID.String(), handle).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == "" {
		return nil, agentdom.ErrAgentNotFound
	}
	return agentFromReadRow(row), nil
}

// CreateAgent inserts a new agent record.
func (r *AgentRepository) CreateAgent(ctx context.Context, a *agentdom.Agent) error {
	rec := agentToRecord(a)
	return r.db.WithContext(ctx).Create(&rec).Error
}

// UpdateAgent patches the mutable fields of an existing agent.
func (r *AgentRepository) UpdateAgent(ctx context.Context, a *agentdom.Agent) error {
	updates := map[string]any{
		"name":            a.Name,
		"handle":          a.Handle,
		"avatar_url":      a.AvatarURL,
		"llm_provider":    a.LLMProvider,
		"llm_model":       a.LLMModel,
		"llm_base_url":    a.LLMBaseURL,
		"system_prompt":   a.SystemPrompt,
		"can_clone_repos": a.CanCloneRepos,
		"can_create_prs":  a.CanCreatePRs,
		"max_iterations":  a.MaxIterations,
		"timeout_minutes": a.TimeoutMinutes,
		"updated_at":      time.Now(),
	}
	if a.LLMAPIKeySecret != "" {
		updates["llm_api_key_secret"] = a.LLMAPIKeySecret
	}
	return r.db.WithContext(ctx).Model(&agentRecord{}).
		Where("id = ?", a.ID.String()).
		Updates(updates).Error
}

// SoftDeleteAgent sets deleted_at on the agent row.
func (r *AgentRepository) SoftDeleteAgent(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&agentRecord{}).
		Where("id = ?", id.String()).
		Update("deleted_at", time.Now()).Error
}

// SetAgentMemberID is a no-op; membership is derived from project_members JOIN.
func (r *AgentRepository) SetAgentMemberID(_ context.Context, _, _ uuid.UUID) error {
	// Member ID is derived from the project_members table by JOIN; no separate column needed.
	return nil
}

// CreateAgentWithMembership atomically inserts the agent and its project_members
// row within a single database transaction.
func (r *AgentRepository) CreateAgentWithMembership(ctx context.Context, a *agentdom.Agent, memberID, projectID, roleID uuid.UUID) error {
	return WithTx(ctx, r.db, func(tx *gorm.DB) error {
		if err := tx.Create(agentToRecord(a)).Error; err != nil {
			return err
		}
		return tx.Exec(`
			INSERT INTO project_members (id, project_id, agent_id, project_role_id, member_type, user_id, created_at, deleted_at)
			VALUES (?, ?, ?, ?, 'agent', NULL, NOW(), NULL)`,
			memberID.String(), projectID.String(), a.ID.String(), roleID.String(),
		).Error
	})
}

// SoftDeleteAgentWithMembership atomically soft-deletes both the agent and its
// project_members row within a single database transaction.
func (r *AgentRepository) SoftDeleteAgentWithMembership(ctx context.Context, projectID, agentID uuid.UUID) error {
	now := time.Now()
	return WithTx(ctx, r.db, func(tx *gorm.DB) error {
		if err := tx.Model(&agentRecord{}).
			Where("id = ?", agentID.String()).
			Update("deleted_at", now).Error; err != nil {
			return err
		}
		// Soft-delete the membership row; 0 rows affected is fine for orphaned agents.
		return tx.Model(&projectMemberRecord{}).
			Where("project_id = ? AND agent_id = ? AND member_type = 'agent'", projectID.String(), agentID.String()).
			Update("deleted_at", now).Error
	})
}

// -------------------------------------------------------------------------
// MCP Servers
// -------------------------------------------------------------------------

// ListMCPServers returns all MCP server records for the given agent.
func (r *AgentRepository) ListMCPServers(ctx context.Context, agentID uuid.UUID) ([]*agentdom.AgentMCPServer, error) {
	var recs []agentMCPServerRecord
	if err := r.db.WithContext(ctx).Where("agent_id = ?", agentID.String()).Find(&recs).Error; err != nil {
		return nil, err
	}
	result := make([]*agentdom.AgentMCPServer, 0, len(recs))
	for _, rec := range recs {
		s, err := mcpServerFromRecord(rec)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

// FindMCPServerByID returns a single MCP server by its primary key.
func (r *AgentRepository) FindMCPServerByID(ctx context.Context, id uuid.UUID) (*agentdom.AgentMCPServer, error) {
	var rec agentMCPServerRecord
	if err := r.db.WithContext(ctx).First(&rec, "id = ?", id.String()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, agentdom.ErrMCPServerNotFound
		}
		return nil, err
	}
	return mcpServerFromRecord(rec)
}

// CreateMCPServer inserts a new MCP server record.
func (r *AgentRepository) CreateMCPServer(ctx context.Context, s *agentdom.AgentMCPServer) error {
	rec, err := mcpServerToRecord(s)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(&rec).Error
}

// UpdateMCPServer saves the full MCP server record.
func (r *AgentRepository) UpdateMCPServer(ctx context.Context, s *agentdom.AgentMCPServer) error {
	rec, err := mcpServerToRecord(s)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(&rec).Error
}

// DeleteMCPServer permanently removes an MCP server record.
func (r *AgentRepository) DeleteMCPServer(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&agentMCPServerRecord{}, "id = ?", id.String()).Error
}

// -------------------------------------------------------------------------
// Skills
// -------------------------------------------------------------------------

// ListSkills returns all skill records for the given agent.
func (r *AgentRepository) ListSkills(ctx context.Context, agentID uuid.UUID) ([]*agentdom.AgentSkill, error) {
	var recs []agentSkillRecord
	if err := r.db.WithContext(ctx).Where("agent_id = ?", agentID.String()).Find(&recs).Error; err != nil {
		return nil, err
	}
	result := make([]*agentdom.AgentSkill, 0, len(recs))
	for _, rec := range recs {
		s, err := skillFromRecord(rec)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

// FindSkillByID returns a single skill by its primary key.
func (r *AgentRepository) FindSkillByID(ctx context.Context, id uuid.UUID) (*agentdom.AgentSkill, error) {
	var rec agentSkillRecord
	if err := r.db.WithContext(ctx).First(&rec, "id = ?", id.String()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, agentdom.ErrSkillNotFound
		}
		return nil, err
	}
	return skillFromRecord(rec)
}

// CreateSkill inserts a new skill record.
func (r *AgentRepository) CreateSkill(ctx context.Context, s *agentdom.AgentSkill) error {
	rec, err := skillToRecord(s)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(&rec).Error
}

// UpdateSkill saves the full skill record.
func (r *AgentRepository) UpdateSkill(ctx context.Context, s *agentdom.AgentSkill) error {
	rec, err := skillToRecord(s)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(&rec).Error
}

// DeleteSkill permanently removes a skill record.
func (r *AgentRepository) DeleteSkill(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&agentSkillRecord{}, "id = ?", id.String()).Error
}

// -------------------------------------------------------------------------
// Conversations
// -------------------------------------------------------------------------

// ListConversations returns a paginated list of conversations matching the filter.
func (r *AgentRepository) ListConversations(ctx context.Context, in agentdom.ListConversationsFilter) ([]*agentdom.AgentConversation, int64, error) {
	q := r.db.WithContext(ctx).Model(&agentConversationRecord{})
	if in.AgentID != nil {
		q = q.Where("agent_id = ?", in.AgentID.String())
	}
	if in.ProjectID != nil {
		q = q.Where("project_id = ?", in.ProjectID.String())
	}
	if in.TaskID != nil {
		q = q.Where("task_id = ?", in.TaskID.String())
	}
	if in.Status != nil {
		q = q.Where("status = ?", *in.Status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var recs []agentConversationRecord
	if err := q.Order("created_at DESC").Offset(in.Offset).Limit(in.Limit).Find(&recs).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*agentdom.AgentConversation, 0, len(recs))
	for _, rec := range recs {
		result = append(result, conversationFromRecord(rec))
	}
	return result, total, nil
}

// FindConversationByID returns a single conversation by its primary key.
func (r *AgentRepository) FindConversationByID(ctx context.Context, id uuid.UUID) (*agentdom.AgentConversation, error) {
	var rec agentConversationRecord
	if err := r.db.WithContext(ctx).First(&rec, "id = ?", id.String()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, agentdom.ErrConversationNotFound
		}
		return nil, err
	}
	return conversationFromRecord(rec), nil
}

// CreateConversation inserts a new conversation record.
func (r *AgentRepository) CreateConversation(ctx context.Context, c *agentdom.AgentConversation) error {
	rec := conversationToRecord(c)
	return r.db.WithContext(ctx).Create(&rec).Error
}

// UpdateConversationStatus sets the status field of a conversation.
func (r *AgentRepository) UpdateConversationStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.db.WithContext(ctx).
		Model(&agentConversationRecord{}).
		Where("id = ?", id.String()).
		Updates(map[string]any{"status": status, "updated_at": time.Now()}).Error
}

// UpdateConversation saves the full conversation record.
func (r *AgentRepository) UpdateConversation(ctx context.Context, c *agentdom.AgentConversation) error {
	rec := conversationToRecord(c)
	return r.db.WithContext(ctx).Save(&rec).Error
}

// ListConversationEvents returns a paginated list of events for a conversation.
func (r *AgentRepository) ListConversationEvents(ctx context.Context, conversationID uuid.UUID, offset, limit int) ([]*agentdom.AgentConversationEvent, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&agentConversationEventRecord{}).
		Where("conversation_id = ?", conversationID.String()).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var recs []agentConversationEventRecord
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID.String()).
		Order("event_index ASC").
		Offset(offset).Limit(limit).
		Find(&recs).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*agentdom.AgentConversationEvent, 0, len(recs))
	for _, rec := range recs {
		e, err := conversationEventFromRecord(rec)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, e)
	}
	return result, total, nil
}

// CreateConversationEvent inserts a new conversation event record.
func (r *AgentRepository) CreateConversationEvent(ctx context.Context, e *agentdom.AgentConversationEvent) error {
	rec, err := conversationEventToRecord(e)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(&rec).Error
}

// -------------------------------------------------------------------------
// Chat Sessions
// -------------------------------------------------------------------------

// ListChatSessions returns all chat sessions for the given agent and member.
func (r *AgentRepository) ListChatSessions(ctx context.Context, agentID, memberID uuid.UUID) ([]*agentdom.AgentChatSession, error) {
	var recs []agentChatSessionRecord
	if err := r.db.WithContext(ctx).
		Where("agent_id = ? AND member_id = ?", agentID.String(), memberID.String()).
		Order("created_at DESC").
		Find(&recs).Error; err != nil {
		return nil, err
	}
	result := make([]*agentdom.AgentChatSession, 0, len(recs))
	for _, rec := range recs {
		result = append(result, chatSessionFromRecord(rec))
	}
	return result, nil
}

// FindChatSessionByID returns a single chat session by its primary key.
func (r *AgentRepository) FindChatSessionByID(ctx context.Context, id uuid.UUID) (*agentdom.AgentChatSession, error) {
	var rec agentChatSessionRecord
	if err := r.db.WithContext(ctx).First(&rec, "id = ?", id.String()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, agentdom.ErrChatSessionNotFound
		}
		return nil, err
	}
	return chatSessionFromRecord(rec), nil
}

// CreateChatSession inserts a new chat session record.
func (r *AgentRepository) CreateChatSession(ctx context.Context, s *agentdom.AgentChatSession) error {
	rec := chatSessionToRecord(s)
	return r.db.WithContext(ctx).Create(&rec).Error
}

// UpdateChatSession saves the full chat session record.
func (r *AgentRepository) UpdateChatSession(ctx context.Context, s *agentdom.AgentChatSession) error {
	rec := chatSessionToRecord(s)
	return r.db.WithContext(ctx).Save(&rec).Error
}

// -------------------------------------------------------------------------
// Mapping helpers
// -------------------------------------------------------------------------

func agentFromReadRow(row agentReadRow) *agentdom.Agent {
	a := &agentdom.Agent{
		ID:              mustParseUUID(row.ID),
		ProjectID:       mustParseUUID(row.ProjectID),
		Name:            row.Name,
		Handle:          row.Handle,
		AvatarURL:       row.AvatarURL,
		LLMProvider:     row.LLMProvider,
		LLMModel:        row.LLMModel,
		LLMAPIKeySecret: row.LLMAPIKeySecret,
		LLMBaseURL:      row.LLMBaseURL,
		SystemPrompt:    row.SystemPrompt,
		CanCloneRepos:   row.CanCloneRepos,
		CanCreatePRs:    row.CanCreatePRs,
		MaxIterations:   row.MaxIterations,
		TimeoutMinutes:  row.TimeoutMinutes,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
	if row.CreatedBy != nil {
		id := mustParseUUID(*row.CreatedBy)
		a.CreatedBy = &id
	}
	if row.DeletedAt.Valid {
		t := row.DeletedAt.Time
		a.DeletedAt = &t
	}
	if row.MemberID != nil {
		mid := mustParseUUID(*row.MemberID)
		a.MemberID = &mid
	}
	return a
}

func agentToRecord(a *agentdom.Agent) agentRecord {
	rec := agentRecord{
		ID:              a.ID.String(),
		ProjectID:       a.ProjectID.String(),
		Name:            a.Name,
		Handle:          a.Handle,
		AvatarURL:       a.AvatarURL,
		LLMProvider:     a.LLMProvider,
		LLMModel:        a.LLMModel,
		LLMAPIKeySecret: a.LLMAPIKeySecret,
		LLMBaseURL:      a.LLMBaseURL,
		SystemPrompt:    a.SystemPrompt,
		CanCloneRepos:   a.CanCloneRepos,
		CanCreatePRs:    a.CanCreatePRs,
		MaxIterations:   a.MaxIterations,
		TimeoutMinutes:  a.TimeoutMinutes,
		CreatedAt:       a.CreatedAt,
		UpdatedAt:       a.UpdatedAt,
	}
	if a.CreatedBy != nil {
		s := a.CreatedBy.String()
		rec.CreatedBy = &s
	}
	return rec
}

func mcpServerFromRecord(rec agentMCPServerRecord) (*agentdom.AgentMCPServer, error) {
	var args []string
	if err := json.Unmarshal(rec.Args, &args); err != nil {
		return nil, err
	}
	var env map[string]string
	if err := json.Unmarshal(rec.Env, &env); err != nil {
		return nil, err
	}
	return &agentdom.AgentMCPServer{
		ID:         mustParseUUID(rec.ID),
		AgentID:    mustParseUUID(rec.AgentID),
		ServerName: rec.ServerName,
		Transport:  rec.Transport,
		Command:    rec.Command,
		Args:       args,
		URL:        rec.URL,
		Env:        env,
		IsEnabled:  rec.IsEnabled,
		CreatedAt:  rec.CreatedAt,
		UpdatedAt:  rec.UpdatedAt,
	}, nil
}

func mcpServerToRecord(s *agentdom.AgentMCPServer) (agentMCPServerRecord, error) {
	args, err := json.Marshal(s.Args)
	if err != nil {
		return agentMCPServerRecord{}, err
	}
	env, err := json.Marshal(s.Env)
	if err != nil {
		return agentMCPServerRecord{}, err
	}
	return agentMCPServerRecord{
		ID:         s.ID.String(),
		AgentID:    s.AgentID.String(),
		ServerName: s.ServerName,
		Transport:  s.Transport,
		Command:    s.Command,
		Args:       args,
		URL:        s.URL,
		Env:        env,
		IsEnabled:  s.IsEnabled,
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	}, nil
}

func skillFromRecord(rec agentSkillRecord) (*agentdom.AgentSkill, error) {
	var triggers []string
	if err := json.Unmarshal(rec.Triggers, &triggers); err != nil {
		return nil, err
	}
	return &agentdom.AgentSkill{
		ID:           mustParseUUID(rec.ID),
		AgentID:      mustParseUUID(rec.AgentID),
		SkillName:    rec.SkillName,
		SkillSource:  rec.SkillSource,
		SkillContent: rec.SkillContent,
		SourceURL:    rec.SourceURL,
		Triggers:     triggers,
		IsEnabled:    rec.IsEnabled,
		CreatedAt:    rec.CreatedAt,
		UpdatedAt:    rec.UpdatedAt,
	}, nil
}

func skillToRecord(s *agentdom.AgentSkill) (agentSkillRecord, error) {
	triggers, err := json.Marshal(s.Triggers)
	if err != nil {
		return agentSkillRecord{}, err
	}
	return agentSkillRecord{
		ID:           s.ID.String(),
		AgentID:      s.AgentID.String(),
		SkillName:    s.SkillName,
		SkillSource:  s.SkillSource,
		SkillContent: s.SkillContent,
		SourceURL:    s.SourceURL,
		Triggers:     triggers,
		IsEnabled:    s.IsEnabled,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}, nil
}

func conversationFromRecord(rec agentConversationRecord) *agentdom.AgentConversation {
	c := &agentdom.AgentConversation{
		ID:                  mustParseUUID(rec.ID),
		AgentID:             mustParseUUID(rec.AgentID),
		ProjectID:           mustParseUUID(rec.ProjectID),
		TriggerType:         rec.TriggerType,
		TriggeredByMemberID: mustParseUUID(rec.TriggeredByMemberID),
		Status:              rec.Status,
		ContainerID:         rec.ContainerID,
		HostPort:            rec.HostPort,
		IterationCount:      rec.IterationCount,
		ErrorMessage:        rec.ErrorMessage,
		RepoCloneURL:        rec.RepoCloneURL,
		BranchName:          rec.BranchName,
		PRUrl:               rec.PRUrl,
		PersistenceDir:      rec.PersistenceDir,
		StartedAt:           rec.StartedAt,
		FinishedAt:          rec.FinishedAt,
		CreatedAt:           rec.CreatedAt,
		UpdatedAt:           rec.UpdatedAt,
	}
	if rec.TaskID != nil {
		id := mustParseUUID(*rec.TaskID)
		c.TaskID = &id
	}
	if rec.CommentID != nil {
		id := mustParseUUID(*rec.CommentID)
		c.CommentID = &id
	}
	if rec.ChatSessionID != nil {
		id := mustParseUUID(*rec.ChatSessionID)
		c.ChatSessionID = &id
	}
	if rec.RepoPluginID != nil {
		id := mustParseUUID(*rec.RepoPluginID)
		c.RepoPluginID = &id
	}
	return c
}

func conversationToRecord(c *agentdom.AgentConversation) agentConversationRecord {
	rec := agentConversationRecord{
		ID:                  c.ID.String(),
		AgentID:             c.AgentID.String(),
		ProjectID:           c.ProjectID.String(),
		TriggerType:         c.TriggerType,
		TriggeredByMemberID: c.TriggeredByMemberID.String(),
		Status:              c.Status,
		ContainerID:         c.ContainerID,
		HostPort:            c.HostPort,
		IterationCount:      c.IterationCount,
		ErrorMessage:        c.ErrorMessage,
		RepoCloneURL:        c.RepoCloneURL,
		BranchName:          c.BranchName,
		PRUrl:               c.PRUrl,
		PersistenceDir:      c.PersistenceDir,
		StartedAt:           c.StartedAt,
		FinishedAt:          c.FinishedAt,
		CreatedAt:           c.CreatedAt,
		UpdatedAt:           c.UpdatedAt,
	}
	if c.TaskID != nil {
		s := c.TaskID.String()
		rec.TaskID = &s
	}
	if c.CommentID != nil {
		s := c.CommentID.String()
		rec.CommentID = &s
	}
	if c.ChatSessionID != nil {
		s := c.ChatSessionID.String()
		rec.ChatSessionID = &s
	}
	if c.RepoPluginID != nil {
		s := c.RepoPluginID.String()
		rec.RepoPluginID = &s
	}
	return rec
}

func conversationEventFromRecord(rec agentConversationEventRecord) (*agentdom.AgentConversationEvent, error) {
	var payload map[string]any
	if err := json.Unmarshal(rec.Payload, &payload); err != nil {
		return nil, err
	}
	return &agentdom.AgentConversationEvent{
		ID:             mustParseUUID(rec.ID),
		ConversationID: mustParseUUID(rec.ConversationID),
		EventIndex:     rec.EventIndex,
		EventType:      rec.EventType,
		EventSource:    rec.EventSource,
		Payload:        payload,
		CreatedAt:      rec.CreatedAt,
	}, nil
}

func conversationEventToRecord(e *agentdom.AgentConversationEvent) (agentConversationEventRecord, error) {
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		return agentConversationEventRecord{}, err
	}
	return agentConversationEventRecord{
		ID:             e.ID.String(),
		ConversationID: e.ConversationID.String(),
		EventIndex:     e.EventIndex,
		EventType:      e.EventType,
		EventSource:    e.EventSource,
		Payload:        payload,
		CreatedAt:      e.CreatedAt,
	}, nil
}

func chatSessionFromRecord(rec agentChatSessionRecord) *agentdom.AgentChatSession {
	return &agentdom.AgentChatSession{
		ID:            mustParseUUID(rec.ID),
		AgentID:       mustParseUUID(rec.AgentID),
		ProjectID:     mustParseUUID(rec.ProjectID),
		MemberID:      mustParseUUID(rec.MemberID),
		Title:         rec.Title,
		LastMessageAt: rec.LastMessageAt,
		CreatedAt:     rec.CreatedAt,
		UpdatedAt:     rec.UpdatedAt,
	}
}

func chatSessionToRecord(s *agentdom.AgentChatSession) agentChatSessionRecord {
	return agentChatSessionRecord{
		ID:            s.ID.String(),
		AgentID:       s.AgentID.String(),
		ProjectID:     s.ProjectID.String(),
		MemberID:      s.MemberID.String(),
		Title:         s.Title,
		LastMessageAt: s.LastMessageAt,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}
