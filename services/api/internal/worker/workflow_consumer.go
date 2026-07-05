package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	taskdom "github.com/Paca-AI/api/internal/domain/task"
	userdom "github.com/Paca-AI/api/internal/domain/user"
	workflowdom "github.com/Paca-AI/api/internal/domain/workflow"
	"github.com/Paca-AI/api/internal/events"
	"github.com/Paca-AI/api/internal/platform/messaging"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	workflowConsumerGroup = "api.workflow_engine"
	workflowReadBlock     = 5 * time.Second
	workflowReadCount     = 50
)

// workflowGraphReader is the minimal workflowdom.Repository surface the
// engine needs to walk the graph and evaluate rules.
type workflowGraphReader interface {
	FindWorkflowByID(ctx context.Context, id uuid.UUID) (*workflowdom.Workflow, error)
	FindNodeByID(ctx context.Context, id uuid.UUID) (*workflowdom.Node, error)
	ListActiveNodesByTaskID(ctx context.Context, taskID uuid.UUID) ([]*workflowdom.Node, error)
	ListStatusRulesByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]*workflowdom.StatusRule, error)
	ListStatusTransitionsByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]*workflowdom.StatusTransition, error)
	ListEdgesByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]*workflowdom.Edge, error)
	ListIncomingEdges(ctx context.Context, targetNodeID uuid.UUID) ([]*workflowdom.Edge, error)
}

// workflowTaskReader is the minimal task-domain surface the engine needs to
// read authoritative task/status state (the activity stream payload only
// carries resolved status *names*, not IDs).
type workflowTaskReader interface {
	FindTaskByID(ctx context.Context, id uuid.UUID) (*taskdom.Task, error)
	FindTaskStatusByID(ctx context.Context, id uuid.UUID) (*taskdom.TaskStatus, error)
}

// workflowTaskUpdater applies an assignment change through the normal task
// service, so it gets the same validation and side effects as a human PATCH.
type workflowTaskUpdater interface {
	UpdateTask(ctx context.Context, projectID, id uuid.UUID, in taskdom.UpdateTaskInput) (*taskdom.Task, error)
}

// workflowActivityRecorder posts the workflow.assigned activity entry.
type workflowActivityRecorder interface {
	RecordActivity(ctx context.Context, in taskdom.RecordActivityInput) error
}

// WorkflowConsumer reads task-activity events from StreamTaskActivities and
// evaluates automation workflows whenever a task's status changes:
//
//   - Event 1 (status changed): the task's new status is looked up in the
//     workflow's status rules and, if found, the task is reassigned.
//   - Event 2 (predecessor done): once a node's task reaches the workflow's
//     derived done status (see isNodeDone) — and, for nodes with multiple
//     incoming edges, once ALL predecessors have — each downstream node's
//     task is reassigned using its OWN current status against the same
//     workflow rules.
//
// Both events reuse the same status->assignee lookup (applyStatusRule); the
// only difference is which (node, task) pair is being evaluated. Because the
// workflow graph is a DAG (enforced at edge-creation time) and can't change
// while active, cascades are guaranteed to terminate.
type WorkflowConsumer struct {
	client       *redis.Client
	workflowRepo workflowGraphReader
	taskRepo     workflowTaskReader
	taskSvc      workflowTaskUpdater
	activityRec  workflowActivityRecorder
	publisher    *messaging.Publisher
	log          *slog.Logger
	consumerName string
	stopCh       chan struct{}
	doneCh       chan struct{}
}

// NewWorkflowConsumer creates a consumer that is ready to be started.
func NewWorkflowConsumer(
	client *redis.Client,
	workflowRepo workflowGraphReader,
	taskRepo workflowTaskReader,
	taskSvc workflowTaskUpdater,
	activityRec workflowActivityRecorder,
	publisher *messaging.Publisher,
	log *slog.Logger,
) *WorkflowConsumer {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = uuid.New().String()
	}
	return &WorkflowConsumer{
		client:       client,
		workflowRepo: workflowRepo,
		taskRepo:     taskRepo,
		taskSvc:      taskSvc,
		activityRec:  activityRec,
		publisher:    publisher,
		log:          log,
		consumerName: fmt.Sprintf("%s.%s", workflowConsumerGroup, hostname),
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}
}

