// Package osutil provides extensions to to the stdlib os package.
//
// It primary interprets "-" as stdin or stdout, and the os-independent
// version of /dev/null
package osutil

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/bufbuild/buf/internal/pkg/errs"
	ioutilext "github.com/bufbuild/buf/internal/pkg/ioutil"
)

// FilePathIsDevNull returns true if the file path is the equivalent of /dev/null.
func FilePathIsDevNull(filePath string) bool {
	switch runtime.GOOS {
	case "darwin", "linux":
		return filePath == "/dev/null"
	case "windows":
		return filePath == "nul"
	default:
		return false
	}
}

// DevNull outputs the equivalent of /dev/null for darwin, linux, and windows.
func DevNull() (string, error) {
	switch runtime.GOOS {
	case "darwin", "linux":
		return "/dev/null", nil
	case "windows":
		return "nul", nil
	default:
		return "", fmt.Errorf("unknown operating system: %q", runtime.GOOS)
	}
}

// FilePathIsStdout returns true if filePath == "-".
func FilePathIsStdout(filePath string) bool {
	return filePath == "-"
}

// WriteCloserForFilePath returns an io.WriteCloser for the filePath.
//
// If the filePath is "-", this is interpreted as stdout and stdout is returned.
// If the filePath is the equivalent of /dev/null, this returns ioutil.Discard.
// If the filePath is "", this returns system error.
// If stdout is nil and filePath is "-", returns user error.
func WriteCloserForFilePath(stdout io.Writer, filePath string) (io.WriteCloser, error) {
	if filePath == "" {
		return nil, errors.New("no filePath")
	}
	if FilePathIsStdout(filePath) {
		if stdout == nil {
			return nil, errs.NewUserError("file path was - but cannot write to stdout")
		}
		return ioutilext.NopWriteCloser(stdout), nil
	}
	if FilePathIsDevNull(filePath) {
		return ioutilext.NopWriteCloser(ioutil.Discard), nil
	}
	file, err := os.Create(filePath)
	if err != nil {
		return nil, errs.NewUserErrorf("error creating %s: %v", filePath, err)
	}
	return file, nil
}

// FilePathIsStdin returns true if filePath == "-".
func FilePathIsStdin(filePath string) bool {
	return filePath == "-"
}

// ReadCloserForFilePath returns an io.ReadCloser for the filePath.
//
// If the filePath is "-", this is interpreted as stdin and stdin is returned.
// If the filePath is the equivalent of /dev/null, this returns a DiscardReader.
// If the filePath is "", this returns system error.
// If stdin is nil and filePath is "-", returns user error.
func ReadCloserForFilePath(stdin io.Reader, filePath string) (io.ReadCloser, error) {
	if filePath == "" {
		return nil, errors.New("no filePath")
	}
	if FilePathIsStdin(filePath) {
		if stdin == nil {
			return nil, errs.NewUserError("file path was - but cannot read from stdin")
		}
		return ioutil.NopCloser(stdin), nil
	}
	if FilePathIsDevNull(filePath) {
		return ioutil.NopCloser(ioutilext.DiscardReader), nil
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, errs.NewUserErrorf("error opening %s: %v", filePath, err)
	}
	return file, nil
}
