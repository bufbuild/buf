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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
	otelconnect "connectrpc.com/otelconnect"
	"github.com/bufbuild/buf/private/buf/bufapp"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufwire"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufapimodule"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufconnect"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimageutil"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulecache"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/buftransport"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/app/appname"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/httpauth"
	"github.com/bufbuild/buf/private/pkg/netrc"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/transport/http/httpclient"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"golang.org/x/term"
)

const (
	// Version is the CLI version of buf.
	Version = "1.29.0"

	inputHTTPSUsernameEnvKey      = "BUF_INPUT_HTTPS_USERNAME"
	inputHTTPSPasswordEnvKey      = "BUF_INPUT_HTTPS_PASSWORD"
	inputSSHKeyFileEnvKey         = "BUF_INPUT_SSH_KEY_FILE"
	inputSSHKnownHostsFilesEnvKey = "BUF_INPUT_SSH_KNOWN_HOSTS_FILES"

	alphaSuppressWarningsEnvKey = "BUF_ALPHA_SUPPRESS_WARNINGS"
	betaSuppressWarningsEnvKey  = "BUF_BETA_SUPPRESS_WARNINGS"

	// AlphaEnableWASMEnvKey is an env var to enable WASM local plugin execution
	AlphaEnableWASMEnvKey = "BUF_ALPHA_ENABLE_WASM"

	inputHashtagFlagName      = "__hashtag__"
	inputHashtagFlagShortName = "#"

	userPromptAttempts = 3

	publicVisibility  = "public"
	privateVisibility = "private"

	// WASMCompilationCacheDir compiled WASM plugin cache directory
	WASMCompilationCacheDir = "wasmplugin-bin"
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

	// AllCacheModuleRelDirPaths are all directory paths for all time concerning the module cache.
	//
	// These are normalized.
	// These are relative to container.CacheDirPath().
	//
	// This variable is used for clearing the cache.
	AllCacheModuleRelDirPaths = []string{
		v1beta1CacheModuleDataRelDirPath,
		v1beta1CacheModuleLockRelDirPath,
		v1CacheModuleDataRelDirPath,
		v1CacheModuleLockRelDirPath,
		v1CacheModuleSumRelDirPath,
		v2CacheModuleRelDirPath,
	}

	// ErrNotATTY is returned when an input io.Reader is not a TTY where it is expected.
	ErrNotATTY = errors.New("reader was not a TTY as expected")

	// v1CacheModuleDataRelDirPath is the relative path to the cache directory where module data
	// was stored in v1beta1.
	//
	// Normalized.
	v1beta1CacheModuleDataRelDirPath = "mod"

	// v1CacheModuleLockRelDirPath is the relative path to the cache directory where module lock files
	// were stored in v1beta1.
	//
	// Normalized.
	v1beta1CacheModuleLockRelDirPath = normalpath.Join("lock", "mod")

	// v1CacheModuleDataRelDirPath is the relative path to the cache directory where module data is stored.
	//
	// Normalized.
	// This is where the actual "clones" of the modules are located.
	v1CacheModuleDataRelDirPath = normalpath.Join("v1", "module", "data")
	// v1CacheModuleLockRelDirPath is the relative path to the cache directory where module lock files are stored.
	//
	// Normalized.
	// These lock files are used to make sure that multiple buf processes do not corrupt the cache.
	v1CacheModuleLockRelDirPath = normalpath.Join("v1", "module", "lock")
	// v1CacheModuleSumRelDirPath is the relative path to the cache directory where module digests are stored.
	//
	// Normalized.
	// These digests are used to make sure that the data written is actually what we expect, and if it is not,
	// we clear an entry from the cache, i.e. delete the relevant data directory.
	v1CacheModuleSumRelDirPath = normalpath.Join("v1", "module", "sum")
	// v2CacheModuleRelDirPath is the relative path to the cache directory for content addressable storage.
	//
	// Normalized.
	// This directory replaces the use of v1CacheModuleDataRelDirPath, v1CacheModuleLockRelDirPath, and
	// v1CacheModuleSumRelDirPath with a cache implementation using content addressable storage.
	v2CacheModuleRelDirPath = normalpath.Join("v2", "module")

	// allVisibiltyStrings are the possible options that a user can set the visibility flag with.
	allVisibiltyStrings = []string{
		publicVisibility,
		privateVisibility,
	}
)

