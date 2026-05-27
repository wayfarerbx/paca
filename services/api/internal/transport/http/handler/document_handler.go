package handler

import (
	"encoding/json"

	"github.com/Paca-AI/api/internal/apierr"
	docdom "github.com/Paca-AI/api/internal/domain/doc"
	"github.com/Paca-AI/api/internal/transport/http/dto"
	"github.com/Paca-AI/api/internal/transport/http/middleware"
	"github.com/Paca-AI/api/internal/transport/http/presenter"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DocumentHandler handles documentation endpoints.
type DocumentHandler struct {
	svc         docdom.Service
	activitySvc docdom.ActivityService
}

// NewDocumentHandler returns a DocumentHandler wired to the doc service and
// activity service.
func NewDocumentHandler(svc docdom.Service, activitySvc docdom.ActivityService) *DocumentHandler {
	return &DocumentHandler{svc: svc, activitySvc: activitySvc}
}

// =============================================================================
// Folder endpoints
// =============================================================================

// ListFolders handles GET /projects/:projectId/docs/folders.
func (h *DocumentHandler) ListFolders(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	folders, err := h.svc.ListFolders(c.Request.Context(), projectID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.DocFolderResponse, 0, len(folders))
	for _, f := range folders {
		resp = append(resp, dto.DocFolderFromEntity(f))
	}
	presenter.OK(c, gin.H{"items": resp})
}

// CreateFolder handles POST /projects/:projectId/docs/folders.
func (h *DocumentHandler) CreateFolder(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.CreateFolderRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	var createdBy *uuid.UUID
	if actorID, ok := middleware.ActorIDFromContext(c.Request.Context()); ok {
		createdBy = &actorID
	}

	f, err := h.svc.CreateFolder(c.Request.Context(), docdom.CreateFolderInput{
		ProjectID: projectID,
		ParentID:  req.ParentID,
		Name:      req.Name,
		CreatedBy: createdBy,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Created(c, dto.DocFolderFromEntity(f))
}

// UpdateFolder handles PATCH /projects/:projectId/docs/folders/:folderId.
func (h *DocumentHandler) UpdateFolder(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	folderID, err := parseDocFolderID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.UpdateFolderRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	f, err := h.svc.UpdateFolder(c.Request.Context(), folderID, docdom.UpdateFolderInput{
		ProjectID: projectID,
		Name:      req.Name,
		ParentID:  req.ParentID.Ptr(),
		Position:  req.Position,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.DocFolderFromEntity(f))
}

// DeleteFolder handles DELETE /projects/:projectId/docs/folders/:folderId.
func (h *DocumentHandler) DeleteFolder(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	folderID, err := parseDocFolderID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if err := h.svc.DeleteFolder(c.Request.Context(), folderID, projectID); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.NoContent(c)
}

// =============================================================================
// Document endpoints
// =============================================================================

// ListDocuments handles GET /projects/:projectId/docs.
// Optional query param: folder_id — filter by folder.
func (h *DocumentHandler) ListDocuments(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var folderID *uuid.UUID
	if raw := c.Query("folder_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			presenter.Error(c, apierr.New(apierr.CodeBadRequest, "invalid folder_id"))
			return
		}
		folderID = &id
	}

	docs, err := h.svc.ListDocuments(c.Request.Context(), projectID, folderID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.DocumentListItemResponse, 0, len(docs))
	for _, d := range docs {
		resp = append(resp, dto.DocumentListItemFromEntity(d))
	}
	presenter.OK(c, gin.H{"items": resp})
}

// GetDocument handles GET /projects/:projectId/docs/:docId.
func (h *DocumentHandler) GetDocument(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	docID, err := parseDocID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	d, err := h.svc.GetDocument(c.Request.Context(), docID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	if d.ProjectID != projectID {
		presenter.Error(c, docdom.ErrDocNotFound)
		return
	}
	presenter.OK(c, dto.DocumentFromEntity(d))
}

// CreateDocument handles POST /projects/:projectId/docs.
func (h *DocumentHandler) CreateDocument(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.CreateDocumentRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	var createdBy *uuid.UUID
	var createdByAgent *uuid.UUID
	if actorID, ok := middleware.ActorIDFromContext(c.Request.Context()); ok {
		createdBy = &actorID
	}
	if agentID, _ := middleware.AgentIDFromContext(c.Request.Context()); agentID != uuid.Nil {
		createdByAgent = &agentID
	}

	d, err := h.svc.CreateDocument(c.Request.Context(), docdom.CreateDocumentInput{
		ProjectID: projectID,
		FolderID:  req.FolderID,
		Title:     req.Title,
		Content:   req.Content,
		CreatedBy: createdBy,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	// Record creation activity (best-effort).
	if createdBy != nil {
		content, _ := json.Marshal(map[string]string{"title": d.Title})
		_ = h.activitySvc.RecordActivity(c.Request.Context(), docdom.RecordActivityInput{
			DocumentID:   d.ID,
			ProjectID:    projectID,
			ActorID:      createdBy,
			ActorAgentID: createdByAgent,
			ActivityType: docdom.ActivityTypeDocCreated,
			Content:      content,
		})
	}

	presenter.Created(c, dto.DocumentFromEntity(d))
}

// UpdateDocument handles PATCH /projects/:projectId/docs/:docId.
func (h *DocumentHandler) UpdateDocument(c *gin.Context) {
	docID, err := parseDocID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.UpdateDocumentRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	var updatedBy *uuid.UUID
	var updatedByAgent *uuid.UUID
	if actorID, ok := middleware.ActorIDFromContext(c.Request.Context()); ok {
		updatedBy = &actorID
	}
	if agentID, _ := middleware.AgentIDFromContext(c.Request.Context()); agentID != uuid.Nil {
		updatedByAgent = &agentID
	}

	// Capture old state for activity recording.
	oldDoc, _ := h.svc.GetDocument(c.Request.Context(), docID)

	d, err := h.svc.UpdateDocument(c.Request.Context(), docID, docdom.UpdateDocumentInput{
		Title:     req.Title,
		Content:   req.Content.Ptr(),
		FolderID:  req.FolderID.Ptr(),
		Position:  req.Position,
		UpdatedBy: updatedBy,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	// Record update activity (best-effort).
	if updatedBy != nil && oldDoc != nil {
		changes := docChangedFields(oldDoc, req)
		if len(changes) > 0 {
			content, _ := json.Marshal(map[string]any{"changes": changes})
			_ = h.activitySvc.RecordActivity(c.Request.Context(), docdom.RecordActivityInput{
				DocumentID:   docID,
				ProjectID:    oldDoc.ProjectID,
				ActorID:      updatedBy,
				ActorAgentID: updatedByAgent,
				ActivityType: docdom.ActivityTypeDocUpdated,
				Content:      content,
			})
		}
	}

	presenter.OK(c, dto.DocumentFromEntity(d))
}

// DeleteDocument handles DELETE /projects/:projectId/docs/:docId.
func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	docID, err := parseDocID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	if err := h.svc.DeleteDocument(c.Request.Context(), docID); err != nil {
		presenter.Error(c, err)
		return
	}

	// Record deletion activity (best-effort).
	if actorID, ok := middleware.ActorIDFromContext(c.Request.Context()); ok {
		agentID, _ := middleware.AgentIDFromContext(c.Request.Context())
		var agentIDPtr *uuid.UUID
		if agentID != uuid.Nil {
			agentIDPtr = &agentID
		}
		_ = h.activitySvc.RecordActivity(c.Request.Context(), docdom.RecordActivityInput{
			DocumentID:   docID,
			ProjectID:    projectID,
			ActorID:      &actorID,
			ActorAgentID: agentIDPtr,
			ActivityType: docdom.ActivityTypeDocDeleted,
		})
	}

	presenter.NoContent(c)
}

// =============================================================================
// Snapshot endpoints
// =============================================================================

// ListSnapshots handles GET /projects/:projectId/docs/:docId/snapshots.
func (h *DocumentHandler) ListSnapshots(c *gin.Context) {
	docID, err := parseDocID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	snaps, err := h.svc.ListSnapshots(c.Request.Context(), docID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.DocSnapshotResponse, 0, len(snaps))
	for _, s := range snaps {
		resp = append(resp, dto.DocSnapshotFromEntity(s))
	}
	presenter.OK(c, gin.H{"items": resp})
}

// GetSnapshot handles GET /projects/:projectId/docs/:docId/snapshots/:snapshotId.
func (h *DocumentHandler) GetSnapshot(c *gin.Context) {
	snapshotID, err := parseDocSnapshotID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	snap, err := h.svc.GetSnapshot(c.Request.Context(), snapshotID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.DocSnapshotFromEntity(snap))
}

// =============================================================================
// Activity / comment endpoints
// =============================================================================

// ListActivities handles GET /projects/:projectId/docs/:docId/activities.
func (h *DocumentHandler) ListActivities(c *gin.Context) {
	docID, err := parseDocID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	activities, err := h.activitySvc.ListActivities(c.Request.Context(), docID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	resp := make([]dto.DocActivityResponse, 0, len(activities))
	for _, a := range activities {
		resp = append(resp, dto.DocActivityFromEntity(a))
	}
	presenter.OK(c, gin.H{"items": resp})
}

// AddComment handles POST /projects/:projectId/docs/:docId/comments.
func (h *DocumentHandler) AddComment(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	docID, err := parseDocID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	actorID, ok := middleware.ActorIDFromContext(c.Request.Context())
	if !ok {
		presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
		return
	}

	var req dto.AddDocCommentRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	agentID, _ := middleware.AgentIDFromContext(c.Request.Context())
	var agentIDPtr *uuid.UUID
	if agentID != uuid.Nil {
		agentIDPtr = &agentID
	}

	a, err := h.activitySvc.AddComment(c.Request.Context(), docdom.AddCommentInput{
		DocumentID: docID,
		ProjectID:  projectID,
		ActorID:    actorID,
		AgentID:    agentIDPtr,
		Content:    req.Content,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Created(c, dto.DocActivityFromEntity(a))
}

// UpdateComment handles PATCH /projects/:projectId/docs/:docId/comments/:commentId.
func (h *DocumentHandler) UpdateComment(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	commentID, err := parseDocCommentID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	actorID, ok := middleware.ActorIDFromContext(c.Request.Context())
	if !ok {
		presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
		return
	}

	var req dto.UpdateDocCommentRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	agentID, _ := middleware.AgentIDFromContext(c.Request.Context())
	var agentIDPtr *uuid.UUID
	if agentID != uuid.Nil {
		agentIDPtr = &agentID
	}

	a, err := h.activitySvc.UpdateComment(c.Request.Context(), commentID, projectID, actorID, agentIDPtr, req.Content)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.OK(c, dto.DocActivityFromEntity(a))
}

// DeleteComment handles DELETE /projects/:projectId/docs/:docId/comments/:commentId.
func (h *DocumentHandler) DeleteComment(c *gin.Context) {
	projectID, err := parseProjectID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	commentID, err := parseDocCommentID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	actorID, ok := middleware.ActorIDFromContext(c.Request.Context())
	if !ok {
		presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
		return
	}

	agentID, _ := middleware.AgentIDFromContext(c.Request.Context())
	var agentIDPtr *uuid.UUID
	if agentID != uuid.Nil {
		agentIDPtr = &agentID
	}

	if err := h.activitySvc.DeleteComment(c.Request.Context(), commentID, projectID, actorID, agentIDPtr); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.NoContent(c)
}

// =============================================================================
// Parse helpers
// =============================================================================

func parseDocID(c *gin.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("docId"))
	if err != nil {
		return uuid.Nil, apierr.New(apierr.CodeBadRequest, "invalid document id")
	}
	return id, nil
}

func parseDocFolderID(c *gin.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("folderId"))
	if err != nil {
		return uuid.Nil, apierr.New(apierr.CodeBadRequest, "invalid folder id")
	}
	return id, nil
}

func parseDocSnapshotID(c *gin.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("snapshotId"))
	if err != nil {
		return uuid.Nil, apierr.New(apierr.CodeBadRequest, "invalid snapshot id")
	}
	return id, nil
}

func parseDocCommentID(c *gin.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("commentId"))
	if err != nil {
		return uuid.Nil, apierr.New(apierr.CodeBadRequest, "invalid comment id")
	}
	return id, nil
}

// =============================================================================
// Activity helpers
// =============================================================================

// docChangedFields compares the old document against the patch request and
// returns a FieldChange for each field that actually changed.
func docChangedFields(old *docdom.Document, req dto.UpdateDocumentRequest) []docdom.FieldChange {
	var changes []docdom.FieldChange

	if req.Title != nil && *req.Title != old.Title {
		changes = append(changes, docdom.FieldChange{Field: "title", Old: old.Title, New: *req.Title})
	}

	if req.Content.Set {
		if string(req.Content.Value) != string(old.Content) {
			changes = append(changes, docdom.FieldChange{Field: "content"})
		}
	}

	if req.FolderID.Set {
		oldVal := uuidPtrToStr(old.FolderID)
		newVal := ""
		if req.FolderID.Value != nil {
			newVal = req.FolderID.Value.String()
		}
		if oldVal != newVal {
			changes = append(changes, docdom.FieldChange{Field: "folder_id", Old: oldVal, New: newVal})
		}
	}

	return changes
}