// Start creates the consumer group if needed and begins processing in a
// background goroutine. Call Stop to drain and exit cleanly.
func (c *WorkflowConsumer) Start(ctx context.Context) {
	err := c.client.XGroupCreateMkStream(ctx, events.StreamTaskActivities, workflowConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		c.log.Warn("workflow consumer: could not create consumer group", "err", err)
	}
	go c.run()
}

// Stop signals the consumer to stop and waits for the goroutine to exit.
func (c *WorkflowConsumer) Stop() {
	close(c.stopCh)
	<-c.doneCh
}

func (c *WorkflowConsumer) run() {
	defer close(c.doneCh)
	c.log.Info("workflow consumer: started", "stream", events.StreamTaskActivities)

	c.processPending(context.Background())

	for {
		select {
		case <-c.stopCh:
			c.log.Info("workflow consumer: stopping")
			return
		default:
		}

		ctx, cancel := context.WithTimeout(context.Background(), workflowReadBlock+time.Second)
		msgs, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    workflowConsumerGroup,
			Consumer: c.consumerName,
			Streams:  []string{events.StreamTaskActivities, ">"},
			Count:    workflowReadCount,
			Block:    workflowReadBlock,
		}).Result()
		cancel()

		if err != nil {
			if err == redis.Nil {
				continue
			}
			c.log.Error("workflow consumer: xreadgroup error", "err", err)
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

func (c *WorkflowConsumer) processPending(ctx context.Context) {
	msgs, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    workflowConsumerGroup,
		Consumer: c.consumerName,
		Streams:  []string{events.StreamTaskActivities, "0"},
		Count:    workflowReadCount,
	}).Result()
	if err != nil && err != redis.Nil {
		c.log.Warn("workflow consumer: could not read pending messages", "err", err)
		return
	}
	for _, stream := range msgs {
		for _, msg := range stream.Messages {
			c.handle(msg)
		}
	}
}

// workflowActivityStreamPayload mirrors the JSON shape produced by
// activityPayload() in activity_service.go — the same payload ActivityConsumer
// decodes, so only the fields this consumer needs are declared here.
type workflowActivityStreamPayload struct {
	TaskID       string `json:"task_id"`
	ProjectID    string `json:"project_id"`
	ActivityType string `json:"activity_type"`
	Content      string `json:"content"`
}

// taskUpdatedContent mirrors the {"changes": [...]} shape task_handler.go
// marshals into a task.updated activity's Content.
type taskUpdatedContent struct {
	Changes []taskdom.FieldChange `json:"changes"`
}

func (p workflowActivityStreamPayload) hasStatusChange() bool {
	var content taskUpdatedContent
	if err := json.Unmarshal([]byte(p.Content), &content); err != nil {
		return false
	}
	for _, c := range content.Changes {
		if c.Field == "status" {
			return true
		}
	}
	return false
}

func (c *WorkflowConsumer) handle(msg redis.XMessage) {
	ctx := context.Background()

	raw, ok := msg.Values["payload"].(string)
	if !ok {
		c.ack(ctx, msg.ID)
		return
	}

	var p workflowActivityStreamPayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		c.log.Warn("workflow consumer: failed to decode payload", "id", msg.ID, "err", err)
		c.ack(ctx, msg.ID)
		return
	}

	// Only task.updated events with a status field change can affect a
	// workflow — everything else (comments, links, other field edits) is a
	// cheap no-op skip.
	if p.ActivityType != string(taskdom.ActivityTypeTaskUpdated) || !p.hasStatusChange() {
		c.ack(ctx, msg.ID)
		return
	}

	taskID, err := uuid.Parse(p.TaskID)
	if err != nil {
		c.log.Warn("workflow consumer: invalid task_id", "id", msg.ID)
		c.ack(ctx, msg.ID)
		return
	}
	projectID, err := uuid.Parse(p.ProjectID)
	if err != nil {
		c.log.Warn("workflow consumer: invalid project_id", "id", msg.ID)
		c.ack(ctx, msg.ID)
		return
	}

	if err := c.processTaskStatusChange(ctx, projectID, taskID); err != nil {
		c.log.Error("workflow consumer: failed to process task status change", "id", msg.ID, "task_id", taskID, "err", err)
		// Do not ack — retried via processPending on next restart.
		return
	}
	c.ack(ctx, msg.ID)
}

