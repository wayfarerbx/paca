package agentdom

// ConversationStatus is a conversation's lifecycle state, matching the
// agent_conversations_status_check constraint in
// services/api/migrations/000008_add_ai_agents.sql exactly.
type ConversationStatus string

const (
	// ConversationStatusQueued indicates a conversation is waiting to be processed.
	ConversationStatusQueued ConversationStatus = "queued"
	// ConversationStatusRunning indicates a conversation is actively being processed.
	ConversationStatusRunning ConversationStatus = "running"
	// ConversationStatusPaused indicates a conversation is temporarily paused.
	ConversationStatusPaused ConversationStatus = "paused"
	// ConversationStatusFinished indicates a conversation has completed successfully.
	ConversationStatusFinished ConversationStatus = "finished"
	// ConversationStatusFailed indicates a conversation has failed.
	ConversationStatusFailed ConversationStatus = "failed"
	// ConversationStatusStopped indicates a conversation was stopped by the user.
	ConversationStatusStopped ConversationStatus = "stopped"
)

// IsTerminal reports whether this status ends the conversation's lifecycle.
// Terminal conversations cannot be resumed — a new conversation must be
// created instead (see SendChatMessage, which only reuses a conversation
// when its status is ConversationStatusPaused).
func (s ConversationStatus) IsTerminal() bool {
	return s == ConversationStatusFinished || s == ConversationStatusFailed || s == ConversationStatusStopped
}
