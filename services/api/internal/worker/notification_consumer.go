package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	notificationdom "github.com/paca/api/internal/domain/notification"
	"github.com/paca/api/internal/events"
	"github.com/redis/go-redis/v9"
)

const (
	notificationConsumerGroup = "api.notification_writer"
	notificationReadBlock     = 5 * time.Second
	notificationReadCount     = 50
)

// NotificationConsumer reads task-assignment events from StreamTaskAssignments
// and creates in-app notifications via the notification service.
//
// The notification service internally resolves member IDs, prevents
// self-notifications, and publishes real-time events to the Valkey Pub/Sub
// channel so the realtime service can push them to connected clients.
type NotificationConsumer struct {
	client          *redis.Client
	notificationSvc notificationdom.Service
	log             *slog.Logger
	consumerName    string
	stopCh          chan struct{}
	doneCh          chan struct{}
}

// NewNotificationConsumer creates a consumer that is ready to be started.
func NewNotificationConsumer(client *redis.Client, notificationSvc notificationdom.Service, log *slog.Logger) *NotificationConsumer {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = uuid.New().String()
	}
	return &NotificationConsumer{
		client:          client,
		notificationSvc: notificationSvc,
		log:             log,
		consumerName:    fmt.Sprintf("%s.%s", notificationConsumerGroup, hostname),
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
	}
}

// Start creates the consumer group if needed and begins processing in a
// background goroutine. Call Stop to drain and exit cleanly.
func (c *NotificationConsumer) Start(ctx context.Context) {
	err := c.client.XGroupCreateMkStream(ctx, events.StreamTaskAssignments, notificationConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		c.log.Warn("notification consumer: could not create consumer group", "err", err)
	}

	go c.run()
}

// Stop signals the consumer to stop and waits for the goroutine to exit.
func (c *NotificationConsumer) Stop() {
	close(c.stopCh)
	<-c.doneCh
}

func (c *NotificationConsumer) run() {
	defer close(c.doneCh)
	c.log.Info("notification consumer: started", "stream", events.StreamTaskAssignments)

	c.processPending(context.Background())

	for {
		select {
		case <-c.stopCh:
			c.log.Info("notification consumer: stopping")
			return
		default:
		}

		ctx, cancel := context.WithTimeout(context.Background(), notificationReadBlock+time.Second)
		msgs, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    notificationConsumerGroup,
			Consumer: c.consumerName,
			Streams:  []string{events.StreamTaskAssignments, ">"},
			Count:    notificationReadCount,
			Block:    notificationReadBlock,
		}).Result()
		cancel()

		if err != nil {
			if err == redis.Nil {
				continue
			}
			c.log.Error("notification consumer: xreadgroup error", "err", err)
			time.Sleep(2 * time.Second)
			continue
		}

		for _, stream := range msgs {
			for _, msg := range stream.Messages {
				c.handle(msg)
			}
		}
	}
}

func (c *NotificationConsumer) processPending(ctx context.Context) {
	msgs, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    notificationConsumerGroup,
		Consumer: c.consumerName,
		Streams:  []string{events.StreamTaskAssignments, "0"},
		Count:    notificationReadCount,
	}).Result()
	if err != nil && err != redis.Nil {
		c.log.Warn("notification consumer: could not read pending messages", "err", err)
		return
	}
	for _, stream := range msgs {
		for _, msg := range stream.Messages {
			c.handle(msg)
		}
	}
}

// assignmentStreamPayload mirrors the JSON shape produced by the task handler
// when it appends to StreamTaskAssignments.
type assignmentStreamPayload struct {
	TaskID              string `json:"task_id"`
	ProjectID           string `json:"project_id"`
	NewAssigneeMemberID string `json:"new_assignee_member_id"`
	OldAssigneeMemberID string `json:"old_assignee_member_id,omitempty"`
	ActorUserID         string `json:"actor_user_id"`
}

func (c *NotificationConsumer) handle(msg redis.XMessage) {
	ctx := context.Background()

	raw, ok := msg.Values["payload"].(string)
	if !ok {
		c.log.Warn("notification consumer: message has no payload field", "id", msg.ID)
		c.ack(ctx, msg.ID)
		return
	}

	var p assignmentStreamPayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		c.log.Warn("notification consumer: failed to decode payload", "id", msg.ID, "err", err)
		c.ack(ctx, msg.ID)
		return
	}

	taskID, err := uuid.Parse(p.TaskID)
	if err != nil {
		c.log.Warn("notification consumer: invalid task_id", "id", msg.ID, "task_id", p.TaskID)
		c.ack(ctx, msg.ID)
		return
	}
	projectID, err := uuid.Parse(p.ProjectID)
	if err != nil {
		c.log.Warn("notification consumer: invalid project_id", "id", msg.ID)
		c.ack(ctx, msg.ID)
		return
	}
	newAssigneeMemberID, err := uuid.Parse(p.NewAssigneeMemberID)
	if err != nil {
		c.log.Warn("notification consumer: invalid new_assignee_member_id", "id", msg.ID)
		c.ack(ctx, msg.ID)
		return
	}
	actorUserID, err := uuid.Parse(p.ActorUserID)
	if err != nil {
		c.log.Warn("notification consumer: invalid actor_user_id", "id", msg.ID)
		c.ack(ctx, msg.ID)
		return
	}

	if err := c.notificationSvc.NotifyAssigned(ctx, notificationdom.NotifyAssignedInput{
		TaskID:              taskID,
		ProjectID:           projectID,
		NewAssigneeMemberID: newAssigneeMemberID,
		ActorUserID:         actorUserID,
	}); err != nil {
		c.log.Error("notification consumer: NotifyAssigned failed", "id", msg.ID, "err", err)
		// Do not ack — will be retried via processPending on next restart.
		return
	}

	c.ack(ctx, msg.ID)
}

func (c *NotificationConsumer) ack(ctx context.Context, id string) {
	if err := c.client.XAck(ctx, events.StreamTaskAssignments, notificationConsumerGroup, id).Err(); err != nil {
		c.log.Warn("notification consumer: xack failed", "id", id, "err", err)
	}
}
