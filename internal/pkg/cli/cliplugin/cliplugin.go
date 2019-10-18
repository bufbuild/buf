// Package cliplugin contains helper functionality for protoc plugins.
package cliplugin

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/bufbuild/buf/internal/pkg/cli"
	"github.com/bufbuild/buf/internal/pkg/cli/internal"
	"github.com/bufbuild/buf/internal/pkg/errs"
	"github.com/golang/protobuf/proto"
	plugin_go "github.com/golang/protobuf/protoc-gen-go/plugin"
	"go.uber.org/multierr"
)

// Handler handles protoc plugun functionality.
type Handler interface {
	// Handle handles the request.

	// If Handle returns a user error and no system errors, this will be added to the error field.
	// If Handle returns any system error, the whole error will be printed as an error to stderr
	// and the plugin will exit with code 1.
	// Only one of files and error can be returned.
	Handle(
		stderr io.Writer,
		request *plugin_go.CodeGeneratorRequest,
	) ([]*plugin_go.CodeGeneratorResponse_File, error)
}

// HandlerFunc is a function that implements Handler.
type HandlerFunc func(io.Writer, *plugin_go.CodeGeneratorRequest) ([]*plugin_go.CodeGeneratorResponse_File, error)

// Handle implements Handler.
func (h HandlerFunc) Handle(
	stderr io.Writer,
	request *plugin_go.CodeGeneratorRequest,
) ([]*plugin_go.CodeGeneratorResponse_File, error) {
	return h(stderr, request)
}

// Main runs the application using the OS runtime and calling os.Exit on the return value of Run.
func Main(handler Handler) {
	os.Exit(Run(handler, internal.NewOSRunEnv()))
}

// Run runs the application, returning the exit code.
//
// RunEnv will be modified to have dummy values if fields are not set.
func Run(handler Handler, runEnv *cli.RunEnv) int {
	start := time.Now()
	internal.SetRunEnvDefaults(runEnv)
	if err := runHandler(handler, start, runEnv); err != nil {
		if errString := err.Error(); errString != "" {
			_, _ = fmt.Fprintln(runEnv.Stderr, errString)
		}
		return 1
	}
	return 0
}

// runHandler only returns an error that should be treated as a system error
func runHandler(handler Handler, start time.Time, runEnv *cli.RunEnv) error {
	input, err := ioutil.ReadAll(runEnv.Stdin)
	if err != nil {
		return err
	}
	request := &plugin_go.CodeGeneratorRequest{}
	if err := proto.Unmarshal(input, request); err != nil {
		return err
	}

	handlerFiles, handlerErr := handler.Handle(runEnv.Stderr, request)
	if handlerErr != nil {
		var userErrs []error
		var systemErrs []error
		for _, err := range multierr.Errors(handlerErr) {
			// really need to replace this with xerrors
			if errs.IsUserError(err) {
				userErrs = append(userErrs, err)
			} else {
				systemErrs = append(systemErrs, err)
			}
		}
		if len(systemErrs) > 0 {
			return handlerErr
		}
		if len(userErrs) > 0 {
			response := &plugin_go.CodeGeneratorResponse{}
			errStrings := make([]string, 0, len(userErrs))
			for _, err := range userErrs {
				if errString := strings.TrimSpace(err.Error()); errString != "" {
					errStrings = append(errStrings, errString)
				}
			}
			if len(errStrings) > 0 {
				response.Error = proto.String(strings.Join(errStrings, "\n"))
			}
			data, err := proto.Marshal(response)
			if err != nil {
				return err
			}
			_, err = runEnv.Stdout.Write(data)
			return err
		}
		return nil
	}

	response := &plugin_go.CodeGeneratorResponse{
		File: handlerFiles,
	}
	data, err := proto.Marshal(response)
	if err != nil {
		return err
	}
	_, err = runEnv.Stdout.Write(data)
	return err
}
