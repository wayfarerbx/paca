package taskdom

import "errors"

// Sentinel domain errors for the task aggregate.
var (
	ErrTaskNotFound     = errors.New("task: not found")
	ErrTaskTitleInvalid = errors.New("task: title is empty or invalid")

	ErrTypeNotFound     = errors.New("task type: not found")
	ErrTypeNameInvalid  = errors.New("task type: name is empty or invalid")
	ErrTypeIsSystem     = errors.New("task type: system types cannot be modified")
	ErrTypeNameReserved = errors.New("task type: name is reserved for system types")

	ErrStatusNotFound        = errors.New("task status: not found")
	ErrStatusNameInvalid     = errors.New("task status: name is empty or invalid")
	ErrStatusCategoryInvalid = errors.New("task status: invalid category value")

	ErrCustomFieldNotFound    = errors.New("custom field: not found")
	ErrCustomFieldKeyInvalid  = errors.New("custom field: key is empty or invalid")
	ErrCustomFieldKeyTaken    = errors.New("custom field: key already in use within project")
	ErrCustomFieldTypeInvalid = errors.New("custom field: invalid field type")
	ErrCustomFieldNameInvalid = errors.New("custom field: display name is empty or invalid")

	ErrBDDScenarioNotFound     = errors.New("bdd scenario: not found")
	ErrBDDScenarioTitleInvalid = errors.New("bdd scenario: title is empty or invalid")

	// Activity / comment errors.
	ErrActivityNotFound    = errors.New("activity: not found")
	ErrActivityForbidden   = errors.New("activity: only the author can modify this comment")
	ErrActivityNotAComment = errors.New("activity: this entry is not a comment and cannot be edited")
	ErrCommentTextInvalid  = errors.New("activity: comment text must not be empty")
)
