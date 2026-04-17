// Package apierr defines machine-readable error codes used in API error
// responses. Clients should switch on Code values rather than HTTP status
// codes or human-readable messages, because messages are subject to change.
package apierr

// Code is a stable, machine-readable string that identifies a specific error
// condition. Codes are grouped by domain prefix (e.g. AUTH_, USER_).
type Code string

const (
	// CodeInvalidCredentials represents invalid authentication credentials.
	CodeInvalidCredentials Code = "AUTH_INVALID_CREDENTIALS"
	// CodeMissingToken represents a missing authentication token.
	CodeMissingToken Code = "AUTH_MISSING_TOKEN"
	// CodeTokenInvalid represents an invalid authentication token.
	CodeTokenInvalid Code = "AUTH_TOKEN_INVALID"
	// CodeUnauthenticated represents an unauthenticated request.
	CodeUnauthenticated Code = "AUTH_UNAUTHENTICATED"

	// CodeUserNotFound represents a user that was not found.
	CodeUserNotFound Code = "USER_NOT_FOUND"
	// CodeUsernameTaken represents a username that is already taken.
	CodeUsernameTaken Code = "USER_USERNAME_TAKEN"
	// CodeForbidden represents a forbidden action.
	CodeForbidden Code = "FORBIDDEN"
	// CodeGlobalRoleNotFound represents a global role that was not found.
	CodeGlobalRoleNotFound Code = "GLOBAL_ROLE_NOT_FOUND"
	// CodeGlobalRoleNameTaken represents a duplicate global role name.
	CodeGlobalRoleNameTaken Code = "GLOBAL_ROLE_NAME_TAKEN"
	// CodeGlobalRoleNameInvalid represents an invalid global role name.
	CodeGlobalRoleNameInvalid Code = "GLOBAL_ROLE_NAME_INVALID"

	// CodeGlobalRoleHasUsers indicates the role cannot be deleted because it
	// still has assigned users.
	CodeGlobalRoleHasUsers Code = "GLOBAL_ROLE_HAS_ASSIGNED_USERS"
	// CodeBadRequest represents a bad request.
	CodeBadRequest Code = "BAD_REQUEST"
	// CodeInternalError represents an internal server error.
	CodeInternalError Code = "INTERNAL_ERROR"
	// CodePasswordChangeRequired indicates the user must change their password
	// before accessing any other endpoint.
	CodePasswordChangeRequired Code = "AUTH_PASSWORD_CHANGE_REQUIRED"
	// CodeInvalidCurrentPassword indicates the supplied current password is wrong.
	CodeInvalidCurrentPassword Code = "USER_INVALID_CURRENT_PASSWORD"

	// CodeProjectNotFound indicates the requested project does not exist.
	CodeProjectNotFound Code = "PROJECT_NOT_FOUND"
	// CodeProjectNameTaken indicates the project name is already in use.
	CodeProjectNameTaken Code = "PROJECT_NAME_TAKEN"
	// CodeProjectNameInvalid indicates the project name is empty or invalid.
	CodeProjectNameInvalid Code = "PROJECT_NAME_INVALID"
	// CodeProjectPrefixInvalid indicates the task ID prefix is not valid.
	CodeProjectPrefixInvalid Code = "PROJECT_PREFIX_INVALID"

	// CodeProjectRoleNotFound indicates the requested project role does not exist.
	CodeProjectRoleNotFound Code = "PROJECT_ROLE_NOT_FOUND"
	// CodeProjectRoleNameTaken indicates the role name is already in use within the project.
	CodeProjectRoleNameTaken Code = "PROJECT_ROLE_NAME_TAKEN"
	// CodeProjectRoleNameInvalid indicates an invalid or empty role name.
	CodeProjectRoleNameInvalid Code = "PROJECT_ROLE_NAME_INVALID"
	// CodeProjectRoleHasMembers indicates the role cannot be deleted because members still use it.
	CodeProjectRoleHasMembers Code = "PROJECT_ROLE_HAS_MEMBERS"

	// CodeProjectMemberNotFound indicates the membership record was not found.
	CodeProjectMemberNotFound Code = "PROJECT_MEMBER_NOT_FOUND"
	// CodeProjectMemberAlreadyAdded indicates the user is already a member of the project.
	CodeProjectMemberAlreadyAdded Code = "PROJECT_MEMBER_ALREADY_ADDED"

	// CodeTaskNotFound indicates the requested task does not exist.
	CodeTaskNotFound Code = "TASK_NOT_FOUND"
	// CodeTaskTitleInvalid indicates an empty or invalid task title.
	CodeTaskTitleInvalid Code = "TASK_TITLE_INVALID"

	// CodeTaskTypeNotFound indicates the requested task type does not exist.
	CodeTaskTypeNotFound Code = "TASK_TYPE_NOT_FOUND"
	// CodeTaskTypeNameInvalid indicates an empty or invalid task type name.
	CodeTaskTypeNameInvalid Code = "TASK_TYPE_NAME_INVALID"
	// CodeTaskTypeIsSystem indicates an attempt to modify a system task type.
	CodeTaskTypeIsSystem Code = "TASK_TYPE_IS_SYSTEM"
	// CodeTaskTypeNameReserved indicates an attempt to use a reserved system type name.
	CodeTaskTypeNameReserved Code = "TASK_TYPE_NAME_RESERVED"

	// CodeTaskStatusNotFound indicates the requested task status does not exist.
	CodeTaskStatusNotFound Code = "TASK_STATUS_NOT_FOUND"
	// CodeTaskStatusNameInvalid indicates an empty or invalid task status name.
	CodeTaskStatusNameInvalid Code = "TASK_STATUS_NAME_INVALID"
	// CodeTaskStatusCategoryInvalid indicates an invalid status category value.
	CodeTaskStatusCategoryInvalid Code = "TASK_STATUS_CATEGORY_INVALID"

	// CodeSprintNotFound indicates the requested sprint does not exist.
	CodeSprintNotFound Code = "SPRINT_NOT_FOUND"
	// CodeSprintNameInvalid indicates an empty or invalid sprint name.
	CodeSprintNameInvalid Code = "SPRINT_NAME_INVALID"
	// CodeSprintStatusInvalid indicates an invalid sprint status value.
	CodeSprintStatusInvalid Code = "SPRINT_STATUS_INVALID"
	// CodeSprintAlreadyComplete indicates the sprint is already completed.
	CodeSprintAlreadyComplete Code = "SPRINT_ALREADY_COMPLETE"

	// CodeViewNotFound indicates the requested sprint view does not exist.
	CodeViewNotFound Code = "VIEW_NOT_FOUND"
	// CodeViewNameInvalid indicates an empty or invalid view name.
	CodeViewNameInvalid Code = "VIEW_NAME_INVALID"
	// CodeViewTypeInvalid indicates an invalid view type value.
	CodeViewTypeInvalid Code = "VIEW_TYPE_INVALID"
	// CodeViewIsLastView indicates the view cannot be deleted because it is the last remaining view.
	CodeViewIsLastView Code = "VIEW_IS_LAST_VIEW"
	// CodeViewReorderInvalid indicates the provided view IDs do not match the interaction's views.
	CodeViewReorderInvalid Code = "VIEW_REORDER_INVALID"

	// CodeCustomFieldNotFound indicates the requested custom field definition does not exist.
	CodeCustomFieldNotFound Code = "CUSTOM_FIELD_NOT_FOUND"
	// CodeCustomFieldKeyInvalid indicates an empty or invalid field key.
	CodeCustomFieldKeyInvalid Code = "CUSTOM_FIELD_KEY_INVALID"
	// CodeCustomFieldKeyTaken indicates the field key is already in use within the project.
	CodeCustomFieldKeyTaken Code = "CUSTOM_FIELD_KEY_TAKEN"
	// CodeCustomFieldTypeInvalid indicates an invalid field type value.
	CodeCustomFieldTypeInvalid Code = "CUSTOM_FIELD_TYPE_INVALID"
	// CodeCustomFieldNameInvalid indicates an empty or invalid display name.
	CodeCustomFieldNameInvalid Code = "CUSTOM_FIELD_NAME_INVALID"

	// CodeBDDScenarioNotFound indicates the requested BDD scenario does not exist.
	CodeBDDScenarioNotFound Code = "BDD_SCENARIO_NOT_FOUND"
	// CodeBDDScenarioTitleInvalid indicates an empty or invalid BDD scenario title.
	CodeBDDScenarioTitleInvalid Code = "BDD_SCENARIO_TITLE_INVALID"

	// CodeFileNotFound indicates the requested file record does not exist.
	CodeFileNotFound Code = "FILE_NOT_FOUND"
	// CodeAttachmentNotFound indicates the requested task attachment does not exist.
	CodeAttachmentNotFound Code = "ATTACHMENT_NOT_FOUND"
	// CodeUploadNotPending indicates the file is not in the pending upload state.
	CodeUploadNotPending Code = "ATTACHMENT_UPLOAD_NOT_PENDING"
	// CodeAttachmentInvalid indicates invalid input for creating an attachment.
	CodeAttachmentInvalid Code = "ATTACHMENT_INVALID"
	// CodeMultipartUploadIDRequired indicates that a multipart upload_id was not provided.
	CodeMultipartUploadIDRequired Code = "ATTACHMENT_MULTIPART_UPLOAD_ID_REQUIRED"
	// CodeNotMultipartUpload indicates that an upload_id was provided for a non-multipart file.
	CodeNotMultipartUpload Code = "ATTACHMENT_NOT_MULTIPART_UPLOAD"
	// CodeUploadIDMismatch indicates that the provided upload_id does not match the stored one.
	CodeUploadIDMismatch Code = "ATTACHMENT_UPLOAD_ID_MISMATCH"
	// CodeMultipartPartsEmpty indicates that no parts were provided for a multipart complete request.
	CodeMultipartPartsEmpty Code = "ATTACHMENT_MULTIPART_PARTS_EMPTY"

	// CodeActivityNotFound indicates the requested activity entry does not exist.
	CodeActivityNotFound Code = "ACTIVITY_NOT_FOUND"
	// CodeActivityForbidden indicates the caller is not the author of the comment.
	CodeActivityForbidden Code = "ACTIVITY_FORBIDDEN"
	// CodeActivityNotAComment indicates the entry is system-generated and cannot be edited.
	CodeActivityNotAComment Code = "ACTIVITY_NOT_A_COMMENT"
	// CodeCommentTextInvalid indicates an empty or invalid comment text.
	CodeCommentTextInvalid Code = "ACTIVITY_COMMENT_TEXT_INVALID"
)

// Error carries a machine-readable Code alongside a human-readable Message.
// It implements the error interface so it can propagate through service layers
// and be detected by the transport presenter.
type Error struct {
	Code    Code
	Message string
}

func (e *Error) Error() string { return e.Message }

// New returns a new *Error with the given code and message.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}
