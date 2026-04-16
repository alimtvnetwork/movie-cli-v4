// Package apperror provides standardised error wrapping for the CLI.
//
// Use Wrap to attach context to an existing error (preserves the cause chain).
// Use Wrapf when the context message needs format arguments.
// Use New to create a standalone error with a formatted message.
package apperror

import "fmt"

// Wrap returns a new error that prepends msg to err's message.
// The original error is preserved and can be unwrapped.
//
//	return apperror.Wrap("open database", err)
//	// → "open database: original message"
func Wrap(msg string, err error) error {
	return fmt.Errorf("%s: %w", msg, err)
}

// Wrapf is like Wrap but accepts a format string with arguments.
// The last argument is NOT included in formatting — it is the wrapped error.
//
//	return apperror.Wrapf(err, "cannot open file %s", path)
func Wrapf(err error, format string, args ...any) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

// New creates a new error with a formatted message (no cause chain).
//
//	return apperror.New("media not found for ID %d", id)
func New(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
