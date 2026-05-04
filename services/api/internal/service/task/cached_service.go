package tasksvc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	taskdom "github.com/paca/api/internal/domain/task"
	"github.com/paca/api/internal/platform/cache"
)

// CachedService decorates a taskdom.Service with a Valkey/Redis-backed cache.
//
// # What is cached
//
//   - ListTaskTypes       – keyed by project; invalidated on any type mutation.
//   - ListTaskStatuses    – keyed by project; invalidated on any status mutation.
//   - ListCustomFieldDefinitions – keyed by project; invalidated on any field mutation.
//
// All other methods (tasks, BDD scenarios) are delegated directly to the
// underlying service without caching.
//
// Cache errors are non-fatal: on a read error the decorator falls through to
// the real service; on a write/delete error it logs and continues so that
// mutations always succeed even when the cache is temporarily unavailable.
type CachedService struct {
	svc taskdom.Service
	st  *cache.Store
	ttl time.Duration
	log *slog.Logger
}

// NewCachedService wraps svc with a caching layer backed by st.
// ttl controls how long each cached entry lives; zero disables caching.
// log receives non-fatal cache warnings.
func NewCachedService(svc taskdom.Service, st *cache.Store, ttl time.Duration, log *slog.Logger) *CachedService {
	return &CachedService{svc: svc, st: st, ttl: ttl, log: log}
}

// --- cache key helpers -------------------------------------------------------

func taskTypesKey(projectID uuid.UUID) string {
	return fmt.Sprintf("project:%s:task-types", projectID)
}

func taskStatusesKey(projectID uuid.UUID) string {
	return fmt.Sprintf("project:%s:task-statuses", projectID)
}

func customFieldsKey(projectID uuid.UUID) string {
	return fmt.Sprintf("project:%s:custom-fields", projectID)
}

// --- Task Types --------------------------------------------------------------