// BindAsFileDescriptorSet binds the exclude-imports flag.
func BindAsFileDescriptorSet(flagSet *pflag.FlagSet, addr *bool, flagName string) {
	flagSet.BoolVar(
		addr,
		flagName,
		false,
		`Output as a google.protobuf.FileDescriptorSet instead of an image
Note that images are wire compatible with FileDescriptorSets, but this flag strips
the additional metadata added for Buf usage`,
	)
}

// BindExcludeImports binds the exclude-imports flag.
func BindExcludeImports(flagSet *pflag.FlagSet, addr *bool, flagName string) {
	flagSet.BoolVar(
		addr,
		flagName,
		false,
		"Exclude imports.",
	)
}

// BindExcludeSourceInfo binds the exclude-source-info flag.
func BindExcludeSourceInfo(flagSet *pflag.FlagSet, addr *bool, flagName string) {
	flagSet.BoolVar(
		addr,
		flagName,
		false,
		"Exclude source info",
	)
}

// BindPaths binds the paths flag.
func BindPaths(
	flagSet *pflag.FlagSet,
	pathsAddr *[]string,
	pathsFlagName string,
) {
	flagSet.StringSliceVar(
		pathsAddr,
		pathsFlagName,
		nil,
		`Limit to specific files or directories, e.g. "proto/a/a.proto", "proto/a"
If specified multiple times, the union is taken`,
	)
}

// BindInputHashtag binds the input hashtag flag.
//
// This needs to be added to any command that has the input as the first argument.
// This deals with the situation "buf build -#format=json" which results in
// a parse error from pflag.
func BindInputHashtag(flagSet *pflag.FlagSet, addr *string) {
	flagSet.StringVarP(
		addr,
		inputHashtagFlagName,
		inputHashtagFlagShortName,
		"",
		"",
	)
	_ = flagSet.MarkHidden(inputHashtagFlagName)
}

// BindExcludePaths binds the exclude-path flag.
func BindExcludePaths(
	flagSet *pflag.FlagSet,
	excludePathsAddr *[]string,
	excludePathsFlagName string,
) {
	flagSet.StringSliceVar(
		excludePathsAddr,
		excludePathsFlagName,
		nil,
		`Exclude specific files or directories, e.g. "proto/a/a.proto", "proto/a"
If specified multiple times, the union is taken`,
	)
}

// BindDisableSymlinks binds the disable-symlinks flag.
func BindDisableSymlinks(flagSet *pflag.FlagSet, addr *bool, flagName string) {
	flagSet.BoolVar(
		addr,
		flagName,
		false,
		`Do not follow symlinks when reading sources or configuration from the local filesystem
By default, symlinks are followed in this CLI, but never followed on the Buf Schema Registry`,
	)
}

// BindVisibility binds the visibility flag.
func BindVisibility(flagSet *pflag.FlagSet, addr *string, flagName string) {
	flagSet.StringVar(
		addr,
		flagName,
		"",
		fmt.Sprintf(`The repository's visibility setting. Must be one of %s`, stringutil.SliceToString(allVisibiltyStrings)),
	)
}

// BindCreateVisibility binds the create-visibility flag. Kept in this package
// so we can keep allVisibilityStrings private.
func BindCreateVisibility(flagSet *pflag.FlagSet, addr *string, flagName string, createFlagName string) {
	flagSet.StringVar(
		addr,
		flagName,
		"",
		fmt.Sprintf(`The repository's visibility setting, if created. Must be set if --%s is set. Must be one of %s`, createFlagName, stringutil.SliceToString(allVisibiltyStrings)),
	)
}

// GetInputLong gets the long command description for an input-based command.
func GetInputLong(inputArgDescription string) string {
	return fmt.Sprintf(
		`The first argument is %s.
The first argument must be one of format %s.
Defaults to "." if no argument is specified.`,
		inputArgDescription,
		buffetch.AllFormatsString,
	)
}

// GetSourceLong gets the long command description for an input-based command.
func GetSourceLong(inputArgDescription string) string {
	return fmt.Sprintf(
		`The first argument is %s.
The first argument must be one of format %s.
Defaults to "." if no argument is specified.`,
		inputArgDescription,
		buffetch.SourceFormatsString,
	)
}

// GetSourceDirLong gets the long command description for a directory-based command.
func GetSourceDirLong(inputArgDescription string) string {
	return fmt.Sprintf(
		`The first argument is %s.
The first argument must be one of format %s.
Defaults to "." if no argument is specified.`,
		inputArgDescription,
		buffetch.SourceDirFormatsString,
	)
}

