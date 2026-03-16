package output

import (
	"errors"
)

// Category is the machine-facing error category used for exit-code mapping.
type Category string

const (
	CategoryValidation Category = "validation"
	CategoryAuth       Category = "auth"
	CategoryNotFound   Category = "not_found"
	CategoryError      Category = "error"
)

// categorized is the interface app errors implement.
type categorized interface {
	Category() string
}

// AppError carries an error category alongside the underlying error so the
// process boundary can map it to the correct exit code.
type AppError struct {
	Cat     Category
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func (e *AppError) Category() string { return string(e.Cat) }

// categoryOf extracts the Category from an error that implements the
// categorized interface, or returns CategoryError as default.
func categoryOf(err error) Category {
	var c categorized
	if errors.As(err, &c) {
		switch Category(c.Category()) {
		case CategoryValidation, CategoryAuth, CategoryNotFound:
			return Category(c.Category())
		}
	}
	return CategoryError
}

// ExitCode returns the stable process exit code for the given error.
func ExitCode(err error) int {
	switch categoryOf(err) {
	case CategoryValidation:
		return 2
	case CategoryAuth:
		return 3
	case CategoryNotFound:
		return 4
	}
	return 1
}

// StateForError returns the output State that corresponds to an error.
func StateForError(err error) State {
	switch categoryOf(err) {
	case CategoryAuth:
		return StateNotLoggedIn
	case CategoryNotFound:
		return StateNotFound
	case CategoryValidation:
		return StateError
	}
	return StateError
}

// ErrorEnvelope builds a failure Envelope from an error.
func ErrorEnvelope(command string, err error) Envelope {
	msg := "an error occurred"
	if err != nil {
		msg = err.Error()
	}
	return Envelope{
		OK:      false,
		Command: command,
		State:   StateForError(err),
		Message: msg,
	}
}
