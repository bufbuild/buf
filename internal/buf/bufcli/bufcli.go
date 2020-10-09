// Copyright 2020 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/bufwire"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/git"
	"github.com/bufbuild/buf/internal/pkg/httpauth"
	"go.uber.org/zap"
)

// Constants used by the buf CLI
const (
	inputHTTPSUsernameEnvKey      = "BUF_INPUT_HTTPS_USERNAME"
	inputHTTPSPasswordEnvKey      = "BUF_INPUT_HTTPS_PASSWORD"
	inputSSHKeyFileEnvKey         = "BUF_INPUT_SSH_KEY_FILE"
	inputSSHKnownHostsFilesEnvKey = "BUF_INPUT_SSH_KNOWN_HOSTS_FILES"
)

var (
	// defaultHTTPClient is the client we use for HTTP requests.
	// Timeout should be set through context for calls to EnvReader, not through http.Client
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

// NewFetchReader creates a new buffetch.Reader with the default HTTP client
// and git cloner.
func NewFetchReader(
	logger *zap.Logger,
	moduleResolver bufmodule.ModuleResolver,
	moduleReader bufmodule.ModuleReader,
) buffetch.Reader {
	return buffetch.NewReader(
		logger,
		defaultHTTPClient,
		defaultHTTPAuthenticator,
		git.NewCloner(logger, defaultGitClonerOptions),
		moduleResolver,
		moduleReader,
	)
}

// NewFetchSourceReader creates a new buffetch.SourceReader with the default HTTP client
// and git cloner.
func NewFetchSourceReader(logger *zap.Logger) buffetch.SourceReader {
	return buffetch.NewSourceReader(
		logger,
		defaultHTTPClient,
		defaultHTTPAuthenticator,
		git.NewCloner(logger, defaultGitClonerOptions),
	)
}

// NewFetchImageReader creates a new buffetch.ImageReader with the default HTTP client
// and git cloner.
func NewFetchImageReader(logger *zap.Logger) buffetch.ImageReader {
	return buffetch.NewImageReader(
		logger,
		defaultHTTPClient,
		defaultHTTPAuthenticator,
		git.NewCloner(logger, defaultGitClonerOptions),
	)
}

// NewWireEnvReader returns a new EnvReader.
func NewWireEnvReader(
	logger *zap.Logger,
	configOverrideFlagName string,
	configProvider bufconfig.Provider,
	moduleResolver bufmodule.ModuleResolver,
	moduleReader bufmodule.ModuleReader,
) bufwire.EnvReader {
	return bufwire.NewEnvReader(
		logger,
		NewFetchReader(logger, moduleResolver, moduleReader),
		configProvider,
		bufmodulebuild.NewModuleBucketBuilder(logger),
		bufmodulebuild.NewModuleFileSetBuilder(logger, moduleReader),
		bufimagebuild.NewBuilder(logger),
		configOverrideFlagName,
	)
}

// NewWireFileLister returns a new FileLister.
func NewWireFileLister(
	logger *zap.Logger,
	configOverrideFlagName string,
	configProvider bufconfig.Provider,
	moduleResolver bufmodule.ModuleResolver,
	moduleReader bufmodule.ModuleReader,
) bufwire.FileLister {
	return bufwire.NewFileLister(
		logger,
		NewFetchReader(logger, moduleResolver, moduleReader),
		configProvider,
		bufmodulebuild.NewModuleBucketBuilder(logger),
		bufimagebuild.NewBuilder(logger),
		configOverrideFlagName,
	)
}

// NewWireImageReader returns a new ImageReader.
func NewWireImageReader(
	logger *zap.Logger,
) bufwire.ImageReader {
	return bufwire.NewImageReader(
		logger,
		NewFetchImageReader(logger),
	)
}

// NewWireImageWriter returns a new ImageWriter.
func NewWireImageWriter(
	logger *zap.Logger,
) bufwire.ImageWriter {
	return bufwire.NewImageWriter(
		logger,
		buffetch.NewWriter(
			logger,
		),
	)
}

// NewWireConfigReader returns a new ConfigReader.
func NewWireConfigReader(
	logger *zap.Logger,
	configOverrideFlagName string,
	configProvider bufconfig.Provider,
) bufwire.ConfigReader {
	return bufwire.NewConfigReader(
		logger,
		configProvider,
		configOverrideFlagName,
	)
}

// WarnBeta warns that the command is beta.
func WarnBeta(logger *zap.Logger) {
	logger.Warn(`This command has been released for early evaluation only and is experimental. It is not ready for production, and is likely to to have significant changes.`)
}

// ModuleResolverReaderProvider provides ModuleResolvers and ModuleReaders.
type ModuleResolverReaderProvider interface {
	GetModuleReader(context.Context, appflag.Container) (bufmodule.ModuleReader, error)
	GetModuleResolver(context.Context, appflag.Container) (bufmodule.ModuleResolver, error)
}

// NopModuleResolverReaderProvider is a no-op ModuleResolverReaderProvider.
type NopModuleResolverReaderProvider struct{}

// GetModuleReader returns a no-op module reader.
func (NopModuleResolverReaderProvider) GetModuleReader(_ context.Context, _ appflag.Container) (bufmodule.ModuleReader, error) {
	return bufmodule.NewNopModuleReader(), nil
}

// GetModuleResolver returns a no-op module resolver.
func (NopModuleResolverReaderProvider) GetModuleResolver(_ context.Context, _ appflag.Container) (bufmodule.ModuleResolver, error) {
	return bufmodule.NewNopModuleResolver(), nil
}
