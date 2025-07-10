// Package errors provides structured error handling for MemGOS
package errors

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/memtensor/memgos/pkg/types"
)

// ErrorCode represents specific error codes
type ErrorCode string

const (
	// Validation errors
	ErrCodeValidation     ErrorCode = "VALIDATION_ERROR"
	ErrCodeInvalidInput   ErrorCode = "INVALID_INPUT"
	ErrCodeMissingField   ErrorCode = "MISSING_FIELD"
	ErrCodeInvalidFormat  ErrorCode = "INVALID_FORMAT"
	
	// Authentication/Authorization errors
	ErrCodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden      ErrorCode = "FORBIDDEN"
	ErrCodeTokenExpired   ErrorCode = "TOKEN_EXPIRED"
	ErrCodeInvalidToken   ErrorCode = "INVALID_TOKEN"
	
	// Resource errors
	ErrCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrCodeAlreadyExists  ErrorCode = "ALREADY_EXISTS"
	ErrCodeConflict       ErrorCode = "CONFLICT"
	ErrCodeResourceLocked ErrorCode = "RESOURCE_LOCKED"
	
	// System errors
	ErrCodeInternal       ErrorCode = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeTimeout        ErrorCode = "TIMEOUT"
	ErrCodeRateLimited    ErrorCode = "RATE_LIMITED"
	
	// Database errors
	ErrCodeDatabaseError  ErrorCode = "DATABASE_ERROR"
	ErrCodeConnectionFailed ErrorCode = "CONNECTION_FAILED"
	ErrCodeQueryFailed    ErrorCode = "QUERY_FAILED"
	ErrCodeTransactionFailed ErrorCode = "TRANSACTION_FAILED"
	
	// Memory errors
	ErrCodeMemoryError    ErrorCode = "MEMORY_ERROR"
	ErrCodeMemoryNotFound ErrorCode = "MEMORY_NOT_FOUND"
	ErrCodeMemoryCorrupted ErrorCode = "MEMORY_CORRUPTED"
	ErrCodeMemoryFull     ErrorCode = "MEMORY_FULL"
	
	// LLM errors
	ErrCodeLLMError       ErrorCode = "LLM_ERROR"
	ErrCodeLLMTimeout     ErrorCode = "LLM_TIMEOUT"
	ErrCodeLLMAPIError    ErrorCode = "LLM_API_ERROR"
	ErrCodeLLMRateLimited ErrorCode = "LLM_RATE_LIMITED"
	
	// Configuration errors
	ErrCodeConfigError    ErrorCode = "CONFIG_ERROR"
	ErrCodeConfigNotFound ErrorCode = "CONFIG_NOT_FOUND"
	ErrCodeConfigInvalid  ErrorCode = "CONFIG_INVALID"
	
	// File system errors
	ErrCodeFileError      ErrorCode = "FILE_ERROR"
	ErrCodeFileNotFound   ErrorCode = "FILE_NOT_FOUND"
	ErrCodeFileCorrupted  ErrorCode = "FILE_CORRUPTED"
	ErrCodePermissionDenied ErrorCode = "PERMISSION_DENIED"
)

