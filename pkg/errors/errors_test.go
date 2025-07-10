package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/memtensor/memgos/pkg/types"
)

func TestMemGOSError(t *testing.T) {
	t.Run("NewMemGOSError", func(t *testing.T) {
		err := NewMemGOSError(types.ErrorTypeValidation, ErrCodeValidation, "test error")
		
		assert.Equal(t, types.ErrorTypeValidation, err.Type)
		assert.Equal(t, ErrCodeValidation, err.Code)
		assert.Equal(t, "test error", err.Message)
		assert.Nil(t, err.Cause)
		assert.Empty(t, err.Details)
		assert.Empty(t, err.StackTrace)
		assert.Empty(t, err.RequestID)
	})

	t.Run("NewMemGOSErrorWithCause", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := NewMemGOSErrorWithCause(types.ErrorTypeInternal, ErrCodeInternal, "wrapped error", cause)
		
		assert.Equal(t, types.ErrorTypeInternal, err.Type)
		assert.Equal(t, ErrCodeInternal, err.Code)
		assert.Equal(t, "wrapped error", err.Message)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("Error", func(t *testing.T) {
		err := NewMemGOSError(types.ErrorTypeValidation, ErrCodeValidation, "test error")
		expected := "[VALIDATION_ERROR] validation: test error"
		assert.Equal(t, expected, err.Error())
		
		// Test with cause
		cause := errors.New("underlying error")
		errWithCause := NewMemGOSErrorWithCause(types.ErrorTypeInternal, ErrCodeInternal, "wrapped error", cause)
		expectedWithCause := "[INTERNAL_ERROR] internal: wrapped error (caused by: underlying error)"
		assert.Equal(t, expectedWithCause, errWithCause.Error())
	})

	t.Run("Unwrap", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := NewMemGOSErrorWithCause(types.ErrorTypeInternal, ErrCodeInternal, "wrapped error", cause)
		
		unwrapped := err.Unwrap()
		assert.Equal(t, cause, unwrapped)
		
		// Test without cause
		errWithoutCause := NewMemGOSError(types.ErrorTypeValidation, ErrCodeValidation, "test error")
		assert.Nil(t, errWithoutCause.Unwrap())
	})

	t.Run("WithDetail", func(t *testing.T) {
		err := NewMemGOSError(types.ErrorTypeValidation, ErrCodeValidation, "test error")
		
		result := err.WithDetail("field", "username")
		assert.Same(t, err, result) // Should return the same instance
		assert.Equal(t, "username", err.Details["field"])
		
		// Add multiple details
		err.WithDetail("value", 123).WithDetail("required", true)
		assert.Equal(t, 123, err.Details["value"])
		assert.Equal(t, true, err.Details["required"])
	})

	t.Run("WithRequestID", func(t *testing.T) {
		err := NewMemGOSError(types.ErrorTypeValidation, ErrCodeValidation, "test error")
		
		result := err.WithRequestID("req-123")
		assert.Same(t, err, result)
		assert.Equal(t, "req-123", err.RequestID)
	})

	t.Run("WithStackTrace", func(t *testing.T) {
		err := NewMemGOSError(types.ErrorTypeValidation, ErrCodeValidation, "test error")
		
		result := err.WithStackTrace()
		assert.Same(t, err, result)
		assert.NotEmpty(t, err.StackTrace)
		assert.Contains(t, err.StackTrace, "TestMemGOSError")
	})

	t.Run("ToTypes", func(t *testing.T) {
		err := NewMemGOSError(types.ErrorTypeValidation, ErrCodeValidation, "test error")
		err.WithDetail("field", "username").WithRequestID("req-123")
		
		typesErr := err.ToTypes()
		assert.Equal(t, types.ErrorTypeValidation, typesErr.Type)
		assert.Equal(t, "test error", typesErr.Message)
		assert.Equal(t, "VALIDATION_ERROR", typesErr.Code)
		assert.Equal(t, "username", typesErr.Details["field"])
	})
}

