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

package bufmodule

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/bufbreakingconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/buflintconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
)

type module struct {
	sourceReadBucket     storage.ReadBucket
	dependencyModulePins []bufmoduleref.ModulePin
	moduleIdentity       bufmoduleref.ModuleIdentity
	commit               string
	documentation        string
	breakingConfig       *bufbreakingconfig.Config
	lintConfig           *buflintconfig.Config
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
	protoBreakingConfig := protoModule.GetBreakingConfig()
	var breakingConfig *bufbreakingconfig.Config
	if protoBreakingConfig.GetVersion() == bufconfig.V1Version {
		var err error
		breakingConfig, err = bufbreakingconfig.NewConfigV1ForProto(protoBreakingConfig)
		if err != nil {
			return nil, err
		}
	}
	if protoBreakingConfig.GetVersion() == bufconfig.V1Beta1Version {
		var err error
		breakingConfig, err = bufbreakingconfig.NewConfigV1Beta1ForProto(protoBreakingConfig)
		if err != nil {
			return nil, err
		}
	}
	protoLintConfig := protoModule.GetLintConfig()
	var lintConfig *buflintconfig.Config
	if protoLintConfig.GetVersion() == bufconfig.V1Version {
		var err error
		lintConfig, err = buflintconfig.NewConfigV1ForProto(protoLintConfig)
		if err != nil {
			return nil, err
		}
	}
	if protoLintConfig.GetVersion() == bufconfig.V1Beta1Version {
		var err error
		lintConfig, err = buflintconfig.NewConfigV1Beta1ForProto(protoLintConfig)
		if err != nil {
			return nil, err
		}
	}
	return newModule(
		ctx,
		readWriteBucket,
		dependencyModulePins,
		protoModule.GetDocumentation(),
		breakingConfig,
		lintConfig,
		options...,
	)
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
	documentation, err := getDocumentationForBucket(ctx, sourceReadBucket)
	if err != nil {
		return nil, err
	}
	moduleConfig, err := bufconfig.GetConfigForBucket(ctx, sourceReadBucket)
	if err != nil {
		return nil, err
	}
	return newModule(
		ctx,
		storage.MapReadBucket(sourceReadBucket, storage.MatchPathExt(".proto")),
		dependencyModulePins,
		documentation,
		moduleConfig.Breaking,
		moduleConfig.Lint,
		options...,
	)
}

// this should only be called by other newModule constructors
func newModule(
	ctx context.Context,
	// must only contain .proto files
	sourceReadBucket storage.ReadBucket,
	dependencyModulePins []bufmoduleref.ModulePin,
	documentation string,
	breakingConfig *bufbreakingconfig.Config,
	lintConfig *buflintconfig.Config,
	options ...ModuleOption,
) (_ *module, retErr error) {
	if err := bufmoduleref.ValidateModulePinsUniqueByIdentity(dependencyModulePins); err != nil {
		return nil, err
	}
	// we rely on this being sorted here
	bufmoduleref.SortModulePins(dependencyModulePins)
	module := &module{
		sourceReadBucket:     sourceReadBucket,
		dependencyModulePins: dependencyModulePins,
		documentation:        documentation,
		breakingConfig:       breakingConfig,
		lintConfig:           lintConfig,
	}
	for _, option := range options {
		option(module)
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
			false,
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
		false,
		m.moduleIdentity,
		m.commit,
	)
	if err != nil {
		return nil, err
	}
	return newModuleFile(fileInfo, readObjectCloser), nil
}

func (m *module) DependencyModulePins() []bufmoduleref.ModulePin {
	// already sorted in constructor
	return m.dependencyModulePins
}

func (m *module) Documentation() string {
	return m.documentation
}

func (m *module) getSourceReadBucket() storage.ReadBucket {
	return m.sourceReadBucket
}

func (m *module) getModuleIdentity() bufmoduleref.ModuleIdentity {
	return m.moduleIdentity
}

func (m *module) getCommit() string {
	return m.commit
}

func (m *module) isModule() {}
