// Package core holds PicBerth's application services: every state change, authorization decision and
// business rule lives here, behind Principal-checked methods. It never imports net/http.
package core

import (
	"fmt"
	"time"
)

// Stable error codes (docs/api.md §2).
const (
	CodeValidation          = "validation"
	CodeUnauthorized        = "unauthorized"
	CodeForbidden           = "forbidden"
	CodeNotFound            = "not-found"
	CodeConflict            = "conflict"
	CodeTooLarge            = "too-large"
	CodeUnsupportedFormat   = "unsupported-format"
	CodeInvalidImage        = "invalid-image"
	CodeTooManyPixels       = "too-many-pixels"
	CodeSetupRequired       = "setup-required"
	CodeSetupDone           = "setup-done"
	CodeRateLimited         = "rate-limited"
	CodeInsufficientStorage = "insufficient-storage"
	CodeFolderExists        = "folder-exists"
	CodeFolderNotEmpty      = "folder-not-empty"
	CodeFolderDepth         = "folder-depth"
	CodeInvalidScope        = "invalid-scope"
	CodeInternal            = "internal"
)

// Error is a domain error with a stable code and a human message that is safe to show to clients.
type Error struct {
	Code       string
	Detail     string
	RetryAfter time.Duration // rate-limited only
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return e.Code + ": " + e.Detail
}

// Is lets errors.Is(err, core.ErrNotFound) match any *Error with the same code.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Code == e.Code
}

// Sentinels for errors.Is.
var (
	ErrValidation          = &Error{Code: CodeValidation}
	ErrUnauthorized        = &Error{Code: CodeUnauthorized}
	ErrForbidden           = &Error{Code: CodeForbidden}
	ErrNotFound            = &Error{Code: CodeNotFound}
	ErrConflict            = &Error{Code: CodeConflict}
	ErrTooLarge            = &Error{Code: CodeTooLarge}
	ErrUnsupportedFormat   = &Error{Code: CodeUnsupportedFormat}
	ErrInvalidImage        = &Error{Code: CodeInvalidImage}
	ErrTooManyPixels       = &Error{Code: CodeTooManyPixels}
	ErrSetupRequired       = &Error{Code: CodeSetupRequired}
	ErrSetupDone           = &Error{Code: CodeSetupDone}
	ErrRateLimited         = &Error{Code: CodeRateLimited}
	ErrInsufficientStorage = &Error{Code: CodeInsufficientStorage}
)

func newErr(code, format string, args ...any) *Error {
	return &Error{Code: code, Detail: fmt.Sprintf(format, args...)}
}

func validation(format string, args ...any) *Error { return newErr(CodeValidation, format, args...) }
func notFound() *Error                             { return &Error{Code: CodeNotFound, Detail: "Not found."} }
func forbidden(detail string) *Error               { return &Error{Code: CodeForbidden, Detail: detail} }
func conflict(format string, args ...any) *Error   { return newErr(CodeConflict, format, args...) }

func folderExists() *Error {
	return &Error{Code: CodeFolderExists, Detail: "A folder with that name already exists here."}
}

func folderNotEmpty() *Error {
	return &Error{Code: CodeFolderNotEmpty, Detail: "The folder contains images or folders."}
}

func folderDepth() *Error {
	return &Error{Code: CodeFolderDepth, Detail: "Folders can be at most 3 levels deep."}
}
func unauthorized() *Error {
	return &Error{Code: CodeUnauthorized, Detail: "Authentication required or credentials invalid."}
}

func rateLimited(after time.Duration) *Error {
	secs := int((after + time.Second - 1) / time.Second)
	if secs < 1 {
		secs = 1
	}
	return &Error{Code: CodeRateLimited, Detail: fmt.Sprintf("Too many requests. Try again in %d s.", secs), RetryAfter: time.Duration(secs) * time.Second}
}

// ErrorCode returns the code of err when it is (or wraps) an *Error, else CodeInternal.
func ErrorCode(err error) string {
	for e := err; e != nil; {
		if ce, ok := e.(*Error); ok {
			return ce.Code
		}
		u, ok := e.(interface{ Unwrap() error })
		if !ok {
			break
		}
		e = u.Unwrap()
	}
	return CodeInternal
}