// GetSourceOrModuleLong gets the long command description for an input-based command.
func GetSourceOrModuleLong(inputArgDescription string) string {
	return fmt.Sprintf(
		`The first argument is %s.
The first argument must be one of format %s.
Defaults to "." if no argument is specified.`,
		inputArgDescription,
		buffetch.SourceOrModuleFormatsString,
	)
}

// GetInputValue gets the first arg.
//
// Also parses the special input hashtag flag that deals with the situation "buf build -#format=json".
// The existence of 0 or 1 args should be handled by the Args field on Command.
func GetInputValue(
	container app.ArgContainer,
	inputHashtag string,
	defaultValue string,
) (string, error) {
	var arg string
	switch numArgs := container.NumArgs(); numArgs {
	case 0:
		if inputHashtag != "" {
			arg = "-#" + inputHashtag
		}
	case 1:
		arg = container.Arg(0)
		if arg == "" {
			return "", errors.New("first argument is present but empty")
		}
		// if arg is non-empty and inputHashtag is non-empty, this means two arguments were specified
		if inputHashtag != "" {
			return "", errors.New("only 1 argument allowed but 2 arguments specified")
		}
	default:
		return "", fmt.Errorf("only 1 argument allowed but %d arguments specified", numArgs)
	}
	if arg != "" {
		return arg, nil
	}
	return defaultValue, nil
}

// WarnAlphaCommand prints a warning for a alpha command unless the alphaSuppressWarningsEnvKey
// environment variable is set.
func WarnAlphaCommand(_ context.Context, container appflag.Container) {
	if container.Env(alphaSuppressWarningsEnvKey) == "" {
		container.Logger().Warn("This command is in alpha. It is hidden for a reason. This command is purely for development purposes, and may never even be promoted to beta, do not rely on this command's functionality. To suppress this warning, set " + alphaSuppressWarningsEnvKey + "=1")
	}
}

// WarnBetaCommand prints a warning for a beta command unless the betaSuppressWarningsEnvKey
// environment variable is set.
func WarnBetaCommand(_ context.Context, container appflag.Container) {
	if container.Env(betaSuppressWarningsEnvKey) == "" {
		container.Logger().Warn("This command is in beta. It is unstable and likely to change. To suppress this warning, set " + betaSuppressWarningsEnvKey + "=1")
	}
}

// NewStorageosProvider returns a new storageos.Provider based on the value of the disable-symlinks flag.
func NewStorageosProvider(disableSymlinks bool) storageos.Provider {
	if disableSymlinks {
		return storageos.NewProvider()
	}
	return storageos.NewProvider(storageos.ProviderWithSymlinks())
}

// NewWireImageConfigReader returns a new ImageConfigReader.
func NewWireImageConfigReader(
	container appflag.Container,
	storageosProvider storageos.Provider,
	runner command.Runner,
	clientConfig *connectclient.Config,
) (bufwire.ImageConfigReader, error) {
	logger := container.Logger()
	moduleResolver := bufapimodule.NewModuleResolver(
		logger,
		bufapimodule.NewRepositoryCommitServiceClientFactory(clientConfig),
	)
	moduleReader, err := NewModuleReaderAndCreateCacheDirs(container, clientConfig)
	if err != nil {
		return nil, err
	}
	return bufwire.NewImageConfigReader(
		logger,
		storageosProvider,
		NewFetchReader(logger, storageosProvider, runner, moduleResolver, moduleReader),
		bufmodulebuild.NewModuleBucketBuilder(),
		bufimagebuild.NewBuilder(logger, moduleReader),
	), nil
}

// NewWireModuleConfigReader returns a new ModuleConfigReader.
func NewWireModuleConfigReader(
	container appflag.Container,
	storageosProvider storageos.Provider,
	runner command.Runner,
	clientConfig *connectclient.Config,
) (bufwire.ModuleConfigReader, error) {
	logger := container.Logger()
	moduleResolver := bufapimodule.NewModuleResolver(
		logger,
		bufapimodule.NewRepositoryCommitServiceClientFactory(clientConfig),
	)
	moduleReader, err := NewModuleReaderAndCreateCacheDirs(container, clientConfig)
	if err != nil {
		return nil, err
	}
	return bufwire.NewModuleConfigReader(
		logger,
		storageosProvider,
		NewFetchReader(logger, storageosProvider, runner, moduleResolver, moduleReader),
		bufmodulebuild.NewModuleBucketBuilder(),
	), nil
}

