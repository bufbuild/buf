// Package errs implements a user error abstraction.
//
// This helps programs determine what errors should be returned to the user, such as
// configuration or input validation errors, and what errors are actual system errors.
//
// This should be replaced with https://godoc.org/golang.org/x/xerrors.
package errs

import "fmt"

// NewUserError returns a new user error.
func NewUserError(value string) error {
	return &userError{
		value: value,
	}
}

// NewUserErrorf returns a new formatted user error.
func NewUserErrorf(format string, args ...interface{}) error {
	return &userError{
		value: fmt.Sprintf(format, args...),
	}
}

// IsUserError returns true if the given error is a user error.
func IsUserError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*userError)
	return ok
}

type userError struct {
	value string
}

func (u *userError) Error() string {
	return u.value
}
