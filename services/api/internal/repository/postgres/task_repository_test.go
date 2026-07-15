package postgres

import (
	"context"
	"testing"

	taskdom "github.com/Paca-AI/api/internal/domain/task"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// openTaskRepoTestDB sets up an in-memory SQLite DB with a minimal tasks
// table for exercising applyTaskFilter via CountTasks.
func openTaskRepoTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema := `
		CREATE TABLE tasks (
			id          TEXT PRIMARY KEY,
			project_id  TEXT NOT NULL,
			assignee_id TEXT,
			deleted_at  DATETIME
		);`
	if _, err := db.ExecContext(context.Background(), schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

// TestTaskRepository_CountTasks_AssigneeNullWithAssigneeIDs is a regression
// test for https://github.com/Paca-AI/paca/issues/272: combining the
// "unassigned" filter with an "assigned to specific users" filter used to
// produce a WHERE clause with unbalanced parentheses, which crashed with a
// SQL syntax error instead of returning matching tasks.
func TestTaskRepository_CountTasks_AssigneeNullWithAssigneeIDs(t *testing.T) {
	db := openTaskRepoTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	projectID := uuid.New()
	assigneeIn := uuid.New()
	assigneeOut := uuid.New()

	db.MustExec(`INSERT INTO tasks (id, project_id, assignee_id) VALUES ($1, $2, NULL)`, uuid.New().String(), projectID.String())
	db.MustExec(`INSERT INTO tasks (id, project_id, assignee_id) VALUES ($1, $2, $3)`, uuid.New().String(), projectID.String(), assigneeIn.String())
	db.MustExec(`INSERT INTO tasks (id, project_id, assignee_id) VALUES ($1, $2, $3)`, uuid.New().String(), projectID.String(), assigneeOut.String())

	filter := taskdom.TaskFilter{
		AssigneeNull: true,
		AssigneeIDs:  []uuid.UUID{assigneeIn},
	}

	count, err := repo.CountTasks(ctx, projectID, filter)
	if err != nil {
		t.Fatalf("expected no error combining AssigneeNull with AssigneeIDs, got: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 tasks (unassigned + matching assignee), got %d", count)
	}
}
