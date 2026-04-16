// Package apperror provides structured error wrapping for the CLI.
//
// SHARED: used across all packages to replace raw fmt.Errorf/errors.New.
package apperror

import "fmt"

// AppError wraps an inner error with a descriptive message.
type AppError struct {
	Message string
	Err     error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the inner error for errors.Is/As.
func (e *AppError) Unwrap() error {
	return e.Err
}

// Wrap creates an AppError wrapping an existing error.
func Wrap(msg string, err error) *AppError {
	return &AppError{Message: msg, Err: err}
}

// New creates an AppError with no inner error.
func New(msg string) *AppError {
	return &AppError{Message: msg}
}

// Newf creates an AppError with a formatted message and no inner error.
func Newf(format string, args ...interface{}) *AppError {
	return &AppError{Message: fmt.Sprintf(format, args...)}
}

// Wrapf creates an AppError wrapping an existing error with a formatted message.
func Wrapf(err error, format string, args ...interface{}) *AppError {
	return &AppError{Message: fmt.Sprintf(format, args...), Err: err}
}
