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

package bufctl

import (
	"context"
	"net/http"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/bufnew/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/applog"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/httpauth"
)

const (
	// ExitCodeFileAnnotation is the exit code used when we print file annotations.
	//
	// We use a different exit code to be able to distinguish user-parsable errors from system errors.
	ExitCodeFileAnnotation = 100
)

var (
	// ErrFileAnnotation is used when we print file annotations and want to return an error.
	//
	// The app package works on the concept that an error results in a non-zero exit
	// code, and we already print the messages with PrintFileAnnotations, so we do
	// not want to print any additional error message.
	//
	// We also exit with 100 to be able to distinguish user-parsable errors from system errors.
	ErrFileAnnotation = app.NewError(ExitCodeFileAnnotation, "")
)

type Container interface {
	app.EnvStdioContainer
	applog.Container
}

type Controller interface {
	GetWorkspace(
		ctx context.Context,
		sourceOrModuleInput string,
		options ...FunctionOption,
	) (bufworkspace.Workspace, error)
	GetImage(
		ctx context.Context,
		input string,
		options ...FunctionOption,
	) (bufimage.Image, error)
	PutImage(
		ctx context.Context,
		imageOutput string,
		image bufimage.Image,
		options ...FunctionOption,
	) error
}

func NewController(
	container Container,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	httpClient *http.Client,
	httpauthAuthenticator httpauth.Authenticator,
	gitClonerOptions git.ClonerOptions,
	options ...ControllerOption,
) (Controller, error) {
	return newController(
		container,
		moduleKeyProvider,
		moduleDataProvider,
		httpClient,
		httpauthAuthenticator,
		gitClonerOptions,
		options...,
	)
}

type ControllerOption func(*controller)

func WithDisableSymlinks(disableSymlinks bool) ControllerOption {
	return func(controller *controller) {
		controller.disableSymlinks = disableSymlinks
	}
}

func WithErrorFormat(errorFormat string) ControllerOption {
	return func(controller *controller) {
		controller.errorFormat = errorFormat
	}
}

func WithFileAnnotationsToStdout() ControllerOption {
	return func(controller *controller) {
		controller.fileAnnotationsToStdout = true
	}
}

type FunctionOption func(*functionOptions)

func WithTargetPaths(targetPaths []string, targetExcludePaths []string) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.targetPaths = targetPaths
		functionOptions.targetExcludePaths = targetExcludePaths
	}
}

func WithExcludeSourceInfo(excludeSourceInfo bool) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.excludeSourceInfo = excludeSourceInfo
	}
}

func WithExcludeImports(excludeImports bool) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.excludeImports = excludeImports
	}
}

func WithAsFileDescriptorSet(asFileDescriptorSet bool) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.asFileDescriptorSet = asFileDescriptorSet
	}
}
