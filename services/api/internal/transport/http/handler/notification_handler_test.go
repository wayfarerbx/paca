package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	notificationdom "github.com/paca/api/internal/domain/notification"
	"github.com/paca/api/internal/transport/http/handler"
	"github.com/paca/api/internal/transport/http/middleware"
)

type mockNotificationSvc struct {
	mu sync.RWMutex

	listNotifications func(ctx context.Context, userID uuid.UUID, limit int) ([]*notificationdom.Notification, error)
	unreadCount       func(ctx context.Context, userID uuid.UUID) (int64, error)
	markAsRead        func(ctx context.Context, id, userID uuid.UUID) error
	markAllAsRead     func(ctx context.Context, userID uuid.UUID) error
}

func (m *mockNotificationSvc) NotifyAssigned(_ context.Context, _ notificationdom.NotifyAssignedInput) error {
	return nil
}

func (m *mockNotificationSvc) NotifyMentioned(_ context.Context, _ notificationdom.NotifyMentionedInput) error {
	return nil
}

func (m *mockNotificationSvc) ListNotifications(ctx context.Context, userID uuid.UUID, limit int) ([]*notificationdom.Notification, error) {
	if m.listNotifications != nil {
		return m.listNotifications(ctx, userID, limit)
	}
	return nil, nil
}

func (m *mockNotificationSvc) UnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	if m.unreadCount != nil {
		return m.unreadCount(ctx, userID)
	}
	return 0, nil
}

func (m *mockNotificationSvc) MarkAsRead(ctx context.Context, id, userID uuid.UUID) error {
	if m.markAsRead != nil {
		return m.markAsRead(ctx, id, userID)
	}
	return nil
}

func (m *mockNotificationSvc) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	if m.markAllAsRead != nil {
		return m.markAllAsRead(ctx, userID)
	}
	return nil
}

var _ notificationdom.Service = (*mockNotificationSvc)(nil)

func buildNotificationRouter(svc *mockNotificationSvc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := handler.NewNotificationHandler(svc)
	r := gin.New()
	g := r.Group("/users")
	g.GET("/me/notifications", h.List)
	g.PATCH("/me/notifications/:notificationId/read", h.MarkAsRead)
	g.POST("/me/notifications/read-all", h.MarkAllAsRead)
	return r
}

