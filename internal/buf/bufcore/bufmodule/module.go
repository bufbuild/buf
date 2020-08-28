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

package bufmodule

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufcore"
	modulev1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/module/v1"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
)

type module struct {
	sourceReadBucket storage.ReadBucket
	dependencies     []ModuleName
}

func newModuleForProto(ctx context.Context, protoModule *modulev1.Module) (*module, error) {
	if err := validateProtoModule(protoModule); err != nil {
		return nil, err
	}
	pathToData := make(map[string][]byte)
	dependencies, err := NewModuleNamesForProtos(protoModule.Dependencies...)
	if err != nil {
		return nil, err
	}
	sortModuleNames(dependencies)
	lockFileContent, err := putDependenciesToData(dependencies)
	if err != nil {
		return nil, err
	}
	pathToData[LockFilePath] = lockFileContent
	for _, moduleFile := range protoModule.Files {
		// we already know that paths are unique from validation
		pathToData[moduleFile.Path] = moduleFile.Content
	}
	sourceReadBucket, err := storagemem.NewReadBucket(pathToData)
	if err != nil {
		return nil, err
	}
	return newModuleForBucket(
		ctx,
		sourceReadBucket,
	)
}
func newModuleForBucket(
	ctx context.Context,
	sourceReadBucket storage.ReadBucket,
) (*module, error) {
	dependencies, err := dependenciesForBucket(ctx, sourceReadBucket)
	if err != nil {
		return nil, err
	}
	return newModuleForBucketWithDependencies(ctx, sourceReadBucket, dependencies)
}

func newModuleForBucketWithDependencies(
	ctx context.Context,
	sourceReadBucket storage.ReadBucket,
	dependencies []ModuleName,
) (*module, error) {
	seenModuleNames := make(map[string]struct{})
	for _, dependency := range dependencies {
		moduleIdentity := moduleNameIdentity(dependency)
		if _, ok := seenModuleNames[moduleIdentity]; ok {
			return nil, fmt.Errorf("module %s appeared twice", moduleIdentity)
		}
		if dependency.Digest() == "" {
			return nil, NewNoDigestError(dependency)
		}
		seenModuleNames[moduleIdentity] = struct{}{}
	}
	sortModuleNames(dependencies)
	return &module{
		// Now that we've captured the dependencies, we prune it from
		// the source read bucket so that it can be validated as a closure of .proto files.
		sourceReadBucket: storage.MapReadBucket(sourceReadBucket, storage.MatchPathExt(".proto")),
		dependencies:     dependencies,
	}, nil
}

func (m *module) TargetFileInfos(ctx context.Context) ([]bufcore.FileInfo, error) {
	return m.SourceFileInfos(ctx)
}

func (m *module) SourceFileInfos(ctx context.Context) ([]bufcore.FileInfo, error) {
	var fileInfos []bufcore.FileInfo
	if err := m.sourceReadBucket.Walk(ctx, "", func(objectInfo storage.ObjectInfo) error {
		// super overkill but ok
		if err := validateModuleFilePathWithoutNormalization(objectInfo.Path()); err != nil {
			return err
		}
		fileInfos = append(fileInfos, bufcore.NewFileInfoForObjectInfo(objectInfo, false))
		return nil
	}); err != nil {
		return nil, err
	}
	if len(fileInfos) == 0 {
		return nil, ErrNoTargetFiles
	}
	sortFileInfos(fileInfos)
	return fileInfos, nil
}

func (m *module) GetFile(ctx context.Context, path string) (ModuleFile, error) {
	// super overkill but ok
	if err := validateModuleFilePath(path); err != nil {
		return nil, err
	}
	readObjectCloser, err := m.sourceReadBucket.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	return newModuleFile(bufcore.NewFileInfoForObjectInfo(readObjectCloser, false), readObjectCloser), nil
}

// Dependencies gets the dependency ModuleNames.
// The returned dependencies are sorted by server, owner, repository, version, and digest.
func (m *module) Dependencies() []ModuleName {
	// already sorted
	return m.dependencies
}

func (m *module) getSourceReadBucket() storage.ReadBucket {
	return m.sourceReadBucket
}

func (m *module) isModule() {}