func TestValidationErrors(t *testing.T) {
	t.Run("NewValidationError", func(t *testing.T) {
		err := NewValidationError("invalid input")
		assert.Equal(t, types.ErrorTypeValidation, err.Type)
		assert.Equal(t, ErrCodeValidation, err.Code)
		assert.Equal(t, "invalid input", err.Message)
	})

	t.Run("NewInvalidInputError", func(t *testing.T) {
		err := NewInvalidInputError("input is malformed")
		assert.Equal(t, types.ErrorTypeValidation, err.Type)
		assert.Equal(t, ErrCodeInvalidInput, err.Code)
		assert.Equal(t, "input is malformed", err.Message)
	})

	t.Run("NewMissingFieldError", func(t *testing.T) {
		err := NewMissingFieldError("username")
		assert.Equal(t, types.ErrorTypeValidation, err.Type)
		assert.Equal(t, ErrCodeMissingField, err.Code)
		assert.Contains(t, err.Message, "username")
		assert.Equal(t, "username", err.Details["field"])
	})

	t.Run("NewInvalidFormatError", func(t *testing.T) {
		err := NewInvalidFormatError("email", "user@example.com")
		assert.Equal(t, types.ErrorTypeValidation, err.Type)
		assert.Equal(t, ErrCodeInvalidFormat, err.Code)
		assert.Contains(t, err.Message, "email")
		assert.Contains(t, err.Message, "user@example.com")
		assert.Equal(t, "email", err.Details["field"])
		assert.Equal(t, "user@example.com", err.Details["expected_format"])
	})
}

func TestAuthenticationErrors(t *testing.T) {
	t.Run("NewUnauthorizedError", func(t *testing.T) {
		err := NewUnauthorizedError("access denied")
		assert.Equal(t, types.ErrorTypeUnauthorized, err.Type)
		assert.Equal(t, ErrCodeUnauthorized, err.Code)
		assert.Equal(t, "access denied", err.Message)
	})

	t.Run("NewForbiddenError", func(t *testing.T) {
		err := NewForbiddenError("insufficient permissions")
		assert.Equal(t, types.ErrorTypeUnauthorized, err.Type)
		assert.Equal(t, ErrCodeForbidden, err.Code)
		assert.Equal(t, "insufficient permissions", err.Message)
	})

	t.Run("NewTokenExpiredError", func(t *testing.T) {
		err := NewTokenExpiredError()
		assert.Equal(t, types.ErrorTypeUnauthorized, err.Type)
		assert.Equal(t, ErrCodeTokenExpired, err.Code)
		assert.Equal(t, "token has expired", err.Message)
	})

	t.Run("NewInvalidTokenError", func(t *testing.T) {
		err := NewInvalidTokenError()
		assert.Equal(t, types.ErrorTypeUnauthorized, err.Type)
		assert.Equal(t, ErrCodeInvalidToken, err.Code)
		assert.Equal(t, "invalid token", err.Message)
	})
}

func TestResourceErrors(t *testing.T) {
	t.Run("NewNotFoundError", func(t *testing.T) {
		err := NewNotFoundError("user")
		assert.Equal(t, types.ErrorTypeNotFound, err.Type)
		assert.Equal(t, ErrCodeNotFound, err.Code)
		assert.Contains(t, err.Message, "user")
		assert.Equal(t, "user", err.Details["resource"])
	})

	t.Run("NewAlreadyExistsError", func(t *testing.T) {
		err := NewAlreadyExistsError("user")
		assert.Equal(t, types.ErrorTypeValidation, err.Type)
		assert.Equal(t, ErrCodeAlreadyExists, err.Code)
		assert.Contains(t, err.Message, "user")
		assert.Equal(t, "user", err.Details["resource"])
	})

	t.Run("NewConflictError", func(t *testing.T) {
		err := NewConflictError("resource conflict")
		assert.Equal(t, types.ErrorTypeValidation, err.Type)
		assert.Equal(t, ErrCodeConflict, err.Code)
		assert.Equal(t, "resource conflict", err.Message)
	})

	t.Run("NewResourceLockedError", func(t *testing.T) {
		err := NewResourceLockedError("file")
		assert.Equal(t, types.ErrorTypeValidation, err.Type)
		assert.Equal(t, ErrCodeResourceLocked, err.Code)
		assert.Contains(t, err.Message, "file")
		assert.Equal(t, "file", err.Details["resource"])
	})
}

