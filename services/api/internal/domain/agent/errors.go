package agentdom

import "errors"

// Agent errors
var (
	ErrAgentNotFound      = errors.New("agent not found")
	ErrAgentHandleTaken   = errors.New("agent handle already in use")
	ErrAgentHandleInvalid = errors.New("agent handle is invalid")
	ErrAgentNameInvalid   = errors.New("agent name is empty or invalid")
)

// MCP server errors
var (
	ErrMCPServerNotFound  = errors.New("MCP server not found")
	ErrMCPServerNameTaken = errors.New("MCP server name already in use on this agent")
)

// Skill errors
var (
	ErrSkillNotFound  = errors.New("skill not found")
	ErrSkillNameTaken = errors.New("skill name already in use on this agent")
)

// Conversation errors
var (
	ErrConversationNotFound       = errors.New("conversation not found")
	ErrConversationNotRunning     = errors.New("conversation is not running")
	ErrConversationAlreadyStopped = errors.New("conversation is already stopped")
)

// Chat session errors
var (
	ErrChatSessionNotFound = errors.New("chat session not found")
)
