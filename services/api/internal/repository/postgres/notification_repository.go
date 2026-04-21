package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	notificationdom "github.com/paca/api/internal/domain/notification"
	"gorm.io/gorm"
)

// --- GORM model -------------------------------------------------------------

type notificationRecord struct {
	ID              string  `gorm:"primarykey;type:uuid"`
	RecipientUserID string  `gorm:"type:uuid;not null;column:recipient_user_id"`
	ActorMemberID   *string `gorm:"type:uuid;column:actor_member_id"`
	Type            string  `gorm:"not null;column:type"`
	TaskID          *string `gorm:"type:uuid;column:task_id"`
	ProjectID       string  `gorm:"type:uuid;not null;column:project_id"`
	ReadAt          *time.Time
	CreatedAt       time.Time
}

func (notificationRecord) TableName() string { return "notifications" }

// notificationReadRow is the result of the enriched SELECT … JOIN query.
type notificationReadRow struct {
	ID              string
	RecipientUserID string
	ActorMemberID   *string
	Type            string
	TaskID          *string
	ProjectID       string
	ReadAt          *time.Time
	CreatedAt       time.Time

	// Joined fields.
	ActorFullName *string `gorm:"column:actor_full_name"`
	ActorUsername *string `gorm:"column:actor_username"`
	TaskTitle     *string `gorm:"column:task_title"`
	TaskNumber    *int    `gorm:"column:task_number"`
	ProjectName   string  `gorm:"column:project_name"`
}

// --- Repository struct -------------------------------------------------------

// NotificationRepository implements notificationdom.Repository.
type NotificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository returns a new NotificationRepository backed by db.
func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// --- Helpers ----------------------------------------------------------------

const notificationReadCols = `
	n.id, n.recipient_user_id, n.actor_member_id, n.type,
	n.task_id, n.project_id, n.read_at, n.created_at,
	u.full_name AS actor_full_name, u.username AS actor_username,
	t.title AS task_title, t.task_number,
	p.name AS project_name`

func notificationFromRow(r notificationReadRow) *notificationdom.Notification {
	n := &notificationdom.Notification{
		ID:              uuid.MustParse(r.ID),
		RecipientUserID: uuid.MustParse(r.RecipientUserID),
		Type:            notificationdom.NotificationType(r.Type),
		ProjectID:       uuid.MustParse(r.ProjectID),
		ProjectName:     r.ProjectName,
		ReadAt:          r.ReadAt,
		CreatedAt:       r.CreatedAt,
	}
	if r.ActorMemberID != nil {
		id := uuid.MustParse(*r.ActorMemberID)
		n.ActorMemberID = &id
	}
	if r.ActorFullName != nil {
		n.ActorFullName = *r.ActorFullName
	}
	if r.ActorUsername != nil {
		n.ActorUsername = *r.ActorUsername
	}
	if r.TaskID != nil {
		id := uuid.MustParse(*r.TaskID)
		n.TaskID = &id
	}
	if r.TaskTitle != nil {
		n.TaskTitle = *r.TaskTitle
	}
	if r.TaskNumber != nil {
		n.TaskNumber = *r.TaskNumber
	}
	return n
}

// --- Repository methods -----------------------------------------------------

// Create persists a new notification.
func (r *NotificationRepository) Create(ctx context.Context, n *notificationdom.Notification) error {
	rec := notificationRecord{
		ID:              n.ID.String(),
		RecipientUserID: n.RecipientUserID.String(),
		Type:            string(n.Type),
		ProjectID:       n.ProjectID.String(),
		ReadAt:          n.ReadAt,
		CreatedAt:       n.CreatedAt,
	}
	if n.ActorMemberID != nil {
		s := n.ActorMemberID.String()
		rec.ActorMemberID = &s
	}
	if n.TaskID != nil {
		s := n.TaskID.String()
		rec.TaskID = &s
	}
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		return fmt.Errorf("notification repo: create: %w", err)
	}
	return nil
}

// ListForUser returns up to limit notifications for the given user, newest first.
func (r *NotificationRepository) ListForUser(ctx context.Context, userID uuid.UUID, limit int) ([]*notificationdom.Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []notificationReadRow
	err := r.db.WithContext(ctx).
		Table("notifications n").
		Select(notificationReadCols).
		Joins("LEFT JOIN project_members pm ON pm.id = n.actor_member_id").
		Joins("LEFT JOIN users u ON u.id = pm.user_id").
		Joins("LEFT JOIN tasks t ON t.id = n.task_id AND t.deleted_at IS NULL").
		Joins("JOIN projects p ON p.id = n.project_id").
		Where("n.recipient_user_id = ?", userID.String()).
		Order("n.created_at DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("notification repo: list: %w", err)
	}

	out := make([]*notificationdom.Notification, 0, len(rows))
	for _, row := range rows {
		out = append(out, notificationFromRow(row))
	}
	return out, nil
}

// UnreadCount returns the count of unread notifications for the given user.
func (r *NotificationRepository) UnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("notifications").
		Where("recipient_user_id = ? AND read_at IS NULL", userID.String()).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("notification repo: unread count: %w", err)
	}
	return count, nil
}

// MarkAsRead sets read_at on a notification owned by userID.
func (r *NotificationRepository) MarkAsRead(ctx context.Context, id, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Table("notifications").
		Where("id = ? AND recipient_user_id = ? AND read_at IS NULL", id.String(), userID.String()).
		UpdateColumn("read_at", time.Now())
	if result.Error != nil {
		return fmt.Errorf("notification repo: mark as read: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return notificationdom.ErrNotificationNotFound
	}
	return nil
}

// MarkAllAsRead sets read_at on all unread notifications for userID.
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Table("notifications").
		Where("recipient_user_id = ? AND read_at IS NULL", userID.String()).
		UpdateColumn("read_at", time.Now()).Error
	if err != nil {
		return fmt.Errorf("notification repo: mark all as read: %w", err)
	}
	return nil
}
