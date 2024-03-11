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

package bufmodule

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/bufbreakingconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/buflintconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	breakingv1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/breaking/v1"
	lintv1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/lint/v1"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
)

type module struct {
	sourceReadBucket           storage.ReadBucket
	declaredDirectDependencies []bufmoduleref.ModuleReference
	dependencyModulePins       []bufmoduleref.ModulePin
	moduleIdentity             bufmoduleref.ModuleIdentity
	commit                     string
	documentation              string
	documentationPath          string
	license                    string
	breakingConfig             *bufbreakingconfig.Config
	lintConfig                 *buflintconfig.Config
	fileSet                    bufcas.FileSet
	workspaceDirectory         string
}

func newModuleForProto(
	ctx context.Context,
	protoModule *modulev1alpha1.Module,
	options ...ModuleOption,
) (*module, error) {
	if err := ValidateProtoModule(protoModule); err != nil {
		return nil, err
	}
	// We store this as a ReadBucket as this should never be modified outside of this function.
	readWriteBucket := storagemem.NewReadWriteBucket()
	for _, moduleFile := range protoModule.Files {
		if normalpath.Ext(moduleFile.Path) != ".proto" {
			return nil, fmt.Errorf("expected .proto file but got %q", moduleFile)
		}
		// we already know that paths are unique from validation
		if err := storage.PutPath(ctx, readWriteBucket, moduleFile.Path, moduleFile.Content); err != nil {
			return nil, err
		}
	}
	dependencyModulePins, err := bufmoduleref.NewModulePinsForProtos(protoModule.Dependencies...)
	if err != nil {
		return nil, err
	}
	breakingConfig, lintConfig, err := configsForProto(protoModule.GetBreakingConfig(), protoModule.GetLintConfig())
	if err != nil {
		return nil, err
	}
	allDependenciesRefs := make([]bufmoduleref.ModuleReference, len(dependencyModulePins))
	for i, dep := range dependencyModulePins {
		allDependenciesRefs[i], err = bufmoduleref.NewModuleReference(
			dep.Remote(), dep.Owner(), dep.Repository(), dep.Commit(),
		)
		if err != nil {
			return nil, fmt.Errorf("cannot build module reference from dependency pin %s: %w", dep.String(), err)
		}
	}
	return newModule(
		readWriteBucket,
		allDependenciesRefs, // Since proto has no distinction between direct/transitive dependencies, we'll need to set them all as direct, otherwise the build will fail.
		dependencyModulePins,
		nil, // The module identity is not stored on the proto. We rely on the layer above, (e.g. `ModuleReader`) to set this as needed.
		protoModule.GetDocumentation(),
		protoModule.GetDocumentationPath(),
		protoModule.GetLicense(),
		breakingConfig,
		lintConfig,
		options...,
	)
}

func configsForProto(
	protoBreakingConfig *breakingv1.Config,
	protoLintConfig *lintv1.Config,
) (*bufbreakingconfig.Config, *buflintconfig.Config, error) {
	var breakingConfig *bufbreakingconfig.Config
	var breakingConfigVersion string
	if protoBreakingConfig != nil {
		breakingConfig = bufbreakingconfig.ConfigForProto(protoBreakingConfig)
		breakingConfigVersion = breakingConfig.Version
	}
	var lintConfig *buflintconfig.Config
	var lintConfigVersion string
	if protoLintConfig != nil {
		lintConfig = buflintconfig.ConfigForProto(protoLintConfig)
		lintConfigVersion = lintConfig.Version
	}
	if lintConfigVersion != breakingConfigVersion {
		return nil, nil, fmt.Errorf("mismatched breaking config version %q and lint config version %q found", breakingConfigVersion, lintConfigVersion)
	}
	// If there is no breaking and lint configs, we want to default to the v1 version.
	if breakingConfig == nil && lintConfig == nil {
		breakingConfig = &bufbreakingconfig.Config{
			Version: bufconfig.V1Version,
		}
		lintConfig = &buflintconfig.Config{
			Version: bufconfig.V1Version,
		}
	} else if breakingConfig == nil {
		// In the case that only breaking config is nil, we'll use generated an empty default config
		// using the lint config version.
		breakingConfig = &bufbreakingconfig.Config{
			Version: lintConfigVersion,
		}
	} else if lintConfig == nil {
		// In the case that only lint config is nil, we'll use generated an empty default config
		// using the breaking config version.
		lintConfig = &buflintconfig.Config{
			Version: breakingConfigVersion,
		}
	}
	// Finally, validate the config versions are valid. This should always pass in the case of
	// the default values.
	if err := bufconfig.ValidateVersion(breakingConfig.Version); err != nil {
		return nil, nil, err
	}
	if err := bufconfig.ValidateVersion(lintConfig.Version); err != nil {
		return nil, nil, err
	}
	return breakingConfig, lintConfig, nil
}

func newModuleForBucket(
	ctx context.Context,
	sourceReadBucket storage.ReadBucket,
	options ...ModuleOption,
) (*module, error) {
	dependencyModulePins, err := bufmoduleref.DependencyModulePinsForBucket(ctx, sourceReadBucket)
	if err != nil {
		return nil, err
	}
	var documentation string
	var documentationPath string
	for _, docPath := range AllDocumentationPaths {
		documentation, err = getFileContentForBucket(ctx, sourceReadBucket, docPath)
		if err != nil {
			return nil, err
		}
		if documentation != "" {
			documentationPath = docPath
			break
		}
	}
	license, err := getFileContentForBucket(ctx, sourceReadBucket, LicenseFilePath)
	if err != nil {
		return nil, err
	}
	moduleConfig, err := bufconfig.GetConfigForBucket(ctx, sourceReadBucket)
	if err != nil {
		return nil, err
	}
	var moduleIdentity bufmoduleref.ModuleIdentity
	// if the module config has an identity, set the module identity
	if moduleConfig.ModuleIdentity != nil {
		moduleIdentity = moduleConfig.ModuleIdentity
	}
	return newModule(
		storage.MapReadBucket(sourceReadBucket, storage.MatchPathExt(".proto")),
		moduleConfig.Build.DependencyModuleReferences, // straight copy from the buf.yaml file
		dependencyModulePins,
		moduleIdentity,
		documentation,
		documentationPath,
		license,
		moduleConfig.Breaking,
		moduleConfig.Lint,
		options...,
	)
}

