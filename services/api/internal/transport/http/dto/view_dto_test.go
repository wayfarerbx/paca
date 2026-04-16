package dto_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	sprintdom "github.com/paca/api/internal/domain/sprint"
	"github.com/paca/api/internal/transport/http/dto"
)

// ---------------------------------------------------------------------------
// ViewFromEntity roundtrip
// ---------------------------------------------------------------------------

// TestViewFromEntity_FiltersRoundtrip verifies that all ViewFilters fields
// using FilterConfig are correctly mapped to ViewResponse.
func TestViewFromEntity_FiltersRoundtrip(t *testing.T) {
	sprintID := uuid.New()
	projectID := uuid.New()
	viewID := uuid.New()
	statusID := uuid.NewString()
	assigneeID := uuid.NewString()

	entity := &sprintdom.SprintView{
		ID:        viewID,
		SprintID:  &sprintID,
		ProjectID: projectID,
		Name:      "Board",
		ViewType:  sprintdom.ViewTypeBoard,
		Position:  1,
		Config: sprintdom.ViewConfig{
			ColumnBy: "status",
			Filters: &sprintdom.ViewFilters{
				TaskTypes: &sprintdom.FilterConfig{
					All: false,
					Items: map[string]sprintdom.FilterEntry{
						"normal": sprintdom.FilterEntryNested(sprintdom.FilterConfig{All: true}),
					},
				},
				Sprints: &sprintdom.FilterConfig{
					All: false,
					Items: map[string]sprintdom.FilterEntry{
						sprintID.String(): sprintdom.FilterEntryInclude(),
						uuid.NewString():  sprintdom.FilterEntryInclude(),
					},
				},
				Statuses: &sprintdom.FilterConfig{
					All: false,
					Items: map[string]sprintdom.FilterEntry{
						statusID: sprintdom.FilterEntryInclude(),
					},
				},
				Assignees: &sprintdom.FilterConfig{
					All: false,
					Items: map[string]sprintdom.FilterEntry{
						assigneeID: sprintdom.FilterEntryInclude(),
					},
				},
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	resp := dto.ViewFromEntity(entity)

	if resp.Config.Filters == nil {
		t.Fatal("expected non-nil Filters in response")
	}
	f := resp.Config.Filters

	if f.TaskTypes == nil {
		t.Fatal("expected non-nil TaskTypes in response filters")
	}
	normalEntry, ok := f.TaskTypes.Items["normal"]
	if !ok {
		t.Fatal("expected 'normal' key in TaskTypes.Items")
	}
	if !normalEntry.IsNested() {
		t.Error("expected 'normal' entry to be a nested FilterConfig")
	}
	if normalEntry.Config() == nil || !normalEntry.Config().All {
		t.Error("expected 'normal' nested config to have All=true")
	}

	if f.Sprints == nil {
		t.Fatal("expected non-nil Sprints in response filters")
	}
	if len(f.Sprints.Items) != 2 {
		t.Errorf("Sprints.Items: want 2 entries, got %d", len(f.Sprints.Items))
	}
	if _, ok := f.Sprints.Items[sprintID.String()]; !ok {
		t.Errorf("Sprints.Items missing original sprintID %s", sprintID)
	}

	if f.Statuses == nil || len(f.Statuses.Items) != 1 {
		t.Error("Statuses.Items: want 1 entry")
	}
	if f.Assignees == nil || len(f.Assignees.Items) != 1 {
		t.Error("Assignees.Items: want 1 entry")
	}
}

// TestViewFromEntity_NilFilters verifies that a nil Filters field maps to nil
// in the DTO (no panic).
func TestViewFromEntity_NilFilters(t *testing.T) {
	entity := &sprintdom.SprintView{
		ID:        uuid.New(),
		ProjectID: uuid.New(),
		Name:      "Table",
		ViewType:  sprintdom.ViewTypeTable,
		Config:    sprintdom.ViewConfig{},
	}
	resp := dto.ViewFromEntity(entity)
	if resp.Config.Filters != nil {
		t.Errorf("expected nil Filters for entity with nil filters, got %+v", resp.Config.Filters)
	}
}

// ---------------------------------------------------------------------------
// toViewConfig (via CreateViewRequest / UpdateViewRequest) roundtrip
// ---------------------------------------------------------------------------

// TestCreateViewRequest_ToCreateInput_FiltersRoundtrip exercises toViewConfig
// indirectly through the CreateViewRequest helper.
func TestCreateViewRequest_ToCreateInput_FiltersRoundtrip(t *testing.T) {
	sprintID := uuid.New()
	projectID := uuid.New()
	systemTypeID := uuid.NewString()

	req := dto.CreateViewRequest{
		Name:     "Board",
		ViewType: sprintdom.ViewTypeBoard,
		Config: &dto.ViewConfigDTO{
			ColumnBy: "status",
			Filters: &dto.ViewFiltersDTO{
				TaskTypes: &sprintdom.FilterConfig{
					All: false,
					Items: map[string]sprintdom.FilterEntry{
						"normal":     sprintdom.FilterEntryNested(sprintdom.FilterConfig{All: true}),
						systemTypeID: sprintdom.FilterEntryInclude(),
					},
				},
				Sprints: &sprintdom.FilterConfig{
					All: false,
					Items: map[string]sprintdom.FilterEntry{
						sprintID.String(): sprintdom.FilterEntryInclude(),
					},
				},
			},
		},
	}

	input := req.ToCreateInput(sprintID, projectID)

	f := input.Config.Filters
	if f == nil {
		t.Fatal("expected non-nil Filters in domain input")
	}
	if f.TaskTypes == nil {
		t.Fatal("expected non-nil TaskTypes in domain input filters")
	}
	if len(f.TaskTypes.Items) != 2 {
		t.Errorf("TaskTypes.Items: want 2, got %d", len(f.TaskTypes.Items))
	}
	normalEntry, ok := f.TaskTypes.Items["normal"]
	if !ok || !normalEntry.IsNested() {
		t.Error("expected 'normal' entry to be a nested FilterConfig")
	}
	sysEntry, ok := f.TaskTypes.Items[systemTypeID]
	if !ok || sysEntry.IsNested() || !sysEntry.Flag() {
		t.Errorf("expected system type entry to be a plain include flag, got nested=%v flag=%v", sysEntry.IsNested(), sysEntry.Flag())
	}
	if f.Sprints == nil || len(f.Sprints.Items) != 1 {
		t.Errorf("Sprints.Items: want 1, got %v", f.Sprints)
	}
	sprintEntry, ok := f.Sprints.Items[sprintID.String()]
	if !ok || !sprintEntry.Flag() {
		t.Errorf("expected sprint entry to be included, got %+v", sprintEntry)
	}
}

// TestViewFiltersDTO_NilTaskTypesPreserved verifies that a nil TaskTypes does
// not get set to a non-nil value after a roundtrip through entity → DTO.
func TestViewFiltersDTO_NilTaskTypesPreserved(t *testing.T) {
	entity := &sprintdom.SprintView{
		ID:        uuid.New(),
		ProjectID: uuid.New(),
		Name:      "Table",
		ViewType:  sprintdom.ViewTypeTable,
		Config: sprintdom.ViewConfig{
			Filters: &sprintdom.ViewFilters{
				Statuses: &sprintdom.FilterConfig{
					All: false,
					Items: map[string]sprintdom.FilterEntry{
						uuid.NewString(): sprintdom.FilterEntryInclude(),
					},
				},
				// TaskTypes intentionally absent.
			},
		},
	}

	resp := dto.ViewFromEntity(entity)
	if resp.Config.Filters == nil {
		t.Fatal("expected non-nil Filters")
	}
	if resp.Config.Filters.TaskTypes != nil {
		t.Errorf("expected nil TaskTypes, got %+v", resp.Config.Filters.TaskTypes)
	}
}
