package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	docdom "github.com/Paca-AI/api/internal/domain/doc"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =============================================================================
// GORM models
// =============================================================================

type docFolderRecord struct {
	ID        string    `gorm:"primarykey;type:uuid"`
	ProjectID string    `gorm:"type:uuid;not null;column:project_id"`
	ParentID  *string   `gorm:"type:uuid;column:parent_id"`
	Name      string    `gorm:"not null;column:name"`
	Position  int       `gorm:"not null;default:0;column:position"`
	CreatedBy *string   `gorm:"type:uuid;column:created_by"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (docFolderRecord) TableName() string { return "doc_folders" }

type documentRecord struct {
	ID        string          `gorm:"primarykey;type:uuid"`
	ProjectID string          `gorm:"type:uuid;not null;column:project_id"`
	FolderID  *string         `gorm:"type:uuid;column:folder_id"`
	Title     string          `gorm:"not null;column:title"`
	Content   json.RawMessage `gorm:"type:jsonb;column:content"`
	Position  int             `gorm:"not null;default:0;column:position"`
	CreatedBy *string         `gorm:"type:uuid;column:created_by"`
	UpdatedBy *string         `gorm:"type:uuid;column:updated_by"`
	CreatedAt time.Time       `gorm:"column:created_at"`
	UpdatedAt time.Time       `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt  `gorm:"column:deleted_at"`
}

func (documentRecord) TableName() string { return "documents" }

type docSnapshotRecord struct {
	ID             string          `gorm:"primarykey;type:uuid"`
	DocumentID     string          `gorm:"type:uuid;not null;column:document_id"`
	Title          string          `gorm:"not null;column:title"`
	Content        json.RawMessage `gorm:"type:jsonb;column:content"`
	SnapshotNumber int64           `gorm:"not null;default:0;column:snapshot_number"`
	CreatedBy      *string         `gorm:"type:uuid;column:created_by"`
	CreatedAt      time.Time       `gorm:"column:created_at"`

	// Joined from project_members → users.
	CreatedByName *string `gorm:"->;column:created_by_name"`
}

func (docSnapshotRecord) TableName() string { return "doc_snapshots" }

type docActivityRecord struct {
	ID           string          `gorm:"primarykey;type:uuid"`
	DocumentID   string          `gorm:"type:uuid;not null;column:document_id"`
	ActorID      *string         `gorm:"type:uuid;column:actor_id"`
	ActivityType string          `gorm:"not null;column:activity_type"`
	Content      json.RawMessage `gorm:"type:jsonb;not null;column:content"`
	CreatedAt    time.Time       `gorm:"column:created_at"`
	UpdatedAt    time.Time       `gorm:"column:updated_at"`
	DeletedAt    gorm.DeletedAt  `gorm:"column:deleted_at"`

	// Joined from project_members + users.
	ActorFullName *string `gorm:"->;column:actor_full_name"`
	ActorUsername *string `gorm:"->;column:actor_username"`
}

func (docActivityRecord) TableName() string { return "doc_activities" }

// =============================================================================
// DocumentRepository
// =============================================================================

// DocumentRepository is the GORM implementation of docdom.Repository.
type DocumentRepository struct {
	db *gorm.DB
}

