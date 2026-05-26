package output

import (
	"fmt"
	"os"
)

// Exit code constants.
const (
	ExitOK      = 0
	ExitGeneral = 1
	ExitInput   = 2
	ExitAuth    = 3
	ExitNetwork = 4
	ExitSystem  = 5
)

// ExitError is a structured error that carries an exit code, error
// type, and message. Command handlers return this instead of
// calling fmt.Printf and os.Exit directly.
type ExitError struct {
	Code    int
	Type    string
	Message string
	Err     error
}

func (e *ExitError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit code %d", e.Code)
}

func (e *ExitError) Unwrap() error { return e.Err }

// ErrValidation returns an input/usage error (exit code 2).
func ErrValidation(format string, args ...any) error {
	return &ExitError{
		Code:    ExitInput,
		Type:    "validation",
		Message: fmt.Sprintf(format, args...),
	}
}

// ErrAuth returns an auth/not-logged-in error (exit code 3).
func ErrAuth(format string, args ...any) error {
	return &ExitError{
		Code:    ExitAuth,
		Type:    "auth",
		Message: fmt.Sprintf(format, args...),
	}
}

// ErrNetwork returns a network/API error (exit code 4).
func ErrNetwork(format string, args ...any) error {
	return &ExitError{
		Code:    ExitNetwork,
		Type:    "network",
		Message: fmt.Sprintf(format, args...),
	}
}

// ErrSystem returns a system/runtime error (exit code 5).
func ErrSystem(format string, args ...any) error {
	return &ExitError{
		Code:    ExitSystem,
		Type:    "system",
		Message: fmt.Sprintf(format, args...),
	}
}

// ErrBare returns an exit code without an error envelope.
// Used when output has already been written (e.g. JSON mode).
func ErrBare(code int) error {
	return &ExitError{Code: code}
}

// WriteError outputs a structured error envelope to stderr
// and returns the exit code. Used by the central error handler.
func WriteError(err error) int {
	if err == nil {
		return ExitOK
	}
	var ee *ExitError
	if AsExitError(err, &ee) {
		if ee.Type != "" {
			_ = WriteJSON(os.Stderr, Envelope{
				OK:    false,
				Code:  ee.Type,
				Error: ee.Message,
			})
		}
		return ee.Code
	}
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	return ExitGeneral
}

// AsExitError checks if err is or wraps an ExitError.
func AsExitError(err error, target **ExitError) bool {
	if err == nil {
		return false
	}
	if ee, ok := err.(*ExitError); ok {
		*target = ee
		return true
	}
	return false
}
