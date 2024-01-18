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

// Package bufwork defines the primitives used to enable workspaces.
//
// If a buf.work.yaml file exists in a parent directory (up to the root of
// the filesystem), the directory containing the file is used as the root of
// one or more modules. With this, modules can import from one another, and a
// variety of commands work on multiple modules rather than one. For example, if
// `buf lint` is run for an input that contains a buf.work.yaml, each of
// the modules contained within the workspace will be linted. Other commands, such
// as `buf build`, will merge workspace modules into one (i.e. a "supermodule")
// so that all of the files contained are consolidated into a single image.
//
// In the following example, the workspace consists of two modules: the module
// defined in the petapis directory can import definitions from the paymentapis
// module without vendoring the definitions under a common root. To be clear,
// `import "acme/payment/v2/payment.proto";` from the acme/pet/v1/pet.proto file
// will suffice as long as the buf.work.yaml file exists.
//
//	// buf.work.yaml
//	version: v1
//	directories:
//	  - paymentapis
//	  - petapis
//
//	$ tree
//	.
//	├── buf.work.yaml
//	├── paymentapis
//	│   ├── acme
//	│   │   └── payment
//	│   │       └── v2
//	│   │           └── payment.proto
//	│   └── buf.yaml
//	└── petapis
//	    ├── acme
//	    │   └── pet
//	    │       └── v1
//	    │           └── pet.proto
//	    └── buf.yaml
//
// Note that inputs MUST NOT overlap with any of the directories defined in the buf.work.yaml
// file. For example, it's not possible to build input "paymentapis/acme" since the image
// would otherwise include the content defined in paymentapis/acme/payment/v2/payment.proto as
// acme/payment/v2/payment.proto and payment/v2/payment.proto.
//
// EVERYTHING IN THIS PACKAGE SHOULD ONLY BE CALLED BY THE CLI AND CANNOT BE USED IN SERVICES.
package bufwork

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
)

const (
	// ExternalConfigV1FilePath is the default configuration file path for v1.
	ExternalConfigV1FilePath = "buf.work.yaml"
	// V1Version is the version string used to indicate the v1 version of the buf.work.yaml file.
	V1Version = "v1"

	// BackupExternalConfigV1FilePath is another acceptable configuration file path for v1.
	//
	// Originally we thought we were going to use buf.work, and had this around for
	// a while, but then moved to buf.work.yaml. We still need to support buf.work as
	// we released with it, however.
	BackupExternalConfigV1FilePath = "buf.work"
)

var (
	// AllConfigFilePaths are all acceptable config file paths without overrides.
	//
	// These are in the order we should check.
	AllConfigFilePaths = []string{
		ExternalConfigV1FilePath,
		BackupExternalConfigV1FilePath,
	}
)

// WorkspaceBuilder builds workspaces. A single WorkspaceBuilder should NOT be persisted
// acorss calls because the WorkspaceBuilder caches the modules used in each workspace.
type WorkspaceBuilder interface {
	// BuildWorkspace builds a bufmodule.Workspace.
	//
	// The given targetSubDirPath is the only path that will have the configOverride applied to it.
	// TODO: delete targetSubDirPath entirely. We are building a Workspace, we don't necessarily
	// have a specific target directory within it. This would mean doing the config override at
	// a higher level for any specific modules within the Workspace. The only thing in the config
	// we care about is the build.excludes, so in theory we should be able to figure out a way
	// to say "exclude these files from these modules when you are building". Even better, the
	// WorkspaceBuilder has nothing to do with building modules.
	BuildWorkspace(
		ctx context.Context,
		workspaceConfig *Config,
		readBucket storage.ReadBucket,
		relativeRootPath string,
		targetSubDirPath string,
		configOverride string,
		externalDirOrFilePaths []string,
		externalExcludeDirOrFilePaths []string,
		externalDirOrFilePathsAllowNotExist bool,
	) (bufmodule.Workspace, error)

	// GetModuleConfig returns the bufmodule.Module and *bufconfig.Config, associated with the given
	// targetSubDirPath, if it exists.
	GetModuleConfig(targetSubDirPath string) (bufmodule.Module, *bufconfig.Config, bool)
}

// NewWorkspaceBuilder returns a new WorkspaceBuilder.
func NewWorkspaceBuilder() WorkspaceBuilder {
	return newWorkspaceBuilder()
}

