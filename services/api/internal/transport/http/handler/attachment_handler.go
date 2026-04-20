package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/paca/api/internal/apierr"
	attachmentdom "github.com/paca/api/internal/domain/attachment"
	"github.com/paca/api/internal/transport/http/dto"
	"github.com/paca/api/internal/transport/http/middleware"
	"github.com/paca/api/internal/transport/http/presenter"
)

// AttachmentHandler handles task-attachment endpoints.
type AttachmentHandler struct {
	svc attachmentdom.Service
}

// NewAttachmentHandler returns an AttachmentHandler wired to the attachment service.
func NewAttachmentHandler(svc attachmentdom.Service) *AttachmentHandler {
	return &AttachmentHandler{svc: svc}
}

// InitiateUpload handles POST /projects/:projectId/tasks/:taskId/attachments/initiate-upload.
// It creates a pending file record and returns either a single presigned PUT URL
// (for files < 5 MiB) or a multipart upload session with per-part URLs.
func (h *AttachmentHandler) InitiateUpload(c *gin.Context) {
	taskID, err := parseTaskID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.InitiateUploadRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	claims := middleware.ClaimsFrom(c)
	if claims == nil {
		presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
		return
	}
	uploaderID, err := uuid.Parse(claims.Subject)
	if err != nil {
		presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "invalid subject in token"))
		return
	}

	session, err := h.svc.InitiateUpload(c.Request.Context(), attachmentdom.InitiateUploadInput{
		TaskID:      taskID,
		FileName:    req.FileName,
		ContentType: req.ContentType,
		FileSize:    req.FileSize,
		UploadedBy:  uploaderID,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Created(c, dto.UploadSessionFromDomain(session))
}

// CompleteUpload handles POST /projects/:projectId/tasks/:taskId/attachments/complete-upload.
// The client calls this after successfully uploading the file (or all parts) to
// the object store.  For multipart uploads the completed parts (with ETags) must
// be supplied so the server can reassemble the object.
func (h *AttachmentHandler) CompleteUpload(c *gin.Context) {
	taskID, err := parseTaskID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	var req dto.CompleteUploadRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	claims := middleware.ClaimsFrom(c)
	if claims == nil {
		presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "unauthenticated"))
		return
	}
	creatorID, err := uuid.Parse(claims.Subject)
	if err != nil {
		presenter.Error(c, apierr.New(apierr.CodeUnauthenticated, "invalid subject in token"))
		return
	}

	parts := make([]attachmentdom.CompletedPart, 0, len(req.Parts))
	for _, p := range req.Parts {
		parts = append(parts, attachmentdom.CompletedPart{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
		})
	}

	attachment, err := h.svc.CompleteUpload(c.Request.Context(), attachmentdom.CompleteUploadInput{
		FileID:    req.FileID,
		TaskID:    taskID,
		CreatedBy: creatorID,
		UploadID:  req.UploadID,
		Parts:     parts,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Created(c, dto.TaskAttachmentFromEntity(attachment))
}

// ListTaskAttachments handles GET /projects/:projectId/tasks/:taskId/attachments.
func (h *AttachmentHandler) ListTaskAttachments(c *gin.Context) {
	taskID, err := parseTaskID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	attachments, err := h.svc.ListTaskAttachments(c.Request.Context(), taskID)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	resp := make([]dto.TaskAttachmentResponse, 0, len(attachments))
	for _, a := range attachments {
		resp = append(resp, dto.TaskAttachmentFromEntity(a))
	}
	presenter.OK(c, gin.H{"items": resp})
}

// GetDownloadURL handles GET /projects/:projectId/tasks/:taskId/attachments/:attachmentId/download-url.
// Returns a short-lived presigned URL valid for 15 minutes.
// Add ?download=true to receive a URL with Content-Disposition: attachment
// so the browser forces a file download instead of inline preview.
func (h *AttachmentHandler) GetDownloadURL(c *gin.Context) {
	taskID, err := parseTaskID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	attachmentID, err := parseAttachmentID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	forceDownload := c.Query("download") == "true"

	url, err := h.svc.GetDownloadURL(c.Request.Context(), taskID, attachmentID, 15*time.Minute, forceDownload)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.OK(c, dto.DownloadURLResponse{URL: url})
}

// DeleteTaskAttachment handles DELETE /projects/:projectId/tasks/:taskId/attachments/:attachmentId.
func (h *AttachmentHandler) DeleteTaskAttachment(c *gin.Context) {
	taskID, err := parseTaskID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	attachmentID, err := parseAttachmentID(c)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	if err := h.svc.DeleteTaskAttachment(c.Request.Context(), taskID, attachmentID); err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.NoContent(c)
}

// --- helpers ---------------------------------------------------------------

func parseAttachmentID(c *gin.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("attachmentId"))
	if err != nil {
		return uuid.Nil, apierr.New(apierr.CodeBadRequest, "invalid attachment id")
	}
	return id, nil
}