// NewWireModuleConfigReaderForModuleReader returns a new ModuleConfigReader using
// the given ModuleReader.
func NewWireModuleConfigReaderForModuleReader(
	container appflag.Container,
	storageosProvider storageos.Provider,
	runner command.Runner,
	clientConfig *connectclient.Config,
	moduleReader bufmodule.ModuleReader,
) (bufwire.ModuleConfigReader, error) {
	logger := container.Logger()
	moduleResolver := bufapimodule.NewModuleResolver(
		logger,
		bufapimodule.NewRepositoryCommitServiceClientFactory(clientConfig),
	)
	return bufwire.NewModuleConfigReader(
		logger,
		storageosProvider,
		NewFetchReader(logger, storageosProvider, runner, moduleResolver, moduleReader),
		bufmodulebuild.NewModuleBucketBuilder(),
	), nil
}

// NewWireFileLister returns a new FileLister.
func NewWireFileLister(
	container appflag.Container,
	storageosProvider storageos.Provider,
	runner command.Runner,
	clientConfig *connectclient.Config,
) (bufwire.FileLister, error) {
	logger := container.Logger()
	moduleResolver := bufapimodule.NewModuleResolver(
		logger,
		bufapimodule.NewRepositoryCommitServiceClientFactory(clientConfig),
	)
	moduleReader, err := NewModuleReaderAndCreateCacheDirs(container, clientConfig)
	if err != nil {
		return nil, err
	}
	return bufwire.NewFileLister(
		logger,
		storageosProvider,
		NewFetchReader(logger, storageosProvider, runner, moduleResolver, moduleReader),
		bufmodulebuild.NewModuleBucketBuilder(),
		bufimagebuild.NewBuilder(logger, moduleReader),
	), nil
}

// NewWireImageReader returns a new ImageReader.
func NewWireImageReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	runner command.Runner,
) bufwire.ImageReader {
	return bufwire.NewImageReader(
		logger,
		newFetchMessageReader(logger, storageosProvider, runner),
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

// NewWireProtoEncodingReader returns a new ProtoEncodingReader.
func NewWireProtoEncodingReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	runner command.Runner,
) bufwire.ProtoEncodingReader {
	return bufwire.NewProtoEncodingReader(
		logger,
		newFetchMessageReader(logger, storageosProvider, runner),
	)
}

// NewWireProtoEncodingWriter returns a new ProtoEncodingWriter.
func NewWireProtoEncodingWriter(
	logger *zap.Logger,
) bufwire.ProtoEncodingWriter {
	return bufwire.NewProtoEncodingWriter(
		logger,
		buffetch.NewWriter(
			logger,
		),
	)
}

// NewModuleReaderAndCreateCacheDirs returns a new ModuleReader while creating the
// required cache directories.
func NewModuleReaderAndCreateCacheDirs(
	container appflag.Container,
	clientConfig *connectclient.Config,
) (bufmodule.ModuleReader, error) {
	return newModuleReaderAndCreateCacheDirs(container, clientConfig)
}

func newModuleReaderAndCreateCacheDirs(
	container appflag.Container,
	clientConfig *connectclient.Config,
) (bufmodule.ModuleReader, error) {
	cacheModuleDirPathV2 := normalpath.Join(container.CacheDirPath(), v2CacheModuleRelDirPath)
	if err := checkExistingCacheDirs(container.CacheDirPath(), cacheModuleDirPathV2); err != nil {
		return nil, err
	}
	if err := createCacheDirs(cacheModuleDirPathV2); err != nil {
		return nil, err
	}
	delegateReader := bufapimodule.NewModuleReader(
		container.Logger(),
		bufapimodule.NewDownloadServiceClientFactory(clientConfig),
		bufapimodule.ModuleReaderWithDeprecationWarning(
			bufapimodule.NewRepositoryServiceClientFactory(clientConfig),
		),
	)
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	var moduleReader bufmodule.ModuleReader
	casModuleBucket, err := storageosProvider.NewReadWriteBucket(cacheModuleDirPathV2)
	if err != nil {
		return nil, err
	}
	moduleReader = bufmodulecache.NewModuleReader(
		container.Logger(),
		container.VerbosePrinter(),
		casModuleBucket,
		delegateReader,
	)
	return moduleReader, nil
}

// NewConfig creates a new Config.
func NewConfig(container appflag.Container) (*bufapp.Config, error) {
	externalConfig := bufapp.ExternalConfig{}
	if err := appname.ReadConfig(container, &externalConfig); err != nil {
		return nil, err
	}
	return bufapp.NewConfig(container, externalConfig)
}

// Returns a registry provider with the given options applied in addition to default ones for all providers
func newConnectClientConfigWithOptions(container appflag.Container, opts ...connectclient.ConfigOption) (*connectclient.Config, error) {
	config, err := NewConfig(container)
	if err != nil {
		return nil, err
	}
	otelconnectInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		return nil, err
	}
	client := httpclient.NewClient(config.TLS)
	options := []connectclient.ConfigOption{
		connectclient.WithAddressMapper(func(address string) string {
			if config.TLS == nil {
				return buftransport.PrependHTTP(address)
			}
			return buftransport.PrependHTTPS(address)
		}),
		connectclient.WithInterceptors([]connect.Interceptor{
			bufconnect.NewSetCLIVersionInterceptor(Version),
			bufconnect.NewCLIWarningInterceptor(container),
			otelconnectInterceptor,
		}),
	}
	options = append(options, opts...)

	return connectclient.NewConfig(client, options...), nil
}