// BuildOptionsForWorkspaceDirectory returns the bufmodulebuild.BuildOptions required for
// the given subDirPath based on the workspace configuration.
//
// The subDirRelPaths are the relative paths of the externalDirOrFilePaths that map to the
// provided subDirPath.
// The subDirRelExcludePaths are the relative paths of the externalExcludeDirOrFilePaths that map to the
// provided subDirPath.
func BuildOptionsForWorkspaceDirectory(
	ctx context.Context,
	workspaceConfig *Config,
	moduleConfig *bufconfig.Config,
	externalDirOrFilePaths []string,
	externalExcludeDirOrFilePaths []string,
	subDirRelPaths []string,
	subDirRelExcludePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
) ([]bufmodulebuild.BuildOption, error) {
	buildOptions := []bufmodulebuild.BuildOption{
		// We can't determine the module's commit from the local file system.
		// This also may be nil.
		//
		// This is particularly useful for the GoPackage modifier used in
		// managed mode, which supports module-specific overrides.
		bufmodulebuild.WithModuleIdentity(moduleConfig.ModuleIdentity),
	}
	if len(externalDirOrFilePaths) == 0 && len(externalExcludeDirOrFilePaths) == 0 {
		return buildOptions, nil
	}
	if len(externalDirOrFilePaths) > 0 {
		// Note that subDirRelPaths can be empty. If so, this represents
		// the case where externalDirOrFilePaths were provided, but none
		// matched.
		if externalDirOrFilePathsAllowNotExist {
			buildOptions = append(buildOptions, bufmodulebuild.WithPathsAllowNotExist(subDirRelPaths))
		} else {
			buildOptions = append(buildOptions, bufmodulebuild.WithPaths(subDirRelPaths))
		}
	}
	if len(externalExcludeDirOrFilePaths) > 0 {
		// Same as above, subDirRelExcludepaths can be empty. If so, this represents the case
		// where excludes were provided by were not matched.
		if externalDirOrFilePathsAllowNotExist {
			buildOptions = append(buildOptions, bufmodulebuild.WithExcludePathsAllowNotExist(subDirRelExcludePaths))
		} else {
			buildOptions = append(buildOptions, bufmodulebuild.WithExcludePaths(subDirRelExcludePaths))
		}
	}
	return buildOptions, nil
}

// ExternalPathsToSubDirRelPaths returns a map of the external paths provided to their relative
// path to the provided subDirPath.
//
// Note not every external path provided may have a relative path mapped to the subDirPath.
func ExternalPathsToSubDirRelPaths(
	relativeRootPath string,
	subDirPath string,
	externalDirOrFilePaths []string,
) (map[string]string, error) {
	// We first need to reformat the externalDirOrFilePaths so that they accommodate
	// for the relativeRootPath (the path to the directory containing the buf.work.yaml).
	//
	// For example,
	//
	//  $ buf build ../../proto --path ../../proto/buf
	//
	//  // buf.work.yaml
	//  version: v1
	//  directories:
	//    - proto
	//    - enterprise/proto
	//
	// Note that we CANNOT simply use the sourceRef because we would not be able to
	// determine which workspace directory the paths apply to afterwards. To be clear,
	// if we determined the bucketRelPath from the sourceRef, the bucketRelPath would be equal
	// to ./buf/... which is ambiguous to the workspace directories ('proto' and 'enterprise/proto'
	// in this case).
	//
	// Also note that we need to use absolute paths because it's possible that the externalDirOrFilePath
	// is not relative to the relativeRootPath. For example, supppose that the buf.work.yaml is found at ../../..,
	// whereas the current working directory is nested within one of the workspace directories like so:
	//
	//  $ buf build ../../.. --path ../proto/buf
	//
	// Although absolute paths don't apply to ArchiveRefs and GitRefs, this logic continues to work in
	// these cases. Both ArchiveRefs and GitRefs might have a relativeRootPath nested within the bucket's
	// root, e.g. an archive that defines a buf.work.yaml in a nested 'proto/buf.work.yaml' directory like so:
	//
	//  $ buf build weather.zip#subdir=proto --path proto/acme/weather/v1/weather.proto
	//
	//  $ zipinfo weather.zip
	//  Archive:  weather.zip
	//  ...
	//  ... proto/
	//
	// In this case, the relativeRootPath is equal to 'proto', so we still need to determine the relative path
	// between 'proto' and 'proto/acme/weather/v1/weather.proto' and assign it to the correct workspace directory.
	// So even though it's impossible for ArchiveRefs and GitRefs to jump context (i.e. '../..'), the transformation
	// from [relative -> absolute -> relative] will always yield valid results. In the example above, we would have
	// something along the lines:
	//
	//  * '/Users/me/path/to/wd' is the current working directory
	//
	//  absRelativeRootPath      == '/Users/me/path/to/wd/proto'
	//  absExternalDirOrFilePath == '/Users/me/path/to/wd/proto/acme/weather/v1/weather.proto'
	//
	//  ==> relativeRootRelPath  == 'acme/weather/v1/weather.proto'
	//
	// The paths, such as '/Users/me/path/to/wd/proto/acme/weather/v1/weather.proto', might not exist on the local
	// file system at all, but the [relative -> absolute -> relative] transformation works as expected.
	//
	// Alternatively, we could special-case this logic so that we only work with relative paths when we have an ArchiveRef
	// or GitRef, but this would violate the abstraction boundary for buffetch.
	externalToAbsExternalDirOrFilePaths := make(map[string]string)
	// We know that if the file is actually buf.work for legacy reasons, this will be wrong,
	// but we accept that as this shouldn't happen often anymore and this is just
	// used for error messages.
	workspaceID := filepath.Join(normalpath.Unnormalize(relativeRootPath), ExternalConfigV1FilePath)
	for _, externalDirOrFilePath := range externalDirOrFilePaths {
		absExternalDirOrFilePath, err := normalpath.NormalizeAndAbsolute(externalDirOrFilePath)
		if err != nil {
			return nil, fmt.Errorf(
				`path "%s" could not be resolved`,
				normalpath.Unnormalize(externalDirOrFilePath),
			)
		}
		externalToAbsExternalDirOrFilePaths[externalDirOrFilePath] = absExternalDirOrFilePath
	}
	absRelativeRootPath, err := normalpath.NormalizeAndAbsolute(relativeRootPath)
	if err != nil {
		return nil, err
	}
	externalToRelativeRootRelPaths := make(map[string]string)
	for externalDirOrFilePath, absExternalDirOrFilePath := range externalToAbsExternalDirOrFilePaths {
		if absRelativeRootPath == absExternalDirOrFilePath {
			return nil, fmt.Errorf(
				`path "%s" is equal to the workspace defined in "%s"`,
				normalpath.Unnormalize(externalDirOrFilePath),
				workspaceID,
			)
		}
		if normalpath.ContainsPath(absRelativeRootPath, absExternalDirOrFilePath, normalpath.Absolute) {
			relativeRootRelPath, err := normalpath.Rel(absRelativeRootPath, absExternalDirOrFilePath)
			if err != nil {
				return nil, fmt.Errorf(
					`a relative path could not be resolved between "%s" and "%s"`,
					normalpath.Unnormalize(externalDirOrFilePath),
					workspaceID,
				)
			}
			externalToRelativeRootRelPaths[externalDirOrFilePath] = relativeRootRelPath
		}
	}
	// Now that the paths are relative to the relativeRootPath, the paths need to be scoped to
	// the directory they belong to.
	//
	// For example, after the paths have been processed above, the arguments can be imagined like so:
	//
	//  $ buf build proto --path proto/buf
	//
	//  // buf.work.yaml
	//  version: v1
	//  directories:
	//    - proto
	//    - enterprise/proto
	//
	// The 'proto' directory will receive the ./proto/buf/... files as ./buf/... whereas the
	// 'enterprise/proto' directory will have no matching paths.
	externalToSubDirRelPaths := make(map[string]string)
	for externalDirOrFilePath, relativeRootRelPath := range externalToRelativeRootRelPaths {
		if subDirPath == relativeRootRelPath {
			return nil, fmt.Errorf(
				`path "%s" is equal to workspace directory "%s" defined in "%s"`,
				normalpath.Unnormalize(externalDirOrFilePath),
				normalpath.Unnormalize(subDirPath),
				workspaceID,
			)
		}
		if normalpath.ContainsPath(subDirPath, relativeRootRelPath, normalpath.Relative) {
			subDirRelPath, err := normalpath.Rel(subDirPath, relativeRootRelPath)
			if err != nil {
				return nil, fmt.Errorf(
					`a relative path could not be resolved between "%s" and "%s"`,
					normalpath.Unnormalize(externalDirOrFilePath),
					subDirPath,
				)
			}
			externalToSubDirRelPaths[externalDirOrFilePath] = subDirRelPath
		}
	}
	return externalToSubDirRelPaths, nil
}

