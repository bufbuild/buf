// Package errs provides errors for both APIs and CLIs.
//
// This package is primarily meant to provde a transport-agnostic abstraction of
// errors that can be easily mapped to both Twirp and gRPC.
//
// Errors in this package are not meant to be sent across the wire, so there is
// no parsing functionality. Rely on the transport-specific mechanism for over-the-wire
// operations.
package errs

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// CodeOK is a sentinel code for no error.
	//
	// No error should use this directly. This is returned by GetCode if the error is nil.
	CodeOK Code = 0

	// CodeCanceled indicates the operation was canceled.
	//
	// HTTP equivalent: 408 REQUEST TIMEOUT
	CodeCanceled Code = 1

	// CodeUnknown indicates an unknown error.
	//
	// This is for errors that do not have specific error information.
	//
	// HTTP equivalent: 500 INTERNAL SERVER ERROR
	CodeUnknown Code = 2

	// CodeInvalidArgument indicates that the an invalid argument was specified, for
	// example missing required arguments and invalid arguments.
	//
	// HTTP equivalent: 400 BAD REQUEST
	CodeInvalidArgument Code = 3

	// CodeDeadlineExceeded indicates that a deadline was exceeded, for example a timeout.
	//
	// HTTP equivalent: 504 GATEWAY TIMEOUT
	// Note that Twirp treats this as a 408 REQUEST TIMEOUT, but grpc-gateway treats this
	// as 504 GATEWAY TIMEOUT.
	CodeDeadlineExceeded Code = 4

	// CodeNotFound indicates that an entity was not found.
	//
	// HTTP equivalent: 404 NOT FOUND
	CodeNotFound Code = 5

	// CodeAlreadyExists indicates that entity creation was unsuccesful as the entity
	// already exists.
	//
	// HTTP equivalent: 409 CONFLICT
	CodeAlreadyExists Code = 6

	// CodePermissionDenied indicates the caller does not have permission to perform
	// the requested operation.
	//
	// HTTP equivalent: 403 FORBIDDEN
	CodePermissionDenied Code = 7

	// CodeResourceExhausted indicates some resource has been exhausted, for example
	// throttling or out-of-space errors.
	//
	// HTTP equivalent: 429 TOO MANY REQUESTS
	// Note that Twirp treats this as 403 FORBIDDEN, but grpc-gateway treats this
	// as 429 TOO MANY REQUESTS.
	CodeResourceExhausted Code = 8

	// CodeFailedPrecondition indicates operation was rejected because the system is not
	// in a state required for the operation's execution, for example a non-recursive
	// non-empty directory deletion.
	//
	// HTTP equivalent: 400 BAD REQUEST
	// Note that Twirp treats this as 412 PRECONDITION FAILED, but grpc-gateway treats this
	// as 400 BAD REQUEST, and has a note saying this is on purpose (and it makes sense).
	CodeFailedPrecondition Code = 9

	// CodeAborted indicates the operation was aborted, for example when a transaction
	// is aborted.
	//
	// HTTP equivalent: 409 CONFLICT
	CodeAborted Code = 10

	// CodeOutOfRange indicates an operation was attempted past the valid range, for example
	// seeking or reading past the end of a paginated collection.
	//
	// Unlike InvalidArgument, this error indicates a problem that may be fixed if
	// the system state changes (i.e. adding more items to the collection).
	//
	// There is a fair bit of overlap between FailedPrecondition and OutOfRange.
	// We recommend using OutOfRange (the more specific error) when it applies so
	// that callers who are iterating through a space can easily look for an
	// OutOfRange error to detect when they are done.
	//
	// HTTP equivant: 400 BAD REQUEST
	CodeOutOfRange Code = 11

	// CodeUnimplemented indicates operation is not implemented or not
	// supported/enabled in this service.
	//
	// HTTP equivalent: 501 NOT IMPLEMENTED
	CodeUnimplemented Code = 12

	// CodeInternal indicates an internal system error.
	//
	// HTTP equivalent: 500 INTERNAL SERVER ERROR
	CodeInternal Code = 13

	// CodeUnavailable indicates the service is currently unavailable.
	// This is a most likely a transient condition and may be corrected
	// by retrying with a backoff.
	//
	// HTTP equivalent: 503 SERVICE UNAVAILABLE
	CodeUnavailable Code = 14

	// CodeDataLoss indicates unrecoverable data loss or corruption.
	//
	// HTTP equivalent: 500 INTERNAL SERVER ERROR
	CodeDataLoss Code = 15

	// CodeUnauthenticated indicates the request does not have valid
	// authentication credentials for the operation. This is different than
	// PermissionDenied, which deals with authorization.
	//
	// HTTP equivalent: 401 UNAUTHORIZED
	CodeUnauthenticated Code = 16
)

// Code is an error code.
//
// Unlike gRPC and Twirp, there is no zero code for success.
//
// All errors must have a valid code. If an error does not have a valid
// code when performing error operations, a new error with CodeInternal
// will be returned.
type Code int