// ListTaskTypes returns task types for a project, reading from cache when
// available and populating it on a miss.
func (c *CachedService) ListTaskTypes(ctx context.Context, projectID uuid.UUID) ([]*taskdom.TaskType, error) {
	if c.ttl == 0 {
		return c.svc.ListTaskTypes(ctx, projectID)
	}
	key := taskTypesKey(projectID)
	var result []*taskdom.TaskType
	if ok, err := c.st.Get(ctx, key, &result); ok {
		return result, nil
	} else if err != nil {
		c.log.WarnContext(ctx, "cache: ListTaskTypes get", "err", err)
	}

	result, err := c.svc.ListTaskTypes(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if err := c.st.Set(ctx, key, result, c.ttl); err != nil {
		c.log.WarnContext(ctx, "cache: ListTaskTypes set", "err", err)
	}
	return result, nil
}

func (c *CachedService) GetTaskType(ctx context.Context, id uuid.UUID) (*taskdom.TaskType, error) {
	return c.svc.GetTaskType(ctx, id)
}

func (c *CachedService) CreateTaskType(ctx context.Context, in taskdom.CreateTaskTypeInput) (*taskdom.TaskType, error) {
	t, err := c.svc.CreateTaskType(ctx, in)
	if err != nil {
		return nil, err
	}
	if err := c.st.Delete(ctx, taskTypesKey(in.ProjectID)); err != nil {
		c.log.WarnContext(ctx, "cache: CreateTaskType delete", "err", err)
	}
	return t, nil
}

func (c *CachedService) UpdateTaskType(ctx context.Context, projectID, id uuid.UUID, in taskdom.UpdateTaskTypeInput) (*taskdom.TaskType, error) {
	t, err := c.svc.UpdateTaskType(ctx, projectID, id, in)
	if err != nil {
		return nil, err
	}
	if err := c.st.Delete(ctx, taskTypesKey(projectID)); err != nil {
		c.log.WarnContext(ctx, "cache: UpdateTaskType delete", "err", err)
	}
	return t, nil
}

func (c *CachedService) DeleteTaskType(ctx context.Context, projectID, id uuid.UUID) error {
	if err := c.svc.DeleteTaskType(ctx, projectID, id); err != nil {
		return err
	}
	if err := c.st.Delete(ctx, taskTypesKey(projectID)); err != nil {
		c.log.WarnContext(ctx, "cache: DeleteTaskType delete", "err", err)
	}
	return nil
}

func (c *CachedService) SetDefaultTaskType(ctx context.Context, projectID, typeID uuid.UUID) (*taskdom.TaskType, error) {
	t, err := c.svc.SetDefaultTaskType(ctx, projectID, typeID)
	if err != nil {
		return nil, err
	}
	if err := c.st.Delete(ctx, taskTypesKey(projectID)); err != nil {
		c.log.WarnContext(ctx, "cache: SetDefaultTaskType delete", "err", err)
	}
	return t, nil
}

// --- Task Statuses -----------------------------------------------------------

// ListTaskStatuses returns task statuses for a project, reading from cache
// when available and populating it on a miss.
func (c *CachedService) ListTaskStatuses(ctx context.Context, projectID uuid.UUID) ([]*taskdom.TaskStatus, error) {
	if c.ttl == 0 {
		return c.svc.ListTaskStatuses(ctx, projectID)
	}
	key := taskStatusesKey(projectID)
	var result []*taskdom.TaskStatus
	if ok, err := c.st.Get(ctx, key, &result); ok {
		return result, nil
	} else if err != nil {
		c.log.WarnContext(ctx, "cache: ListTaskStatuses get", "err", err)
	}

	result, err := c.svc.ListTaskStatuses(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if err := c.st.Set(ctx, key, result, c.ttl); err != nil {
		c.log.WarnContext(ctx, "cache: ListTaskStatuses set", "err", err)
	}
	return result, nil
}

func (c *CachedService) GetTaskStatus(ctx context.Context, id uuid.UUID) (*taskdom.TaskStatus, error) {
	return c.svc.GetTaskStatus(ctx, id)
}

func (c *CachedService) CreateTaskStatus(ctx context.Context, in taskdom.CreateTaskStatusInput) (*taskdom.TaskStatus, error) {
	st, err := c.svc.CreateTaskStatus(ctx, in)
	if err != nil {
		return nil, err
	}
	if err := c.st.Delete(ctx, taskStatusesKey(in.ProjectID)); err != nil {
		c.log.WarnContext(ctx, "cache: CreateTaskStatus delete", "err", err)
	}
	return st, nil
}

func (c *CachedService) UpdateTaskStatus(ctx context.Context, projectID, id uuid.UUID, in taskdom.UpdateTaskStatusInput) (*taskdom.TaskStatus, error) {
	st, err := c.svc.UpdateTaskStatus(ctx, projectID, id, in)
	if err != nil {
		return nil, err
	}
	if err := c.st.Delete(ctx, taskStatusesKey(projectID)); err != nil {
		c.log.WarnContext(ctx, "cache: UpdateTaskStatus delete", "err", err)
	}
	return st, nil
}

func (c *CachedService) DeleteTaskStatus(ctx context.Context, projectID, id uuid.UUID) error {
	if err := c.svc.DeleteTaskStatus(ctx, projectID, id); err != nil {
		return err
	}
	if err := c.st.Delete(ctx, taskStatusesKey(projectID)); err != nil {
		c.log.WarnContext(ctx, "cache: DeleteTaskStatus delete", "err", err)
	}
	return nil
}

func (c *CachedService) SetDefaultTaskStatus(ctx context.Context, projectID, statusID uuid.UUID) (*taskdom.TaskStatus, error) {
	st, err := c.svc.SetDefaultTaskStatus(ctx, projectID, statusID)
	if err != nil {
		return nil, err
	}
	if err := c.st.Delete(ctx, taskStatusesKey(projectID)); err != nil {
		c.log.WarnContext(ctx, "cache: SetDefaultTaskStatus delete", "err", err)
	}
	return st, nil
}

// --- Tasks (pass-through) ----------------------------------------------------
// Tasks are not cached because they are highly mutable and queries are filtered
// in many different ways, making cache key cardinality prohibitively large.

func (c *CachedService) ListTasks(ctx context.Context, projectID uuid.UUID, filter taskdom.TaskFilter, page, pageSize int) ([]*taskdom.Task, int64, error) {
	return c.svc.ListTasks(ctx, projectID, filter, page, pageSize)
}

func (c *CachedService) GetTask(ctx context.Context, projectID, id uuid.UUID) (*taskdom.Task, error) {
	return c.svc.GetTask(ctx, projectID, id)
}

func (c *CachedService) GetTaskByNumber(ctx context.Context, projectID uuid.UUID, taskNumber int64) (*taskdom.Task, error) {
	return c.svc.GetTaskByNumber(ctx, projectID, taskNumber)
}

func (c *CachedService) CreateTask(ctx context.Context, in taskdom.CreateTaskInput) (*taskdom.Task, error) {
	return c.svc.CreateTask(ctx, in)
}

func (c *CachedService) UpdateTask(ctx context.Context, projectID, id uuid.UUID, in taskdom.UpdateTaskInput) (*taskdom.Task, error) {
	return c.svc.UpdateTask(ctx, projectID, id, in)
}

func (c *CachedService) DeleteTask(ctx context.Context, projectID, id uuid.UUID) error {
	return c.svc.DeleteTask(ctx, projectID, id)
}

// --- Custom Field Definitions ------------------------------------------------

// ListCustomFieldDefinitions returns custom field definitions for a project,
// reading from cache when available and populating it on a miss.
func (c *CachedService) ListCustomFieldDefinitions(ctx context.Context, projectID uuid.UUID) ([]*taskdom.CustomFieldDefinition, error) {
	if c.ttl == 0 {
		return c.svc.ListCustomFieldDefinitions(ctx, projectID)
	}
	key := customFieldsKey(projectID)
	var result []*taskdom.CustomFieldDefinition
	if ok, err := c.st.Get(ctx, key, &result); ok {
		return result, nil
	} else if err != nil {
		c.log.WarnContext(ctx, "cache: ListCustomFieldDefinitions get", "err", err)
	}

	result, err := c.svc.ListCustomFieldDefinitions(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if err := c.st.Set(ctx, key, result, c.ttl); err != nil {
		c.log.WarnContext(ctx, "cache: ListCustomFieldDefinitions set", "err", err)
	}
	return result, nil
}

func (c *CachedService) GetCustomFieldDefinition(ctx context.Context, projectID, id uuid.UUID) (*taskdom.CustomFieldDefinition, error) {
	return c.svc.GetCustomFieldDefinition(ctx, projectID, id)
}

func (c *CachedService) CreateCustomFieldDefinition(ctx context.Context, in taskdom.CreateCustomFieldDefinitionInput) (*taskdom.CustomFieldDefinition, error) {
	f, err := c.svc.CreateCustomFieldDefinition(ctx, in)
	if err != nil {
		return nil, err
	}
	if err := c.st.Delete(ctx, customFieldsKey(in.ProjectID)); err != nil {
		c.log.WarnContext(ctx, "cache: CreateCustomFieldDefinition delete", "err", err)
	}
	return f, nil
}

func (c *CachedService) UpdateCustomFieldDefinition(ctx context.Context, projectID, id uuid.UUID, in taskdom.UpdateCustomFieldDefinitionInput) (*taskdom.CustomFieldDefinition, error) {
	f, err := c.svc.UpdateCustomFieldDefinition(ctx, projectID, id, in)
	if err != nil {
		return nil, err
	}
	if err := c.st.Delete(ctx, customFieldsKey(projectID)); err != nil {
		c.log.WarnContext(ctx, "cache: UpdateCustomFieldDefinition delete", "err", err)
	}
	return f, nil
}

func (c *CachedService) DeleteCustomFieldDefinition(ctx context.Context, projectID, id uuid.UUID) error {
	if err := c.svc.DeleteCustomFieldDefinition(ctx, projectID, id); err != nil {
		return err
	}
	if err := c.st.Delete(ctx, customFieldsKey(projectID)); err != nil {
		c.log.WarnContext(ctx, "cache: DeleteCustomFieldDefinition delete", "err", err)
	}
	return nil
}

// --- BDD Scenarios (pass-through) --------------------------------------------
// BDD scenarios are task-level and change frequently enough that caching adds
// complexity without meaningful benefit.

func (c *CachedService) ListBDDScenarios(ctx context.Context, projectID, taskID uuid.UUID) ([]*taskdom.BDDScenario, error) {
	return c.svc.ListBDDScenarios(ctx, projectID, taskID)
}

func (c *CachedService) GetBDDScenario(ctx context.Context, projectID, taskID, id uuid.UUID) (*taskdom.BDDScenario, error) {
	return c.svc.GetBDDScenario(ctx, projectID, taskID, id)
}

func (c *CachedService) CreateBDDScenario(ctx context.Context, in taskdom.CreateBDDScenarioInput) (*taskdom.BDDScenario, error) {
	return c.svc.CreateBDDScenario(ctx, in)
}

func (c *CachedService) UpdateBDDScenario(ctx context.Context, projectID, taskID, id uuid.UUID, in taskdom.UpdateBDDScenarioInput) (*taskdom.BDDScenario, error) {
	return c.svc.UpdateBDDScenario(ctx, projectID, taskID, id, in)
}

func (c *CachedService) DeleteBDDScenario(ctx context.Context, projectID, taskID, id uuid.UUID) error {
	return c.svc.DeleteBDDScenario(ctx, projectID, taskID, id)
}
