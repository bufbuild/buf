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

// Package bufwork defines the primitives used to enable workspaces.
//
// If a buf.work file exists in a parent directory (up to the root of
// the filesystem), the directory containing the file is used as the root of
// one or more modules. With this, modules can import from one another, and a
// variety of commands work on multiple modules rather than one. For example, if
// `buf lint` is run for an input that contains a buf.work, each of
// the modules contained within the workspace will be linted. Other commands, such
// as `buf build`, will merge workspace modules into one (i.e. a "supermodule")
// so that all of the files contained are consolidated into a single image.
//
// In the following example, the workspace consists of two modules: the module
// defined in the petapis directory can import definitions from the paymentapis
// module without vendoring the definitions under a common root. To be clear,
// `import "acme/payment/v2/payment.proto";` from the acme/pet/v1/pet.proto file
// will suffice as long as the buf.work file exists.
//
//   // buf.work
//   version: v1
//   directories:
//     - paymentapis
//     - petapis
//
//   $ tree
//   .
//   ├── buf.work
//   ├── paymentapis
//   │   ├── acme
//   │   │   └── payment
//   │   │       └── v2
//   │   │           └── payment.proto
//   │   └── buf.mod
//   └── petapis
//       ├── acme
//       │   └── pet
//       │       └── v1
//       │           └── pet.proto
//       └── buf.mod
//
// Note that inputs MUST NOT overlap with any of the directories defined in the buf.work
// file. For example, it's not possible to build input "paymentapis/acme" since the image
// would otherwise include the content defined in paymentapis/acme/payment/v2/payment.proto as
// acme/payment/v2/payment.proto and payment/v2/payment.proto.
package bufwork

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

const (
	// ExternalConfigFilePath is the default configuration file path for v1.
	ExternalConfigFilePath = "buf.work"
	// V1Version is the version string used to indicate the v1 version of the buf.work file.
	V1Version = "v1"
)

// NewWorkspace returns a new workspace.
func NewWorkspace(
	ctx context.Context,
	config *Config,
	readBucket storage.ReadBucket,
	configProvider bufconfig.Provider,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
	relativeRootPath string,
	targetSubDirPath string,
	configOverride string,
	externalDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
) (bufmodule.Workspace, error) {
	return newWorkspace(
		ctx,
		config,
		readBucket,
		configProvider,
		moduleBucketBuilder,
		relativeRootPath,
		targetSubDirPath,
		configOverride,
		externalDirOrFilePaths,
		externalDirOrFilePathsAllowNotExist,
	)
}