// NewDocumentRepository returns a new DocumentRepository backed by db.
func NewDocumentRepository(db *gorm.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

// =============================================================================
// Mapping helpers
// =============================================================================

func folderFromRecord(r docFolderRecord) *docdom.DocFolder {
	f := &docdom.DocFolder{
		ID:        uuid.MustParse(r.ID),
		ProjectID: uuid.MustParse(r.ProjectID),
		Name:      r.Name,
		Position:  r.Position,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
	if r.ParentID != nil {
		id := uuid.MustParse(*r.ParentID)
		f.ParentID = &id
	}
	if r.CreatedBy != nil {
		id := uuid.MustParse(*r.CreatedBy)
		f.CreatedBy = &id
	}
	return f
}

func folderToRecord(f *docdom.DocFolder) docFolderRecord {
	r := docFolderRecord{
		ID:        f.ID.String(),
		ProjectID: f.ProjectID.String(),
		Name:      f.Name,
		Position:  f.Position,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
	if f.ParentID != nil {
		s := f.ParentID.String()
		r.ParentID = &s
	}
	if f.CreatedBy != nil {
		s := f.CreatedBy.String()
		r.CreatedBy = &s
	}
	return r
}

func documentFromRecord(r documentRecord) *docdom.Document {
	d := &docdom.Document{
		ID:        uuid.MustParse(r.ID),
		ProjectID: uuid.MustParse(r.ProjectID),
		Title:     r.Title,
		Content:   r.Content,
		Position:  r.Position,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
	if r.FolderID != nil {
		id := uuid.MustParse(*r.FolderID)
		d.FolderID = &id
	}
	if r.CreatedBy != nil {
		id := uuid.MustParse(*r.CreatedBy)
		d.CreatedBy = &id
	}
	if r.UpdatedBy != nil {
		id := uuid.MustParse(*r.UpdatedBy)
		d.UpdatedBy = &id
	}
	if r.DeletedAt.Valid {
		d.DeletedAt = &r.DeletedAt.Time
	}
	return d
}

func documentToRecord(d *docdom.Document) documentRecord {
	r := documentRecord{
		ID:        d.ID.String(),
		ProjectID: d.ProjectID.String(),
		Title:     d.Title,
		Content:   d.Content,
		Position:  d.Position,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
	if d.FolderID != nil {
		s := d.FolderID.String()
		r.FolderID = &s
	}
	if d.CreatedBy != nil {
		s := d.CreatedBy.String()
		r.CreatedBy = &s
	}
	if d.UpdatedBy != nil {
		s := d.UpdatedBy.String()
		r.UpdatedBy = &s
	}
	return r
}

func snapshotFromRecord(r docSnapshotRecord) *docdom.DocSnapshot {
	s := &docdom.DocSnapshot{
		ID:             uuid.MustParse(r.ID),
		DocumentID:     uuid.MustParse(r.DocumentID),
		Title:          r.Title,
		Content:        r.Content,
		SnapshotNumber: r.SnapshotNumber,
		CreatedAt:      r.CreatedAt,
	}
	if r.CreatedBy != nil {
		id := uuid.MustParse(*r.CreatedBy)
		s.CreatedBy = &id
	}
	if r.CreatedByName != nil {
		s.CreatedByName = *r.CreatedByName
	}
	return s
}

func activityFromDocRecord(r docActivityRecord) *docdom.Activity {
	var deletedAt *time.Time
	if r.DeletedAt.Valid {
		deletedAt = &r.DeletedAt.Time
	}
	a := &docdom.Activity{
		ID:           uuid.MustParse(r.ID),
		DocumentID:   uuid.MustParse(r.DocumentID),
		ActivityType: docdom.ActivityType(r.ActivityType),
		Content:      r.Content,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
		DeletedAt:    deletedAt,
	}
	if r.ActorID != nil {
		id := uuid.MustParse(*r.ActorID)
		a.ActorID = &id
	}
	if r.ActorFullName != nil {
		a.ActorName = *r.ActorFullName
	}
	if r.ActorUsername != nil {
		a.ActorUsername = *r.ActorUsername
	}
	return a
}

// =============================================================================
// Folder CRUD
// =============================================================================

// ListFolders returns all folders for a project.
func (r *DocumentRepository) ListFolders(_ context.Context, projectID uuid.UUID) ([]*docdom.DocFolder, error) {
	var records []docFolderRecord
	if err := r.db.Where("project_id = ?", projectID.String()).Order("position ASC, name ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]*docdom.DocFolder, 0, len(records))
	for _, rec := range records {
		out = append(out, folderFromRecord(rec))
	}
	return out, nil
}

// FindFolderByID returns a single folder.
func (r *DocumentRepository) FindFolderByID(_ context.Context, id uuid.UUID) (*docdom.DocFolder, error) {
	var rec docFolderRecord
	if err := r.db.Where("id = ?", id.String()).First(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, docdom.ErrFolderNotFound
		}
		return nil, err
	}
	return folderFromRecord(rec), nil
}

// CreateFolder persists a new folder.
func (r *DocumentRepository) CreateFolder(_ context.Context, f *docdom.DocFolder) error {
	rec := folderToRecord(f)
	return r.db.Create(&rec).Error
}

// UpdateFolder persists mutable changes to a folder.
func (r *DocumentRepository) UpdateFolder(_ context.Context, f *docdom.DocFolder) error {
	rec := folderToRecord(f)
	return r.db.Save(&rec).Error
}

// DeleteFolder permanently deletes a folder.
func (r *DocumentRepository) DeleteFolder(_ context.Context, id uuid.UUID) error {
	return r.db.Where("id = ?", id.String()).Delete(&docFolderRecord{}).Error
}

// =============================================================================
// Document CRUD
// =============================================================================

// ListDocuments returns non-deleted documents for a project.
func (r *DocumentRepository) ListDocuments(_ context.Context, projectID uuid.UUID, folderID *uuid.UUID) ([]*docdom.Document, error) {
	q := r.db.Where("project_id = ? AND deleted_at IS NULL", projectID.String())
	if folderID != nil {
		q = q.Where("folder_id = ?", folderID.String())
	}
	var records []documentRecord
	if err := q.Order("position ASC, title ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]*docdom.Document, 0, len(records))
	for _, rec := range records {
		out = append(out, documentFromRecord(rec))
	}
	return out, nil
}

// FindDocumentByID returns a single non-deleted document.
func (r *DocumentRepository) FindDocumentByID(_ context.Context, id uuid.UUID) (*docdom.Document, error) {
	var rec documentRecord
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id.String()).First(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, docdom.ErrDocNotFound
		}
		return nil, err
	}
	return documentFromRecord(rec), nil
}

// CreateDocument persists a new document.
func (r *DocumentRepository) CreateDocument(_ context.Context, d *docdom.Document) error {
	rec := documentToRecord(d)
	return r.db.Create(&rec).Error
}

// UpdateDocument persists mutable changes to a document.
func (r *DocumentRepository) UpdateDocument(_ context.Context, d *docdom.Document) error {
	rec := documentToRecord(d)
	return r.db.Save(&rec).Error
}

// DeleteDocument soft-deletes a document.
func (r *DocumentRepository) DeleteDocument(_ context.Context, id uuid.UUID) error {
	return r.db.Model(&documentRecord{}).
		Where("id = ?", id.String()).
		Update("deleted_at", time.Now()).Error
}

// =============================================================================
// Snapshot CRUD
// =============================================================================

func (r *DocumentRepository) snapshotQuery() *gorm.DB {
	return r.db.Table("doc_snapshots ds").
		Select("ds.*, u.full_name AS created_by_name").
		Joins("LEFT JOIN project_members pm ON pm.id = ds.created_by").
		Joins("LEFT JOIN users u ON u.id = pm.user_id")
}

// ListSnapshots returns all snapshots for a document, newest first.
func (r *DocumentRepository) ListSnapshots(_ context.Context, documentID uuid.UUID) ([]*docdom.DocSnapshot, error) {
	var records []docSnapshotRecord
	if err := r.snapshotQuery().
		Where("ds.document_id = ?", documentID.String()).
		Order("ds.snapshot_number DESC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]*docdom.DocSnapshot, 0, len(records))
	for _, rec := range records {
		out = append(out, snapshotFromRecord(rec))
	}
	return out, nil
}

// FindSnapshotByID returns a single snapshot.
func (r *DocumentRepository) FindSnapshotByID(_ context.Context, id uuid.UUID) (*docdom.DocSnapshot, error) {
	var rec docSnapshotRecord
	if err := r.snapshotQuery().
		Where("ds.id = ?", id.String()).
		First(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, docdom.ErrSnapshotNotFound
		}
		return nil, err
	}
	return snapshotFromRecord(rec), nil
}

// FindLatestSnapshot returns the snapshot with the highest snapshot_number.
func (r *DocumentRepository) FindLatestSnapshot(_ context.Context, documentID uuid.UUID) (*docdom.DocSnapshot, error) {
	var rec docSnapshotRecord
	if err := r.snapshotQuery().
		Where("ds.document_id = ?", documentID.String()).
		Order("ds.snapshot_number DESC").
		First(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // no snapshots yet — not an error
		}
		return nil, err
	}
	return snapshotFromRecord(rec), nil
}

// CreateSnapshot persists a new snapshot.
func (r *DocumentRepository) CreateSnapshot(_ context.Context, s *docdom.DocSnapshot) error {
	rec := docSnapshotRecord{
		ID:         s.ID.String(),
		DocumentID: s.DocumentID.String(),
		Title:      s.Title,
		Content:    s.Content,
		CreatedAt:  s.CreatedAt,
	}
	if s.CreatedBy != nil {
		str := s.CreatedBy.String()
		rec.CreatedBy = &str
	}
	return r.db.Create(&rec).Error
}

// DeleteRecentSnapshotsExcept deletes all snapshots for a document created at
// or after `since` whose ID is not `excludeID`. This consolidates rapid saves
// so that at most one snapshot exists per time window.
func (r *DocumentRepository) DeleteRecentSnapshotsExcept(_ context.Context, documentID uuid.UUID, excludeID uuid.UUID, since time.Time) error {
	return r.db.
		Where("document_id = ? AND id != ? AND created_at >= ?", documentID.String(), excludeID.String(), since).
		Delete(&docSnapshotRecord{}).Error
}

// =============================================================================
// Activity CRUD
// =============================================================================

func (r *DocumentRepository) activityQuery() *gorm.DB {
	return r.db.Table("doc_activities da").
		Select("da.*, COALESCE(u.full_name, ag.name) AS actor_full_name, COALESCE(u.username, ag.handle) AS actor_username").
		Joins("LEFT JOIN project_members pm ON pm.id = da.actor_id").
		Joins("LEFT JOIN users u ON u.id = pm.user_id").
		Joins("LEFT JOIN agents ag ON ag.id = pm.agent_id")
}

// ListActivities returns non-deleted activities for a document, oldest first.
func (r *DocumentRepository) ListActivities(_ context.Context, documentID uuid.UUID) ([]*docdom.Activity, error) {
	var records []docActivityRecord
	if err := r.activityQuery().
		Where("da.document_id = ? AND da.deleted_at IS NULL", documentID.String()).
		Order("da.created_at ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]*docdom.Activity, 0, len(records))
	for _, rec := range records {
		out = append(out, activityFromDocRecord(rec))
	}
	return out, nil
}

// FindActivityByID returns a single activity (including soft-deleted).
func (r *DocumentRepository) FindActivityByID(_ context.Context, id uuid.UUID) (*docdom.Activity, error) {
	var rec docActivityRecord
	if err := r.activityQuery().
		Where("da.id = ?", id.String()).
		First(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, docdom.ErrActivityNotFound
		}
		return nil, err
	}
	return activityFromDocRecord(rec), nil
}

// CreateActivity persists a new activity record.
func (r *DocumentRepository) CreateActivity(_ context.Context, a *docdom.Activity) error {
	content := a.Content
	if content == nil {
		content = json.RawMessage("{}")
	}
	rec := docActivityRecord{
		ID:           a.ID.String(),
		DocumentID:   a.DocumentID.String(),
		ActivityType: string(a.ActivityType),
		Content:      content,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
	if a.ActorID != nil {
		s := a.ActorID.String()
		rec.ActorID = &s
	}
	return r.db.Create(&rec).Error
}

// UpdateActivity persists mutable changes to an activity.
func (r *DocumentRepository) UpdateActivity(_ context.Context, a *docdom.Activity) error {
	content := a.Content
	if content == nil {
		content = json.RawMessage("{}")
	}
	return r.db.Model(&docActivityRecord{}).
		Where("id = ?", a.ID.String()).
		Updates(map[string]any{
			"content":    content,
			"updated_at": a.UpdatedAt,
		}).Error
}

// DeleteActivity soft-deletes an activity.
func (r *DocumentRepository) DeleteActivity(_ context.Context, id uuid.UUID) error {
	return r.db.Model(&docActivityRecord{}).
		Where("id = ?", id.String()).
		Update("deleted_at", time.Now()).Error
}
