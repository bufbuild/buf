package output

import (
	"fmt"
	"io"
)

// PrintError prints the error.
func PrintError(stderr io.Writer, err error) {
	if errString := err.Error(); errString != "" {
		_, _ = fmt.Fprintln(stderr, errString)
	}
}
