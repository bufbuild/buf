// Copyright 2020-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package bufpluginexec provides protoc plugin handling and execution.
//
// Note this is currently implicitly tested through buf's protoc command.
// If this were split out into a separate package, testing would need to be moved to this package.
package bufpluginexec

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufwasm"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

const (
	// Note on versions: while Protobuf is on vX.0 where X >=21, and we
	// download protoc vX.0, the version reported by protoc --version is 3.X.0.
	// This is what we want to report here.

	// DefaultMajorVersion is the default major version.
	defaultMajorVersion = 4
	// DefaultMinorVersion is the default minor version.
	defaultMinorVersion = 25
	// DefaultPatchVersion is the default patch version.
	defaultPatchVersion = 0
	// DefaultSuffixVersion is the default suffix version.
	defaultSuffixVersion = ""
)

var (
	// ProtocProxyPluginNames are the names of the plugins that should be proxied through protoc
	// in the absence of a binary.
	ProtocProxyPluginNames = map[string]struct{}{
		"cpp":    {},
		"csharp": {},
		"java":   {},
		"js":     {},
		"objc":   {},
		"php":    {},
		"python": {},
		"pyi":    {},
		"ruby":   {},
		"kotlin": {},
	}

	// DefaultVersion represents the default version to use as compiler version for codegen requests.
	DefaultVersion = newVersion(
		defaultMajorVersion,
		defaultMinorVersion,
		defaultPatchVersion,
		defaultSuffixVersion,
	)
)

// Generator is used to generate code with plugins found on the local filesystem.
type Generator interface {
	// Generate generates a CodeGeneratorResponse for the given pluginName. The
	// pluginName must be available on the system's PATH or one of the plugins
	// built-in to protoc. The plugin path can be overridden via the
	// GenerateWithPluginPath option.
	Generate(
		ctx context.Context,
		container app.EnvStderrContainer,
		pluginName string,
		requests []*pluginpb.CodeGeneratorRequest,
		options ...GenerateOption,
	) (*pluginpb.CodeGeneratorResponse, error)
}

// NewGenerator returns a new Generator.
func NewGenerator(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	runner command.Runner,
	wasmPluginExecutor bufwasm.PluginExecutor,
) Generator {
	return newGenerator(logger, storageosProvider, runner, wasmPluginExecutor)
}

// GenerateOption is an option for Generate.
type GenerateOption func(*generateOptions)

// GenerateWithPluginPath returns a new GenerateOption that uses the given path to the plugin.
// If the path has more than one element, the first is the plugin binary and the others are
// optional additional arguments to pass to the binary.
func GenerateWithPluginPath(pluginPath ...string) GenerateOption {
	return func(generateOptions *generateOptions) {
		generateOptions.pluginPath = pluginPath
	}
}

// GenerateWithProtocPath returns a new GenerateOption that uses the given protoc
// path to the plugin.
func GenerateWithProtocPath(protocPath string) GenerateOption {
	return func(generateOptions *generateOptions) {
		generateOptions.protocPath = protocPath
	}
}

// GenerateWithWASMEnabled returns a new GenerateOption that sets wasmEnabled according to the env variable.
func GenerateWithWASMEnabled() GenerateOption {
	return func(generateOptions *generateOptions) {
		generateOptions.wasmEnabled = true
	}
}

