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
	"io"

	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/internal"
	modulev1alpha1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
)

type module struct {
	sourceReadBucket     storage.ReadBucket
	dependencyModulePins []ModulePin
	moduleCommit         ModuleCommit
	documentation        string
}

func newModuleForProto(
	ctx context.Context,
	protoModule *modulev1alpha1.Module,
	options ...ModuleOption,
) (*module, error) {
	if err := ValidateProtoModule(protoModule); err != nil {
		return nil, err
	}
	readBucketBuilder := storagemem.NewReadBucketBuilder()
	for _, moduleFile := range protoModule.Files {
		// we already know that paths are unique from validation
		if err := storage.PutPath(ctx, readBucketBuilder, moduleFile.Path, moduleFile.Content); err != nil {
			return nil, err
		}
	}
	if docs := protoModule.GetDocumentation(); docs != "" {
		if err := storage.PutPath(ctx, readBucketBuilder, DocumentationFilePath, []byte(docs)); err != nil {
			return nil, err
		}
	}
	sourceReadBucket, err := readBucketBuilder.ToReadBucket()
	if err != nil {
		return nil, err
	}
	dependencyModulePins, err := NewModulePinsForProtos(protoModule.Dependencies...)
	if err != nil {
		return nil, err
	}
	return newModuleForBucketWithDependencyModulePins(
		ctx,
		sourceReadBucket,
		dependencyModulePins,
		options...,
	)
}

func newModuleForBucket(
	ctx context.Context,
	sourceReadBucket storage.ReadBucket,
	options ...ModuleOption,
) (*module, error) {
	dependencyModulePins, err := getDependencyModulePinsForBucket(ctx, sourceReadBucket)
	if err != nil {
		return nil, err
	}
	return newModuleForBucketWithDependencyModulePins(
		ctx,
		sourceReadBucket,
		dependencyModulePins,
		options...,
	)
}

func newModuleForBucketWithDependencyModulePins(
	ctx context.Context,
	sourceReadBucket storage.ReadBucket,
	dependencyModulePins []ModulePin,
	options ...ModuleOption,
) (*module, error) {
	if err := ValidateModulePinsUniqueByIdentity(dependencyModulePins); err != nil {
		return nil, err
	}
	moduleOptions := &moduleOptions{}
	for _, option := range options {
		option(moduleOptions)
	}
	documentationReader, err := sourceReadBucket.Get(ctx, DocumentationFilePath)
	// we allow the lack of documentation file
	if err != nil && !storage.IsNotExist(err) {
		return nil, err
	}
	documentationContents := ""
	if documentationReader != nil {
		documentationBytes, err := io.ReadAll(documentationReader)
		if err != nil {
			return nil, err
		}
		documentationContents = string(documentationBytes)
	}
	// we rely on this being sorted here
	SortModulePins(dependencyModulePins)
	return &module{
		sourceReadBucket:     storage.MapReadBucket(sourceReadBucket, storage.MatchPathExt(".proto")),
		dependencyModulePins: dependencyModulePins,
		moduleCommit:         moduleOptions.moduleCommit,
		documentation:        documentationContents,
	}, nil
}

func (m *module) TargetFileInfos(ctx context.Context) ([]FileInfo, error) {
	return m.SourceFileInfos(ctx)
}

func (m *module) SourceFileInfos(ctx context.Context) ([]FileInfo, error) {
	var fileInfos []FileInfo
	if err := m.sourceReadBucket.Walk(ctx, "", func(objectInfo storage.ObjectInfo) error {
		// super overkill but ok
		if err := validateModuleFilePathWithoutNormalization(objectInfo.Path()); err != nil {
			return err
		}
		coreFileInfo := bufcore.NewFileInfoForObjectInfo(objectInfo, false)
		fileInfos = append(fileInfos, NewFileInfo(coreFileInfo, m.moduleCommit))
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to enumerate module files: %w", err)
	}
	if len(fileInfos) == 0 {
		return nil, internal.ErrNoTargetFiles
	}
	sortFileInfos(fileInfos)
	return fileInfos, nil
}

func (m *module) GetModuleFile(ctx context.Context, path string) (ModuleFile, error) {
	// super overkill but ok
	if err := ValidateModuleFilePath(path); err != nil {
		return nil, err
	}
	readObjectCloser, err := m.sourceReadBucket.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	coreFileInfo := bufcore.NewFileInfoForObjectInfo(readObjectCloser, false)
	return newModuleFile(NewFileInfo(coreFileInfo, m.moduleCommit), readObjectCloser), nil
}

func (m *module) DependencyModulePins() []ModulePin {
	// already sorted in constructor
	return m.dependencyModulePins
}

func (m *module) Documentation() string {
	return m.documentation
}

func (m *module) getSourceReadBucket() storage.ReadBucket {
	return m.sourceReadBucket
}

func (m *module) getModuleCommit() ModuleCommit {
	return m.moduleCommit
}

func (m *module) isModule() {}
