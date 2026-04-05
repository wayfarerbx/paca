package events

// Valkey Pub/Sub channel that services/realtime subscribes to for immediate
// fan-out to connected Socket.IO clients.
const ChannelRealtime = "paca.events"

// Valkey Stream key used for durable analytics and audit log events.
const StreamAnalytics = "paca.analytics"

// Event type constants used in both Pub/Sub messages and Stream entries.
const (
	TopicUserCreated = "user.created"
	TopicUserDeleted = "user.deleted"
	TopicAuthLogin   = "auth.login"
	TopicAuthLogout  = "auth.logout"
)