// Config is the workspace config.
type Config struct {
	// Directories are normalized and validated.
	//
	// Must be non-empty to be a valid configuration.
	Directories []string
}

// GetConfigForBucket gets the Config for the YAML data at ConfigFilePath.
//
// This function expects that there is a valid non-empty configuration in the bucket. Otherwise, this errors.
func GetConfigForBucket(ctx context.Context, readBucket storage.ReadBucket, relativeRootPath string) (*Config, error) {
	return getConfigForBucket(ctx, readBucket, relativeRootPath)
}

// GetConfigForData gets the Config for the given JSON or YAML data.
//
// This function expects that there is a valid non-empty configuration. Otherwise, this errors.
func GetConfigForData(ctx context.Context, data []byte) (*Config, error) {
	return getConfigForData(ctx, data)
}

// ExistingConfigFilePath checks if a configuration file exists, and if so, returns the path
// within the ReadBucket of this configuration file.
//
// Returns empty string and no error if no configuration file exists.
func ExistingConfigFilePath(ctx context.Context, readBucket storage.ReadBucket) (string, error) {
	for _, configFilePath := range AllConfigFilePaths {
		exists, err := storage.Exists(ctx, readBucket, configFilePath)
		if err != nil {
			return "", err
		}
		if exists {
			return configFilePath, nil
		}
	}
	return "", nil
}

// ExternalConfigV1 represents the on-disk representation
// of the workspace configuration at version v1.
type ExternalConfigV1 struct {
	Version     string   `json:"version,omitempty" yaml:"version,omitempty"`
	Directories []string `json:"directories,omitempty" yaml:"directories,omitempty"`
}

type externalConfigVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}