var (
	codeToString = map[Code]string{
		CodeCanceled:           "CANCELED",
		CodeUnknown:            "UNKNOWN",
		CodeInvalidArgument:    "INVALID_ARGUMENT",
		CodeDeadlineExceeded:   "DEADLINE_EXCEEDED",
		CodeNotFound:           "NOT_FOUND",
		CodeAlreadyExists:      "ALREADY_EXISTS",
		CodePermissionDenied:   "PERMISSION_DENIED",
		CodeResourceExhausted:  "RESOURCE_EXHAUSTED",
		CodeFailedPrecondition: "FAILED_PRECONDITION",
		CodeAborted:            "ABORTED",
		CodeOutOfRange:         "OUT_OF_RANGE",
		CodeUnimplemented:      "UNIMPLEMENTED",
		CodeInternal:           "INTERNAL",
		CodeUnavailable:        "UNAVAILABLE",
		CodeDataLoss:           "DATA_LOSS",
		CodeUnauthenticated:    "UNAUTHENTICATED",
	}
)

// String returns the string value of c.
func (c Code) String() string {
	s, ok := codeToString[c]
	if !ok {
		return strconv.Itoa(int(c))
	}
	return s
}

// NewError returns a new error with a code.
//
// The value of Error() will only contain the message. If you would like to
// also print the code, you must do this manually.
//
// If the code is invalid, an error with CodeInternal will be returned.
func NewError(code Code, message string) error {
	return newMultiError(code, message)
}

// NewErrorf returns a new error.
//
// The value of Error() will only contain the message. If you would like to
// also print the code, you must do this manually.
//
// If the code is invalid, an error with CodeInternal will be returned.
func NewErrorf(code Code, format string, args ...interface{}) error {
	return newMultiError(code, fmt.Sprintf(format, args...))
}

// NewCanceled is a convenience function for errors with CodeCanceled.
func NewCanceled(message string) error {
	return NewError(CodeCanceled, message)
}

// NewCanceledf is a convenience function for errors with CodeCanceled.
func NewCanceledf(format string, args ...interface{}) error {
	return NewErrorf(CodeCanceled, format, args...)
}

// NewUnknown is a convenience function for errors with CodeUnknown.
func NewUnknown(message string) error {
	return NewError(CodeUnknown, message)
}

// NewUnknownf is a convenience function for errors with CodeUnknown.
func NewUnknownf(format string, args ...interface{}) error {
	return NewErrorf(CodeUnknown, format, args...)
}

// NewInvalidArgument is a convenience function for errors with CodeInvalidArgument.
func NewInvalidArgument(message string) error {
	return NewError(CodeInvalidArgument, message)
}

// NewInvalidArgumentf is a convenience function for errors with CodeInvalidArgument.
func NewInvalidArgumentf(format string, args ...interface{}) error {
	return NewErrorf(CodeInvalidArgument, format, args...)
}

// NewDeadlineExceeded is a convenience function for errors with CodeDeadlineExceeded.
func NewDeadlineExceeded(message string) error {
	return NewError(CodeDeadlineExceeded, message)
}

// NewDeadlineExceededf is a convenience function for errors with CodeDeadlineExceeded.
func NewDeadlineExceededf(format string, args ...interface{}) error {
	return NewErrorf(CodeDeadlineExceeded, format, args...)
}

// NewNotFound is a convenience function for errors with CodeNotFound.
func NewNotFound(message string) error {
	return NewError(CodeNotFound, message)
}

// NewNotFoundf is a convenience function for errors with CodeNotFound.
func NewNotFoundf(format string, args ...interface{}) error {
	return NewErrorf(CodeNotFound, format, args...)
}

// NewAlreadyExists is a convenience function for errors with CodeAlreadyExists.
func NewAlreadyExists(message string) error {
	return NewError(CodeAlreadyExists, message)
}

// NewAlreadyExistsf is a convenience function for errors with CodeAlreadyExists.
func NewAlreadyExistsf(format string, args ...interface{}) error {
	return NewErrorf(CodeAlreadyExists, format, args...)
}

// NewPermissionDenied is a convenience function for errors with CodePermissionDenied.
func NewPermissionDenied(message string) error {
	return NewError(CodePermissionDenied, message)
}

// NewPermissionDeniedf is a convenience function for errors with CodePermissionDenied.
func NewPermissionDeniedf(format string, args ...interface{}) error {
	return NewErrorf(CodePermissionDenied, format, args...)
}

// NewResourceExhausted is a convenience function for errors with CodeResourceExhausted.
func NewResourceExhausted(message string) error {
	return NewError(CodeResourceExhausted, message)
}

// NewResourceExhaustedf is a convenience function for errors with CodeResourceExhausted.
func NewResourceExhaustedf(format string, args ...interface{}) error {
	return NewErrorf(CodeResourceExhausted, format, args...)
}

// NewFailedPrecondition is a convenience function for errors with CodeFailedPrecondition.
func NewFailedPrecondition(message string) error {
	return NewError(CodeFailedPrecondition, message)
}

