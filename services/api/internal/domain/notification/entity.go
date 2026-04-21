// Package notificationdom defines the notification aggregate and its domain contracts.
package notificationdom

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// NotificationType categorises a notification.
type NotificationType string

const (
	// NotificationTypeAssigned is sent when a task is assigned to a user.
	NotificationTypeAssigned NotificationType = "assigned"
	// NotificationTypeMentioned is sent when a user is @mentioned in a comment.
	NotificationTypeMentioned NotificationType = "mentioned"
)

// Notification is a single notification entry.
type Notification struct {
	ID uuid.UUID
	// RecipientUserID is the users.id of the user who receives this notification.
	RecipientUserID uuid.UUID
	// ActorMemberID is the project_members.id of the user who triggered the
	// notification.  Nil when the actor account has been deleted.
	ActorMemberID *uuid.UUID
	// ActorFullName and ActorUsername are denormalised from the actor's user record.
	ActorFullName string
	ActorUsername string
	// Type is one of the NotificationType constants.
	Type NotificationType
	// TaskID is the task this notification is about.
	TaskID *uuid.UUID
	// TaskTitle and TaskNumber are denormalised from the task record.
	TaskTitle  string
	TaskNumber int
	// ProjectID is the project the task belongs to.
	ProjectID uuid.UUID
	// ProjectName is denormalised from the project record.
	ProjectName string
	// ReadAt is nil when the notification has not yet been read.
	ReadAt    *time.Time
	CreatedAt time.Time
}

// CreateNotificationInput carries the data needed to create a notification.
type CreateNotificationInput struct {
	RecipientUserID uuid.UUID
	ActorMemberID   *uuid.UUID
	Type            NotificationType
	TaskID          *uuid.UUID
	ProjectID       uuid.UUID
}

// Repository defines persistence operations for notifications.
type Repository interface {
	// Create persists a new notification.
	Create(ctx context.Context, n *Notification) error
	// ListForUser returns up to limit notifications for the given user,
	// ordered newest first.
	ListForUser(ctx context.Context, userID uuid.UUID, limit int) ([]*Notification, error)
	// UnreadCount returns the number of unread notifications for the given user.
	UnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)
	// MarkAsRead sets read_at on a notification owned by userID.
	// Returns ErrNotificationNotFound when the notification does not exist or
	// does not belong to userID.
	MarkAsRead(ctx context.Context, id, userID uuid.UUID) error
	// MarkAllAsRead sets read_at on all unread notifications for userID.
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
}

// Service defines the notification use-cases exposed to callers.
type Service interface {
	// NotifyAssigned creates a notification for the new assignee when a task
	// is assigned.  actorMemberID is the project_members.id of the caller.
	// newAssigneeMemberID is the project_members.id of the new assignee.
	// Does nothing if the new assignee is nil or is the same as the actor.
	NotifyAssigned(ctx context.Context, in NotifyAssignedInput) error
	// NotifyMentioned creates notifications for all @mentioned users found in
	// commentText who are members of the project.
	NotifyMentioned(ctx context.Context, in NotifyMentionedInput) error
	// ListNotifications returns up to limit notifications for the authenticated user.
	ListNotifications(ctx context.Context, userID uuid.UUID, limit int) ([]*Notification, error)
	// UnreadCount returns the count of unread notifications for the user.
	UnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)
	// MarkAsRead marks a single notification as read.
	MarkAsRead(ctx context.Context, id, userID uuid.UUID) error
	// MarkAllAsRead marks all notifications for the user as read.
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
}

// NotifyAssignedInput carries data for an assignment notification.
type NotifyAssignedInput struct {
	TaskID    uuid.UUID
	ProjectID uuid.UUID
	// NewAssigneeMemberID is the project_members.id of the new assignee.
	NewAssigneeMemberID uuid.UUID
	// ActorUserID is the users.id of the person who made the assignment.
	// Used to resolve the actor to a member and to skip self-assignment.
	ActorUserID uuid.UUID
}

// NotifyMentionedInput carries data for a mention notification.
type NotifyMentionedInput struct {
	TaskID      uuid.UUID
	ProjectID   uuid.UUID
	CommentText string
	// ActorMemberID is the project_members.id of the commenter.
	ActorMemberID uuid.UUID
	// ActorUserID is the users.id of the commenter (used to exclude self-mention).
	ActorUserID uuid.UUID
}

// ErrNotificationNotFound is returned when a notification does not exist or
// does not belong to the requesting user.
var ErrNotificationNotFound = errNotificationNotFound("notification not found")

type errNotificationNotFound string

func (e errNotificationNotFound) Error() string { return string(e) }
