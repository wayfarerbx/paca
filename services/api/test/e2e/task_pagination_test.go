package e2e_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

// listTasksPage issues GET /projects/:id/tasks with the given query params and
// asserts a 200 response, returning the decoded data map.
func listTasksPage(t *testing.T, env *e2eEnv, client *http.Client, token, projID string, q url.Values) map[string]any {
	t.Helper()
	reqURL := fmt.Sprintf("%s/api/v1/projects/%s/tasks", env.base, projID)
	if len(q) > 0 {
		reqURL += "?" + q.Encode()
	}
	req := mustRequest(env.ctx, t, http.MethodGet, reqURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := mustDo(t, client, req)
	defer func() { _ = resp.Body.Close() }()
	assertStatus(t, resp, http.StatusOK)
	var env2 envelope
	decodeJSON(t, resp, &env2)
	return assertDataMap(t, env2)
}

// nextCursorStr extracts the next_cursor string from a list-tasks data map.
// Returns "" when next_cursor is absent or null.
func nextCursorStr(data map[string]any) string {
	if v, ok := data["next_cursor"]; ok && v != nil {
		s, _ := v.(string)
		return s
	}
	return ""
}

// itemIDs extracts the "id" field from every item in a list-tasks data map.
func itemIDs(data map[string]any) []string {
	items, _ := data["items"].([]any)
	ids := make([]string, 0, len(items))
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		id, _ := item["id"].(string)
		ids = append(ids, id)
	}
	return ids
}

// ---------------------------------------------------------------------------
// TestE2EListTaskPagination_CursorBased
// Tests the cursor-based pagination on the general ListTasks endpoint
// (GET /projects/:projectId/tasks).
// ---------------------------------------------------------------------------

