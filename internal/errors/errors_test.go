package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroveError(t *testing.T) {
	t.Run("basic error creation", func(t *testing.T) {
		err := NewGroveError(ErrCodeGitNotFound, "git not found", nil)

		assert.Equal(t, ErrCodeGitNotFound, err.Code)
		assert.Equal(t, "git not found", err.Message)
		assert.Nil(t, err.Cause)
		assert.Equal(t, "git not found", err.Error())
	})

	t.Run("error with cause", func(t *testing.T) {
		cause := fmt.Errorf("underlying error")
		err := NewGroveError(ErrCodeGitOperation, "git operation failed", cause)

		assert.Equal(t, ErrCodeGitOperation, err.Code)
		assert.Equal(t, "git operation failed", err.Message)
		assert.Equal(t, cause, err.Cause)
		assert.Equal(t, "git operation failed: underlying error", err.Error())
	})

	t.Run("error with context", func(t *testing.T) {
		err := NewGroveError(ErrCodeDirectoryAccess, "directory access failed", nil)
		err = err.WithContext("path", "/test/path")
		err = err.WithContext("operation", "read")

		assert.Equal(t, "/test/path", err.Context["path"])
		assert.Equal(t, "read", err.Context["operation"])
	})

	t.Run("error unwrapping", func(t *testing.T) {
		cause := fmt.Errorf("underlying error")
		err := NewGroveError(ErrCodeGitOperation, "git operation failed", cause)

		assert.Equal(t, cause, err.Unwrap())
	})
}

func TestGroveErrorf(t *testing.T) {
	t.Run("formatted error creation", func(t *testing.T) {
		cause := fmt.Errorf("underlying error")
		err := NewGroveErrorf(ErrCodeRepoNotFound, cause, "repository not found at %s", "/test/path")

		assert.Equal(t, ErrCodeRepoNotFound, err.Code)
		assert.Equal(t, "repository not found at /test/path", err.Message)
		assert.Equal(t, cause, err.Cause)
	})
}

func TestErrorFactoryFunctions(t *testing.T) {
	t.Run("ErrGitNotFound", func(t *testing.T) {
		cause := fmt.Errorf("exec error")
		err := ErrGitNotFound(cause)

		assert.Equal(t, ErrCodeGitNotFound, err.Code)
		assert.Contains(t, err.Message, "git is not available in PATH")
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("ErrRepoExists", func(t *testing.T) {
		path := "/test/repo"
		err := ErrRepoExists(path)

		assert.Equal(t, ErrCodeRepoExists, err.Code)
		assert.Contains(t, err.Message, "already exists")
		assert.Equal(t, path, err.Context["path"])
	})

	t.Run("ErrInvalidURL", func(t *testing.T) {
		url := "invalid://url"
		reason := "unsupported protocol"
		err := ErrInvalidURL(url, reason)

		assert.Equal(t, ErrCodeInvalidURL, err.Code)
		assert.Contains(t, err.Message, url)
		assert.Contains(t, err.Message, reason)
		assert.Equal(t, url, err.Context["url"])
		assert.Equal(t, reason, err.Context["reason"])
	})

	t.Run("ErrPathTraversal", func(t *testing.T) {
		path := "../../../etc/passwd"
		err := ErrPathTraversal(path)

		assert.Equal(t, ErrCodePathTraversal, err.Code)
		assert.Contains(t, err.Message, "directory traversal")
		assert.Equal(t, path, err.Context["path"])
	})
}

func TestErrorHelpers(t *testing.T) {
	t.Run("IsGroveError", func(t *testing.T) {
		err := ErrGitNotFound(nil)

		assert.True(t, IsGroveError(err, ErrCodeGitNotFound))
		assert.False(t, IsGroveError(err, ErrCodeRepoExists))
		assert.False(t, IsGroveError(fmt.Errorf("standard error"), ErrCodeGitNotFound))
	})

	t.Run("GetErrorCode", func(t *testing.T) {
		err := ErrGitNotFound(nil)

		assert.Equal(t, ErrCodeGitNotFound, GetErrorCode(err))
		assert.Equal(t, "", GetErrorCode(fmt.Errorf("standard error")))
	})

	t.Run("GetErrorContext", func(t *testing.T) {
		err := ErrRepoExists("/test/path")

		context := GetErrorContext(err)
		assert.NotNil(t, context)
		assert.Equal(t, "/test/path", context["path"])

		context = GetErrorContext(fmt.Errorf("standard error"))
		assert.Nil(t, context)
	})
}

func TestErrorIs(t *testing.T) {
	t.Run("Grove error Is comparison", func(t *testing.T) {
		err1 := ErrGitNotFound(nil)
		err2 := ErrGitNotFound(fmt.Errorf("different cause"))
		err3 := ErrRepoExists("/path")

		assert.True(t, Is(err1, err2))
		assert.False(t, Is(err1, err3))
	})
}

func TestWrapFunctions(t *testing.T) {
	t.Run("Wrap", func(t *testing.T) {
		original := fmt.Errorf("original error")
		wrapped := Wrap(original, "additional context")

		assert.Contains(t, wrapped.Error(), "additional context")
		assert.Contains(t, wrapped.Error(), "original error")
		assert.True(t, Is(wrapped, original))
	})

	t.Run("Wrapf", func(t *testing.T) {
		original := fmt.Errorf("original error")
		wrapped := Wrapf(original, "context with %s", "parameter")

		assert.Contains(t, wrapped.Error(), "context with parameter")
		assert.Contains(t, wrapped.Error(), "original error")
		assert.True(t, Is(wrapped, original))
	})

	t.Run("WithOperation", func(t *testing.T) {
		groveErr := ErrGitNotFound(nil)
		withOp := WithOperation(groveErr, "clone")

		var result *GroveError
		require.True(t, As(withOp, &result))
		assert.Equal(t, "clone", result.Operation)
	})

	t.Run("WithContext on standard error", func(t *testing.T) {
		standardErr := fmt.Errorf("standard error")
		withContext := WithContext(standardErr, "key", "value")

		var result *GroveError
		require.True(t, As(withContext, &result))
		assert.Equal(t, "value", result.Context["key"])
	})
}

func TestNilErrorHandling(t *testing.T) {
	t.Run("Wrap with nil error", func(t *testing.T) {
		result := Wrap(nil, "context")
		assert.Nil(t, result)
	})

	t.Run("Wrapf with nil error", func(t *testing.T) {
		result := Wrapf(nil, "context %s", "param")
		assert.Nil(t, result)
	})

	t.Run("WithOperation with nil error", func(t *testing.T) {
		result := WithOperation(nil, "operation")
		assert.Nil(t, result)
	})

	t.Run("WithContext with nil error", func(t *testing.T) {
		result := WithContext(nil, "key", "value")
		assert.Nil(t, result)
	})
}
