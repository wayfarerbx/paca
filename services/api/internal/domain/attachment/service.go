package attachmentdom

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/paca/api/internal/platform/storage"
)

// Service is the combined attachment management contract.
type Service interface {
	// InitiateUpload creates a pending File record and returns either a
	// single-part presigned PUT URL (files < MultipartThreshold) or the
	// multipart upload descriptor with per-part presigned URLs.
	InitiateUpload(ctx context.Context, in InitiateUploadInput) (*UploadSession, error)

	// CompleteUpload marks the File as uploaded and creates the
	// TaskAttachment join record.  For multipart uploads the caller must
	// supply the completed parts so the server can call
	// CompleteMultipartUpload on the object store.
	CompleteUpload(ctx context.Context, in CompleteUploadInput) (*TaskAttachment, error)

	// GetDownloadURL returns a short-lived presigned GET URL for the file
	// that backs the given attachment. Verifies the attachment belongs to taskID.
	// Set forceDownload true to embed Content-Disposition: attachment so the
	// browser downloads the file rather than previewing it inline.
	GetDownloadURL(ctx context.Context, taskID, attachmentID uuid.UUID, ttl time.Duration, forceDownload bool) (string, error)

	// ListTaskAttachments returns all confirmed attachments for the given task.
	ListTaskAttachments(ctx context.Context, taskID uuid.UUID) ([]*TaskAttachment, error)

	// DeleteTaskAttachment removes the task→file association for the given
	// attachment. Verifies the attachment belongs to taskID before deleting.
	DeleteTaskAttachment(ctx context.Context, taskID, attachmentID uuid.UUID) error
}

// UploadSession is returned by InitiateUpload and carries everything the
// client needs to upload the file directly to the object store.
type UploadSession struct {
	FileID      uuid.UUID `json:"file_id"`
	IsMultipart bool      `json:"is_multipart"`
	// UploadURL is set for single-part uploads (IsMultipart=false).
	UploadURL string `json:"upload_url,omitempty"`
	// Multipart is set for large files (IsMultipart=true).
	Multipart *storage.MultipartUpload `json:"multipart,omitempty"`
}

// InitiateUploadInput carries the client-supplied file metadata for starting
// an upload.
type InitiateUploadInput struct {
	TaskID      uuid.UUID
	FileName    string
	ContentType string
	FileSize    int64
	UploadedBy  uuid.UUID // user performing the upload
}

// CompletedPart mirrors storage.CompletedPart for multipart completion.
type CompletedPart struct {
	PartNumber int
	ETag       string
}

// CompleteUploadInput carries parameters for finishing an upload.
type CompleteUploadInput struct {
	FileID    uuid.UUID
	TaskID    uuid.UUID
	CreatedBy uuid.UUID
	// UploadID and Parts are required only for multipart uploads.
	UploadID *string
	Parts    []CompletedPart
}