// NewConnectClientConfig creates a new connect.ClientConfig which uses a token reader to look
// up the token in the container or in netrc based on the address of each individual client.
// It is then set in the header of all outgoing requests from clients created using this config.
func NewConnectClientConfig(container appflag.Container) (*connectclient.Config, error) {
	envTokenProvider, err := bufconnect.NewTokenProviderFromContainer(container)
	if err != nil {
		return nil, err
	}
	netrcTokenProvider := bufconnect.NewNetrcTokenProvider(container, netrc.GetMachineForName)
	return newConnectClientConfigWithOptions(
		container,
		connectclient.WithAuthInterceptorProvider(
			bufconnect.NewAuthorizationInterceptorProvider(envTokenProvider, netrcTokenProvider),
		),
	)
}

// NewConnectClientConfigWithToken creates a new connect.ClientConfig with a given token. The provided token is
// set in the header of all outgoing requests from this provider
func NewConnectClientConfigWithToken(container appflag.Container, token string) (*connectclient.Config, error) {
	tokenProvider, err := bufconnect.NewTokenProviderFromString(token)
	if err != nil {
		return nil, err
	}
	return newConnectClientConfigWithOptions(
		container,
		connectclient.WithAuthInterceptorProvider(
			bufconnect.NewAuthorizationInterceptorProvider(tokenProvider),
		),
	)
}

// NewFetchReader creates a new buffetch.Reader with the default HTTP client
// and git cloner.
func NewFetchReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	runner command.Runner,
	moduleResolver bufmodule.ModuleResolver,
	moduleReader bufmodule.ModuleReader,
) buffetch.Reader {
	return buffetch.NewReader(
		logger,
		storageosProvider,
		defaultHTTPClient,
		defaultHTTPAuthenticator,
		git.NewCloner(logger, storageosProvider, runner, defaultGitClonerOptions),
		moduleResolver,
		moduleReader,
	)
}

// PromptUserForDelete is used to receive user confirmation that a specific
// entity should be deleted. If the user's answer does not match the expected
// answer, an error is returned.
// ErrNotATTY is returned if the input containers Stdin is not a terminal.
func PromptUserForDelete(container app.Container, entityType string, expectedAnswer string) error {
	confirmation, err := PromptUser(
		container,
		fmt.Sprintf(
			"Please confirm that you want to DELETE this %s by entering its name (%s) again."+
				"\nWARNING: This action is NOT reversible!\n",
			entityType,
			expectedAnswer,
		),
	)
	if err != nil {
		if errors.Is(err, ErrNotATTY) {
			return errors.New("cannot perform an interactive delete from a non-TTY device")
		}
		return err
	}
	if confirmation != expectedAnswer {
		return fmt.Errorf(
			"expected %q, but received %q",
			expectedAnswer,
			confirmation,
		)
	}
	return nil
}

// PromptUser reads a line from Stdin, prompting the user with the prompt first.
// The prompt is repeatedly shown until the user provides a non-empty response.
// ErrNotATTY is returned if the input containers Stdin is not a terminal.
func PromptUser(container app.Container, prompt string) (string, error) {
	return promptUser(container, prompt, false)
}

