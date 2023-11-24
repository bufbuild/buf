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
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/applog"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/httpauth"
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
	) (bufimage.Image, []bufanalysis.FileAnnotation, error)
	PutImage(
		ctx context.Context,
		container app.EnvStdoutContainer,
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
) Controller {
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
