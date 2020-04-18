package app

import (
	"fmt"
	"strconv"
)

type appError struct {
	exitCode int
	message  string
}

func newAppError(exitCode int, message string) *appError {
	if exitCode == 0 {
		message = fmt.Sprintf(
			"got invalid exit code %d when constructing error (original message was %q)",
			exitCode,
			message,
		)
		exitCode = 1
	}
	return &appError{
		exitCode: exitCode,
		message:  message,
	}
}

func (e *appError) Error() string {
	if e.message != "" {
		return e.message
	}
	return "exit status " + strconv.Itoa(e.exitCode)
}

func printError(container StderrContainer, err error) {
	if errString := err.Error(); errString != "" {
		_, _ = fmt.Fprintln(container.Stderr(), errString)
	}
}
