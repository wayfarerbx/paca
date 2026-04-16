// Package attachmentdom defines the file/attachment aggregate and its domain contracts.
package attachmentdom

import (
	"time"

	"github.com/google/uuid"
)

// UploadStatus represents the lifecycle of a file upload.
type UploadStatus string

// Upload status values for the File lifecycle.
const (
	UploadStatusPending  UploadStatus = "pending"
	UploadStatusUploaded UploadStatus = "uploaded"
	UploadStatusFailed   UploadStatus = "failed"
)

// File is the central metadata record for every object stored in the
// object store.  It is shared across all entities that can hold attachments.
type File struct {
	ID                uuid.UUID
	StorageKey        string
	Bucket            string
	FileName          string
	ContentType       string
	FileSize          int64
	UploadStatus      UploadStatus
	MultipartUploadID *string
	UploadedBy        *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TaskAttachment is the join record that links a confirmed File to a Task.
type TaskAttachment struct {
	ID        uuid.UUID
	TaskID    uuid.UUID
	FileID    uuid.UUID
	CreatedBy *uuid.UUID
	CreatedAt time.Time
	// Eagerly-loaded file details (populated by ListTaskAttachments).
	File *File
}
