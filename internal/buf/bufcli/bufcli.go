// Copyright 2020-2021 Buf Technologies, Inc.
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
	"net/http"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufapiclient"
	"github.com/bufbuild/buf/internal/buf/bufapp"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/buftransport"
	"github.com/bufbuild/buf/internal/buf/bufwire"
	"github.com/bufbuild/buf/internal/buf/bufwork"
	"github.com/bufbuild/buf/internal/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/app/appname"
	"github.com/bufbuild/buf/internal/pkg/git"
	"github.com/bufbuild/buf/internal/pkg/httpauth"
	"github.com/bufbuild/buf/internal/pkg/netrc"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/rpc"
	"github.com/bufbuild/buf/internal/pkg/rpc/rpcauth"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

const (
	// Version is the version of buf.
	Version = "0.45.0-dev"
	// VersionHeaderName is the name of the header carrying the bufcli version.
	VersionHeaderName = "buf-version"

	// FlagDeprecationMessageSuffix is the suffix for flag deprecation messages.
	FlagDeprecationMessageSuffix = `
We recommend migrating, however this flag continues to work.
See https://docs.buf.build/faq for more details.`

	inputHTTPSUsernameEnvKey      = "BUF_INPUT_HTTPS_USERNAME"
	inputHTTPSPasswordEnvKey      = "BUF_INPUT_HTTPS_PASSWORD"
	inputSSHKeyFileEnvKey         = "BUF_INPUT_SSH_KEY_FILE"
	inputSSHKnownHostsFilesEnvKey = "BUF_INPUT_SSH_KNOWN_HOSTS_FILES"

	inputHashtagFlagName      = "__hashtag__"
	inputHashtagFlagShortName = "#"

	userPromptAttempts = 3
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
	}

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
)

// BindAsFileDescriptorSet binds the exclude-imports flag.
func BindAsFileDescriptorSet(flagSet *pflag.FlagSet, addr *bool, flagName string) {
	flagSet.BoolVar(
		addr,
		flagName,
		false,
		`Output as a google.protobuf.FileDescriptorSet instead of an image.
Note that images are wire-compatible with FileDescriptorSets, however this flag will strip
the additional metadata added for Buf usage.`,
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
		"Exclude source info.",
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
		`Limit to specific files or directories, for example "proto/a/a.proto" or "proto/a".
If specified multiple times, the union will be taken.`,
	)
}

// BindPathAndDeprecatedFiles binds the paths flag and the deprecated files flag.
func BindPathsAndDeprecatedFiles(
	flagSet *pflag.FlagSet,
	pathsAddr *[]string,
	pathsFlagName string,
	filesAddr *[]string,
	filesFlagName string,
) {
	BindPaths(flagSet, pathsAddr, pathsFlagName)
	flagSet.StringSliceVar(
		filesAddr,
		filesFlagName,
		nil,
		`Limit to specific files.
If specified multiple times, the union will be taken.`,
	)
	_ = flagSet.MarkHidden(filesFlagName)
	_ = flagSet.MarkDeprecated(
		filesFlagName,
		fmt.Sprintf("use --%s instead.%s", pathsFlagName, FlagDeprecationMessageSuffix),
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

// GetInputLong gets the long command description for an input-based command.
func GetInputLong(inputArgDescription string) string {
	return fmt.Sprintf(
		`The first argument is %s.
The first argument must be one of format %s.
If no argument is specified, defaults to ".".`,
		inputArgDescription,
		buffetch.AllFormatsString,
	)
}

// GetSourceOrModuleLong gets the long command description for an input-based command.
func GetSourceOrModuleLong(inputArgDescription string) string {
	return fmt.Sprintf(
		`The first argument is %s.
The first argument must be one of format %s.
If no argument is specified, defaults to ".".`,
		inputArgDescription,
		buffetch.SourceOrModuleFormatsString,
	)
}

// GetInputValue gets either the first arg or the deprecated flag, but not both.
//
// Also parses the special input hashtag flag that deals with the situation "buf build -#format=json".
// The existence of 0 or 1 args should be handled by the Args field on Command.
func GetInputValue(
	container appflag.Container,
	inputHashtag string,
	deprecatedFlag string,
	deprecatedFlagName string,
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
	if arg != "" && deprecatedFlag != "" {
		return "", fmt.Errorf("cannot specify both first argument and deprecated flag --%s", deprecatedFlagName)
	}
	if arg != "" {
		return arg, nil
	}
	if deprecatedFlag != "" {
		return deprecatedFlag, nil
	}
	return defaultValue, nil
}

// GetStringFlagOrDeprecatedFlag gets the flag, or the deprecated flag.
func GetStringFlagOrDeprecatedFlag(
	flag string,
	flagName string,
	deprecatedFlag string,
	deprecatedFlagName string,
) (string, error) {
	if flag != "" && deprecatedFlag != "" {
		return "", fmt.Errorf("cannot specify both --%s and --%s", flagName, deprecatedFlagName)
	}
	if flag != "" {
		return flag, nil
	}
	return deprecatedFlag, nil
}

// GetStringSliceFlagOrDeprecatedFlag gets the flag, or the deprecated flag.
func GetStringSliceFlagOrDeprecatedFlag(
	flag []string,
	flagName string,
	deprecatedFlag []string,
	deprecatedFlagName string,
) ([]string, error) {
	if len(flag) > 0 && len(deprecatedFlag) > 0 {
		return nil, fmt.Errorf("cannot specify both --%s and --%s", flagName, deprecatedFlagName)
	}
	if len(flag) > 0 {
		return flag, nil
	}
	return deprecatedFlag, nil
}

// NewFetchReader creates a new buffetch.Reader with the default HTTP client
// and git cloner.
func NewFetchReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	moduleResolver bufmodule.ModuleResolver,
	moduleReader bufmodule.ModuleReader,
) buffetch.Reader {
	return buffetch.NewReader(
		logger,
		storageosProvider,
		defaultHTTPClient,
		defaultHTTPAuthenticator,
		git.NewCloner(logger, storageosProvider, defaultGitClonerOptions),
		moduleResolver,
		moduleReader,
	)
}

// NewFetchSourceReader creates a new buffetch.SourceReader with the default HTTP client
// and git cloner.
func NewFetchSourceReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
) buffetch.SourceReader {
	return buffetch.NewSourceReader(
		logger,
		storageosProvider,
		defaultHTTPClient,
		defaultHTTPAuthenticator,
		git.NewCloner(logger, storageosProvider, defaultGitClonerOptions),
	)
}