func TestE2EListTaskPagination_CursorBased(t *testing.T) {
	env := newE2EEnv(t)
	seedTaskMemberUser(t, env, "cursor-pag-user", "cursorpagpass1")
	client, token := taskMemberLogin(t, env, "cursor-pag-user", "cursorpagpass1")
	projID := createProjectForTasksViaAPI(t, env, client, token)

	// createBacklogTask creates a task with no sprint assignment.
	createBacklogTask := func(title string) string {
		t.Helper()
		body := jsonBody(t, map[string]any{"title": title})
		req := mustRequest(env.ctx, t, http.MethodPost,
			fmt.Sprintf("%s/api/v1/projects/%s/tasks", env.base, projID), body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusCreated)
		var env2 envelope
		decodeJSON(t, resp, &env2)
		data := assertDataMap(t, env2)
		id, _ := data["id"].(string)
		return id
	}

	// Create 5 tasks for all sub-tests in this group.
	var allTaskIDs []string
	for i := 0; i < 5; i++ {
		allTaskIDs = append(allTaskIDs, createBacklogTask(fmt.Sprintf("Cursor Task %d", i+1)))
	}

	t.Run("first_page_returns_page_size_items_with_next_cursor", func(t *testing.T) {
		data := listTasksPage(t, env, client, token, projID, url.Values{"page_size": {"3"}})
		items, _ := data["items"].([]any)
		if len(items) != 3 {
			t.Errorf("expected 3 items on first page, got %d", len(items))
		}
		if nextCursorStr(data) == "" {
			t.Error("expected next_cursor to be set when more tasks exist beyond first page")
		}
	})

	t.Run("second_page_via_cursor_has_remaining_items_and_no_cursor", func(t *testing.T) {
		firstPage := listTasksPage(t, env, client, token, projID, url.Values{"page_size": {"3"}})
		cursor := nextCursorStr(firstPage)
		if cursor == "" {
			t.Fatal("expected non-empty next_cursor from first page")
		}

		secondPage := listTasksPage(t, env, client, token, projID, url.Values{
			"page_size": {"3"},
			"cursor":    {cursor},
		})
		items, _ := secondPage["items"].([]any)
		if len(items) != 2 {
			t.Errorf("expected 2 remaining items on second page (5 total, 3 on first), got %d", len(items))
		}
		if nextCursorStr(secondPage) != "" {
			t.Error("expected next_cursor to be absent on the last page")
		}
	})

	t.Run("no_next_cursor_when_all_tasks_fit_in_one_page", func(t *testing.T) {
		data := listTasksPage(t, env, client, token, projID, url.Values{"page_size": {"10"}})
		items, _ := data["items"].([]any)
		if len(items) != 5 {
			t.Errorf("expected 5 items when page_size exceeds task count, got %d", len(items))
		}
		if nextCursorStr(data) != "" {
			t.Error("expected no next_cursor when all tasks fit in one page")
		}
	})

	t.Run("full_traversal_returns_all_tasks_without_duplicates", func(t *testing.T) {
		seen := make(map[string]int)
		cursor := ""
		for {
			q := url.Values{"page_size": {"2"}}
			if cursor != "" {
				q.Set("cursor", cursor)
			}
			data := listTasksPage(t, env, client, token, projID, q)
			for _, id := range itemIDs(data) {
				seen[id]++
			}
			cursor = nextCursorStr(data)
			if cursor == "" {
				break
			}
		}
		if len(seen) != 5 {
			t.Errorf("expected 5 unique tasks after full traversal, got %d", len(seen))
		}
		for _, id := range allTaskIDs {
			if seen[id] == 0 {
				t.Errorf("task %q was not returned during full traversal", id)
			}
			if seen[id] > 1 {
				t.Errorf("task %q was returned %d times (duplicate)", id, seen[id])
			}
		}
	})

	t.Run("page_size_zero_is_clamped_to_default", func(t *testing.T) {
		data := listTasksPage(t, env, client, token, projID, url.Values{"page_size": {"0"}})
		if ps, _ := data["page_size"].(float64); ps != 20 {
			t.Errorf("expected page_size=20 when 0 requested (out-of-range clamped), got %v", ps)
		}
	})

	t.Run("page_size_over_max_is_clamped_to_default", func(t *testing.T) {
		data := listTasksPage(t, env, client, token, projID, url.Values{"page_size": {"201"}})
		if ps, _ := data["page_size"].(float64); ps != 20 {
			t.Errorf("expected page_size=20 when 201 requested (over max, clamped), got %v", ps)
		}
	})

	t.Run("invalid_cursor_returns_error", func(t *testing.T) {
		q := url.Values{"cursor": {"not-a-valid-base64-cursor"}}
		reqURL := fmt.Sprintf("%s/api/v1/projects/%s/tasks?%s", env.base, projID, q.Encode())
		req := mustRequest(env.ctx, t, http.MethodGet, reqURL, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == http.StatusOK {
			t.Errorf("expected a non-200 error response for invalid cursor, got 200")
		}
	})
}

// ---------------------------------------------------------------------------
// TestE2EListTaskPagination_CursorWithSprintFilter
// Tests that cursor pagination works correctly when combined with a
// sprint_id filter — only sprint tasks are paginated; backlog tasks are excluded.
// ---------------------------------------------------------------------------

func TestE2EListTaskPagination_CursorWithSprintFilter(t *testing.T) {
	env := newE2EEnv(t)
	seedTaskMemberUser(t, env, "cursor-sprint-filter-user", "cursorsprintfilterpass1")
	client, token := taskMemberLogin(t, env, "cursor-sprint-filter-user", "cursorsprintfilterpass1")
	projID := createProjectForTasksViaAPI(t, env, client, token)
	sprintID := createSprintViaAPI(t, env, client, token, projID, "Pagination Sprint")

	createSprintTask := func(title string) string {
		t.Helper()
		body := jsonBody(t, map[string]any{"title": title, "sprint_id": sprintID})
		req := mustRequest(env.ctx, t, http.MethodPost,
			fmt.Sprintf("%s/api/v1/projects/%s/tasks", env.base, projID), body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusCreated)
		var env2 envelope
		decodeJSON(t, resp, &env2)
		data := assertDataMap(t, env2)
		id, _ := data["id"].(string)
		return id
	}

	// 4 sprint tasks + 2 backlog tasks that must not appear in sprint results.
	var sprintTaskIDs []string
	for i := 0; i < 4; i++ {
		sprintTaskIDs = append(sprintTaskIDs, createSprintTask(fmt.Sprintf("Sprint Pag Task %d", i+1)))
	}
	for i := 0; i < 2; i++ {
		body := jsonBody(t, map[string]any{"title": fmt.Sprintf("Backlog Noise Task %d", i+1)})
		req := mustRequest(env.ctx, t, http.MethodPost,
			fmt.Sprintf("%s/api/v1/projects/%s/tasks", env.base, projID), body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		_ = resp.Body.Close()
		assertStatus(t, resp, http.StatusCreated)
	}

	t.Run("first_sprint_page_has_cursor_when_more_exist", func(t *testing.T) {
		data := listTasksPage(t, env, client, token, projID, url.Values{
			"sprint_id": {sprintID},
			"page_size": {"2"},
		})
		items, _ := data["items"].([]any)
		if len(items) != 2 {
			t.Errorf("expected 2 sprint tasks on first page, got %d", len(items))
		}
		if nextCursorStr(data) == "" {
			t.Error("expected next_cursor when more sprint tasks exist beyond first page")
		}
	})

	t.Run("full_sprint_traversal_returns_only_sprint_tasks_without_duplicates", func(t *testing.T) {
		seen := make(map[string]int)
		cursor := ""
		for {
			q := url.Values{"sprint_id": {sprintID}, "page_size": {"2"}}
			if cursor != "" {
				q.Set("cursor", cursor)
			}
			data := listTasksPage(t, env, client, token, projID, q)
			for _, id := range itemIDs(data) {
				seen[id]++
			}
			cursor = nextCursorStr(data)
			if cursor == "" {
				break
			}
		}
		if len(seen) != 4 {
			t.Errorf("expected 4 sprint tasks from full traversal, got %d", len(seen))
		}
		for _, id := range sprintTaskIDs {
			if seen[id] == 0 {
				t.Errorf("sprint task %q was not returned during traversal", id)
			}
			if seen[id] > 1 {
				t.Errorf("sprint task %q appeared %d times (duplicate)", id, seen[id])
			}
		}
	})
}

// ---------------------------------------------------------------------------
// TestE2EListTaskPagination_CursorWithBacklogFilter
// Tests that cursor pagination works correctly with the sprint_id=null backlog
// filter — only backlog tasks are paginated; sprint tasks are excluded.
// ---------------------------------------------------------------------------

func TestE2EListTaskPagination_CursorWithBacklogFilter(t *testing.T) {
	env := newE2EEnv(t)
	seedTaskMemberUser(t, env, "cursor-backlog-filter-user", "cursorbacklogfilterpass1")
	client, token := taskMemberLogin(t, env, "cursor-backlog-filter-user", "cursorbacklogfilterpass1")
	projID := createProjectForTasksViaAPI(t, env, client, token)
	sprintID := createSprintViaAPI(t, env, client, token, projID, "Sprint for Backlog Pagination Test")

	createTask := func(title string, inSprint bool) string {
		t.Helper()
		body := map[string]any{"title": title}
		if inSprint {
			body["sprint_id"] = sprintID
		}
		req := mustRequest(env.ctx, t, http.MethodPost,
			fmt.Sprintf("%s/api/v1/projects/%s/tasks", env.base, projID), jsonBody(t, body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp := mustDo(t, client, req)
		defer func() { _ = resp.Body.Close() }()
		assertStatus(t, resp, http.StatusCreated)
		var env2 envelope
		decodeJSON(t, resp, &env2)
		data := assertDataMap(t, env2)
		id, _ := data["id"].(string)
		return id
	}

	// 4 backlog tasks + 2 sprint tasks that must not appear in backlog results.
	var backlogTaskIDs []string
	for i := 0; i < 4; i++ {
		backlogTaskIDs = append(backlogTaskIDs, createTask(fmt.Sprintf("Backlog Pag Task %d", i+1), false))
	}
	for i := 0; i < 2; i++ {
		createTask(fmt.Sprintf("Sprint Noise Task %d", i+1), true)
	}

	t.Run("first_backlog_page_has_cursor_when_more_exist", func(t *testing.T) {
		data := listTasksPage(t, env, client, token, projID, url.Values{
			"sprint_id": {"null"},
			"page_size": {"2"},
		})
		items, _ := data["items"].([]any)
		if len(items) != 2 {
			t.Errorf("expected 2 backlog tasks on first page, got %d", len(items))
		}
		if nextCursorStr(data) == "" {
			t.Error("expected next_cursor when more backlog tasks exist beyond first page")
		}
	})

	t.Run("full_backlog_traversal_returns_only_backlog_tasks_without_duplicates", func(t *testing.T) {
		seen := make(map[string]int)
		cursor := ""
		for {
			q := url.Values{"sprint_id": {"null"}, "page_size": {"2"}}
			if cursor != "" {
				q.Set("cursor", cursor)
			}
			data := listTasksPage(t, env, client, token, projID, q)
			for _, id := range itemIDs(data) {
				seen[id]++
			}
			cursor = nextCursorStr(data)
			if cursor == "" {
				break
			}
		}
		if len(seen) != 4 {
			t.Errorf("expected 4 backlog tasks from full traversal, got %d", len(seen))
		}
		for _, id := range backlogTaskIDs {
			if seen[id] == 0 {
				t.Errorf("backlog task %q was not returned during traversal", id)
			}
			if seen[id] > 1 {
				t.Errorf("backlog task %q appeared %d times (duplicate)", id, seen[id])
			}
		}
	})
}
