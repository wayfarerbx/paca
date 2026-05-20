// Package agentdom defines the AI Agent aggregate and its domain contracts.
package agentdom

import (
	"time"

	"github.com/google/uuid"
)

// Agent represents an AI agent belonging to a project.
type Agent struct {
	ID              uuid.UUID
	ProjectID       uuid.UUID
	Name            string
	Handle          string
	AvatarURL       *string
	LLMProvider     string
	LLMModel        string
	LLMAPIKeySecret string // reference to secrets store entry
	LLMBaseURL      *string
	SystemPrompt    string
	CanCloneRepos   bool
	CanCreatePRs    bool
	MaxIterations   int
	TimeoutMinutes  int
	CreatedBy       *uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
	// Member ID in project_members (populated on create / list)
	MemberID   *uuid.UUID
	MCPServers []*AgentMCPServer
	Skills     []*AgentSkill
}

// AgentMCPServer is a custom MCP server configuration attached to an agent.
type AgentMCPServer struct {
	ID         uuid.UUID
	AgentID    uuid.UUID
	ServerName string
	Transport  string // stdio | sse | http
	Command    *string
	Args       []string
	URL        *string
	Env        map[string]string
	IsEnabled  bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// AgentSkill is a skill associated with an agent.
type AgentSkill struct {
	ID           uuid.UUID
	AgentID      uuid.UUID
	SkillName    string
	SkillSource  string // inline | marketplace | github_url
	SkillContent string
	SourceURL    *string
	Triggers     []string
	IsEnabled    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AgentConversation tracks each OpenHands conversation session.
type AgentConversation struct {
	ID                  uuid.UUID
	AgentID             uuid.UUID
	ProjectID           uuid.UUID
	TriggerType         string // task_assigned | comment_mention | chat_message
	TaskID              *uuid.UUID
	CommentID           *uuid.UUID
	ChatSessionID       *uuid.UUID
	TriggeredByMemberID uuid.UUID
	Status              string // queued | running | paused | finished | failed | stopped
	ContainerID         *string
	HostPort            *int
	IterationCount      int
	ErrorMessage        *string
	RepoPluginID        *uuid.UUID
	RepoCloneURL        *string
	BranchName          *string
	PRUrl               *string
	PersistenceDir      *string
	StartedAt           *time.Time
	FinishedAt          *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	// Populated by JOIN
	AgentName   string
	AgentHandle string
	TaskTitle   *string
}

// AgentConversationEvent is a single event emitted by the OpenHands SDK.
type AgentConversationEvent struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	EventIndex     int
	EventType      string
	EventSource    string // agent | user | system
	Payload        map[string]any
	CreatedAt      time.Time
}

// AgentChatSession is a persistent chat session between a user and an agent.
type AgentChatSession struct {
	ID            uuid.UUID
	AgentID       uuid.UUID
	ProjectID     uuid.UUID
	MemberID      uuid.UUID
	Title         *string
	LastMessageAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
