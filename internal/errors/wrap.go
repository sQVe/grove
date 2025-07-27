package errors

import (
	"errors"
	"fmt"
)

func New(text string) error {
	return errors.New(text)
}

// Report whether any error in err's chain matches target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// Find the first error in err's chain that matches target.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

func Unwrap(err error) error {
	return errors.Unwrap(err)
}

func Join(errs ...error) error {
	return errors.Join(errs...)
}

func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

func WithOperation(err error, operation string) error {
	if err == nil {
		return nil
	}

	var groveErr *GroveError
	if As(err, &groveErr) {
		groveErr.Operation = operation
		return groveErr
	}

	return NewGroveError(ErrCodeGitOperation, fmt.Sprintf("operation %s failed", operation), err).
		WithContext("operation", operation)
}

func WithContext(err error, key string, value interface{}) error {
	if err == nil {
		return nil
	}

	var groveErr *GroveError
	if As(err, &groveErr) {
		return groveErr.WithContext(key, value)
	}

	newErr := NewGroveError(ErrCodeGitOperation, err.Error(), err)
	return newErr.WithContext(key, value)
}
