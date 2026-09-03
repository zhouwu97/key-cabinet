package errors

import "fmt"

type AppError struct {
	Code    Code
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

// New creates a new AppError
func New(code Code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Newf creates a new AppError with formatted message
func Newf(code Code, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap wraps an existing error with a message
func Wrap(err error, message string) *AppError {
	return &AppError{
		Code:    CodeInternalError,
		Message: message,
		Cause:   err,
	}
}

// Wrapf wraps an existing error with a formatted message
func Wrapf(err error, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:    CodeInternalError,
		Message: fmt.Sprintf(format, args...),
		Cause:   err,
	}
}

// WrapWithCode wraps an existing error with a specific code
func WrapWithCode(err error, code Code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}