// NewHandler returns a new Handler based on the plugin name and optional path.
//
// protocPath and pluginPath are optional.
//
//   - If a WASM plugin path is specified as the plugin name, this returns a WASM handler.
//   - If the plugin path is set, this returns a new binary handler for that path.
//   - If the plugin path is unset, this does exec.LookPath for a binary named protoc-gen-pluginName,
//     and if one is found, a new binary handler is returned for this.
//   - Else, if the name is in ProtocProxyPluginNames, this returns a new protoc proxy handler.
//   - Else, this returns error.
func NewHandler(
	storageosProvider storageos.Provider,
	runner command.Runner,
	wasmPluginExecutor bufwasm.PluginExecutor,
	pluginName string,
	options ...HandlerOption,
) (appproto.Handler, error) {
	handlerOptions := newHandlerOptions()
	for _, option := range options {
		option(handlerOptions)
	}

	// Initialize WASM plugin handler. This is the quickest check we can do in order to
	// branch here. A more stringent check is done inside the handler initialization.
	// In a followup we should unify the following three checks into a strategy pattern.
	if looksLikeWASM(pluginName) && handlerOptions.wasmEnabled {
		return newWasmHandler(wasmPluginExecutor, pluginName)
	}

	// Initialize binary plugin handler when path is specified with optional args. Return
	// on error as something is wrong with the supplied pluginPath option.
	if len(handlerOptions.pluginPath) > 0 {
		return NewBinaryHandler(runner, handlerOptions.pluginPath[0], handlerOptions.pluginPath[1:])
	}

	// Initialize binary plugin handler based on plugin name.
	if handler, err := NewBinaryHandler(runner, "protoc-gen-"+pluginName, nil); err == nil {
		return handler, nil
	}

	// Initialize builtin protoc plugin handler. We always look for protoc-gen-X first,
	// but if not, check the builtins.
	if _, ok := ProtocProxyPluginNames[pluginName]; ok {
		if handlerOptions.protocPath == "" {
			handlerOptions.protocPath = "protoc"
		}
		if protocPath, err := unsafeLookPath(handlerOptions.protocPath); err != nil {
			return nil, err
		} else {
			return newProtocProxyHandler(storageosProvider, runner, protocPath, pluginName), nil
		}
	}
	return nil, fmt.Errorf(
		"could not find protoc plugin for name %s - please make sure protoc-gen-%s is installed and present on your $PATH",
		pluginName,
		pluginName,
	)
}

// HandlerOption is an option for a new Handler.
type HandlerOption func(*handlerOptions)

// HandlerWithProtocPath returns a new HandlerOption that sets the path to the protoc binary.
//
// The default is to do exec.LookPath on "protoc".
// protocPath is expected to be unnormalized.
func HandlerWithProtocPath(protocPath string) HandlerOption {
	return func(handlerOptions *handlerOptions) {
		handlerOptions.protocPath = protocPath
	}
}

// HandlerWithPluginPath returns a new HandlerOption that sets the path to the plugin binary.
//
// The default is to do exec.LookPath on "protoc-gen-" + pluginName. pluginPath is expected
// to be unnormalized. If the path has more than one element, the first is the plugin binary
// and the others are optional additional arguments to pass to the binary
func HandlerWithPluginPath(pluginPath ...string) HandlerOption {
	return func(handlerOptions *handlerOptions) {
		handlerOptions.pluginPath = pluginPath
	}
}

// HandlerWithWASMEnabled returns a new HandlerOption that sets wasmEnabled according to the env variable.
func HandlerWithWASMEnabled() HandlerOption {
	return func(handlerOptions *handlerOptions) {
		handlerOptions.wasmEnabled = true
	}
}

// NewBinaryHandler returns a new Handler that invokes the specific plugin
// specified by pluginPath.
//
// Used by other repositories.
func NewBinaryHandler(runner command.Runner, pluginPath string, pluginArgs []string) (appproto.Handler, error) {
	pluginPath, err := unsafeLookPath(pluginPath)
	if err != nil {
		return nil, err
	}
	return newBinaryHandler(runner, pluginPath, pluginArgs), nil
}

type handlerOptions struct {
	protocPath  string
	pluginPath  []string
	wasmEnabled bool
}

func newHandlerOptions() *handlerOptions {
	return &handlerOptions{}
}

// looksLikeWASM is a minimal check for WASM plugins. A more stringent validation
// of the file is done in the handlers Handle method.
func looksLikeWASM(pluginName string) bool {
	return strings.HasSuffix(pluginName, ".wasm")
}

// unsafeLookPath is a wrapper around exec.LookPath that restores the original
// pre-Go 1.19 behavior of resolving queries that would use relative PATH
// entries. We consider it acceptable for the use case of locating plugins.
//
// https://pkg.go.dev/os/exec#hdr-Executables_in_the_current_directory
func unsafeLookPath(file string) (string, error) {
	path, err := exec.LookPath(file)
	if errors.Is(err, exec.ErrDot) {
		err = nil
	}
	return path, err
}