func TestSystemErrors(t *testing.T) {
	t.Run("NewInternalError", func(t *testing.T) {
		err := NewInternalError("system failure")
		assert.Equal(t, types.ErrorTypeInternal, err.Type)
		assert.Equal(t, ErrCodeInternal, err.Code)
		assert.Equal(t, "system failure", err.Message)
	})

	t.Run("NewInternalErrorWithCause", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := NewInternalErrorWithCause("system failure", cause)
		assert.Equal(t, types.ErrorTypeInternal, err.Type)
		assert.Equal(t, ErrCodeInternal, err.Code)
		assert.Equal(t, "system failure", err.Message)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("NewServiceUnavailableError", func(t *testing.T) {
		err := NewServiceUnavailableError("database")
		assert.Equal(t, types.ErrorTypeInternal, err.Type)
		assert.Equal(t, ErrCodeServiceUnavailable, err.Code)
		assert.Contains(t, err.Message, "database")
		assert.Equal(t, "database", err.Details["service"])
	})

	t.Run("NewTimeoutError", func(t *testing.T) {
		err := NewTimeoutError("query")
		assert.Equal(t, types.ErrorTypeInternal, err.Type)
		assert.Equal(t, ErrCodeTimeout, err.Code)
		assert.Contains(t, err.Message, "query")
		assert.Equal(t, "query", err.Details["operation"])
	})

	t.Run("NewRateLimitedError", func(t *testing.T) {
		err := NewRateLimitedError("too many requests")
		assert.Equal(t, types.ErrorTypeInternal, err.Type)
		assert.Equal(t, ErrCodeRateLimited, err.Code)
		assert.Equal(t, "too many requests", err.Message)
	})
}

func TestDatabaseErrors(t *testing.T) {
	t.Run("NewDatabaseError", func(t *testing.T) {
		err := NewDatabaseError("connection failed")
		assert.Equal(t, types.ErrorTypeInternal, err.Type)
		assert.Equal(t, ErrCodeDatabaseError, err.Code)
		assert.Equal(t, "connection failed", err.Message)
	})

	t.Run("NewDatabaseErrorWithCause", func(t *testing.T) {
		cause := errors.New("network error")
		err := NewDatabaseErrorWithCause("connection failed", cause)
		assert.Equal(t, types.ErrorTypeInternal, err.Type)
		assert.Equal(t, ErrCodeDatabaseError, err.Code)
		assert.Equal(t, "connection failed", err.Message)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("NewConnectionFailedError", func(t *testing.T) {
		err := NewConnectionFailedError("localhost:5432")
		assert.Equal(t, types.ErrorTypeInternal, err.Type)
		assert.Equal(t, ErrCodeConnectionFailed, err.Code)
		assert.Contains(t, err.Message, "localhost:5432")
		assert.Equal(t, "localhost:5432", err.Details["target"])
	})

	t.Run("NewQueryFailedError", func(t *testing.T) {
		cause := errors.New("syntax error")
		err := NewQueryFailedError("SELECT * FROM users", cause)
		assert.Equal(t, types.ErrorTypeInternal, err.Type)
		assert.Equal(t, ErrCodeQueryFailed, err.Code)
		assert.Equal(t, "query execution failed", err.Message)
		assert.Equal(t, cause, err.Cause)
		assert.Equal(t, "SELECT * FROM users", err.Details["query"])
	})

	t.Run("NewTransactionFailedError", func(t *testing.T) {
		cause := errors.New("deadlock detected")
		err := NewTransactionFailedError(cause)
		assert.Equal(t, types.ErrorTypeInternal, err.Type)
		assert.Equal(t, ErrCodeTransactionFailed, err.Code)
		assert.Equal(t, "transaction failed", err.Message)
		assert.Equal(t, cause, err.Cause)
	})
}

func TestMemoryErrors(t *testing.T) {
	t.Run("NewMemoryError", func(t *testing.T) {
		err := NewMemoryError("memory allocation failed")
		assert.Equal(t, types.ErrorTypeInternal, err.Type)
		assert.Equal(t, ErrCodeMemoryError, err.Code)
		assert.Equal(t, "memory allocation failed", err.Message)
	})

	t.Run("NewMemoryNotFoundError", func(t *testing.T) {
		err := NewMemoryNotFoundError("mem-123")
		assert.Equal(t, types.ErrorTypeNotFound, err.Type)
		assert.Equal(t, ErrCodeMemoryNotFound, err.Code)
		assert.Contains(t, err.Message, "mem-123")
		assert.Equal(t, "mem-123", err.Details["memory_id"])
	})

	t.Run("NewMemoryCorruptedError", func(t *testing.T) {
		err := NewMemoryCorruptedError("mem-456")
		assert.Equal(t, types.ErrorTypeInternal, err.Type)
		assert.Equal(t, ErrCodeMemoryCorrupted, err.Code)
		assert.Contains(t, err.Message, "mem-456")
		assert.Equal(t, "mem-456", err.Details["memory_id"])
	})

	t.Run("NewMemoryFullError", func(t *testing.T) {
		err := NewMemoryFullError("cube-789")
		assert.Equal(t, types.ErrorTypeInternal, err.Type)
		assert.Equal(t, ErrCodeMemoryFull, err.Code)
		assert.Contains(t, err.Message, "cube-789")
		assert.Equal(t, "cube-789", err.Details["cube_id"])
	})
}

func TestLLMErrors(t *testing.T) {
	t.Run("NewLLMError", func(t *testing.T) {
		err := NewLLMError("model unavailable")
		assert.Equal(t, types.ErrorTypeExternal, err.Type)
		assert.Equal(t, ErrCodeLLMError, err.Code)
		assert.Equal(t, "model unavailable", err.Message)
	})

	t.Run("NewLLMTimeoutError", func(t *testing.T) {
		err := NewLLMTimeoutError("gpt-3.5-turbo")
		assert.Equal(t, types.ErrorTypeExternal, err.Type)
		assert.Equal(t, ErrCodeLLMTimeout, err.Code)
		assert.Contains(t, err.Message, "gpt-3.5-turbo")
		assert.Equal(t, "gpt-3.5-turbo", err.Details["model"])
	})

	t.Run("NewLLMAPIError", func(t *testing.T) {
		cause := errors.New("HTTP 429 Too Many Requests")
		err := NewLLMAPIError("API request failed", cause)
		assert.Equal(t, types.ErrorTypeExternal, err.Type)
		assert.Equal(t, ErrCodeLLMAPIError, err.Code)
		assert.Equal(t, "API request failed", err.Message)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("NewLLMRateLimitedError", func(t *testing.T) {
		err := NewLLMRateLimitedError("gpt-4")
		assert.Equal(t, types.ErrorTypeExternal, err.Type)
		assert.Equal(t, ErrCodeLLMRateLimited, err.Code)
		assert.Contains(t, err.Message, "gpt-4")
		assert.Equal(t, "gpt-4", err.Details["model"])
	})
}

func TestConfigErrors(t *testing.T) {
	t.Run("NewConfigError", func(t *testing.T) {
		err := NewConfigError("invalid configuration")
		assert.Equal(t, types.ErrorTypeValidation, err.Type)
		assert.Equal(t, ErrCodeConfigError, err.Code)
		assert.Equal(t, "invalid configuration", err.Message)
	})

	t.Run("NewConfigNotFoundError", func(t *testing.T) {
		err := NewConfigNotFoundError("/path/to/config.yaml")
		assert.Equal(t, types.ErrorTypeNotFound, err.Type)
		assert.Equal(t, ErrCodeConfigNotFound, err.Code)
		assert.Contains(t, err.Message, "/path/to/config.yaml")
		assert.Equal(t, "/path/to/config.yaml", err.Details["config_path"])
	})

	t.Run("NewConfigInvalidError", func(t *testing.T) {
		err := NewConfigInvalidError("missing required field")
		assert.Equal(t, types.ErrorTypeValidation, err.Type)
		assert.Equal(t, ErrCodeConfigInvalid, err.Code)
		assert.Equal(t, "missing required field", err.Message)
	})
}

func TestFileErrors(t *testing.T) {
	t.Run("NewFileError", func(t *testing.T) {
		err := NewFileError("file operation failed")
		assert.Equal(t, types.ErrorTypeInternal, err.Type)
		assert.Equal(t, ErrCodeFileError, err.Code)
		assert.Equal(t, "file operation failed", err.Message)
	})

	t.Run("NewFileNotFoundError", func(t *testing.T) {
		err := NewFileNotFoundError("/path/to/file.txt")
		assert.Equal(t, types.ErrorTypeNotFound, err.Type)
		assert.Equal(t, ErrCodeFileNotFound, err.Code)
		assert.Contains(t, err.Message, "/path/to/file.txt")
		assert.Equal(t, "/path/to/file.txt", err.Details["file_path"])
	})

	t.Run("NewFileCorruptedError", func(t *testing.T) {
		err := NewFileCorruptedError("/path/to/corrupted.dat")
		assert.Equal(t, types.ErrorTypeInternal, err.Type)
		assert.Equal(t, ErrCodeFileCorrupted, err.Code)
		assert.Contains(t, err.Message, "/path/to/corrupted.dat")
		assert.Equal(t, "/path/to/corrupted.dat", err.Details["file_path"])
	})

	t.Run("NewPermissionDeniedError", func(t *testing.T) {
		err := NewPermissionDeniedError("/protected/file.txt")
		assert.Equal(t, types.ErrorTypeUnauthorized, err.Type)
		assert.Equal(t, ErrCodePermissionDenied, err.Code)
		assert.Contains(t, err.Message, "/protected/file.txt")
		assert.Equal(t, "/protected/file.txt", err.Details["file_path"])
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("getStackTrace", func(t *testing.T) {
		trace := getStackTrace()
		assert.NotEmpty(t, trace)
		assert.Contains(t, trace, "TestHelperFunctions")
	})

	t.Run("IsMemGOSError", func(t *testing.T) {
		memgosErr := NewValidationError("test error")
		standardErr := errors.New("standard error")
		
		assert.True(t, IsMemGOSError(memgosErr))
		assert.False(t, IsMemGOSError(standardErr))
		assert.False(t, IsMemGOSError(nil))
	})

	t.Run("GetMemGOSError", func(t *testing.T) {
		memgosErr := NewValidationError("test error")
		standardErr := errors.New("standard error")
		
		extracted := GetMemGOSError(memgosErr)
		assert.Equal(t, memgosErr, extracted)
		
		extracted = GetMemGOSError(standardErr)
		assert.Nil(t, extracted)
		
		extracted = GetMemGOSError(nil)
		assert.Nil(t, extracted)
	})

	t.Run("WrapError", func(t *testing.T) {
		cause := errors.New("underlying error")
		wrapped := WrapError(cause, types.ErrorTypeInternal, ErrCodeInternal, "wrapped message")
		
		assert.Equal(t, types.ErrorTypeInternal, wrapped.Type)
		assert.Equal(t, ErrCodeInternal, wrapped.Code)
		assert.Equal(t, "wrapped message", wrapped.Message)
		assert.Equal(t, cause, wrapped.Cause)
	})
}

func TestErrorList(t *testing.T) {
	t.Run("NewErrorList", func(t *testing.T) {
		el := NewErrorList()
		assert.NotNil(t, el)
		assert.Empty(t, el.Errors)
		assert.False(t, el.HasErrors())
	})

	t.Run("Add", func(t *testing.T) {
		el := NewErrorList()
		err1 := NewValidationError("error 1")
		err2 := NewValidationError("error 2")
		
		el.Add(err1)
		assert.Len(t, el.Errors, 1)
		assert.True(t, el.HasErrors())
		
		el.Add(err2)
		assert.Len(t, el.Errors, 2)
	})

	t.Run("Error", func(t *testing.T) {
		el := NewErrorList()
		err1 := NewValidationError("error 1")
		err2 := NewValidationError("error 2")
		
		el.Add(err1)
		el.Add(err2)
		
		errorString := el.Error()
		assert.Contains(t, errorString, "error 1")
		assert.Contains(t, errorString, "error 2")
		assert.Contains(t, errorString, ";")
	})

	t.Run("ToError", func(t *testing.T) {
		el := NewErrorList()
		
		// Empty list should return nil
		err := el.ToError()
		assert.Nil(t, err)
		
		// List with errors should return the list
		el.Add(NewValidationError("test error"))
		err = el.ToError()
		assert.Equal(t, el, err)
	})

	t.Run("Collect", func(t *testing.T) {
		err1 := NewValidationError("error 1")
		err2 := NewValidationError("error 2")
		
		el := Collect(err1, nil, err2, nil)
		assert.Len(t, el.Errors, 2)
		assert.Equal(t, err1, el.Errors[0])
		assert.Equal(t, err2, el.Errors[1])
	})
}

// Benchmark tests
func BenchmarkNewMemGOSError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewMemGOSError(types.ErrorTypeValidation, ErrCodeValidation, "test error")
	}
}

func BenchmarkMemGOSErrorError(b *testing.B) {
	err := NewMemGOSError(types.ErrorTypeValidation, ErrCodeValidation, "test error")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

func BenchmarkMemGOSErrorWithDetail(b *testing.B) {
	err := NewMemGOSError(types.ErrorTypeValidation, ErrCodeValidation, "test error")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err.WithDetail(fmt.Sprintf("field_%d", i), i)
	}
}

func BenchmarkErrorListAdd(b *testing.B) {
	el := NewErrorList()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := NewValidationError(fmt.Sprintf("error %d", i))
		el.Add(err)
	}
}

func BenchmarkIsMemGOSError(b *testing.B) {
	memgosErr := NewValidationError("test error")
	standardErr := errors.New("standard error")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			_ = IsMemGOSError(memgosErr)
		} else {
			_ = IsMemGOSError(standardErr)
		}
	}
}

// Integration tests
func TestErrorIntegration(t *testing.T) {
	t.Run("ComplexErrorChain", func(t *testing.T) {
		// Create a chain of errors
		rootCause := errors.New("network connection failed")
		dbErr := NewDatabaseErrorWithCause("failed to execute query", rootCause)
		appErr := WrapError(dbErr, types.ErrorTypeInternal, ErrCodeInternal, "application error")
		
		appErr.WithDetail("operation", "user_lookup").
			WithDetail("user_id", "123").
			WithRequestID("req-456").
			WithStackTrace()
		
		// Test error chain
		assert.Contains(t, appErr.Error(), "application error")
		assert.Contains(t, appErr.Error(), "failed to execute query")
		assert.Equal(t, dbErr, appErr.Unwrap())
		assert.Equal(t, rootCause, dbErr.Unwrap())
		
		// Test details
		assert.Equal(t, "user_lookup", appErr.Details["operation"])
		assert.Equal(t, "123", appErr.Details["user_id"])
		assert.Equal(t, "req-456", appErr.RequestID)
		assert.NotEmpty(t, appErr.StackTrace)
		
		// Test conversion to types
		typesErr := appErr.ToTypes()
		assert.Equal(t, types.ErrorTypeInternal, typesErr.Type)
		assert.Equal(t, "application error", typesErr.Message)
	})

	t.Run("ErrorListWithMultipleTypes", func(t *testing.T) {
		el := NewErrorList()
		
		el.Add(NewValidationError("validation failed"))
		el.Add(NewUnauthorizedError("access denied"))
		el.Add(NewNotFoundError("resource"))
		el.Add(NewInternalError("system failure"))
		
		assert.Len(t, el.Errors, 4)
		assert.True(t, el.HasErrors())
		
		errorString := el.Error()
		assert.Contains(t, errorString, "validation failed")
		assert.Contains(t, errorString, "access denied")
		assert.Contains(t, errorString, "resource not found")
		assert.Contains(t, errorString, "system failure")
	})
}

// Example test for documentation
func ExampleNewValidationError() {
	err := NewValidationError("username is required")
	fmt.Println(err.Error())
	// Output: [VALIDATION_ERROR] validation: username is required
}

func ExampleMemGOSError_WithDetail() {
	err := NewMissingFieldError("email").
		WithDetail("provided_value", "").
		WithRequestID("req-123")
	
	fmt.Printf("Type: %s\n", err.Type)
	fmt.Printf("Code: %s\n", err.Code)
	fmt.Printf("Field: %s\n", err.Details["field"])
	fmt.Printf("Request ID: %s\n", err.RequestID)
	// Output:
	// Type: validation
	// Code: MISSING_FIELD
	// Field: email
	// Request ID: req-123
}