func (c *WorkflowConsumer) ack(ctx context.Context, id string) {
	if err := c.client.XAck(ctx, events.StreamTaskActivities, workflowConsumerGroup, id).Err(); err != nil {
		c.log.Warn("workflow consumer: xack failed", "id", id, "err", err)
	}
}

// processTaskStatusChange re-evaluates every active-workflow node that
// references taskID. Errors applying an individual node are logged and
// skipped (best-effort) rather than aborting the whole batch — a bug
// affecting one workflow shouldn't block others. An error is returned only
// when the foundational lookups (nodes, task) fail, so the message is
// retried.
func (c *WorkflowConsumer) processTaskStatusChange(ctx context.Context, projectID, taskID uuid.UUID) error {
	nodes, err := c.workflowRepo.ListActiveNodesByTaskID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("list active nodes: %w", err)
	}
	if len(nodes) == 0 {
		return nil
	}

	task, err := c.taskRepo.FindTaskByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("find task: %w", err)
	}
	if task.StatusID == nil {
		return nil
	}

	for _, node := range nodes {
		if err := c.applyNode(ctx, projectID, node, task); err != nil {
			c.log.Error("workflow consumer: failed to apply node", "node_id", node.ID, "workflow_id", node.WorkflowID, "task_id", taskID, "err", err)
		}
	}
	return nil
}

// applyNode runs event 1 (this node's own status rule) and, if the node just
// became done, fans out event 2 to its outgoing edges.
func (c *WorkflowConsumer) applyNode(ctx context.Context, projectID uuid.UUID, node *workflowdom.Node, task *taskdom.Task) error {
	if err := c.applyStatusRule(ctx, projectID, node, task, "status_rule"); err != nil {
		return fmt.Errorf("apply status rule: %w", err)
	}

	done, err := c.isNodeDone(ctx, node, task)
	if err != nil {
		return fmt.Errorf("check done status: %w", err)
	}
	if !done {
		return nil
	}

	edges, err := c.workflowRepo.ListEdgesByWorkflow(ctx, node.WorkflowID)
	if err != nil {
		return fmt.Errorf("list edges: %w", err)
	}
	for _, e := range edges {
		if e.SourceNodeID != node.ID {
			continue
		}
		if err := c.tryFireEdge(ctx, projectID, e); err != nil {
			c.log.Error("workflow consumer: failed to fire edge", "edge_id", e.ID, "err", err)
		}
	}
	return nil
}

// isNodeDone reports whether task's current status equals node's workflow's
// derived done status — the one status-transition entry with no configured
// next status. If the chain has no unique terminal entry (shouldn't happen
// for an active workflow, since Activate enforces it, but handled
// defensively), the node is treated as not done rather than erroring.
func (c *WorkflowConsumer) isNodeDone(ctx context.Context, node *workflowdom.Node, task *taskdom.Task) (bool, error) {
	if task.StatusID == nil {
		return false, nil
	}
	transitions, err := c.workflowRepo.ListStatusTransitionsByWorkflow(ctx, node.WorkflowID)
	if err != nil {
		return false, err
	}
	doneStatusID, ok := workflowdom.DeriveDoneStatusID(transitions)
	if !ok {
		return false, nil
	}
	return *task.StatusID == doneStatusID, nil
}

