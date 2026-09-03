package errors

import (
	"errors"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	err := New(CodeInvalidInput, "invalid email")
	assert.Equal(t, CodeInvalidInput, err.Code)
	assert.Equal(t, "invalid email", err.Message)
	assert.Contains(t, err.Error(), "INVALID_INPUT")
}

func TestWrap(t *testing.T) {
	cause := errors.New("db connection failed")
	err := Wrap(cause, "failed to query user")
	assert.Equal(t, CodeInternalError, err.Code)
	assert.NotNil(t, err.Cause)
	assert.Equal(t, cause, err.Unwrap())
}

func TestIsAppError(t *testing.T) {
	appErr := New(CodeNotFound, "user not found")
	result, ok := IsAppError(appErr)
	assert.True(t, ok)
	assert.Equal(t, CodeNotFound, result.Code)

	stdErr := errors.New("standard error")
	_, ok = IsAppError(stdErr)
	assert.False(t, ok)
}

func TestNewf(t *testing.T) {
	err := Newf(CodeInvalidInput, "invalid field: %s", "email")
	assert.Equal(t, CodeInvalidInput, err.Code)
	assert.Equal(t, "invalid field: email", err.Message)
}

func TestWrapWithCode(t *testing.T) {
	cause := errors.New("network timeout")
	err := WrapWithCode(cause, CodeServiceUnavailable, "service unavailable")
	assert.Equal(t, CodeServiceUnavailable, err.Code)
	assert.Equal(t, "service unavailable", err.Message)
	assert.Equal(t, cause, err.Unwrap())
}
