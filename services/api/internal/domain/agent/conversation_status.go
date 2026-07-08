package agentdom

// ConversationStatus is a conversation's lifecycle state, matching the
// agent_conversations_status_check constraint in
// services/api/migrations/000008_add_ai_agents.sql exactly.
type ConversationStatus string

const (
	ConversationStatusQueued   ConversationStatus = "queued"
	ConversationStatusRunning  ConversationStatus = "running"
	ConversationStatusPaused   ConversationStatus = "paused"
	ConversationStatusFinished ConversationStatus = "finished"
	ConversationStatusFailed   ConversationStatus = "failed"
	ConversationStatusStopped  ConversationStatus = "stopped"
)

// IsTerminal reports whether this status ends the conversation's lifecycle.
// Terminal conversations cannot be resumed — a new conversation must be
// created instead (see SendChatMessage, which only reuses a conversation
// when its status is ConversationStatusPaused).
func (s ConversationStatus) IsTerminal() bool {
	return s == ConversationStatusFinished || s == ConversationStatusFailed || s == ConversationStatusStopped
}
