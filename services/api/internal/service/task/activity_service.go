package tasksvc

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	projectdom "github.com/paca/api/internal/domain/project"
	taskdom "github.com/paca/api/internal/domain/task"
	"github.com/paca/api/internal/events"
	"github.com/paca/api/internal/platform/messaging"
)

// memberLookup is the minimal interface ActivitySvc needs to resolve a user
// UUID to a project member UUID.
type memberLookup interface {
	FindMemberByUserProject(ctx context.Context, userID, projectID uuid.UUID) (*projectdom.ProjectMember, error)
}

// ActivitySvc implements taskdom.ActivityService (which includes
// taskdom.ActivityRecorder via embedding).
type ActivitySvc struct {
	repo       taskdom.ActivityRepository
	memberRepo memberLookup
	publisher  *messaging.Publisher
}

// NewActivityService creates a new ActivitySvc backed by repo.
// memberRepo is used to resolve user UUIDs to project-member UUIDs for comment
// operations; it may be nil (lookups will return ErrMemberNotFound).
// publisher may be nil; events are then skipped silently.
func NewActivityService(repo taskdom.ActivityRepository, memberRepo memberLookup, publisher *messaging.Publisher) *ActivitySvc {
	return &ActivitySvc{repo: repo, memberRepo: memberRepo, publisher: publisher}
}

// --- ActivityRecorder -------------------------------------------------------

// RecordActivity publishes a system-generated activity event to the Valkey
// stream (StreamTaskActivities). The ActivityConsumer worker reads that stream
// and writes the entry to the database, so this method intentionally does NOT
// touch the database itself.
func (s *ActivitySvc) RecordActivity(ctx context.Context, in taskdom.RecordActivityInput) error {
	now := time.Now()
	content := in.Content
	if len(content) == 0 {
		content = json.RawMessage("{}")
	}
	a := &taskdom.Activity{
		ID:           uuid.New(),
		TaskID:       in.TaskID,
		ActorID:      in.ActorID,
		ActivityType: in.ActivityType,
		Content:      content,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.publishToActivityStream(ctx, a, in.ProjectID)
	return nil
}

// --- ActivityService --------------------------------------------------------

// ListActivities returns all non-deleted activities for a task, oldest first.
func (s *ActivitySvc) ListActivities(ctx context.Context, taskID uuid.UUID) ([]*taskdom.Activity, error) {
	return s.repo.ListActivities(ctx, taskID)
}

// AddComment creates a user comment on the task.
func (s *ActivitySvc) AddComment(ctx context.Context, in taskdom.AddCommentInput) (*taskdom.Activity, error) {
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return nil, taskdom.ErrCommentTextInvalid
	}
	// Resolve the authenticated user UUID to the project-member UUID so the
	// stored actor_id satisfies the FK constraint on task_activities(actor_id)
	// → project_members(id).
	member, err := s.memberRepo.FindMemberByUserProject(ctx, in.ActorID, in.ProjectID)
	if err != nil {
		return nil, err
	}
	content, _ := json.Marshal(map[string]string{"text": text})
	now := time.Now()
	a := &taskdom.Activity{
		ID:           uuid.New(),
		TaskID:       in.TaskID,
		ActorID:      &member.ID,
		ActivityType: taskdom.ActivityTypeComment,
		Content:      content,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.CreateActivity(ctx, a); err != nil {
		return nil, err
	}
	s.publishRealtimeOnly(ctx, events.TopicTaskCommentAdded, activityPayload(a, in.ProjectID))
	return a, nil
}

// UpdateComment edits the text of an existing comment.
func (s *ActivitySvc) UpdateComment(ctx context.Context, id uuid.UUID, projectID uuid.UUID, actorID uuid.UUID, text string) (*taskdom.Activity, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, taskdom.ErrCommentTextInvalid
	}
	a, err := s.repo.FindActivityByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.ActivityType != taskdom.ActivityTypeComment {
		return nil, taskdom.ErrActivityNotAComment
	}
	// Resolve caller's user UUID to their member UUID for ownership comparison.
	member, err := s.memberRepo.FindMemberByUserProject(ctx, actorID, projectID)
	if err != nil {
		return nil, err
	}
	if a.ActorID == nil || *a.ActorID != member.ID {
		return nil, taskdom.ErrActivityForbidden
	}
	content, _ := json.Marshal(map[string]string{"text": text})
	a.Content = content
	a.UpdatedAt = time.Now()
	if err := s.repo.UpdateActivity(ctx, a); err != nil {
		return nil, err
	}
	s.publishRealtimeOnly(ctx, events.TopicTaskCommentUpdated, activityPayload(a, uuid.Nil))
	return a, nil
}

// DeleteComment soft-deletes a comment.
func (s *ActivitySvc) DeleteComment(ctx context.Context, id uuid.UUID, projectID uuid.UUID, actorID uuid.UUID) error {
	a, err := s.repo.FindActivityByID(ctx, id)
	if err != nil {
		return err
	}
	if a.ActivityType != taskdom.ActivityTypeComment {
		return taskdom.ErrActivityNotAComment
	}
	// Resolve caller's user UUID to their member UUID for ownership comparison.
	member, err := s.memberRepo.FindMemberByUserProject(ctx, actorID, projectID)
	if err != nil {
		return err
	}
	if a.ActorID == nil || *a.ActorID != member.ID {
		return taskdom.ErrActivityForbidden
	}
	if err := s.repo.DeleteActivity(ctx, id); err != nil {
		return err
	}
	s.publishRealtimeOnly(ctx, events.TopicTaskCommentDeleted, map[string]any{
		"id":       id,
		"task_id":  a.TaskID,
		"actor_id": actorID,
	})
	return nil
}

// --- helpers ----------------------------------------------------------------

// activityPayload builds the full stream message body for an activity entry.
// projectID is included so the consumer can resolve the actor (user UUID) to
// the correct project_members.id.
func activityPayload(a *taskdom.Activity, projectID uuid.UUID) map[string]any {
	p := map[string]any{
		"id":            a.ID,
		"task_id":       a.TaskID,
		"project_id":    projectID,
		"activity_type": string(a.ActivityType),
		"content":       string(a.Content),
		"created_at":    a.CreatedAt,
		"updated_at":    a.UpdatedAt,
	}
	if a.ActorID != nil {
		p["actor_id"] = a.ActorID.String()
	}
	return p
}

// publishToActivityStream appends the activity to the dedicated task-activity
// Valkey stream and also broadcasts a real-time pub/sub notification.
// Errors are intentionally swallowed — a messaging failure must not block
// the primary HTTP response.
func (s *ActivitySvc) publishToActivityStream(ctx context.Context, a *taskdom.Activity, projectID uuid.UUID) {
	if s.publisher == nil {
		return
	}
	payload := activityPayload(a, projectID)
	_ = s.publisher.Append(ctx, events.StreamTaskActivities, string(a.ActivityType), payload)
	_ = s.publisher.Publish(ctx, events.ChannelRealtime, map[string]any{
		"type":    string(a.ActivityType),
		"payload": payload,
	})
}

// publishRealtimeOnly sends a real-time pub/sub notification without writing
// to any stream.  Used for comment operations that already write to the DB
// directly and don't need the consumer-persistence path.
func (s *ActivitySvc) publishRealtimeOnly(ctx context.Context, topic string, payload any) {
	if s.publisher == nil {
		return
	}
	_ = s.publisher.Publish(ctx, events.ChannelRealtime, map[string]any{
		"type":    topic,
		"payload": payload,
	})
}