// tryFireEdge evaluates the AND-join for edge's target node — ALL of the
// target's incoming edges must have a done source — and, if satisfied,
// applies event 2 using the target task's own current status.
func (c *WorkflowConsumer) tryFireEdge(ctx context.Context, projectID uuid.UUID, edge *workflowdom.Edge) error {
	target, err := c.workflowRepo.FindNodeByID(ctx, edge.TargetNodeID)
	if err != nil {
		return fmt.Errorf("find target node: %w", err)
	}

	incoming, err := c.workflowRepo.ListIncomingEdges(ctx, target.ID)
	if err != nil {
		return fmt.Errorf("list incoming edges: %w", err)
	}
	for _, in := range incoming {
		srcNode, err := c.workflowRepo.FindNodeByID(ctx, in.SourceNodeID)
		if err != nil {
			return fmt.Errorf("find source node: %w", err)
		}
		srcTask, err := c.taskRepo.FindTaskByID(ctx, srcNode.TaskID)
		if err != nil {
			return fmt.Errorf("find source task: %w", err)
		}
		done, err := c.isNodeDone(ctx, srcNode, srcTask)
		if err != nil {
			return fmt.Errorf("check predecessor done: %w", err)
		}
		if !done {
			// AND-join not satisfied yet — this is not an error, just not
			// time to fire. It will be re-evaluated when the remaining
			// predecessor(s) also reach done.
			return nil
		}
	}

	targetTask, err := c.taskRepo.FindTaskByID(ctx, target.TaskID)
	if err != nil {
		return fmt.Errorf("find target task: %w", err)
	}
	return c.applyStatusRule(ctx, projectID, target, targetTask, "predecessor_done")
}

// applyStatusRule looks up node's rule for task's current status and, if one
// exists and the assignee actually differs (idempotent no-op otherwise),
// reassigns the task through the normal task service, records a
// workflow.assigned activity, and publishes to StreamTaskAssignments so the
// existing notification/agent-trigger pipeline picks it up uniformly.
func (c *WorkflowConsumer) applyStatusRule(ctx context.Context, projectID uuid.UUID, node *workflowdom.Node, task *taskdom.Task, reason string) error {
	if task.StatusID == nil {
		return nil
	}
	rules, err := c.workflowRepo.ListStatusRulesByWorkflow(ctx, node.WorkflowID)
	if err != nil {
		return fmt.Errorf("list status rules: %w", err)
	}
	var rule *workflowdom.StatusRule
	for _, r := range rules {
		if r.StatusID == *task.StatusID {
			rule = r
			break
		}
	}
	if rule == nil {
		return nil
	}
	if task.AssigneeID != nil && *task.AssigneeID == rule.AssigneeMemberID {
		return nil // already assigned — idempotent no-op
	}

	workflow, err := c.workflowRepo.FindWorkflowByID(ctx, node.WorkflowID)
	if err != nil {
		return fmt.Errorf("find workflow: %w", err)
	}

	oldAssignee := task.AssigneeID
	newAssignee := rule.AssigneeMemberID
	newAssigneePtr := &newAssignee
	if _, err := c.taskSvc.UpdateTask(ctx, projectID, task.ID, taskdom.UpdateTaskInput{AssigneeID: &newAssigneePtr}); err != nil {
		return fmt.Errorf("update task assignee: %w", err)
	}

	if c.activityRec != nil {
		content, _ := json.Marshal(map[string]any{
			"workflow_id":   workflow.ID,
			"workflow_name": workflow.Name,
			"reason":        reason,
			"old_assignee":  oldAssignee,
			"new_assignee":  newAssignee,
		})
		_ = c.activityRec.RecordActivity(ctx, taskdom.RecordActivityInput{
			TaskID:       task.ID,
			ProjectID:    projectID,
			ActivityType: taskdom.ActivityTypeWorkflowAssigned,
			Content:      content,
		})
	}

	if c.publisher != nil {
		payload := map[string]any{
			"task_id":                task.ID.String(),
			"project_id":             projectID.String(),
			"new_assignee_member_id": newAssignee.String(),
			"actor_user_id":          userdom.SystemActorUserID.String(),
			"workflow_id":            workflow.ID.String(),
			"workflow_name":          workflow.Name,
		}
		if oldAssignee != nil {
			payload["old_assignee_member_id"] = oldAssignee.String()
		}
		if transitions, err := c.workflowRepo.ListStatusTransitionsByWorkflow(ctx, node.WorkflowID); err == nil {
			for _, tr := range transitions {
				if tr.StatusID == *task.StatusID && tr.NextStatusID != nil {
					if status, err := c.taskRepo.FindTaskStatusByID(ctx, *tr.NextStatusID); err == nil {
						payload["next_status_name"] = status.Name
					}
					break
				}
			}
		}
		_ = c.publisher.Append(ctx, events.StreamTaskAssignments, "task.assigned", payload)
	}

	return nil
}
