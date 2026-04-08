package e2e_test

import (
	"fmt"
	"net/http"
	"testing"
)

// createViewViaAPI creates a sprint view and returns its ID.
func createViewViaAPI(t *testing.T, env *e2eEnv, client *http.Client, token, projectID, sprintID, name, viewType string) string {
	t.Helper()
	url := fmt.Sprintf("%s/api/v1/projects/%s/sprints/%s/views", env.base, projectID, sprintID)
	body := jsonBody(t, map[string]any{
		"name":      name,
		"view_type": viewType,
	})
	req := mustRequest(env.ctx, t, http.MethodPost, url, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp := mustDo(t, client, req)
	defer func() { _ = resp.Body.Close() }()
	assertStatus(t, resp, http.StatusCreated)
	var env2 envelope
	decodeJSON(t, resp, &env2)
	data := assertDataMap(t, env2)
	id, _ := data["id"].(string)
	if id == "" {
		t.Fatal("missing view id in response")
	}
	return id
}

// listViewIDsViaAPI returns all view IDs for a sprint.
func listViewIDsViaAPI(t *testing.T, env *e2eEnv, client *http.Client, token, projectID, sprintID string) []string {
	t.Helper()
	url := fmt.Sprintf("%s/api/v1/projects/%s/sprints/%s/views", env.base, projectID, sprintID)
	req := mustRequest(env.ctx, t, http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := mustDo(t, client, req)
	defer func() { _ = resp.Body.Close() }()
	assertStatus(t, resp, http.StatusOK)
	var env2 envelope
	decodeJSON(t, resp, &env2)
	data := assertDataMap(t, env2)
	items, _ := data["items"].([]any)
	var ids []string
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			if id, ok := m["id"].(string); ok && id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// deleteViewViaAPI deletes a view, ignoring 404 (already gone).
func deleteViewViaAPI(t *testing.T, env *e2eEnv, client *http.Client, token, projectID, sprintID, viewID string) {
	t.Helper()
	url := fmt.Sprintf("%s/api/v1/projects/%s/sprints/%s/views/%s", env.base, projectID, sprintID, viewID)
	req := mustRequest(env.ctx, t, http.MethodDelete, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := mustDo(t, client, req)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		t.Errorf("deleteViewViaAPI: unexpected status %d", resp.StatusCode)
	}
}

func TestE2EViewManagement_CRUD(t *testing.T) {
	env := newE2EEnv(t)
	seedTaskMemberUser(t, env, "view-crud-user", "viewpass1")
	client, token := taskMemberLogin(t, env, "view-crud-user", "viewpass1")
	projID := createProjectForTasksViaAPI(t, env, client, token)
	sprintID := createSprintViaAPI(t, env, client, token, projID, "Sprint for Views")

	// Sprint creation auto-seeds default views; record their IDs so we can
	// clean them up before testing the "last view" guard.
	autoSeededViewIDs := listViewIDsViaAPI(t, env, client, token, projID, sprintID)

	var viewID string
	var view2ID string

	t.Run("create_view", func(t *testing.T) {
		viewID = createViewViaAPI(t, env, client, token, projID, sprintID, "Board View", "board")
		if viewID == "" {
			t.Fatal("expected non-empty view id")
		}
	})

	t.Run("list_views", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/projects/%s/sprints/%s/views", env.base, projID, sprintID)
		req := mustRequest(env.ctx, t, http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusOK)
		var env2 envelope
		decodeJSON(t, resp, &env2)
		data := assertDataMap(t, env2)
		items, _ := data["items"].([]any)
		if len(items) == 0 {
			t.Error("expected at least 1 view in list")
		}
	})

	t.Run("get_view", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/projects/%s/sprints/%s/views/%s", env.base, projID, sprintID, viewID)
		req := mustRequest(env.ctx, t, http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusOK)
		var env2 envelope
		decodeJSON(t, resp, &env2)
		data := assertDataMap(t, env2)
		if data["id"] != viewID {
			t.Errorf("expected id=%s, got %v", viewID, data["id"])
		}
	})

	t.Run("update_name", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/projects/%s/sprints/%s/views/%s", env.base, projID, sprintID, viewID)
		body := jsonBody(t, map[string]any{"name": "Renamed Board"})
		req := mustRequest(env.ctx, t, http.MethodPatch, url, body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusOK)
		var env2 envelope
		decodeJSON(t, resp, &env2)
		data := assertDataMap(t, env2)
		if data["name"] != "Renamed Board" {
			t.Errorf("expected name=Renamed Board, got %v", data["name"])
		}
	})

	t.Run("update_config", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/projects/%s/sprints/%s/views/%s", env.base, projID, sprintID, viewID)
		body := jsonBody(t, map[string]any{
			"config": map[string]any{
				"column_by": "status",
				"swimlanes": "assignee",
			},
		})
		req := mustRequest(env.ctx, t, http.MethodPatch, url, body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusOK)
		var env2 envelope
		decodeJSON(t, resp, &env2)
		data := assertDataMap(t, env2)
		cfg, _ := data["config"].(map[string]any)
		if cfg["column_by"] != "status" {
			t.Errorf("expected column_by=status, got %v", cfg["column_by"])
		}
	})

	t.Run("create_second_view", func(t *testing.T) {
		view2ID = createViewViaAPI(t, env, client, token, projID, sprintID, "Table View", "table")
	})

	t.Run("delete_first_view", func(t *testing.T) {
		// Delete the manually created first view.
		deleteViewViaAPI(t, env, client, token, projID, sprintID, viewID)
		// Also delete every auto-seeded view (created when the sprint was
		// opened) so that view2ID becomes the sole remaining view, allowing
		// the next subtest to exercise the "last view" guard.
		for _, aid := range autoSeededViewIDs {
			deleteViewViaAPI(t, env, client, token, projID, sprintID, aid)
		}
	})

	t.Run("delete_last_view_rejected", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/projects/%s/sprints/%s/views/%s", env.base, projID, sprintID, view2ID)
		req := mustRequest(env.ctx, t, http.MethodDelete, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusConflict)
		var env2 envelope
		decodeJSON(t, resp, &env2)
		if env2.ErrorCode != "VIEW_IS_LAST_VIEW" {
			t.Errorf("expected VIEW_IS_LAST_VIEW, got %q", env2.ErrorCode)
		}
	})
}