// PromptUserForPassword reads a line from Stdin, prompting the user with the prompt first.
// The prompt is repeatedly shown until the user provides a non-empty response.
// ErrNotATTY is returned if the input containers Stdin is not a terminal.
func PromptUserForPassword(container app.Container, prompt string) (string, error) {
	return promptUser(container, prompt, true)
}

// BucketAndConfigForSource returns a bucket and config. The bucket contains
// just the files that constitute a module. It also checks if config
// exists and defines a module identity, returning ErrNoConfigFile and
// ErrNoModuleName respectfully.
//
// Workspaces are disabled when fetching the source.
func BucketAndConfigForSource(
	ctx context.Context,
	logger *zap.Logger,
	container app.EnvStdinContainer,
	storageosProvider storageos.Provider,
	runner command.Runner,
	source string,
) (storage.ReadBucketCloser, *bufconfig.Config, error) {
	sourceRef, err := buffetch.NewSourceRefParser(
		logger,
	).GetSourceRef(
		ctx,
		source,
	)
	if err != nil {
		return nil, nil, err
	}
	sourceBucket, err := newFetchSourceReader(
		logger,
		storageosProvider,
		runner,
	).GetSourceBucket(
		ctx,
		container,
		sourceRef,
		buffetch.GetSourceBucketWithWorkspacesDisabled(),
	)
	if err != nil {
		return nil, nil, err
	}
	existingConfigFilePath, err := bufconfig.ExistingConfigFilePath(ctx, sourceBucket)
	if err != nil {
		return nil, nil, NewInternalError(err)
	}
	if existingConfigFilePath == "" {
		return nil, nil, ErrNoConfigFile
	}
	// TODO: This should just read a lock file
	sourceConfig, err := bufconfig.GetConfigForBucket(
		ctx,
		sourceBucket,
	)
	if err != nil {
		return nil, nil, err
	}
	if sourceConfig.ModuleIdentity == nil {
		return nil, nil, ErrNoModuleName
	}

	return sourceBucket, sourceConfig, nil
}

// NewImageForSource resolves a single bufimage.Image from the user-provided source with the build options.
func NewImageForSource(
	ctx context.Context,
	container appflag.Container,
	source string,
	errorFormat string,
	disableSymlinks bool,
	configOverride string,
	externalDirOrFilePaths []string,
	externalExcludeDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) (bufimage.Image, error) {
	ref, err := buffetch.NewRefParser(container.Logger()).GetRef(ctx, source)
	if err != nil {
		return nil, err
	}
	storageosProvider := NewStorageosProvider(disableSymlinks)
	runner := command.NewRunner()
	clientConfig, err := NewConnectClientConfig(container)
	if err != nil {
		return nil, err
	}
	imageConfigReader, err := NewWireImageConfigReader(
		container,
		storageosProvider,
		runner,
		clientConfig,
	)
	if err != nil {
		return nil, err
	}
	imageConfigs, fileAnnotations, err := imageConfigReader.GetImageConfigs(
		ctx,
		container,
		ref,
		configOverride,
		externalDirOrFilePaths,
		externalExcludeDirOrFilePaths,
		externalDirOrFilePathsAllowNotExist,
		excludeSourceCodeInfo,
	)
	if err != nil {
		return nil, err
	}
	if len(fileAnnotations) > 0 {
		// stderr since we do output to stdout potentially
		if err := bufanalysis.PrintFileAnnotations(
			container.Stderr(),
			fileAnnotations,
			errorFormat,
		); err != nil {
			return nil, err
		}
		return nil, ErrFileAnnotation
	}
	images := make([]bufimage.Image, 0, len(imageConfigs))
	for _, imageConfig := range imageConfigs {
		images = append(images, imageConfig.Image())
	}
	return bufimage.MergeImages(images...)
}

// WellKnownTypeImage returns the image for the well known type (google.protobuf.Duration for example).
func WellKnownTypeImage(ctx context.Context, logger *zap.Logger, wellKnownType string) (bufimage.Image, error) {
	sourceConfig, err := bufconfig.GetConfigForBucket(
		ctx,
		storage.NopReadBucketCloser(datawkt.ReadBucket),
	)
	if err != nil {
		return nil, err
	}

	module, err := bufmodulebuild.NewModuleBucketBuilder().BuildForBucket(
		ctx,
		datawkt.ReadBucket,
		sourceConfig.Build,
	)
	if err != nil {
		return nil, err
	}
	image, _, err := bufimagebuild.NewBuilder(logger, bufmodule.NewNopModuleReader()).Build(ctx, module)
	if err != nil {
		return nil, err
	}
	return bufimageutil.ImageFilteredByTypes(image, wellKnownType)
}

