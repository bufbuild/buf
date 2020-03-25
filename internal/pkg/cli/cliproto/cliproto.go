// Package cliproto contains helper functionality for protoc plugins.
package cliproto

import (
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/cli/clienv"
	"github.com/bufbuild/buf/internal/pkg/cli/internal/output"
	"github.com/bufbuild/buf/internal/pkg/util/utilproto"
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

// Env is an execution environment for the plugin.
//
// This is a partial of clienv.Env.
type Env interface {
	// Stderr is the stderr.
	//
	// If no value was passed when the Env was created, this will return io.EOF on any call.
	Stderr() io.Writer
	// Getenv is the equivalent of os.Getenv.
	Getenv(key string) string
}

// Handler handles protoc plugin functionality.
type Handler interface {
	// Handle handles the request.
	//
	// Only system errors should be returned.
	Handle(
		env Env,
		responseWriter ResponseWriter,
		request *plugin_go.CodeGeneratorRequest,
	)
}

// HandlerFunc is a function that implements Handler.
type HandlerFunc func(
	Env,
	ResponseWriter,
	*plugin_go.CodeGeneratorRequest,
)

// Handle implements Handler.
func (h HandlerFunc) Handle(
	env Env,
	responseWriter ResponseWriter,
	request *plugin_go.CodeGeneratorRequest,
) {
	h(env, responseWriter, request)
}

// Main runs the application using the OS runtime and calling os.Exit on the return value of Run.
func Main(handler Handler) {
	env, err := clienv.NewOSEnv()
	if err != nil {
		output.PrintError(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(Run(handler, env))
}

// Run runs the application, returning the exit code.
//
// Env will be modified to have dummy values if fields are not set.
func Run(handler Handler, env clienv.Env) int {
	if err := runHandler(handler, env); err != nil {
		output.PrintError(env.Stderr(), err)
		return 1
	}
	return 0
}

// runHandler only returns an error that should be treated as a system error
func runHandler(handler Handler, env clienv.Env) error {
	input, err := ioutil.ReadAll(env.Stdin())
	if err != nil {
		return err
	}
	request := &plugin_go.CodeGeneratorRequest{}
	if err := proto.Unmarshal(input, request); err != nil {
		return err
	}
	responseWriter := newResponseWriter()
	handler.Handle(env, responseWriter, request)
	response := responseWriter.ToCodeGeneratorResponse()
	data, err := utilproto.MarshalWire(response)
	if err != nil {
		return err
	}
	_, err = env.Stdout().Write(data)
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
