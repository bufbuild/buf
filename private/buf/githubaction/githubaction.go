// Copyright 2020-2022 Buf Technologies, Inc.
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

package githubaction

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app"
)

const (
	githubShaEnvKey        = "GITHUB_SHA"
	githubRepositoryEnvKey = "GITHUB_REPOSITORY"
	githubRefNameEnvKey    = "GITHUB_REF_NAME"
	githubRefTypeEnvKey    = "GITHUB_REF_TYPE"
	githubAPIURLEnvKey     = "GITHUB_API_URL"

	githubTokenEnvKey = "GITHUB_TOKEN"
	tokenEnvKey       = "BUF_TOKEN"
	bufInputEnvKey    = "BUF_INPUT"
	bufTrackEnvKey    = "BUF_TRACK"

	inputActionInputName       = "input"
	trackActionInputName       = "track"
	bufTokenActionInputName    = "buf_token"
	githubTokenActionInputName = "github_token"
)

func Push(
	ctx context.Context,
	container app.EnvContainer,
	registryProvider registryv1alpha1apiclient.Provider,
	moduleIdentity bufmoduleref.ModuleIdentity,
	module bufmodule.Module,
	protoModule *modulev1alpha1.Module,
	bufVersion string,
) error {
	// Exit early if this isn't a branch push.
	githubRefType, err := GetGithubRefTypeValue(container)
	if err != nil {
		return err
	}
	if githubRefType != "branch" {
		return nil
	}

	p, err := newPusher(
		ctx,
		container,
		registryProvider,
		moduleIdentity,
		module,
		protoModule,
		bufVersion,
	)
	if err != nil {
		return err
	}
	return p.push(ctx)
}

func GetInputValue(container app.EnvContainer) (string, error) {
	s := container.Env(bufInputEnvKey)
	if s == "" {
		return "", fmt.Errorf("inputs.%s is required", inputActionInputName)
	}
	return s, nil
}

func GetTrackValue(container app.EnvContainer) (string, error) {
	s := container.Env(bufTrackEnvKey)
	if s == "" {
		return "", fmt.Errorf("inputs.%s is required", trackActionInputName)
	}
	return s, nil
}

func GetBufTokenValue(container app.EnvContainer) (string, error) {
	s := container.Env(tokenEnvKey)
	if s == "" {
		return "", fmt.Errorf("inputs.%s is required", bufTokenActionInputName)
	}
	return s, nil
}

func GetGithubTokenValue(container app.EnvContainer) (string, error) {
	s := container.Env(githubTokenEnvKey)
	if s == "" {
		return "", fmt.Errorf("inputs.%s is required", githubTokenActionInputName)
	}
	return s, nil
}

func GetGithubShaValue(container app.EnvContainer) (string, error) {
	s := container.Env(githubShaEnvKey)
	if s == "" {
		return "", fmt.Errorf("environment variable %s is required", githubShaEnvKey)
	}
	return s, nil
}

func GetGithubRepositoryValue(container app.EnvContainer) (string, error) {
	s := container.Env(githubRepositoryEnvKey)
	if s == "" {
		return "", fmt.Errorf("environment variable %s is required", githubRepositoryEnvKey)
	}
	return s, nil
}

func GetGithubRefNameValue(container app.EnvContainer) (string, error) {
	s := container.Env(githubRefNameEnvKey)
	if s == "" {
		return "", fmt.Errorf("environment variable %s is required", githubRefNameEnvKey)
	}
	return s, nil
}

func GetGithubRefTypeValue(container app.EnvContainer) (string, error) {
	s := container.Env(githubRefTypeEnvKey)
	if s == "" {
		return "", fmt.Errorf("environment variable %s is required", githubRefTypeEnvKey)
	}
	return s, nil
}

func GetGithubAPIURLValue(container app.EnvContainer) (string, error) {
	s := container.Env(githubAPIURLEnvKey)
	if s == "" {
		return "", fmt.Errorf("environment variable %s is required", githubAPIURLEnvKey)
	}
	return s, nil
}
