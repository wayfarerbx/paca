package worker

import (
	"context"
	"io"
	"log/slog"
	"testing"

	taskdom "github.com/Paca-AI/api/internal/domain/task"
	workflowdom "github.com/Paca-AI/api/internal/domain/workflow"
	"github.com/google/uuid"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

type fakeGraphStore struct {
	workflows   map[uuid.UUID]*workflowdom.Workflow
	nodes       map[uuid.UUID]*workflowdom.Node
	rules       map[uuid.UUID][]*workflowdom.StatusRule       // keyed by workflow ID
	transitions map[uuid.UUID][]*workflowdom.StatusTransition // keyed by workflow ID
	edges       []*workflowdom.Edge
}

func newFakeGraphStore() *fakeGraphStore {
	return &fakeGraphStore{
		workflows:   make(map[uuid.UUID]*workflowdom.Workflow),
		nodes:       make(map[uuid.UUID]*workflowdom.Node),
		rules:       make(map[uuid.UUID][]*workflowdom.StatusRule),
		transitions: make(map[uuid.UUID][]*workflowdom.StatusTransition),
	}
}

func (f *fakeGraphStore) FindWorkflowByID(_ context.Context, id uuid.UUID) (*workflowdom.Workflow, error) {
	w, ok := f.workflows[id]
	if !ok {
		return nil, workflowdom.ErrNotFound
	}
	return w, nil
}

func (f *fakeGraphStore) FindNodeByID(_ context.Context, id uuid.UUID) (*workflowdom.Node, error) {
	n, ok := f.nodes[id]
	if !ok {
		return nil, workflowdom.ErrNodeNotFound
	}
	return n, nil
}

func (f *fakeGraphStore) ListActiveNodesByTaskID(_ context.Context, taskID uuid.UUID) ([]*workflowdom.Node, error) {
	var out []*workflowdom.Node
	for _, n := range f.nodes {
		if n.TaskID != taskID {
			continue
		}
		if w, ok := f.workflows[n.WorkflowID]; ok && w.Status == workflowdom.StatusActive {
			out = append(out, n)
		}
	}
	return out, nil
}

func (f *fakeGraphStore) ListStatusRulesByWorkflow(_ context.Context, workflowID uuid.UUID) ([]*workflowdom.StatusRule, error) {
	return f.rules[workflowID], nil
}

func (f *fakeGraphStore) ListStatusTransitionsByWorkflow(_ context.Context, workflowID uuid.UUID) ([]*workflowdom.StatusTransition, error) {
	return f.transitions[workflowID], nil
}

func (f *fakeGraphStore) ListEdgesByWorkflow(_ context.Context, workflowID uuid.UUID) ([]*workflowdom.Edge, error) {
	var out []*workflowdom.Edge
	for _, e := range f.edges {
		if e.WorkflowID == workflowID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeGraphStore) ListIncomingEdges(_ context.Context, targetNodeID uuid.UUID) ([]*workflowdom.Edge, error) {
	var out []*workflowdom.Edge
	for _, e := range f.edges {
		if e.TargetNodeID == targetNodeID {
			out = append(out, e)
		}
	}
	return out, nil
}

// fakeTaskStore implements both workflowTaskReader and workflowTaskUpdater.
type fakeTaskStore struct {
	tasks       map[uuid.UUID]*taskdom.Task
	statuses    map[uuid.UUID]*taskdom.TaskStatus
	updateCalls int
}

func newFakeTaskStore() *fakeTaskStore {
	return &fakeTaskStore{
		tasks:    make(map[uuid.UUID]*taskdom.Task),
		statuses: make(map[uuid.UUID]*taskdom.TaskStatus),
	}
}

func (f *fakeTaskStore) FindTaskByID(_ context.Context, id uuid.UUID) (*taskdom.Task, error) {
	t, ok := f.tasks[id]
	if !ok {
		return nil, taskdom.ErrTaskNotFound
	}
	cp := *t
	return &cp, nil
}

func (f *fakeTaskStore) FindTaskStatusByID(_ context.Context, id uuid.UUID) (*taskdom.TaskStatus, error) {
	s, ok := f.statuses[id]
	if !ok {
		return nil, taskdom.ErrStatusNotFound
	}
	return s, nil
}

func (f *fakeTaskStore) UpdateTask(_ context.Context, _, id uuid.UUID, in taskdom.UpdateTaskInput) (*taskdom.Task, error) {
	f.updateCalls++
	t, ok := f.tasks[id]
	if !ok {
		return nil, taskdom.ErrTaskNotFound
	}
	if in.AssigneeID != nil {
		t.AssigneeID = *in.AssigneeID
	}
	cp := *t
	return &cp, nil
}

type fakeActivityRecorder struct{ calls int }

func (f *fakeActivityRecorder) RecordActivity(_ context.Context, _ taskdom.RecordActivityInput) error {
	f.calls++
	return nil
}

// ---------------------------------------------------------------------------
// Test fixture
// ---------------------------------------------------------------------------

type engineFixture struct {
	graph    *fakeGraphStore
	tasks    *fakeTaskStore
	activity *fakeActivityRecorder
	consumer *WorkflowConsumer

	projectID uuid.UUID
	doneStatus,
	readyStatus *taskdom.TaskStatus
}

func newEngineFixture() *engineFixture {
	graph := newFakeGraphStore()
	tasks := newFakeTaskStore()
	activity := &fakeActivityRecorder{}
	projectID := uuid.New()

	doneStatus := &taskdom.TaskStatus{ID: uuid.New(), ProjectID: projectID, Name: "Done", Category: taskdom.StatusCategoryDone}
	readyStatus := &taskdom.TaskStatus{ID: uuid.New(), ProjectID: projectID, Name: "Ready", Category: taskdom.StatusCategoryTodo}
	tasks.statuses[doneStatus.ID] = doneStatus
	tasks.statuses[readyStatus.ID] = readyStatus

	return &engineFixture{
		graph:       graph,
		tasks:       tasks,
		activity:    activity,
		projectID:   projectID,
		doneStatus:  doneStatus,
		readyStatus: readyStatus,
		consumer: &WorkflowConsumer{
			workflowRepo: graph,
			taskRepo:     tasks,
			taskSvc:      tasks,
			activityRec:  activity,
			log:          discardLogger(),
		},
	}
}

func (f *engineFixture) addWorkflow() *workflowdom.Workflow {
	w := &workflowdom.Workflow{ID: uuid.New(), ProjectID: f.projectID, Status: workflowdom.StatusActive, Name: "wf"}
	f.graph.workflows[w.ID] = w
	return w
}

func (f *engineFixture) addNode(w *workflowdom.Workflow, statusID *uuid.UUID) (*workflowdom.Node, *taskdom.Task) {
	task := &taskdom.Task{ID: uuid.New(), ProjectID: f.projectID, StatusID: statusID}
	f.tasks.tasks[task.ID] = task
	node := &workflowdom.Node{ID: uuid.New(), WorkflowID: w.ID, TaskID: task.ID}
	f.graph.nodes[node.ID] = node
	return node, task
}

func (f *engineFixture) addRule(w *workflowdom.Workflow, statusID, memberID uuid.UUID) {
	f.graph.rules[w.ID] = append(f.graph.rules[w.ID], &workflowdom.StatusRule{
		ID: uuid.New(), WorkflowID: w.ID, StatusID: statusID, AssigneeMemberID: memberID,
	})
}

// addTransition sets, for the workflow as a whole, what status comes next
// after statusID. nextStatusID nil marks statusID as the workflow's
// terminal/done status.
func (f *engineFixture) addTransition(w *workflowdom.Workflow, statusID uuid.UUID, nextStatusID *uuid.UUID) {
	f.graph.transitions[w.ID] = append(f.graph.transitions[w.ID], &workflowdom.StatusTransition{
		ID: uuid.New(), WorkflowID: w.ID, StatusID: statusID, NextStatusID: nextStatusID,
	})
}

func (f *engineFixture) addEdge(w *workflowdom.Workflow, source, target *workflowdom.Node) {
	f.graph.edges = append(f.graph.edges, &workflowdom.Edge{
		ID: uuid.New(), WorkflowID: w.ID, SourceNodeID: source.ID, TargetNodeID: target.ID,
	})
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestApplyNode_EventOne_DirectStatusRule(t *testing.T) {
	f := newEngineFixture()
	ctx := context.Background()
	w := f.addWorkflow()
	member := uuid.New()

	_, task := f.addNode(w, &f.doneStatus.ID)
	f.addRule(w, f.doneStatus.ID, member)

	if err := f.consumer.processTaskStatusChange(ctx, f.projectID, task.ID); err != nil {
		t.Fatalf("processTaskStatusChange: %v", err)
	}

	got := f.tasks.tasks[task.ID]
	if got.AssigneeID == nil || *got.AssigneeID != member {
		t.Fatalf("expected task assigned to %v, got %+v", member, got.AssigneeID)
	}
	if f.tasks.updateCalls != 1 {
		t.Fatalf("expected exactly 1 UpdateTask call, got %d", f.tasks.updateCalls)
	}
}

func TestApplyNode_EventOne_IdempotentWhenAlreadyAssigned(t *testing.T) {
	f := newEngineFixture()
	ctx := context.Background()
	w := f.addWorkflow()
	member := uuid.New()

	_, task := f.addNode(w, &f.doneStatus.ID)
	f.addRule(w, f.doneStatus.ID, member)
	task.AssigneeID = &member // already assigned before the event fires

	if err := f.consumer.processTaskStatusChange(ctx, f.projectID, task.ID); err != nil {
		t.Fatalf("processTaskStatusChange: %v", err)
	}
	if f.tasks.updateCalls != 0 {
		t.Fatalf("expected no UpdateTask call when already assigned, got %d", f.tasks.updateCalls)
	}
}

func TestChain_SinglePredecessor_AssignsDownstreamOnDone(t *testing.T) {
	f := newEngineFixture()
	ctx := context.Background()
	w := f.addWorkflow()
	downstreamMember := uuid.New()

	nodeA, taskA := f.addNode(w, &f.doneStatus.ID)
	nodeB, taskB := f.addNode(w, &f.readyStatus.ID)
	f.addTransition(w, f.doneStatus.ID, nil) // doneStatus is this workflow's terminal/done status
	f.addRule(w, f.readyStatus.ID, downstreamMember)
	f.addEdge(w, nodeA, nodeB)

	if err := f.consumer.processTaskStatusChange(ctx, f.projectID, taskA.ID); err != nil {
		t.Fatalf("processTaskStatusChange: %v", err)
	}

	gotB := f.tasks.tasks[taskB.ID]
	if gotB.AssigneeID == nil || *gotB.AssigneeID != downstreamMember {
		t.Fatalf("expected downstream task assigned to %v, got %+v", downstreamMember, gotB.AssigneeID)
	}
}

func TestDiamond_ANDJoin_WaitsForAllPredecessors(t *testing.T) {
	f := newEngineFixture()
	ctx := context.Background()
	w := f.addWorkflow()
	downstreamMember := uuid.New()

	nodeA, taskA := f.addNode(w, &f.readyStatus.ID)
	nodeB, taskB := f.addNode(w, &f.readyStatus.ID)
	nodeC, taskC := f.addNode(w, &f.readyStatus.ID)
	f.addTransition(w, f.doneStatus.ID, nil) // doneStatus is this workflow's terminal/done status
	f.addRule(w, f.readyStatus.ID, downstreamMember)
	f.addEdge(w, nodeA, nodeC)
	f.addEdge(w, nodeB, nodeC)

	// A finishes, but B has not — C must NOT be assigned yet.
	taskA.StatusID = &f.doneStatus.ID
	if err := f.consumer.processTaskStatusChange(ctx, f.projectID, taskA.ID); err != nil {
		t.Fatalf("processTaskStatusChange(A): %v", err)
	}
	if got := f.tasks.tasks[taskC.ID]; got.AssigneeID != nil {
		t.Fatalf("expected C to remain unassigned while B is not done, got %+v", got.AssigneeID)
	}

	// B finishes too — NOW C should be assigned.
	taskB.StatusID = &f.doneStatus.ID
	if err := f.consumer.processTaskStatusChange(ctx, f.projectID, taskB.ID); err != nil {
		t.Fatalf("processTaskStatusChange(B): %v", err)
	}
	gotC := f.tasks.tasks[taskC.ID]
	if gotC.AssigneeID == nil || *gotC.AssigneeID != downstreamMember {
		t.Fatalf("expected C assigned to %v once both predecessors are done, got %+v", downstreamMember, gotC.AssigneeID)
	}
}

func TestIsNodeDone_DerivesFromWorkflowTransitions(t *testing.T) {
	f := newEngineFixture()
	ctx := context.Background()
	w := f.addWorkflow()
	f.addTransition(w, f.readyStatus.ID, &f.doneStatus.ID)
	f.addTransition(w, f.doneStatus.ID, nil) // terminal

	node, task := f.addNode(w, &f.readyStatus.ID)
	task.StatusID = &f.doneStatus.ID

	done, err := f.consumer.isNodeDone(ctx, node, task)
	if err != nil {
		t.Fatalf("isNodeDone: %v", err)
	}
	if !done {
		t.Fatalf("expected node to be considered done once its task reaches the workflow's derived done status")
	}
}

func TestIsNodeDone_FalseWhenChainHasNoUniqueTerminal(t *testing.T) {
	f := newEngineFixture()
	ctx := context.Background()
	w := f.addWorkflow()
	// No transitions configured at all — nothing to derive a done status from.

	node, task := f.addNode(w, &f.readyStatus.ID)
	task.StatusID = &f.doneStatus.ID

	done, err := f.consumer.isNodeDone(ctx, node, task)
	if err != nil {
		t.Fatalf("isNodeDone: %v", err)
	}
	if done {
		t.Fatalf("expected node to not be considered done when no unique terminal status is configured")
	}
}
