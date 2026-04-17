package taskdom

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ActivityType categorises a task activity entry.
type ActivityType string

// Activity type constants.  The "comment" type is user-initiated; all others
// are system-generated when task-related operations occur.
const (
	// --- Task-level events ---------------------------------------------------

	// ActivityTypeTaskCreated is recorded when a task is first created.
	ActivityTypeTaskCreated ActivityType = "task.created"
	// ActivityTypeTaskUpdated is recorded when mutable task fields change.
	ActivityTypeTaskUpdated ActivityType = "task.updated"
	// ActivityTypeTaskDeleted is recorded when a task is soft-deleted.
	ActivityTypeTaskDeleted ActivityType = "task.deleted"

	// --- Attachment events ---------------------------------------------------

	// ActivityTypeAttachmentAdded is recorded when a file is attached to a task.
	ActivityTypeAttachmentAdded ActivityType = "task.attachment.added"
	// ActivityTypeAttachmentRemoved is recorded when an attachment is detached.
	ActivityTypeAttachmentRemoved ActivityType = "task.attachment.removed"

	// --- BDD scenario events -------------------------------------------------

	// ActivityTypeBDDScenarioCreated is recorded when a BDD scenario is added.
	ActivityTypeBDDScenarioCreated ActivityType = "task.bdd_scenario.created"
	// ActivityTypeBDDScenarioUpdated is recorded when a BDD scenario is edited.
	ActivityTypeBDDScenarioUpdated ActivityType = "task.bdd_scenario.updated"
	// ActivityTypeBDDScenarioDeleted is recorded when a BDD scenario is removed.
	ActivityTypeBDDScenarioDeleted ActivityType = "task.bdd_scenario.deleted"

	// --- Checklist events ----------------------------------------------------

	// ActivityTypeChecklistCreated is recorded when a checklist group is added.
	ActivityTypeChecklistCreated ActivityType = "task.checklist.created"
	// ActivityTypeChecklistUpdated is recorded when a checklist group is renamed.
	ActivityTypeChecklistUpdated ActivityType = "task.checklist.updated"
	// ActivityTypeChecklistDeleted is recorded when a checklist group is removed.
	ActivityTypeChecklistDeleted ActivityType = "task.checklist.deleted"
	// ActivityTypeChecklistItemCreated is recorded when a checklist item is added.
	ActivityTypeChecklistItemCreated ActivityType = "task.checklist_item.created"
	// ActivityTypeChecklistItemUpdated is recorded when a checklist item is edited.
	ActivityTypeChecklistItemUpdated ActivityType = "task.checklist_item.updated"
	// ActivityTypeChecklistItemDeleted is recorded when a checklist item is removed.
	ActivityTypeChecklistItemDeleted ActivityType = "task.checklist_item.deleted"

	// --- Comment -------------------------------------------------------------

	// ActivityTypeComment is a user-authored comment on the task.
	ActivityTypeComment ActivityType = "comment"
)

// Activity is a single entry in a task's activity log.  It represents either
// a system-generated change event (e.g. status change) or a user comment.
type Activity struct {
	ID            uuid.UUID
	TaskID        uuid.UUID
	ActorID       *uuid.UUID // nil when the actor account has been deleted
	ActorName     string     // denormalised full name (populated on read)
	ActorUsername string     // denormalised username   (populated on read)
	ActivityType  ActivityType
	Content       json.RawMessage
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time // non-nil for soft-deleted comments
}

// FieldChange records a single before/after value for task.updated events.
type FieldChange struct {
	Field string `json:"field"`
	Old   any    `json:"old"`
	New   any    `json:"new"`
}

// ActivityRepository defines persistence for task activities.
type ActivityRepository interface {
	// ListActivities returns all non-deleted activity entries for a task
	// ordered by created_at ascending.
	ListActivities(ctx context.Context, taskID uuid.UUID) ([]*Activity, error)
	// FindActivityByID returns a single activity (including soft-deleted).
	FindActivityByID(ctx context.Context, id uuid.UUID) (*Activity, error)
	// CreateActivity persists a new activity entry.
	CreateActivity(ctx context.Context, a *Activity) error
	// UpdateActivity persists mutable changes to an existing activity.
	UpdateActivity(ctx context.Context, a *Activity) error
	// DeleteActivity soft-deletes the activity (sets deleted_at).
	DeleteActivity(ctx context.Context, id uuid.UUID) error
}

// ActivityService defines use-cases for task activity and comments.
// It embeds ActivityRecorder so a single interface covers both user-facing
// comment operations and system-generated activity recording.
type ActivityService interface {
	ActivityRecorder
	// ListActivities returns all activities (system + comments) for a task.
	ListActivities(ctx context.Context, taskID uuid.UUID) ([]*Activity, error)
	// AddComment creates a new user comment on the task.
	AddComment(ctx context.Context, in AddCommentInput) (*Activity, error)
	// UpdateComment edits the text of an existing comment.
	// Returns ErrActivityForbidden when actorID != comment's author.
	UpdateComment(ctx context.Context, id uuid.UUID, actorID uuid.UUID, text string) (*Activity, error)
	// DeleteComment soft-deletes a comment.
	// Returns ErrActivityForbidden when actorID != comment's author.
	DeleteComment(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error
}

// ActivityRecorder is the minimal interface used to persist system-generated
// activity entries.  It is embedded in ActivityService so a single concrete
// implementation satisfies both.
type ActivityRecorder interface {
	RecordActivity(ctx context.Context, in RecordActivityInput) error
}

// AddCommentInput carries the data needed to post a comment.
type AddCommentInput struct {
	TaskID  uuid.UUID
	ActorID uuid.UUID
	Text    string
}

// RecordActivityInput carries the data needed to persist a system event.
type RecordActivityInput struct {
	TaskID       uuid.UUID
	ActorID      *uuid.UUID // nil is allowed for system events
	ActivityType ActivityType
	Content      json.RawMessage
}
