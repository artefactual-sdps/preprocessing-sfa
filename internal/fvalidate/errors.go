package fvalidate

import "fmt"

type SystemError struct {
	validator string

	// exitCode is the exit code returned by the system command.
	exitCode int

	// err is the underlying error returned by the system command.
	err error

	// message is a human-readable error message intended for the operator.
	message string
}

func NewSystemError(validator string, exitCode int, err error, msg string) *SystemError {
	return &SystemError{
		validator: validator,
		exitCode:  exitCode,
		err:       err,
		message:   msg,
	}
}

func (e *SystemError) Validator() string {
	return e.validator
}

func (e *SystemError) Error() string {
	return fmt.Sprintf("system error: exit code %d: %s", e.exitCode, e.err)
}

func (e *SystemError) Unwrap() error {
	return e.err
}

func (e *SystemError) Message() string {
	return e.message
}