// MemGOSError represents a structured error in MemGOS
type MemGOSError struct {
	Type       types.ErrorType        `json:"type"`
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Cause      error                  `json:"-"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
}

// Error implements the error interface
func (e *MemGOSError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s (caused by: %v)", e.Code, e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *MemGOSError) Unwrap() error {
	return e.Cause
}

// WithDetail adds a detail to the error
func (e *MemGOSError) WithDetail(key string, value interface{}) *MemGOSError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithRequestID adds a request ID to the error
func (e *MemGOSError) WithRequestID(requestID string) *MemGOSError {
	e.RequestID = requestID
	return e
}

// WithStackTrace adds a stack trace to the error
func (e *MemGOSError) WithStackTrace() *MemGOSError {
	e.StackTrace = getStackTrace()
	return e
}

// ToTypes converts to types.MemGOSError
func (e *MemGOSError) ToTypes() *types.MemGOSError {
	return &types.MemGOSError{
		Type:    e.Type,
		Message: e.Message,
		Code:    string(e.Code),
		Details: e.Details,
	}
}

// NewMemGOSError creates a new MemGOS error
func NewMemGOSError(errType types.ErrorType, code ErrorCode, message string) *MemGOSError {
	return &MemGOSError{
		Type:    errType,
		Code:    code,
		Message: message,
	}
}

// NewMemGOSErrorWithCause creates a new MemGOS error with a cause
func NewMemGOSErrorWithCause(errType types.ErrorType, code ErrorCode, message string, cause error) *MemGOSError {
	return &MemGOSError{
		Type:    errType,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Validation error constructors
func NewValidationError(message string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeValidation, ErrCodeValidation, message)
}

func NewInvalidInputError(message string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeValidation, ErrCodeInvalidInput, message)
}

func NewMissingFieldError(field string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeValidation, ErrCodeMissingField, 
		fmt.Sprintf("missing required field: %s", field)).WithDetail("field", field)
}

func NewInvalidFormatError(field, expectedFormat string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeValidation, ErrCodeInvalidFormat,
		fmt.Sprintf("invalid format for field %s, expected: %s", field, expectedFormat)).
		WithDetail("field", field).WithDetail("expected_format", expectedFormat)
}

// Authentication/Authorization error constructors
func NewUnauthorizedError(message string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeUnauthorized, ErrCodeUnauthorized, message)
}

func NewForbiddenError(message string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeUnauthorized, ErrCodeForbidden, message)
}

func NewTokenExpiredError() *MemGOSError {
	return NewMemGOSError(types.ErrorTypeUnauthorized, ErrCodeTokenExpired, "token has expired")
}

func NewInvalidTokenError() *MemGOSError {
	return NewMemGOSError(types.ErrorTypeUnauthorized, ErrCodeInvalidToken, "invalid token")
}

// Resource error constructors
func NewNotFoundError(resource string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeNotFound, ErrCodeNotFound,
		fmt.Sprintf("%s not found", resource)).WithDetail("resource", resource)
}

func NewAlreadyExistsError(resource string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeValidation, ErrCodeAlreadyExists,
		fmt.Sprintf("%s already exists", resource)).WithDetail("resource", resource)
}

func NewConflictError(message string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeValidation, ErrCodeConflict, message)
}

func NewResourceLockedError(resource string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeValidation, ErrCodeResourceLocked,
		fmt.Sprintf("%s is locked", resource)).WithDetail("resource", resource)
}

// System error constructors
func NewInternalError(message string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeInternal, ErrCodeInternal, message)
}

func NewInternalErrorWithCause(message string, cause error) *MemGOSError {
	return NewMemGOSErrorWithCause(types.ErrorTypeInternal, ErrCodeInternal, message, cause)
}

func NewServiceUnavailableError(service string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeInternal, ErrCodeServiceUnavailable,
		fmt.Sprintf("%s service is unavailable", service)).WithDetail("service", service)
}

func NewTimeoutError(operation string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeInternal, ErrCodeTimeout,
		fmt.Sprintf("%s operation timed out", operation)).WithDetail("operation", operation)
}

func NewRateLimitedError(message string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeInternal, ErrCodeRateLimited, message)
}

// Database error constructors
func NewDatabaseError(message string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeInternal, ErrCodeDatabaseError, message)
}

func NewDatabaseErrorWithCause(message string, cause error) *MemGOSError {
	return NewMemGOSErrorWithCause(types.ErrorTypeInternal, ErrCodeDatabaseError, message, cause)
}

func NewConnectionFailedError(target string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeInternal, ErrCodeConnectionFailed,
		fmt.Sprintf("failed to connect to %s", target)).WithDetail("target", target)
}

func NewQueryFailedError(query string, cause error) *MemGOSError {
	return NewMemGOSErrorWithCause(types.ErrorTypeInternal, ErrCodeQueryFailed,
		"query execution failed", cause).WithDetail("query", query)
}

func NewTransactionFailedError(cause error) *MemGOSError {
	return NewMemGOSErrorWithCause(types.ErrorTypeInternal, ErrCodeTransactionFailed,
		"transaction failed", cause)
}

// Memory error constructors
func NewMemoryError(message string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeInternal, ErrCodeMemoryError, message)
}

func NewMemoryNotFoundError(memoryID string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeNotFound, ErrCodeMemoryNotFound,
		fmt.Sprintf("memory not found: %s", memoryID)).WithDetail("memory_id", memoryID)
}

func NewMemoryCorruptedError(memoryID string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeInternal, ErrCodeMemoryCorrupted,
		fmt.Sprintf("memory corrupted: %s", memoryID)).WithDetail("memory_id", memoryID)
}

func NewMemoryFullError(cubeID string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeInternal, ErrCodeMemoryFull,
		fmt.Sprintf("memory cube full: %s", cubeID)).WithDetail("cube_id", cubeID)
}

// LLM error constructors
func NewLLMError(message string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeExternal, ErrCodeLLMError, message)
}

func NewLLMTimeoutError(model string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeExternal, ErrCodeLLMTimeout,
		fmt.Sprintf("LLM request timed out: %s", model)).WithDetail("model", model)
}

func NewLLMAPIError(message string, cause error) *MemGOSError {
	return NewMemGOSErrorWithCause(types.ErrorTypeExternal, ErrCodeLLMAPIError, message, cause)
}

func NewLLMRateLimitedError(model string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeExternal, ErrCodeLLMRateLimited,
		fmt.Sprintf("LLM rate limited: %s", model)).WithDetail("model", model)
}

// Configuration error constructors
func NewConfigError(message string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeValidation, ErrCodeConfigError, message)
}

func NewConfigNotFoundError(configPath string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeNotFound, ErrCodeConfigNotFound,
		fmt.Sprintf("configuration file not found: %s", configPath)).WithDetail("config_path", configPath)
}

func NewConfigInvalidError(message string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeValidation, ErrCodeConfigInvalid, message)
}

// File system error constructors
func NewFileError(message string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeInternal, ErrCodeFileError, message)
}

func NewFileNotFoundError(filePath string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeNotFound, ErrCodeFileNotFound,
		fmt.Sprintf("file not found: %s", filePath)).WithDetail("file_path", filePath)
}

func NewFileCorruptedError(filePath string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeInternal, ErrCodeFileCorrupted,
		fmt.Sprintf("file corrupted: %s", filePath)).WithDetail("file_path", filePath)
}

func NewPermissionDeniedError(filePath string) *MemGOSError {
	return NewMemGOSError(types.ErrorTypeUnauthorized, ErrCodePermissionDenied,
		fmt.Sprintf("permission denied: %s", filePath)).WithDetail("file_path", filePath)
}

// Helper functions
func getStackTrace() string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	
	var trace strings.Builder
	for {
		frame, more := frames.Next()
		if !more {
			break
		}
		trace.WriteString(fmt.Sprintf("%s:%d %s\n", frame.File, frame.Line, frame.Function))
	}
	
	return trace.String()
}

// IsMemGOSError checks if an error is a MemGOSError
func IsMemGOSError(err error) bool {
	_, ok := err.(*MemGOSError)
	return ok
}

// GetMemGOSError extracts a MemGOSError from an error
func GetMemGOSError(err error) *MemGOSError {
	if memgosErr, ok := err.(*MemGOSError); ok {
		return memgosErr
	}
	return nil
}

// WrapError wraps an error as a MemGOSError
func WrapError(err error, errType types.ErrorType, code ErrorCode, message string) *MemGOSError {
	return NewMemGOSErrorWithCause(errType, code, message, err)
}

// ErrorList represents a list of errors
type ErrorList struct {
	Errors []*MemGOSError `json:"errors"`
}

// Error implements the error interface
func (el *ErrorList) Error() string {
	var messages []string
	for _, err := range el.Errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// Add adds an error to the list
func (el *ErrorList) Add(err *MemGOSError) {
	el.Errors = append(el.Errors, err)
}

// HasErrors returns true if there are errors
func (el *ErrorList) HasErrors() bool {
	return len(el.Errors) > 0
}

// ToError returns the ErrorList as an error if it has errors, otherwise nil
func (el *ErrorList) ToError() error {
	if el.HasErrors() {
		return el
	}
	return nil
}

// NewErrorList creates a new error list
func NewErrorList() *ErrorList {
	return &ErrorList{
		Errors: make([]*MemGOSError, 0),
	}
}

// Collect collects multiple errors into an ErrorList
func Collect(errors ...*MemGOSError) *ErrorList {
	el := NewErrorList()
	for _, err := range errors {
		if err != nil {
			el.Add(err)
		}
	}
	return el
}