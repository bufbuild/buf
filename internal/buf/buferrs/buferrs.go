// Package buferrs provides error functionality for Buf.
package buferrs

import (
	"fmt"
)

// NewUserError returns a new user error.
func NewUserError(message string) error {
	return newBufError(true, message)
}

// NewUserErrorf returns a new user error.
func NewUserErrorf(format string, args ...interface{}) error {
	return newBufError(true, fmt.Sprintf(format, args...))
}

// NewSystemError returns a new system error.
func NewSystemError(message string) error {
	return newBufError(true, message)
}

// NewSystemErrorf returns a new system error.
func NewSystemErrorf(format string, args ...interface{}) error {
	return newBufError(true, fmt.Sprintf(format, args...))
}

// IsUserError returns true if err is a user error.
//
// Returns false if err == nil.
func IsUserError(err error) bool {
	if err == nil {
		return false
	}
	bufError, ok := err.(*bufError)
	if !ok {
		return false
	}
	return bufError.isUser
}

type bufError struct {
	isUser  bool
	message string
}

func newBufError(isUser bool, message string) *bufError {
	return &bufError{
		isUser:  isUser,
		message: message,
	}
}

func (m *bufError) Error() string {
	return m.message
}