// NewFetchImageReader creates a new buffetch.ImageReader with the default HTTP client
// and git cloner.
func NewFetchImageReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
) buffetch.ImageReader {
	return buffetch.NewImageReader(
		logger,
		storageosProvider,
		defaultHTTPClient,
		defaultHTTPAuthenticator,
		git.NewCloner(logger, storageosProvider, defaultGitClonerOptions),
	)
}

// NewWireImageConfigReader returns a new ImageConfigReader.
func NewWireImageConfigReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	configProvider bufconfig.Provider,
	workspaceConfigProvider bufwork.Provider,
	moduleResolver bufmodule.ModuleResolver,
	moduleReader bufmodule.ModuleReader,
) bufwire.ImageConfigReader {
	return bufwire.NewImageConfigReader(
		logger,
		storageosProvider,
		NewFetchReader(logger, storageosProvider, moduleResolver, moduleReader),
		configProvider,
		workspaceConfigProvider,
		bufmodulebuild.NewModuleBucketBuilder(logger),
		bufmodulebuild.NewModuleFileSetBuilder(logger, moduleReader),
		bufimagebuild.NewBuilder(logger),
	)
}

// NewWireModuleConfigReader returns a new ModuleConfigReader.
func NewWireModuleConfigReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	configProvider bufconfig.Provider,
	workspaceConfigProvider bufwork.Provider,
	moduleResolver bufmodule.ModuleResolver,
	moduleReader bufmodule.ModuleReader,
) bufwire.ModuleConfigReader {
	return bufwire.NewModuleConfigReader(
		logger,
		storageosProvider,
		NewFetchReader(logger, storageosProvider, moduleResolver, moduleReader),
		configProvider,
		workspaceConfigProvider,
		bufmodulebuild.NewModuleBucketBuilder(logger),
	)
}

// NewWireFileLister returns a new FileLister.
func NewWireFileLister(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	configProvider bufconfig.Provider,
	workspaceConfigProvider bufwork.Provider,
	moduleResolver bufmodule.ModuleResolver,
	moduleReader bufmodule.ModuleReader,
) bufwire.FileLister {
	return bufwire.NewFileLister(
		logger,
		NewFetchReader(logger, storageosProvider, moduleResolver, moduleReader),
		configProvider,
		workspaceConfigProvider,
		bufmodulebuild.NewModuleBucketBuilder(logger),
		bufimagebuild.NewBuilder(logger),
	)
}