// VisibilityFlagToVisibility parses the given string as a registryv1alpha1.Visibility.
func VisibilityFlagToVisibility(visibility string) (registryv1alpha1.Visibility, error) {
	switch visibility {
	case publicVisibility:
		return registryv1alpha1.Visibility_VISIBILITY_PUBLIC, nil
	case privateVisibility:
		return registryv1alpha1.Visibility_VISIBILITY_PRIVATE, nil
	default:
		return 0, fmt.Errorf("invalid visibility: %s, expected one of %s", visibility, stringutil.SliceToString(allVisibiltyStrings))
	}
}

// VisibilityFlagToVisibilityAllowUnspecified parses the given string as a registryv1alpha1.Visibility,
// where an empty string will be parsed as unspecified
func VisibilityFlagToVisibilityAllowUnspecified(visibility string) (registryv1alpha1.Visibility, error) {
	switch visibility {
	case publicVisibility:
		return registryv1alpha1.Visibility_VISIBILITY_PUBLIC, nil
	case privateVisibility:
		return registryv1alpha1.Visibility_VISIBILITY_PRIVATE, nil
	case "":
		return registryv1alpha1.Visibility_VISIBILITY_UNSPECIFIED, nil
	default:
		return 0, fmt.Errorf("invalid visibility: %s", visibility)
	}
}

// IsAlphaWASMEnabled returns an BUF_ALPHA_ENABLE_WASM is set to true.
func IsAlphaWASMEnabled(container app.EnvContainer) (bool, error) {
	return app.EnvBool(container, AlphaEnableWASMEnvKey, false)
}

// ValidateErrorFormatFlag validates the error format flag for all commands but lint.
func ValidateErrorFormatFlag(errorFormatString string, errorFormatFlagName string) error {
	return validateErrorFormatFlag(bufanalysis.AllFormatStrings, errorFormatString, errorFormatFlagName)
}

// ValidateErrorFormatFlagLint validates the error format flag for lint.
func ValidateErrorFormatFlagLint(errorFormatString string, errorFormatFlagName string) error {
	return validateErrorFormatFlag(buflint.AllFormatStrings, errorFormatString, errorFormatFlagName)
}

// PackageVersionShortDescription returns the long description for the <package>-version command.
func PackageVersionShortDescription(name string) string {
	return fmt.Sprintf("Resolve module and %s plugin reference to a specific Generated SDK version", name)
}

// PackageVersionLongDescription returns the long description for the <package>-version command.
func PackageVersionLongDescription(registryName, commandName, examplePlugin string) string {
	return fmt.Sprintf(`This command returns the version of the %s Generated SDK to be used with the %s registry.
Examples:

Get the version of the eliza module and the %s plugin for use with %s.
    $ buf alpha sdk %s --module=buf.build/connectrpc/eliza --plugin=%s
        v1.7.0-20230609151053-e682db0d9918.1

Use a specific module version and plugin version.
    $ buf alpha sdk %s --module=buf.build/connectrpc/eliza:233fca715f49425581ec0a1b660be886 --plugin=%s:v1.0.0
        v1.0.0-20230609151053-e682db0d9918.1
`, registryName, registryName, examplePlugin, registryName, commandName, examplePlugin, commandName, examplePlugin)
}

// SelectReferenceForRemote receives a list of module references and selects one for remote
// operations. In most cases, all references will have the same remote, which will result in the
// first reference being selected. In cases in which there is a mix of remotes, the first reference
// with a remote different than "buf.build" will be selected. This func is useful for targeting
// single-tenant BSR addresses.
func SelectReferenceForRemote(references []bufmoduleref.ModuleReference) bufmoduleref.ModuleReference {
	if len(references) == 0 {
		return nil
	}
	for _, ref := range references {
		if ref == nil {
			continue
		}
		if ref.Remote() != bufconnect.DefaultRemote {
			return ref
		}
	}
	return references[0]
}

func validateErrorFormatFlag(validFormatStrings []string, errorFormatString string, errorFormatFlagName string) error {
	for _, formatString := range validFormatStrings {
		if errorFormatString == formatString {
			return nil
		}
	}
	return appcmd.NewInvalidArgumentErrorf("--%s: invalid format: %q", errorFormatFlagName, errorFormatString)
}

