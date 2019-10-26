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
	"github.com/golang/protobuf/proto"
	plugin_go "github.com/golang/protobuf/protoc-gen-go/plugin"
)

// ResponseWriter is a response writer.
//
// Not thread-safe.
type ResponseWriter interface {
	// WriteCodeGeneratorResponseFile adds the file to the response.
	//
	// Can be called multiple times.
	WriteCodeGeneratorResponseFile(*plugin_go.CodeGeneratorResponse_File)
	// WriteError writes the error to the response.
	//
	// Can be called multiple times. Errors will be concatenated by newlines.
	// Resulting error string will have spaces trimmed before creating the response.
	WriteError(string)
}

// Handler handles protoc plugin functionality.
type Handler interface {
	// Handle handles the request.
	//
	// Only system errors should be returned.
	Handle(
		stderr io.Writer,
		responseWriter ResponseWriter,
		request *plugin_go.CodeGeneratorRequest,
	)
}

// HandlerFunc is a function that implements Handler.
type HandlerFunc func(
	io.Writer,
	ResponseWriter,
	*plugin_go.CodeGeneratorRequest,
)

// Handle implements Handler.
func (h HandlerFunc) Handle(
	stderr io.Writer,
	responseWriter ResponseWriter,
	request *plugin_go.CodeGeneratorRequest,
) {
	h(stderr, responseWriter, request)
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
	responseWriter := newResponseWriter()
	handler.Handle(runEnv.Stderr, responseWriter, request)
	response := responseWriter.ToCodeGeneratorResponse()
	data, err := proto.Marshal(response)
	if err != nil {
		return err
	}
	_, err = runEnv.Stdout.Write(data)
	return err
}

type responseWriter struct {
	files         []*plugin_go.CodeGeneratorResponse_File
	errorMessages []string
}

func newResponseWriter() *responseWriter {
	return &responseWriter{}
}

func (r *responseWriter) WriteCodeGeneratorResponseFile(file *plugin_go.CodeGeneratorResponse_File) {
	r.files = append(r.files, file)
}

func (r *responseWriter) WriteError(errorMessage string) {
	r.errorMessages = append(r.errorMessages, errorMessage)
}

func (r *responseWriter) ToCodeGeneratorResponse() *plugin_go.CodeGeneratorResponse {
	var err *string
	if errorMessage := strings.TrimSpace(strings.Join(r.errorMessages, "\n")); errorMessage != "" {
		err = proto.String(errorMessage)
	}
	return &plugin_go.CodeGeneratorResponse{
		File:  r.files,
		Error: err,
	}
}
