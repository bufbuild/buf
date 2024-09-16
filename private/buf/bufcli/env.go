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

package bufcli

import (
	"context"
	"net/http"

	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/httpauth"
)

const (
	inputHTTPSUsernameEnvKey      = "BUF_INPUT_HTTPS_USERNAME"
	inputHTTPSPasswordEnvKey      = "BUF_INPUT_HTTPS_PASSWORD"
	inputSSHKeyFileEnvKey         = "BUF_INPUT_SSH_KEY_FILE"
	inputSSHKnownHostsFilesEnvKey = "BUF_INPUT_SSH_KNOWN_HOSTS_FILES"

	alphaSuppressWarningsEnvKey = "BUF_ALPHA_SUPPRESS_WARNINGS"
	betaSuppressWarningsEnvKey  = "BUF_BETA_SUPPRESS_WARNINGS"

	// This is actually much slower with how it is currently implemented if you use --path.
	// Example: Build a repo with 1000 .proto files, but filter to a single path. As this is
	// implemented now, all 1000 .proto file are copied. You could get smarter with caching
	// at a per-file level.
	copyToInMemoryEnvKey = "BUF_BETA_COPY_FILES_TO_MEMORY"

	// This should only be used for testing. This is not part of Buf's API, and should
	// never be documented or part of Buf's contract.
	legacyFederationRegistryEnvKey = "BUF_TESTING_LEGACY_FEDERATION_REGISTRY"
	// This should only be used for testing. This is not part of Buf's API, and should
	// never be documented or part of Buf's contract.
	publicRegistryEnvKey = "BUF_TESTING_PUBLIC_REGISTRY"
)

var (
	// defaultHTTPClient is the client we use for HTTP requests.
	// Timeout should be set through context for calls to ImageConfigReader, not through http.Client
	defaultHTTPClient = &http.Client{}
	// defaultHTTPAuthenticator is the default authenticator
	// used for HTTP requests.
	defaultHTTPAuthenticator = httpauth.NewMultiAuthenticator(
		httpauth.NewNetrcAuthenticator(),
		// must keep this for legacy purposes
		httpauth.NewEnvAuthenticator(
			inputHTTPSPasswordEnvKey,
			inputHTTPSPasswordEnvKey,
		),
	)
	// defaultGitClonerOptions defines the default git clone options.
	defaultGitClonerOptions = git.ClonerOptions{
		HTTPSUsernameEnvKey:      inputHTTPSUsernameEnvKey,
		HTTPSPasswordEnvKey:      inputHTTPSPasswordEnvKey,
		SSHKeyFileEnvKey:         inputSSHKeyFileEnvKey,
		SSHKnownHostsFilesEnvKey: inputSSHKnownHostsFilesEnvKey,
	}
)

// WarnAlphaCommand prints a warning for a alpha command unless the alphaSuppressWarningsEnvKey
// environment variable is set.
func WarnAlphaCommand(_ context.Context, container appext.Container) {
	if container.Env(alphaSuppressWarningsEnvKey) == "" {
		container.Logger().Warn("This command is in alpha. It is hidden for a reason. This command is purely for development purposes, and may never even be promoted to beta, do not rely on this command's functionality. To suppress this warning, set " + alphaSuppressWarningsEnvKey + "=1")
	}
}

// WarnBetaCommand prints a warning for a beta command unless the betaSuppressWarningsEnvKey
// environment variable is set.
func WarnBetaCommand(_ context.Context, container appext.Container) {
	if container.Env(betaSuppressWarningsEnvKey) == "" {
		container.Logger().Warn("This command is in beta. It is unstable and likely to change. To suppress this warning, set " + betaSuppressWarningsEnvKey + "=1")
	}
}