// BuildOptionsForWorkspaceDirectory returns the bufmodulebuild.BuildOptions required for
// the given subDirPath based on the workspace configuration.
func BuildOptionsForWorkspaceDirectory(
	ctx context.Context,
	workspaceConfig *Config,
	moduleConfig *bufconfig.Config,
	relativeRootPath string,
	subDirPath string,
	externalDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
) ([]bufmodulebuild.BuildOption, error) {
	buildOptions := []bufmodulebuild.BuildOption{
		// We can't determine the module's commit from the local file system.
		// This also may be nil.
		//
		// This is particularly useful for the GoPackage modifier used in
		// Managed Mode, which supports module-specific overrides.
		bufmodulebuild.WithModuleIdentity(moduleConfig.ModuleIdentity),
	}
	if len(externalDirOrFilePaths) == 0 {
		return buildOptions, nil
	}
	workspaceID := filepath.Join(normalpath.Unnormalize(relativeRootPath), ExternalConfigFilePath)
	// We first need to reformat the externalDirOrFilePaths so that they accommodate
	// for the relativeRootPath (the path to the directory containing the buf.work).
	//
	// For example,
	//
	//  $ buf build ../../proto --path ../../proto/buf
	//
	//  // buf.work
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
	// is not relative to the relativeRootPath. For example, supppose that the buf.work is found at ../../..,
	// whereas the current working directory is nested within one of the workspace directories like so:
	//
	//  $ buf build ../../.. --path ../proto/buf
	//
	// Although absolute paths don't apply to ArchiveRefs and GitRefs, this logic continues to work in
	// these cases. Both ArchiveRefs and GitRefs might have a relativeRootPath nested within the bucket's
	// root, e.g. an archive that defines a buf.work in a nested 'proto/buf.work' directory like so:
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
	absExternalDirOrFilePaths := make([]string, 0, len(externalDirOrFilePaths))
	for _, externalDirOrFilePath := range externalDirOrFilePaths {
		absExternalDirOrFilePath, err := normalpath.NormalizeAndAbsolute(externalDirOrFilePath)
		if err != nil {
			return nil, fmt.Errorf(
				"path %q could not be resolved",
				normalpath.Unnormalize(externalDirOrFilePath),
			)
		}
		absExternalDirOrFilePaths = append(absExternalDirOrFilePaths, absExternalDirOrFilePath)
	}
	absRelativeRootPath, err := normalpath.NormalizeAndAbsolute(relativeRootPath)
	if err != nil {
		return nil, err
	}
	relativeRootRelPaths := make([]string, 0, len(absExternalDirOrFilePaths))
	for i, absExternalDirOrFilePath := range absExternalDirOrFilePaths {
		if absRelativeRootPath == absExternalDirOrFilePath {
			return nil, fmt.Errorf(
				"path %q is equal to the workspace defined in %q",
				normalpath.Unnormalize(externalDirOrFilePaths[i]),
				workspaceID,
			)
		}
		if normalpath.ContainsPath(absRelativeRootPath, absExternalDirOrFilePath, normalpath.Absolute) {
			relativeRootRelPath, err := normalpath.Rel(absRelativeRootPath, absExternalDirOrFilePath)
			if err != nil {
				return nil, fmt.Errorf(
					"a relative path could not be resolved between %q and %q",
					normalpath.Unnormalize(externalDirOrFilePaths[i]),
					workspaceID,
				)
			}
			relativeRootRelPaths = append(relativeRootRelPaths, relativeRootRelPath)
		}
	}
	// Now that the paths are relative to the relativeRootPath, the paths need to be scoped to
	// the directory they belong to.
	//
	// For example, after the paths have been processed above, the arguments can be imagined like so:
	//
	//  $ buf build proto --path proto/buf
	//
	//  // buf.work
	//  version: v1
	//  directories:
	//    - proto
	//    - enterprise/proto
	//
	// The 'proto' directory will receive the ./proto/buf/... files as ./buf/... whereas the
	// 'enterprise/proto' directory will have no matching paths.
	subDirRelPaths := make([]string, 0, len(relativeRootRelPaths))
	for i, relativeRootRelPath := range relativeRootRelPaths {
		if subDirPath == relativeRootRelPath {
			return nil, fmt.Errorf(
				"path %q is equal to workspace directory %q defined in %q",
				normalpath.Unnormalize(externalDirOrFilePaths[i]),
				normalpath.Unnormalize(subDirPath),
				workspaceID,
			)
		}
		if normalpath.ContainsPath(subDirPath, relativeRootRelPath, normalpath.Relative) {
			subDirRelPath, err := normalpath.Rel(subDirPath, relativeRootRelPath)
			if err != nil {
				return nil, fmt.Errorf(
					"a relative path could not be resolved between %q and %q",
					normalpath.Unnormalize(externalDirOrFilePaths[i]),
					subDirPath,
				)
			}
			subDirRelPaths = append(subDirRelPaths, subDirRelPath)
		}
	}
	// Note that subDirRelPaths can be empty. If so, this represents
	// the case where externalDirOrFilePaths were provided, but none
	// matched.
	if externalDirOrFilePathsAllowNotExist {
		buildOptions = append(buildOptions, bufmodulebuild.WithPathsAllowNotExist(subDirRelPaths))
	} else {
		buildOptions = append(buildOptions, bufmodulebuild.WithPaths(subDirRelPaths))
	}
	return buildOptions, nil
}

// Config is the workspace config.
type Config struct {
	// Directories are normalized and validated.
	Directories []string
}

// Provider provides workspace configurations.
type Provider interface {
	// GetConfig gets the Config for the YAML data at ConfigFilePath.
	//
	// If the data is of length 0, returns the default config.
	GetConfig(ctx context.Context, readBucket storage.ReadBucket, relativeRootPath string) (*Config, error)
	// GetConfig gets the Config for the given JSON or YAML data.
	//
	// If the data is of length 0, returns the default config.
	GetConfigForData(ctx context.Context, data []byte) (*Config, error)
}

// NewProvider returns a new Provider.
func NewProvider(logger *zap.Logger) Provider {
	return newProvider(logger)
}

// ConfigExists checks if a workspace configuration file exists.
func ConfigExists(ctx context.Context, readBucket storage.ReadBucket) (bool, error) {
	return storage.Exists(ctx, readBucket, ExternalConfigFilePath)
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
