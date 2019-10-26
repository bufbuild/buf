// Package bufcheck contains the implementations of the lint and breaking change detection checkers.
//
// There is a lot of shared logic between the two, and originally they were actually combined into
// one logical entity (where some checks happened to be linters, and some checks happen to be
// breaking change detectors), so some of this is historical.
package bufcheck

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/bufbuild/buf/internal/pkg/errs"
)

// Checker is a checker.
type Checker interface {
	json.Marshaler

	// ID returns the ID of the Checker.
	//
	// UPPER_SNAKE_CASE.
	ID() string
	// Categories returns the categories of the Checker.
	//
	// UPPER_SNAKE_CASE.
	// Sorted.
	Categories() []string
	// Purpose returns the purpose of the Checker.
	//
	// Full sentence.
	Purpose() string
}

// PrintCheckers prints the checkers to the writer.
func PrintCheckers(writer io.Writer, checkers []Checker, asJSON bool) (retErr error) {
	if len(checkers) == 0 {
		return nil
	}
	if !asJSON {
		tabWriter := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
		defer func() {
			retErr = errs.Append(retErr, tabWriter.Flush())
		}()
		writer = tabWriter
		if _, err := fmt.Fprintln(writer, "ID\tCATEGORIES\tPURPOSE"); err != nil {
			return err
		}
	}
	for _, checker := range checkers {
		if err := printChecker(writer, checker, asJSON); err != nil {
			return err
		}
	}
	return nil
}

func printChecker(writer io.Writer, checker Checker, asJSON bool) error {
	if asJSON {
		data, err := json.Marshal(checker)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(writer, string(data)); err != nil {
			return err
		}
		return nil
	}
	if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\n", checker.ID(), strings.Join(checker.Categories(), ", "), checker.Purpose()); err != nil {
		return err
	}
	return nil
}