// promptUser reads a line from Stdin, prompting the user with the prompt first.
// The prompt is repeatedly shown until the user provides a non-empty response.
// ErrNotATTY is returned if the input containers Stdin is not a terminal.
func promptUser(container app.Container, prompt string, isPassword bool) (string, error) {
	file, ok := container.Stdin().(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return "", ErrNotATTY
	}
	var attempts int
	for attempts < userPromptAttempts {
		attempts++
		if _, err := fmt.Fprint(
			container.Stdout(),
			prompt,
		); err != nil {
			return "", NewInternalError(err)
		}
		var value string
		if isPassword {
			data, err := term.ReadPassword(int(file.Fd()))
			if err != nil {
				// If the user submitted an EOF (e.g. via ^D) then we
				// should not treat it as an internal error; returning
				// the error directly makes it more clear as to
				// why the command failed.
				if errors.Is(err, io.EOF) {
					return "", err
				}
				return "", NewInternalError(err)
			}
			value = string(data)
		} else {
			scanner := bufio.NewScanner(container.Stdin())
			if !scanner.Scan() {
				// scanner.Err() returns nil on EOF.
				if err := scanner.Err(); err != nil {
					return "", NewInternalError(err)
				}
				return "", io.EOF
			}
			value = scanner.Text()
			if err := scanner.Err(); err != nil {
				return "", NewInternalError(err)
			}
		}
		if len(strings.TrimSpace(value)) != 0 {
			// We want to preserve spaces in user input, so we only apply
			// strings.TrimSpace to verify an answer was provided.
			return value, nil
		}
		if attempts < userPromptAttempts {
			// We only want to ask the user to try again if they actually
			// have another attempt.
			if _, err := fmt.Fprintln(
				container.Stdout(),
				"No answer was provided. Please try again.",
			); err != nil {
				return "", NewInternalError(err)
			}
		}
	}
	return "", NewTooManyEmptyAnswersError(userPromptAttempts)
}

// newFetchSourceReader creates a new buffetch.SourceReader with the default HTTP client
// and git cloner.
func newFetchSourceReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	runner command.Runner,
) buffetch.SourceReader {
	return buffetch.NewSourceReader(
		logger,
		storageosProvider,
		defaultHTTPClient,
		defaultHTTPAuthenticator,
		git.NewCloner(logger, storageosProvider, runner, defaultGitClonerOptions),
	)
}

// newFetchMessageReader creates a new buffetch.MessageReader with the default HTTP client
// and git cloner.
func newFetchMessageReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	runner command.Runner,
) buffetch.MessageReader {
	return buffetch.NewMessageReader(
		logger,
		storageosProvider,
		defaultHTTPClient,
		defaultHTTPAuthenticator,
		git.NewCloner(logger, storageosProvider, runner, defaultGitClonerOptions),
	)
}

func checkExistingCacheDirs(baseCacheDirPath string, dirPaths ...string) error {
	dirPathsToCheck := make([]string, 0, len(dirPaths)+1)
	// Check base cache directory in addition to subdirectories
	dirPathsToCheck = append(dirPathsToCheck, baseCacheDirPath)
	dirPathsToCheck = append(dirPathsToCheck, dirPaths...)
	for _, dirPath := range dirPathsToCheck {
		dirPath = normalpath.Unnormalize(dirPath)
		// OK to use os.Stat instead of os.LStat here as this is CLI-only
		fileInfo, err := os.Stat(dirPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !fileInfo.IsDir() {
			return fmt.Errorf("Expected %q to be a directory. This is used for buf's cache. You can override the base cache directory %q by setting the $BUF_CACHE_DIR environment variable.", dirPath, baseCacheDirPath)
		}
		if fileInfo.Mode().Perm()&0700 != 0700 {
			return fmt.Errorf("Expected %q to be a writeable directory. This is used for buf's cache. You can override the base cache directory %q by setting the $BUF_CACHE_DIR environment variable.", dirPath, baseCacheDirPath)
		}
	}
	return nil
}

func createCacheDirs(dirPaths ...string) error {
	for _, dirPath := range dirPaths {
		// os.MkdirAll does nothing if the directory already exists
		if err := os.MkdirAll(normalpath.Unnormalize(dirPath), 0755); err != nil {
			return err
		}
	}
	return nil
}
