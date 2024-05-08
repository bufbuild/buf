// Copyright 2020-2024 Buf Technologies, Inc.
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

// Package appext contains functionality to work with flags.
package appext

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	configFileName   = "config.yaml"
	secretRelDirPath = "secrets"
)

// NameContainer is a container for named applications.
//
// Application name foo-bar translates to environment variable prefix FOO_BAR_, which is
// used for the various functions that NameContainer provides.
type NameContainer interface {
	// AppName is the application name.
	//
	// The name must be in [a-zA-Z0-9-_].
	AppName() string
	// ConfigDirPath is the config directory path for the named application.
	//
	// First checks for $APP_NAME_CONFIG_DIR.
	// If this is not set, uses app.ConfigDirPath()/app-name.
	// Unnormalized.
	ConfigDirPath() string
	// CacheDirPath is the cache directory path for the named application.
	//
	// First checks for $APP_NAME_CACHE_DIR.
	// If this is not set, uses app.CacheDirPath()/app-name.
	// Unnormalized.
	CacheDirPath() string
	// DataDirPath is the data directory path for the named application.
	//
	// First checks for $APP_NAME_DATA_DIR.
	// If this is not set, uses app.DataDirPath()/app-name.
	// Unnormalized.
	DataDirPath() string
	// Port is the port to use for serving.
	//
	// First checks for $APP_NAME_PORT.
	// If this is not set, checks for $PORT.
	// If this is not set, returns 0, which means no port is known.
	// Returns error on parse.
	Port() (uint16, error)
}

// NewNameContainer returns a new NameContainer.
//
// The name must be in [a-zA-Z0-9-_].
func NewNameContainer(envContainer app.EnvContainer, appName string) (NameContainer, error) {
	return newNameContainer(envContainer, appName)
}

// LoggerContainer provides a *zap.Logger.
type LoggerContainer interface {
	Logger() *zap.Logger
}

// NewLoggerContainer returns a new LoggerContainer.
func NewLoggerContainer(logger *zap.Logger) LoggerContainer {
	return newLoggerContainer(logger)
}

// TracerContainer provides a trace.Tracer based on the application name.
type TracerContainer interface {
	Tracer() trace.Tracer
}

// NewTracerContainer returns a new TracerContainer for the application name.
func NewTracerContainer(appName string) TracerContainer {
	return newTracerContainer(appName)
}

// VerboseContainer provides a verbose.Printer.
type VerboseContainer interface {
	// VerboseEnabled returns true if verbose mode is enabled.
	VerboseEnabled() bool
	// VerbosePrinter returns a verbose.Printer to use for verbose printing.
	VerbosePrinter() verbose.Printer
}

// NewVerboseContainer returns a new VerboseContainer.
func NewVerboseContainer(verbosePrinter verbose.Printer) VerboseContainer {
	return newVerboseContainer(verbosePrinter)
}

// Container contains not just the base app container, but all extended containers.
type Container interface {
	app.Container
	NameContainer
	LoggerContainer
	TracerContainer
	VerboseContainer
}

// NewContainer returns a new Container.
func NewContainer(
	baseContainer app.Container,
	appName string,
	logger *zap.Logger,
	verbosePrinter verbose.Printer,
) (Container, error) {
	return newContainer(
		baseContainer,
		appName,
		logger,
		verbosePrinter,
	)
}

// Interceptor intercepts and adapts the request or response of run functions.
type Interceptor func(func(context.Context, Container) error) func(context.Context, Container) error

// SubCommandBuilder builds run functions for sub-commands.
type SubCommandBuilder interface {
	NewRunFunc(func(context.Context, Container) error) func(context.Context, app.Container) error
}

// Builder builds run functions for both top-level commands and sub-commands.
type Builder interface {
	BindRoot(flagSet *pflag.FlagSet)
	SubCommandBuilder
}

// NewBuilder returns a new Builder.
func NewBuilder(appName string, options ...BuilderOption) Builder {
	return newBuilder(appName, options...)
}

// BuilderOption is an option for a new Builder
type BuilderOption func(*builder)

// BuilderWithTimeout returns a new BuilderOption that adds a timeout flag and the default timeout.
func BuilderWithTimeout(defaultTimeout time.Duration) BuilderOption {
	return func(builder *builder) {
		builder.defaultTimeout = defaultTimeout
	}
}

// BuilderWithTracing enables zap tracing for the builder.
func BuilderWithTracing() BuilderOption {
	return func(builder *builder) {
		builder.tracing = true
	}
}

// BuilderWithInterceptor adds the given interceptor for all run functions.
func BuilderWithInterceptor(interceptor Interceptor) BuilderOption {
	return func(builder *builder) {
		builder.interceptors = append(builder.interceptors, interceptor)
	}
}

// BuilderWithDefaultLogLevel adds the given default log level.
func BuilderWithDefaultLogLevel(defaultLogLevel zapcore.Level) BuilderOption {
	return func(builder *builder) {
		builder.defaultLogLevel = defaultLogLevel
	}
}

// ReadConfig reads the configuration from the YAML configuration file config.yaml
// in the configuration directory.
//
// If the file does not exist, this is a no-op.
// The value should be a pointer to unmarshal into.
func ReadConfig(container NameContainer, value interface{}) error {
	configFilePath := filepath.Join(container.ConfigDirPath(), configFileName)
	data, err := os.ReadFile(configFilePath)
	if !errors.Is(err, os.ErrNotExist) {
		if err != nil {
			return fmt.Errorf("could not read %s configuration file at %s: %w", container.AppName(), configFilePath, err)
		}
		if err := encoding.UnmarshalYAMLStrict(data, value); err != nil {
			return fmt.Errorf("invalid %s configuration file: %w", container.AppName(), err)
		}
	}
	return nil
}

// ReadSecret returns the contents of the file at path
// filepath.Join(container.ConfigDirPath(), secretRelDirPath, name).
func ReadSecret(container NameContainer, name string) (string, error) {
	secretFilePath := filepath.Join(container.ConfigDirPath(), secretRelDirPath, name)
	data, err := os.ReadFile(secretFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read secret at %s: %w", secretFilePath, err)
	}
	return string(data), nil
}

// WriteConfig writes the configuration to the YAML configuration file config.yaml
// in the configuration directory.
//
// The directory is created if it does not exist.
// The value should be a pointer to marshal.
func WriteConfig(container NameContainer, value interface{}) error {
	data, err := encoding.MarshalYAML(value)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(container.ConfigDirPath(), 0755); err != nil {
		return err
	}
	configFilePath := filepath.Join(container.ConfigDirPath(), configFileName)
	fileMode := os.FileMode(0644)
	// OK to use os.Stat instead of os.Lstat here
	if fileInfo, err := os.Stat(configFilePath); err == nil {
		fileMode = fileInfo.Mode()
	}
	return os.WriteFile(configFilePath, data, fileMode)
}

// Listen listens on the container's port, falling back to defaultPort.
func Listen(ctx context.Context, container NameContainer, defaultPort uint16) (net.Listener, error) {
	port, err := container.Port()
	if err != nil {
		return nil, err
	}
	if port == 0 {
		port = defaultPort
	}
	// Must be 0.0.0.0
	var listenConfig net.ListenConfig
	return listenConfig.Listen(ctx, "tcp", fmt.Sprintf("0.0.0.0:%d", port))
}
