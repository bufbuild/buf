package imports

import (
	"errors"
	"io"
)

// ScanForImports should be replaced with protocompile's ScanForImports when available.
//
// TODO: Replave with protocompile's ScanForImports.
func ScanForImports(io.Reader) ([]string, error) {
	return nil, errors.New("TODO")
}
