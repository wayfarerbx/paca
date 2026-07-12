package workflowsvc_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	workflowdom "github.com/Paca-AI/api/internal/domain/workflow"
	"github.com/Paca-AI/api/internal/platform/cache"
	workflowsvc "github.com/Paca-AI/api/internal/service/workflow"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newCacheStore(t *testing.T) *cache.Store {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return cache.NewStore(client, "paca:")
}

func TestCachedRepository_ListStatusRules_CacheMissPopulatesCache(t *testing.T) {
	ctx := context.Background()
	repo := newFakeWorkflowRepo()
	w := &workflowdom.Workflow{ID: uuid.New(), Status: workflowdom.StatusActive}
	_ = repo.CreateWorkflow(ctx, w)
	rule := &workflowdom.StatusRule{ID: uuid.New(), WorkflowID: w.ID, StatusID: uuid.New(), AssigneeMemberID: uuid.New()}
	_ = repo.CreateStatusRule(ctx, rule)
	repo.listStatusRulesCalls = 0 // ignore the seed call above

	cached := workflowsvc.NewCachedRepository(repo, newCacheStore(t), 5*time.Minute, discardLogger())

	// First call: miss.
	got, err := cached.ListStatusRulesByWorkflow(ctx, w.ID)
	if err != nil {
		t.Fatalf("ListStatusRulesByWorkflow (miss): %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(got))
	}
	if repo.listStatusRulesCalls != 1 {
		t.Fatalf("expected 1 underlying call, got %d", repo.listStatusRulesCalls)
	}

	// Second call: hit.
	if _, err := cached.ListStatusRulesByWorkflow(ctx, w.ID); err != nil {
		t.Fatalf("ListStatusRulesByWorkflow (hit): %v", err)
	}
	if repo.listStatusRulesCalls != 1 {
		t.Fatalf("cache hit: underlying repo called again; got %d calls", repo.listStatusRulesCalls)
	}
}

func TestCachedRepository_ListStatusRules_ZeroTTLBypassesCache(t *testing.T) {
	ctx := context.Background()
	repo := newFakeWorkflowRepo()
	cached := workflowsvc.NewCachedRepository(repo, newCacheStore(t), 0, discardLogger())

	for i := 0; i < 3; i++ {
		if _, err := cached.ListStatusRulesByWorkflow(ctx, uuid.New()); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if repo.listStatusRulesCalls != 3 {
		t.Fatalf("TTL=0 should bypass cache; want 3 calls, got %d", repo.listStatusRulesCalls)
	}
}

func TestCachedRepository_CreateStatusRule_InvalidatesList(t *testing.T) {
	ctx := context.Background()
	repo := newFakeWorkflowRepo()
	w := &workflowdom.Workflow{ID: uuid.New(), Status: workflowdom.StatusActive}
	_ = repo.CreateWorkflow(ctx, w)
	cached := workflowsvc.NewCachedRepository(repo, newCacheStore(t), 5*time.Minute, discardLogger())

	if _, err := cached.ListStatusRulesByWorkflow(ctx, w.ID); err != nil {
		t.Fatalf("ListStatusRulesByWorkflow: %v", err)
	}
	rule := &workflowdom.StatusRule{ID: uuid.New(), WorkflowID: w.ID, StatusID: uuid.New(), AssigneeMemberID: uuid.New()}
	if err := cached.CreateStatusRule(ctx, rule); err != nil {
		t.Fatalf("CreateStatusRule: %v", err)
	}
	got, err := cached.ListStatusRulesByWorkflow(ctx, w.ID)
	if err != nil {
		t.Fatalf("ListStatusRulesByWorkflow after Create: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected the newly created rule to be visible immediately, got %d rules", len(got))
	}
	if repo.listStatusRulesCalls != 2 {
		t.Fatalf("expected 2 underlying calls (cache invalidated by the create), got %d", repo.listStatusRulesCalls)
	}
}

// TestCachedRepository_UpdateStatusRule_ReflectsNewAssigneeImmediately is the
// regression case from the review: a rule edited via the API (SetStatusRule,
// which calls UpdateStatusRule for an existing rule) must be visible to the
// very next automation read, not held stale until some cache TTL expires.
func TestCachedRepository_UpdateStatusRule_ReflectsNewAssigneeImmediately(t *testing.T) {
	ctx := context.Background()
	repo := newFakeWorkflowRepo()
	w := &workflowdom.Workflow{ID: uuid.New(), Status: workflowdom.StatusActive}
	_ = repo.CreateWorkflow(ctx, w)
	oldMember := uuid.New()
	newMember := uuid.New()
	rule := &workflowdom.StatusRule{ID: uuid.New(), WorkflowID: w.ID, StatusID: uuid.New(), AssigneeMemberID: oldMember}
	_ = repo.CreateStatusRule(ctx, rule)

	cached := workflowsvc.NewCachedRepository(repo, newCacheStore(t), 5*time.Minute, discardLogger())

	// Populate the cache with the pre-edit value (e.g. an automation event
	// reading the rule list before the edit lands).
	got, err := cached.ListStatusRulesByWorkflow(ctx, w.ID)
	if err != nil || len(got) != 1 || got[0].AssigneeMemberID != oldMember {
		t.Fatalf("expected cached read to see %v, got %+v (err %v)", oldMember, got, err)
	}

	// A concurrent SetStatusRule call updates the assignee.
	updated := *rule
	updated.AssigneeMemberID = newMember
	if err := cached.UpdateStatusRule(ctx, &updated); err != nil {
		t.Fatalf("UpdateStatusRule: %v", err)
	}

	// The next read — even though the list was cached moments ago — must see
	// the new assignee, not the stale cached one.
	got, err = cached.ListStatusRulesByWorkflow(ctx, w.ID)
	if err != nil {
		t.Fatalf("ListStatusRulesByWorkflow after Update: %v", err)
	}
	if len(got) != 1 || got[0].AssigneeMemberID != newMember {
		t.Fatalf("expected the updated assignee %v to be visible immediately, got %+v", newMember, got)
	}
}

func TestCachedRepository_DeleteStatusRule_InvalidatesList(t *testing.T) {
	ctx := context.Background()
	repo := newFakeWorkflowRepo()
	w := &workflowdom.Workflow{ID: uuid.New(), Status: workflowdom.StatusActive}
	_ = repo.CreateWorkflow(ctx, w)
	rule := &workflowdom.StatusRule{ID: uuid.New(), WorkflowID: w.ID, StatusID: uuid.New(), AssigneeMemberID: uuid.New()}
	_ = repo.CreateStatusRule(ctx, rule)
	cached := workflowsvc.NewCachedRepository(repo, newCacheStore(t), 5*time.Minute, discardLogger())

	if got, err := cached.ListStatusRulesByWorkflow(ctx, w.ID); err != nil || len(got) != 1 {
		t.Fatalf("ListStatusRulesByWorkflow: got %+v, err %v", got, err)
	}
	if err := cached.DeleteStatusRule(ctx, rule.ID); err != nil {
		t.Fatalf("DeleteStatusRule: %v", err)
	}
	got, err := cached.ListStatusRulesByWorkflow(ctx, w.ID)
	if err != nil {
		t.Fatalf("ListStatusRulesByWorkflow after Delete: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected the deleted rule to be gone immediately, got %+v", got)
	}
}

func TestCachedRepository_ListStatusRules_PerWorkflowCacheIsolation(t *testing.T) {
	ctx := context.Background()
	repo := newFakeWorkflowRepo()
	w1 := &workflowdom.Workflow{ID: uuid.New(), Status: workflowdom.StatusActive}
	w2 := &workflowdom.Workflow{ID: uuid.New(), Status: workflowdom.StatusActive}
	_ = repo.CreateWorkflow(ctx, w1)
	_ = repo.CreateWorkflow(ctx, w2)
	_ = repo.CreateStatusRule(ctx, &workflowdom.StatusRule{ID: uuid.New(), WorkflowID: w1.ID, StatusID: uuid.New(), AssigneeMemberID: uuid.New()})
	cached := workflowsvc.NewCachedRepository(repo, newCacheStore(t), 5*time.Minute, discardLogger())

	got1, err := cached.ListStatusRulesByWorkflow(ctx, w1.ID)
	if err != nil || len(got1) != 1 {
		t.Fatalf("w1: got %+v, err %v", got1, err)
	}
	got2, err := cached.ListStatusRulesByWorkflow(ctx, w2.ID)
	if err != nil || len(got2) != 0 {
		t.Fatalf("w2: expected no rules, got %+v, err %v", got2, err)
	}

	// Creating a rule on w2 must not disturb w1's already-cached (and still
	// correct) entry.
	_ = cached.CreateStatusRule(ctx, &workflowdom.StatusRule{ID: uuid.New(), WorkflowID: w2.ID, StatusID: uuid.New(), AssigneeMemberID: uuid.New()})
	if got, err := cached.ListStatusRulesByWorkflow(ctx, w1.ID); err != nil || len(got) != 1 {
		t.Fatalf("w1 after w2 create: got %+v, err %v", got, err)
	}
	if repo.listStatusRulesCalls != 2 {
		t.Fatalf("expected w1's cached entry to survive w2's invalidation (2 underlying calls total), got %d", repo.listStatusRulesCalls)
	}
}

// TestCachedRepository_DelegatesUncachedMethods spot-checks that a method
// with no special caching (FindWorkflowByID) passes straight through.
func TestCachedRepository_DelegatesUncachedMethods(t *testing.T) {
	ctx := context.Background()
	repo := newFakeWorkflowRepo()
	w := &workflowdom.Workflow{ID: uuid.New(), Name: "wf", Status: workflowdom.StatusActive}
	_ = repo.CreateWorkflow(ctx, w)
	cached := workflowsvc.NewCachedRepository(repo, newCacheStore(t), 5*time.Minute, discardLogger())

	got, err := cached.FindWorkflowByID(ctx, w.ID)
	if err != nil {
		t.Fatalf("FindWorkflowByID: %v", err)
	}
	if got.Name != "wf" {
		t.Fatalf("expected delegated FindWorkflowByID to return the real workflow, got %+v", got)
	}
}