func newModuleForFileSet(
	ctx context.Context,
	fileSet bufcas.FileSet,
	options ...ModuleOption,
) (*module, error) {
	bucket := storagemem.NewReadWriteBucket()
	if err := bufcas.PutFileSetToBucket(
		ctx,
		fileSet,
		bucket,
	); err != nil {
		return nil, err
	}
	module, err := newModuleForBucket(ctx, bucket, options...)
	if err != nil {
		return nil, err
	}
	module.fileSet = fileSet
	return module, nil
}

// this should only be called by other newModule constructors
func newModule(
	// must only contain .proto files
	sourceReadBucket storage.ReadBucket,
	declaredDirectDependencies []bufmoduleref.ModuleReference,
	dependencyModulePins []bufmoduleref.ModulePin,
	moduleIdentity bufmoduleref.ModuleIdentity,
	documentation string,
	documentationPath string,
	license string,
	breakingConfig *bufbreakingconfig.Config,
	lintConfig *buflintconfig.Config,
	options ...ModuleOption,
) (_ *module, retErr error) {
	if err := bufmoduleref.ValidateModuleReferencesUniqueByIdentity(declaredDirectDependencies); err != nil {
		return nil, err
	}
	if err := bufmoduleref.ValidateModulePinsUniqueByIdentity(dependencyModulePins); err != nil {
		return nil, err
	}
	// we rely on this being sorted here
	bufmoduleref.SortModuleReferences(declaredDirectDependencies)
	bufmoduleref.SortModulePins(dependencyModulePins)
	module := &module{
		sourceReadBucket:           sourceReadBucket,
		declaredDirectDependencies: declaredDirectDependencies,
		dependencyModulePins:       dependencyModulePins,
		moduleIdentity:             moduleIdentity,
		documentation:              documentation,
		documentationPath:          documentationPath,
		license:                    license,
		breakingConfig:             breakingConfig,
		lintConfig:                 lintConfig,
	}
	for _, option := range options {
		option(module)
	}
	if module.moduleIdentity == nil && module.commit != "" {
		return nil, fmt.Errorf("module was constructed with commit %q but no associated ModuleIdentity", module.commit)
	}
	return module, nil
}

func (m *module) TargetFileInfos(ctx context.Context) ([]bufmoduleref.FileInfo, error) {
	return m.SourceFileInfos(ctx)
}

func (m *module) SourceFileInfos(ctx context.Context) ([]bufmoduleref.FileInfo, error) {
	var fileInfos []bufmoduleref.FileInfo
	if walkErr := m.sourceReadBucket.Walk(ctx, "", func(objectInfo storage.ObjectInfo) error {
		// super overkill but ok
		if err := bufmoduleref.ValidateModuleFilePath(objectInfo.Path()); err != nil {
			return err
		}
		fileInfo, err := bufmoduleref.NewFileInfo(
			objectInfo.Path(),
			objectInfo.ExternalPath(),
			m.moduleIdentity,
			m.commit,
		)
		if err != nil {
			return err
		}
		fileInfos = append(fileInfos, fileInfo)
		return nil
	}); walkErr != nil {
		return nil, fmt.Errorf("failed to enumerate module files: %w", walkErr)
	}
	bufmoduleref.SortFileInfos(fileInfos)
	return fileInfos, nil
}

func (m *module) GetModuleFile(ctx context.Context, path string) (ModuleFile, error) {
	// super overkill but ok
	if err := bufmoduleref.ValidateModuleFilePath(path); err != nil {
		return nil, err
	}
	readObjectCloser, err := m.sourceReadBucket.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	fileInfo, err := bufmoduleref.NewFileInfo(
		readObjectCloser.Path(),
		readObjectCloser.ExternalPath(),
		m.moduleIdentity,
		m.commit,
	)
	if err != nil {
		return nil, err
	}
	return newModuleFile(fileInfo, readObjectCloser), nil
}

func (m *module) DeclaredDirectDependencies() []bufmoduleref.ModuleReference {
	// already sorted in constructor
	return m.declaredDirectDependencies
}

func (m *module) DependencyModulePins() []bufmoduleref.ModulePin {
	// already sorted in constructor
	return m.dependencyModulePins
}

func (m *module) Documentation() string {
	return m.documentation
}

func (m *module) DocumentationPath() string {
	return m.documentationPath
}

func (m *module) License() string {
	return m.license
}

func (m *module) BreakingConfig() *bufbreakingconfig.Config {
	return m.breakingConfig
}

func (m *module) LintConfig() *buflintconfig.Config {
	return m.lintConfig
}

func (m *module) FileSet() bufcas.FileSet {
	return m.fileSet
}

func (m *module) ModuleIdentity() bufmoduleref.ModuleIdentity {
	return m.moduleIdentity
}

func (m *module) Commit() string {
	return m.commit
}

func (m *module) WorkspaceDirectory() string {
	return m.workspaceDirectory
}

func (m *module) getSourceReadBucket() storage.ReadBucket {
	return m.sourceReadBucket
}

func (m *module) isModule() {}
