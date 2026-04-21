package events

// ChannelRealtime is the Valkey Pub/Sub channel that services/realtime subscribes to for immediate
// fan-out to connected Socket.IO clients.
const ChannelRealtime = "paca.events"

// StreamAnalytics is the Valkey Stream key used for durable analytics and audit log events.
const StreamAnalytics = "paca.analytics"

// StreamTaskActivities is the Valkey Stream key used to fan out task-activity
// events from the API to the internal consumer that persists them to PostgreSQL.
// System-generated activities (task created, updated, BDD changes, etc.) are
// appended here instead of being written directly to the database; the
// ActivityConsumer worker reads this stream and handles the DB write.
const StreamTaskActivities = "paca.task_activities"

// StreamDocActivities is the Valkey Stream key used to fan out doc-activity
// events from the API to the internal consumer that persists them to PostgreSQL.
// System-generated activities (doc created, updated, etc.) are appended here;
// the DocActivityConsumer worker reads this stream and handles the DB write.
const StreamDocActivities = "paca.doc_activities"

// StreamTaskAssignments is the Valkey Stream key used to fan out task
// assignment events (task created/updated with a new assignee) to the
// NotificationConsumer worker, which creates in-app notifications and
// publishes real-time push events.
const StreamTaskAssignments = "paca.task_assignments"

// Event type constants used in both Pub/Sub messages and Stream entries.
const (
	// --- Auth events --------------------------------------------------------
	TopicUserCreated = "user.created"
	TopicUserDeleted = "user.deleted"
	TopicAuthLogin   = "auth.login"
	TopicAuthLogout  = "auth.logout"

	// --- Task events --------------------------------------------------------
	TopicTaskCreated = "task.created"
	TopicTaskUpdated = "task.updated"
	TopicTaskDeleted = "task.deleted"

	// --- Task attachment events ---------------------------------------------
	TopicTaskAttachmentAdded   = "task.attachment.added"
	TopicTaskAttachmentRemoved = "task.attachment.removed"

	// --- Task BDD scenario events -------------------------------------------
	TopicTaskBDDScenarioCreated = "task.bdd_scenario.created"
	TopicTaskBDDScenarioUpdated = "task.bdd_scenario.updated"
	TopicTaskBDDScenarioDeleted = "task.bdd_scenario.deleted"

	// --- Task checklist events ----------------------------------------------
	TopicTaskChecklistCreated     = "task.checklist.created"
	TopicTaskChecklistUpdated     = "task.checklist.updated"
	TopicTaskChecklistDeleted     = "task.checklist.deleted"
	TopicTaskChecklistItemCreated = "task.checklist_item.created"
	TopicTaskChecklistItemUpdated = "task.checklist_item.updated"
	TopicTaskChecklistItemDeleted = "task.checklist_item.deleted"

	// --- Comment events -----------------------------------------------------
	TopicTaskCommentAdded   = "task.comment.added"
	TopicTaskCommentUpdated = "task.comment.updated"
	TopicTaskCommentDeleted = "task.comment.deleted"

	// --- Doc events ---------------------------------------------------------
	TopicDocCreated = "doc.created"
	TopicDocUpdated = "doc.updated"
	TopicDocDeleted = "doc.deleted"
	TopicDocMoved   = "doc.moved"

	// --- Doc folder events --------------------------------------------------
	TopicDocFolderCreated = "doc.folder.created"
	TopicDocFolderUpdated = "doc.folder.updated"
	TopicDocFolderDeleted = "doc.folder.deleted"

	// --- Doc comment events -------------------------------------------------
	TopicDocCommentAdded   = "doc.comment.added"
	TopicDocCommentUpdated = "doc.comment.updated"
	TopicDocCommentDeleted = "doc.comment.deleted"

	// --- Notification events ------------------------------------------------
	// TopicNotificationCreated is published to ChannelRealtime when a new
	// notification is created.  The payload includes recipient_user_id so the
	// realtime service can route the event to the correct user room.
	TopicNotificationCreated = "notification.created"
)