func doNotifRequest(r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequestWithContext(context.Background(), method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func doNotifRequestWithActor(r *gin.Engine, method, path string, body any, actorID uuid.UUID) *httptest.ResponseRecorder {
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequestWithContext(middleware.WithActorID(context.Background(), actorID), method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeNotificationEnvelope(t *testing.T, body []byte) (bool, string, json.RawMessage) {
	t.Helper()
	var env struct {
		Success   bool            `json:"success"`
		ErrorCode string          `json:"error_code"`
		Data      json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	return env.Success, env.ErrorCode, env.Data
}

func TestNotificationHandler_List_Unauthenticated(t *testing.T) {
	svc := &mockNotificationSvc{}
	r := buildNotificationRouter(svc)

	w := doNotifRequest(r, http.MethodGet, "/users/me/notifications", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	ok, code, _ := decodeNotificationEnvelope(t, w.Body.Bytes())
	if ok {
		t.Fatal("expected success=false")
	}
	if code != "AUTH_UNAUTHENTICATED" {
		t.Fatalf("expected error code AUTH_UNAUTHENTICATED, got %s", code)
	}
}

func TestNotificationHandler_List_Success(t *testing.T) {
	userID := uuid.New()
	now := time.Now().Truncate(time.Millisecond)
	taskID := uuid.New()
	notifs := []*notificationdom.Notification{
		{
			ID:              uuid.New(),
			RecipientUserID: userID,
			Type:            notificationdom.NotificationTypeAssigned,
			ActorFullName:   "Jane Doe",
			ActorUsername:   "janedoe",
			TaskID:          &taskID,
			TaskTitle:       "Do the thing",
			TaskNumber:      7,
			ProjectID:       uuid.New(),
			ProjectName:     "Paca",
			CreatedAt:       now,
		},
	}

	svc := &mockNotificationSvc{}
	svc.listNotifications = func(_ context.Context, gotUserID uuid.UUID, limit int) ([]*notificationdom.Notification, error) {
		if gotUserID != userID {
			t.Fatalf("expected user %s, got %s", userID, gotUserID)
		}
		if limit != 50 {
			t.Fatalf("expected limit 50, got %d", limit)
		}
		return notifs, nil
	}
	svc.unreadCount = func(_ context.Context, _ uuid.UUID) (int64, error) {
		return 3, nil
	}

	r := buildNotificationRouter(svc)
	w := doNotifRequestWithActor(r, http.MethodGet, "/users/me/notifications", nil, userID)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ok, _, data := decodeNotificationEnvelope(t, w.Body.Bytes())
	if !ok {
		t.Fatal("expected success=true")
	}

	var resp struct {
		Items []struct {
			ID            string `json:"id"`
			Type          string `json:"type"`
			ActorFullName string `json:"actor_full_name"`
			ActorUsername string `json:"actor_username"`
			TaskTitle     string `json:"task_title"`
			ProjectName   string `json:"project_name"`
		} `json:"items"`
		UnreadCount int64 `json:"unread_count"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.Items[0].Type != "assigned" {
		t.Fatalf("expected type assigned, got %s", resp.Items[0].Type)
	}
	if resp.Items[0].ActorFullName != "Jane Doe" {
		t.Fatalf("expected actor Jane Doe, got %s", resp.Items[0].ActorFullName)
	}
	if resp.UnreadCount != 3 {
		t.Fatalf("expected unread_count 3, got %d", resp.UnreadCount)
	}
}

func TestNotificationHandler_List_Empty(t *testing.T) {
	userID := uuid.New()
	svc := &mockNotificationSvc{}
	svc.listNotifications = func(_ context.Context, _ uuid.UUID, _ int) ([]*notificationdom.Notification, error) {
		return nil, nil
	}
	svc.unreadCount = func(_ context.Context, _ uuid.UUID) (int64, error) {
		return 0, nil
	}

	r := buildNotificationRouter(svc)
	w := doNotifRequestWithActor(r, http.MethodGet, "/users/me/notifications", nil, userID)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	_, _, data := decodeNotificationEnvelope(t, w.Body.Bytes())
	var resp struct {
		Items       []any `json:"items"`
		UnreadCount int64 `json:"unread_count"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(resp.Items))
	}
}

func TestNotificationHandler_List_ServiceError(t *testing.T) {
	userID := uuid.New()
	svc := &mockNotificationSvc{}
	svc.listNotifications = func(_ context.Context, _ uuid.UUID, _ int) ([]*notificationdom.Notification, error) {
		return nil, errors.New("boom")
	}

	r := buildNotificationRouter(svc)
	w := doNotifRequestWithActor(r, http.MethodGet, "/users/me/notifications", nil, userID)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotificationHandler_List_UnreadCountError(t *testing.T) {
	userID := uuid.New()
	svc := &mockNotificationSvc{}
	svc.listNotifications = func(_ context.Context, _ uuid.UUID, _ int) ([]*notificationdom.Notification, error) {
		return nil, nil
	}
	svc.unreadCount = func(_ context.Context, _ uuid.UUID) (int64, error) {
		return 0, errors.New("unread count failure")
	}

	r := buildNotificationRouter(svc)
	w := doNotifRequestWithActor(r, http.MethodGet, "/users/me/notifications", nil, userID)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotificationHandler_MarkAsRead_Unauthenticated(t *testing.T) {
	svc := &mockNotificationSvc{}
	r := buildNotificationRouter(svc)

	w := doNotifRequest(r, http.MethodPatch, fmt.Sprintf("/users/me/notifications/%s/read", uuid.New()), nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	ok, code, _ := decodeNotificationEnvelope(t, w.Body.Bytes())
	if ok {
		t.Fatal("expected success=false")
	}
	if code != "AUTH_UNAUTHENTICATED" {
		t.Fatalf("expected error code AUTH_UNAUTHENTICATED, got %s", code)
	}
}

func TestNotificationHandler_MarkAsRead_InvalidUUID(t *testing.T) {
	svc := &mockNotificationSvc{}
	r := buildNotificationRouter(svc)
	userID := uuid.New()

	w := doNotifRequestWithActor(r, http.MethodPatch, "/users/me/notifications/not-a-uuid/read", nil, userID)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	ok, code, _ := decodeNotificationEnvelope(t, w.Body.Bytes())
	if ok {
		t.Fatal("expected success=false")
	}
	if code != "BAD_REQUEST" {
		t.Fatalf("expected error code BAD_REQUEST, got %s", code)
	}
}

func TestNotificationHandler_MarkAsRead_Success(t *testing.T) {
	userID := uuid.New()
	notifID := uuid.New()

	var capturedID, capturedUserID uuid.UUID
	svc := &mockNotificationSvc{}
	svc.markAsRead = func(_ context.Context, id, uid uuid.UUID) error {
		capturedID = id
		capturedUserID = uid
		return nil
	}

	r := buildNotificationRouter(svc)
	w := doNotifRequestWithActor(r, http.MethodPatch, fmt.Sprintf("/users/me/notifications/%s/read", notifID), nil, userID)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if capturedID != notifID {
		t.Fatalf("expected notification id %s, got %s", notifID, capturedID)
	}
	if capturedUserID != userID {
		t.Fatalf("expected user id %s, got %s", userID, capturedUserID)
	}
}

func TestNotificationHandler_MarkAsRead_NotFoundError(t *testing.T) {
	userID := uuid.New()
	notifID := uuid.New()

	svc := &mockNotificationSvc{}
	svc.markAsRead = func(_ context.Context, _, _ uuid.UUID) error {
		return notificationdom.ErrNotificationNotFound
	}

	r := buildNotificationRouter(svc)
	w := doNotifRequestWithActor(r, http.MethodPatch, fmt.Sprintf("/users/me/notifications/%s/read", notifID), nil, userID)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	ok, code, _ := decodeNotificationEnvelope(t, w.Body.Bytes())
	if ok {
		t.Fatal("expected success=false")
	}
	if code != "NOTIFICATION_NOT_FOUND" {
		t.Fatalf("expected error code NOTIFICATION_NOT_FOUND, got %s", code)
	}
}

func TestNotificationHandler_MarkAllAsRead_Unauthenticated(t *testing.T) {
	svc := &mockNotificationSvc{}
	r := buildNotificationRouter(svc)

	w := doNotifRequest(r, http.MethodPost, "/users/me/notifications/read-all", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	ok, code, _ := decodeNotificationEnvelope(t, w.Body.Bytes())
	if ok {
		t.Fatal("expected success=false")
	}
	if code != "AUTH_UNAUTHENTICATED" {
		t.Fatalf("expected error code AUTH_UNAUTHENTICATED, got %s", code)
	}
}

func TestNotificationHandler_MarkAllAsRead_Success(t *testing.T) {
	userID := uuid.New()

	var capturedUserID uuid.UUID
	svc := &mockNotificationSvc{}
	svc.markAllAsRead = func(_ context.Context, uid uuid.UUID) error {
		capturedUserID = uid
		return nil
	}

	r := buildNotificationRouter(svc)
	w := doNotifRequestWithActor(r, http.MethodPost, "/users/me/notifications/read-all", nil, userID)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if capturedUserID != userID {
		t.Fatalf("expected user id %s, got %s", userID, capturedUserID)
	}
}

func TestNotificationHandler_MarkAllAsRead_ServiceError(t *testing.T) {
	userID := uuid.New()

	svc := &mockNotificationSvc{}
	svc.markAllAsRead = func(_ context.Context, _ uuid.UUID) error {
		return errors.New("db down")
	}

	r := buildNotificationRouter(svc)
	w := doNotifRequestWithActor(r, http.MethodPost, "/users/me/notifications/read-all", nil, userID)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}