// NewFailedPreconditionf is a convenience function for errors with CodeFailedPrecondition.
func NewFailedPreconditionf(format string, args ...interface{}) error {
	return NewErrorf(CodeFailedPrecondition, format, args...)
}

// NewAborted is a convenience function for errors with CodeAborted.
func NewAborted(message string) error {
	return NewError(CodeAborted, message)
}

// NewAbortedf is a convenience function for errors with CodeAborted.
func NewAbortedf(format string, args ...interface{}) error {
	return NewErrorf(CodeAborted, format, args...)
}

// NewOutOfRange is a convenience function for errors with CodeOutOfRange.
func NewOutOfRange(message string) error {
	return NewError(CodeOutOfRange, message)
}

// NewOutOfRangef is a convenience function for errors with CodeOutOfRange.
func NewOutOfRangef(format string, args ...interface{}) error {
	return NewErrorf(CodeOutOfRange, format, args...)
}

// NewUnimplemented is a convenience function for errors with CodeUnimplemented.
func NewUnimplemented(message string) error {
	return NewError(CodeUnimplemented, message)
}

// NewUnimplementedf is a convenience function for errors with CodeUnimplemented.
func NewUnimplementedf(format string, args ...interface{}) error {
	return NewErrorf(CodeUnimplemented, format, args...)
}

// NewInternal is a convenience function for errors with CodeInternal.
func NewInternal(message string) error {
	return NewError(CodeInternal, message)
}

// NewInternalf is a convenience function for errors with CodeInternal.
func NewInternalf(format string, args ...interface{}) error {
	return NewErrorf(CodeInternal, format, args...)
}

// NewUnavailable is a convenience function for errors with CodeUnavailable.
func NewUnavailable(message string) error {
	return NewError(CodeUnavailable, message)
}

// NewUnavailablef is a convenience function for errors with CodeUnavailable.
func NewUnavailablef(format string, args ...interface{}) error {
	return NewErrorf(CodeUnavailable, format, args...)
}

// NewDataLoss is a convenience function for errors with CodeDataLoss.
func NewDataLoss(message string) error {
	return NewError(CodeDataLoss, message)
}

// NewDataLossf is a convenience function for errors with CodeDataLoss.
func NewDataLossf(format string, args ...interface{}) error {
	return NewErrorf(CodeDataLoss, format, args...)
}

// NewUnauthenticated is a convenience function for errors with CodeUnauthenticated.
func NewUnauthenticated(message string) error {
	return NewError(CodeUnauthenticated, message)
}

// NewUnauthenticatedf is a convenience function for errors with CodeUnauthenticated.
func NewUnauthenticatedf(format string, args ...interface{}) error {
	return NewErrorf(CodeUnauthenticated, format, args...)
}

// GetCode gets the error code.
//
// If the error is nil, this returns CodeOK.
// If the error is not created by this package, it will return CodeInternal.
//
// Note that gRPC maps to CodeUnknown in the same scenario.
func GetCode(err error) Code {
	if err == nil {
		return 0
	}
	if multiError, ok := err.(*multiError); ok {
		return multiError.code
	}
	return CodeInternal
}

// IsError returns true if err is an error created by this package.
//
// Returns false if err == nil.nil
func IsError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*multiError)
	return ok
}

// Append combines the two errors.
//
// If both are nil, this returns nil.
// If one is nil, this returns two.
// If two is nil, this returns one.
//
// If both are non-nil this will convert both to errors for this package.
// The error messages will be concatanated, and the Code will be one of the following:
//
// If both codes are equal, this returns an error with that Code.
// If the codes are unequal, this returns an error with CodeInternal.
func Append(one error, two error) error {
	if one == nil {
		return two
	}
	if two == nil {
		return one
	}

	oneMultiError, ok := one.(*multiError)
	if !ok {
		oneMultiError = newMultiError(CodeInternal, one.Error())
	}
	twoMultiError, ok := two.(*multiError)
	if !ok {
		twoMultiError = newMultiError(CodeInternal, two.Error())
	}
	oneMultiError.appendMultiError(twoMultiError)
	return oneMultiError
}

type multiError struct {
	code     Code
	messages []string
}

func newMultiError(code Code, message string) *multiError {
	if _, ok := codeToString[code]; !ok {
		message = fmt.Sprintf(
			"got invalid code %q when constructing error (original message was %q)",
			code.String(),
			message,
		)
		code = CodeInternal
	}
	return &multiError{
		code:     code,
		messages: []string{message},
	}
}

func (m *multiError) Error() string {
	switch len(m.messages) {
	case 0:
		return ""
	case 1:
		return m.messages[0]
	default:
		return strings.TrimSpace(strings.Join(m.messages, "\n"))
	}
}

func (m *multiError) appendMultiError(other *multiError) {
	if m.code != other.code {
		m.code = CodeInternal
	}
	m.messages = append(m.messages, other.messages...)
}
