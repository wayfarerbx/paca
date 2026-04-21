package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/paca/api/internal/apierr"
	notificationdom "github.com/paca/api/internal/domain/notification"
	"github.com/paca/api/internal/transport/http/dto"
	"github.com/paca/api/internal/transport/http/middleware"
	"github.com/paca/api/internal/transport/http/presenter"
)

const defaultNotificationLimit = 50

// NotificationHandler handles notification endpoints.
type NotificationHandler struct {
	svc notificationdom.Service
}

// NewNotificationHandler returns a NotificationHandler wired to svc.
func NewNotificationHandler(svc notificationdom.Service) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// List handles GET /users/me/notifications.
// Returns the most recent notifications for the authenticated user together
// with the current unread count.
func (h *NotificationHandler) List(c *gin.Context) {
	userID, ok := middleware.ActorIDFromContext(c.Request.Context())
	if !ok {
		presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
		return
	}

	notifications, err := h.svc.ListNotifications(c.Request.Context(), userID, defaultNotificationLimit)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	unreadCount, err := h.svc.UnreadCount(c.Request.Context(), userID)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	items := make([]dto.NotificationResponse, 0, len(notifications))
	for _, n := range notifications {
		items = append(items, dto.NotificationFromEntity(n))
	}

	presenter.OK(c, dto.NotificationListResponse{
		Items:       items,
		UnreadCount: unreadCount,
	})
}

// MarkAsRead handles PATCH /users/me/notifications/:notificationId/read.
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID, ok := middleware.ActorIDFromContext(c.Request.Context())
	if !ok {
		presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
		return
	}

	notificationID, err := uuid.Parse(c.Param("notificationId"))
	if err != nil {
		presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid notification id"))
		return
	}

	if err := h.svc.MarkAsRead(c.Request.Context(), notificationID, userID); err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.NoContent(c)
}

// MarkAllAsRead handles POST /users/me/notifications/read-all.
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, ok := middleware.ActorIDFromContext(c.Request.Context())
	if !ok {
		presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
		return
	}

	if err := h.svc.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.NoContent(c)
}