func TestE2ETaskPositionManagement(t *testing.T) {
	env := newE2EEnv(t)
	seedTaskMemberUser(t, env, "view-pos-user", "pospass1")
	client, token := taskMemberLogin(t, env, "view-pos-user", "pospass1")
	projID := createProjectForTasksViaAPI(t, env, client, token)
	sprintID := createSprintViaAPI(t, env, client, token, projID, "Sprint for Positions")
	viewID := createViewViaAPI(t, env, client, token, projID, sprintID, "Position View", "table")

	// Use a fixed task UUID (doesn't need to exist in DB for position tracking)
	taskID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	t.Run("move_task", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/projects/%s/sprints/%s/views/%s/task-positions/%s",
			env.base, projID, sprintID, viewID, taskID)
		body := jsonBody(t, map[string]any{
			"position":  2,
			"group_key": "in-progress",
		})
		req := mustRequest(env.ctx, t, http.MethodPut, url, body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusNoContent)
	})

	t.Run("list_positions", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/projects/%s/sprints/%s/views/%s/task-positions",
			env.base, projID, sprintID, viewID)
		req := mustRequest(env.ctx, t, http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusOK)
		var env2 envelope
		decodeJSON(t, resp, &env2)
		data := assertDataMap(t, env2)
		items, _ := data["items"].([]any)
		if len(items) != 1 {
			t.Fatalf("expected 1 position, got %d", len(items))
		}
		pos, _ := items[0].(map[string]any)
		if pos["task_id"] != taskID {
			t.Errorf("expected task_id=%s, got %v", taskID, pos["task_id"])
		}
	})

	t.Run("move_to_different_group", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/projects/%s/sprints/%s/views/%s/task-positions/%s",
			env.base, projID, sprintID, viewID, taskID)
		body := jsonBody(t, map[string]any{
			"position":  0,
			"group_key": "done",
		})
		req := mustRequest(env.ctx, t, http.MethodPut, url, body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusNoContent)

		// Verify updated
		listURL := fmt.Sprintf("%s/api/v1/projects/%s/sprints/%s/views/%s/task-positions",
			env.base, projID, sprintID, viewID)
		req2 := mustRequest(env.ctx, t, http.MethodGet, listURL, nil)
		req2.Header.Set("Authorization", "Bearer "+token)
		resp2 := mustDo(t, client, req2)
		defer func() { _ = resp2.Body.Close() }()
		assertStatus(t, resp2, http.StatusOK)
		var env2 envelope
		decodeJSON(t, resp2, &env2)
		data := assertDataMap(t, env2)
		items, _ := data["items"].([]any)
		if len(items) != 1 {
			t.Fatalf("expected 1 position after upsert, got %d", len(items))
		}
		pos, _ := items[0].(map[string]any)
		if pos["group_key"] != "done" {
			t.Errorf("expected group_key=done, got %v", pos["group_key"])
		}
	})
}

