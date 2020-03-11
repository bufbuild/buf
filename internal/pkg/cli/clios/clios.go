// Package clios provides extensions to to the stdlib os package.
package clios

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	internalioutil "github.com/bufbuild/buf/internal/pkg/cli/internal/ioutil"
)

// WriteCloserForFilePath returns an io.WriteCloser for the filePath.
//
// If the filePath is "-", this is interpreted as stdout and stdout is returned.
// If the filePath is the equivalent of /dev/null, this returns ioutil.Discard.
// If the filePath is "", this returns error.
// If stdout is nil and filePath is "-", returns error.
func WriteCloserForFilePath(stdout io.Writer, filePath string) (io.WriteCloser, error) {
	if filePath == "" {
		return nil, errors.New("no filePath")
	}
	if filePathIsStdout(filePath) {
		if stdout == nil {
			return nil, errors.New("file path was - but cannot write to stdout")
		}
		return internalioutil.NopWriteCloser(stdout), nil
	}
	if filePath == DevNull {
		return internalioutil.NopWriteCloser(ioutil.Discard), nil
	}
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("error creating %s: %v", filePath, err)
	}
	return file, nil
}

// ReadCloserForFilePath returns an io.ReadCloser for the filePath.
//
// If the filePath is "-", this is interpreted as stdin and stdin is returned.
// If the filePath is the equivalent of /dev/null, this returns a DiscardReader.
// If the filePath is "", this returns error.
// If stdin is nil and filePath is "-", returns error.
func ReadCloserForFilePath(stdin io.Reader, filePath string) (io.ReadCloser, error) {
	if filePath == "" {
		return nil, errors.New("no filePath")
	}
	if filePathIsStdin(filePath) {
		if stdin == nil {
			return nil, errors.New("file path was - but cannot read from stdin")
		}
		return ioutil.NopCloser(stdin), nil
	}
	if filePath == DevNull {
		return ioutil.NopCloser(internalioutil.DiscardReader), nil
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening %s: %v", filePath, err)
	}
	return file, nil
}

// filePathIsStdin returns true if filePath == "-".
func filePathIsStdin(filePath string) bool {
	return filePath == "-"
}

// filePathIsStdout returns true if filePath == "-".
func filePathIsStdout(filePath string) bool {
	return filePath == "-"
}