// NewWireImageReader returns a new ImageReader.
func NewWireImageReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
) bufwire.ImageReader {
	return bufwire.NewImageReader(
		logger,
		NewFetchImageReader(logger, storageosProvider),
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

// NewConfig creates a new Config.
func NewConfig(container appflag.Container) (*bufapp.Config, error) {
	externalConfig := bufapp.ExternalConfig{}
	if err := appname.ReadConfig(container, &externalConfig); err != nil {
		return nil, err
	}
	return bufapp.NewConfig(container, externalConfig)
}

// NewRegistryProvider creates a new registryv1alpha1apiclient.Provider.
func NewRegistryProvider(ctx context.Context, container appflag.Container) (registryv1alpha1apiclient.Provider, error) {
	config, err := NewConfig(container)
	if err != nil {
		return nil, err
	}
	useGRPC, err := buftransport.UseGRPC(container)
	if err != nil {
		return nil, err
	}
	options := []bufapiclient.RegistryProviderOption{
		bufapiclient.RegistryProviderWithContextModifierProvider(NewContextModifierProvider(container)),
	}
	if buftransport.IsAPISubdomainEnabled(container) {
		options = append(options, bufapiclient.RegistryProviderWithAddressMapper(buftransport.PrependAPISubdomain))
	}
	if useGRPC {
		options = append(options, bufapiclient.RegistryProviderWithGRPC())
	}
	return bufapiclient.NewRegistryProvider(
		ctx,
		container.Logger(),
		config.TLS,
		options...,
	)
}

// NewContextModifierProvider returns a new context modifier provider for API providers.
//
// Public for use in other packages that provide API provider constructors.
func NewContextModifierProvider(
	container appflag.Container,
) func(string) (func(context.Context) context.Context, error) {
	return func(address string) (func(context.Context) context.Context, error) {
		machine, err := netrc.GetMachineForName(container, address)
		if err != nil {
			return nil, fmt.Errorf("failed to read server password from netrc: %w", err)
		}
		var password string
		if machine != nil {
			password = machine.Password()
		}
		return func(ctx context.Context) context.Context {
			return rpcauth.WithToken(
				rpc.WithOutgoingHeader(
					ctx,
					VersionHeaderName,
					Version,
				),
				password,
			)
		}, nil
	}
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

// NewRegistryModuleResolverReaderProvider returns a new registry-backed ModuleResolverReaderProvider.
func NewRegistryModuleResolverReaderProvider() ModuleResolverReaderProvider {
	return newRegistryModuleResolverReaderProvider()
}

// PromptUserForDelete is used to receieve user confirmation that a specific
// entity should be deleted. If the user's answer does not match the expected
// answer, an error is returned.
func PromptUserForDelete(container app.Container, entityType string, expectedAnswer string) error {
	confirmation, err := promptUser(
		container,
		fmt.Sprintf(
			"Please confirm that you want to DELETE this %s by entering its name again."+
				"\nWARNING: This action is NOT reversible!\n",
			entityType,
		),
	)
	if err != nil {
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

// ReadModule gets a module from a source ref.
// Workspaces are disabled for this function.
func ReadModule(
	ctx context.Context,
	container appflag.Container,
	storageosProvider storageos.Provider,
	source string,
) (bufmodule.Module, bufmodule.ModuleIdentity, error) {
	sourceRef, err := buffetch.NewSourceRefParser(
		container.Logger(),
	).GetSourceRef(
		ctx,
		source,
	)
	if err != nil {
		return nil, nil, err
	}
	sourceBucket, err := NewFetchSourceReader(
		container.Logger(),
		storageosProvider,
	).GetSourceBucket(
		ctx,
		container,
		sourceRef,
		buffetch.GetSourceBucketWithWorkspacesDisabled(),
	)
	if err != nil {
		return nil, nil, err
	}
	exists, err := bufconfig.ConfigExists(ctx, sourceBucket)
	if err != nil {
		return nil, nil, NewInternalError(err)
	}
	if !exists {
		return nil, nil, ErrNoConfigFile
	}
	// TODO: This should just read a lock file
	sourceConfig, err := bufconfig.NewProvider(
		container.Logger(),
	).GetConfig(
		ctx,
		sourceBucket,
	)
	if err != nil {
		return nil, nil, err
	}
	moduleIdentity := sourceConfig.ModuleIdentity
	if moduleIdentity == nil {
		return nil, nil, ErrNoModuleName
	}
	module, err := bufmodulebuild.NewModuleBucketBuilder(container.Logger()).BuildForBucket(
		ctx,
		sourceBucket,
		sourceConfig.Build,
	)
	if err != nil {
		return nil, nil, err
	}
	return module, moduleIdentity, err
}

// promptUser reads a line from Stdin, prompting the user with the prompt first.
// The prompt is repeatedly shown until the user provides a non-empty response.
func promptUser(container app.Container, prompt string) (string, error) {
	var attempts int
	for attempts < userPromptAttempts {
		attempts++
		if _, err := fmt.Fprint(
			container.Stdout(),
			prompt,
		); err != nil {
			return "", NewInternalError(err)
		}
		scanner := bufio.NewScanner(container.Stdin())
		if !scanner.Scan() {
			return "", NewInternalError(scanner.Err())
		}
		value := scanner.Text()
		if err := scanner.Err(); err != nil {
			return "", NewInternalError(err)
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
				"An answer was not provided; please try again.",
			); err != nil {
				return "", NewInternalError(err)
			}
		}
	}
	return "", NewTooManyEmptyAnswersError(userPromptAttempts)
}
