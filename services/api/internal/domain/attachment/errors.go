package attachmentdom

import "errors"

// Sentinel errors returned by attachment domain operations.
var (
	ErrFileNotFound       = errors.New("file not found")
	ErrAttachmentNotFound = errors.New("attachment not found")
	ErrUploadNotPending   = errors.New("file upload is not in pending state")
	ErrFileSizeZero       = errors.New("file size must be greater than zero")
	ErrFileNameEmpty      = errors.New("file name must not be empty")
	ErrContentTypeEmpty   = errors.New("content type must not be empty")
)
