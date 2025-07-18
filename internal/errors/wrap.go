package errors

import (
	"errors"
	"fmt"
)

// Standard error functions that wrap the standard library
// This allows us to use errors.New, errors.Is, etc. from our errors package.

// New creates a new error with the given text.
func New(text string) error {
	return errors.New(text)
}

// Is reports whether any error in err's chain matches target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Unwrap returns the result of calling the Unwrap method on err.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// Join returns an error that wraps the given errors.
func Join(errs ...error) error {
	return errors.Join(errs...)
}

// Wrap creates a new error with additional context.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf creates a new error with formatted additional context.
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

// WithOperation adds operation context to a Grove error.
func WithOperation(err error, operation string) error {
	if err == nil {
		return nil
	}

	var groveErr *GroveError
	if As(err, &groveErr) {
		groveErr.Operation = operation
		return groveErr
	}

	// If it's not a Grove error, create a new one
	return NewGroveError(ErrCodeGitOperation, fmt.Sprintf("operation %s failed", operation), err).
		WithContext("operation", operation)
}

// WithContext adds context to a Grove error, or creates a new one if not Grove error.
func WithContext(err error, key string, value interface{}) error {
	if err == nil {
		return nil
	}

	var groveErr *GroveError
	if As(err, &groveErr) {
		return groveErr.WithContext(key, value)
	}

	// If it's not a Grove error, create a new one
	newErr := NewGroveError(ErrCodeGitOperation, err.Error(), err)
	return newErr.WithContext(key, value)
}