// ---------------------------------------------------------------------------
// Product-backlog view helpers
// ---------------------------------------------------------------------------

// createBacklogViewViaAPI creates a product-backlog view and returns its ID.
func createBacklogViewViaAPI(t *testing.T, env *e2eEnv, client *http.Client, token, projectID, name, viewType string) string {
	t.Helper()
	url := fmt.Sprintf("%s/api/v1/projects/%s/product-backlog/views", env.base, projectID)
	body := jsonBody(t, map[string]any{
		"name":      name,
		"view_type": viewType,
	})
	req := mustRequest(env.ctx, t, http.MethodPost, url, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp := mustDo(t, client, req)
	defer func() { _ = resp.Body.Close() }()
	assertStatus(t, resp, http.StatusCreated)
	var env2 envelope
	decodeJSON(t, resp, &env2)
	data := assertDataMap(t, env2)
	id, _ := data["id"].(string)
	if id == "" {
		t.Fatal("missing view id in backlog view response")
	}
	return id
}

// listBacklogViewIDsViaAPI returns all view IDs for the product backlog.
func listBacklogViewIDsViaAPI(t *testing.T, env *e2eEnv, client *http.Client, token, projectID string) []string {
	t.Helper()
	url := fmt.Sprintf("%s/api/v1/projects/%s/product-backlog/views", env.base, projectID)
	req := mustRequest(env.ctx, t, http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := mustDo(t, client, req)
	defer func() { _ = resp.Body.Close() }()
	assertStatus(t, resp, http.StatusOK)
	var env2 envelope
	decodeJSON(t, resp, &env2)
	data := assertDataMap(t, env2)
	items, _ := data["items"].([]any)
	var ids []string
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			if id, ok := m["id"].(string); ok && id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// deleteBacklogViewViaAPI deletes a product-backlog view, ignoring 404.
func deleteBacklogViewViaAPI(t *testing.T, env *e2eEnv, client *http.Client, token, projectID, viewID string) {
	t.Helper()
	url := fmt.Sprintf("%s/api/v1/projects/%s/product-backlog/views/%s", env.base, projectID, viewID)
	req := mustRequest(env.ctx, t, http.MethodDelete, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := mustDo(t, client, req)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		t.Errorf("deleteBacklogViewViaAPI: unexpected status %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Product-backlog view e2e tests
// ---------------------------------------------------------------------------

func TestE2EBacklogViewManagement_CRUD(t *testing.T) {
	env := newE2EEnv(t)
	seedTaskMemberUser(t, env, "backlog-view-crud-user", "backlogpass1")
	client, token := taskMemberLogin(t, env, "backlog-view-crud-user", "backlogpass1")
	projID := createProjectForTasksViaAPI(t, env, client, token)

	// Project creation auto-seeds default backlog views; record their IDs.
	autoSeededViewIDs := listBacklogViewIDsViaAPI(t, env, client, token, projID)

	var viewID string
	var view2ID string

	t.Run("create_view", func(t *testing.T) {
		viewID = createBacklogViewViaAPI(t, env, client, token, projID, "Backlog Board", "board")
		if viewID == "" {
			t.Fatal("expected non-empty view id")
		}
	})

	t.Run("response_has_no_sprint_id", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/projects/%s/product-backlog/views/%s", env.base, projID, viewID)
		req := mustRequest(env.ctx, t, http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusOK)
		var env2 envelope
		decodeJSON(t, resp, &env2)
		data := assertDataMap(t, env2)
		if sid, ok := data["sprint_id"]; ok && sid != nil {
			t.Errorf("expected sprint_id absent/null in backlog view response, got %v", sid)
		}
		if pid, _ := data["project_id"].(string); pid != projID {
			t.Errorf("expected project_id=%s, got %q", projID, pid)
		}
	})

	t.Run("list_views", func(t *testing.T) {
		ids := listBacklogViewIDsViaAPI(t, env, client, token, projID)
		if len(ids) == 0 {
			t.Error("expected at least 1 backlog view in list")
		}
	})

	t.Run("update_name", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/projects/%s/product-backlog/views/%s", env.base, projID, viewID)
		body := jsonBody(t, map[string]any{"name": "Renamed Backlog Board"})
		req := mustRequest(env.ctx, t, http.MethodPatch, url, body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusOK)
		var env2 envelope
		decodeJSON(t, resp, &env2)
		data := assertDataMap(t, env2)
		if data["name"] != "Renamed Backlog Board" {
			t.Errorf("expected name=Renamed Backlog Board, got %v", data["name"])
		}
	})

	t.Run("create_second_view", func(t *testing.T) {
		view2ID = createBacklogViewViaAPI(t, env, client, token, projID, "Backlog Table", "table")
	})

	t.Run("delete_first_view", func(t *testing.T) {
		deleteBacklogViewViaAPI(t, env, client, token, projID, viewID)
		for _, aid := range autoSeededViewIDs {
			deleteBacklogViewViaAPI(t, env, client, token, projID, aid)
		}
	})

	t.Run("delete_last_view_rejected", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/projects/%s/product-backlog/views/%s", env.base, projID, view2ID)
		req := mustRequest(env.ctx, t, http.MethodDelete, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusConflict)
		var env2 envelope
		decodeJSON(t, resp, &env2)
		if env2.ErrorCode != "VIEW_IS_LAST_VIEW" {
			t.Errorf("expected VIEW_IS_LAST_VIEW, got %q", env2.ErrorCode)
		}
	})
}

func TestE2EBacklogTaskPositionManagement(t *testing.T) {
	env := newE2EEnv(t)
	seedTaskMemberUser(t, env, "backlog-pos-user", "backlogpospass1")
	client, token := taskMemberLogin(t, env, "backlog-pos-user", "backlogpospass1")
	projID := createProjectForTasksViaAPI(t, env, client, token)
	viewID := createBacklogViewViaAPI(t, env, client, token, projID, "Backlog Position View", "table")

	taskID := "cccccccc-dddd-eeee-ffff-aaaaaaaaaaaa"

	t.Run("move_task", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/projects/%s/product-backlog/views/%s/task-positions/%s",
			env.base, projID, viewID, taskID)
		body := jsonBody(t, map[string]any{
			"position":  1,
			"group_key": "todo",
		})
		req := mustRequest(env.ctx, t, http.MethodPut, url, body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusNoContent)
	})

	t.Run("list_positions", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/projects/%s/product-backlog/views/%s/task-positions",
			env.base, projID, viewID)
		req := mustRequest(env.ctx, t, http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusOK)
		var env2 envelope
		decodeJSON(t, resp, &env2)
		data := assertDataMap(t, env2)
		items, _ := data["items"].([]any)
		if len(items) != 1 {
			t.Fatalf("expected 1 position, got %d", len(items))
		}
		pos, _ := items[0].(map[string]any)
		if pos["task_id"] != taskID {
			t.Errorf("expected task_id=%s, got %v", taskID, pos["task_id"])
		}
	})

	t.Run("update_group", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/projects/%s/product-backlog/views/%s/task-positions/%s",
			env.base, projID, viewID, taskID)
		body := jsonBody(t, map[string]any{
			"position":  0,
			"group_key": "in-progress",
		})
		req := mustRequest(env.ctx, t, http.MethodPut, url, body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusNoContent)

		listURL := fmt.Sprintf("%s/api/v1/projects/%s/product-backlog/views/%s/task-positions",
			env.base, projID, viewID)
		req2 := mustRequest(env.ctx, t, http.MethodGet, listURL, nil)
		req2.Header.Set("Authorization", "Bearer "+token)
		resp2 := mustDo(t, client, req2)
		defer func() { _ = resp2.Body.Close() }()
		assertStatus(t, resp2, http.StatusOK)
		var env2 envelope
		decodeJSON(t, resp2, &env2)
		data := assertDataMap(t, env2)
		items, _ := data["items"].([]any)
		if len(items) != 1 {
			t.Fatalf("expected 1 position, got %d", len(items))
		}
		pos, _ := items[0].(map[string]any)
		if pos["group_key"] != "in-progress" {
			t.Errorf("expected group_key=in-progress, got %v", pos["group_key"])
		}
	})
}
