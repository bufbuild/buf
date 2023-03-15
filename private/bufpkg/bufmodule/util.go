// Copyright 2020-2023 Buf Technologies, Inc.
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
	"io"
	"sort"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/multierr"
)

func putModuleFileToBucket(ctx context.Context, module Module, path string, writeBucket storage.WriteBucket) (retErr error) {
	moduleFile, err := module.GetModuleFile(ctx, path)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, moduleFile.Close())
	}()
	var copyOptions []storage.CopyOption
	if writeBucket.SetExternalPathSupported() {
		copyOptions = append(copyOptions, storage.CopyWithExternalPaths())
	}
	return storage.CopyReadObject(ctx, writeBucket, moduleFile, copyOptions...)
}

func moduleFileToProto(ctx context.Context, module Module, path string) (_ *modulev1alpha1.ModuleFile, retErr error) {
	protoModuleFile := &modulev1alpha1.ModuleFile{
		Path: path,
	}
	moduleFile, err := module.GetModuleFile(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, moduleFile.Close())
	}()
	protoModuleFile.Content, err = io.ReadAll(moduleFile)
	if err != nil {
		return nil, err
	}
	return protoModuleFile, nil
}

func getFileContentForBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
	path string,
) (string, error) {
	data, err := storage.ReadPath(ctx, readBucket, path)
	if err != nil {
		if storage.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func copyModulePinsSortedByOnlyCommit(modulePins []bufmoduleref.ModulePin) []bufmoduleref.ModulePin {
	s := make([]bufmoduleref.ModulePin, len(modulePins))
	copy(s, modulePins)
	sort.Slice(s, func(i, j int) bool {
		return modulePinLessOnlyCommit(s[i], s[j])
	})
	return s
}

func modulePinLessOnlyCommit(a bufmoduleref.ModulePin, b bufmoduleref.ModulePin) bool {
	return modulePinCompareToOnlyCommit(a, b) < 0
}

// return -1 if less
// return 1 if greater
// return 0 if equal
func modulePinCompareToOnlyCommit(a bufmoduleref.ModulePin, b bufmoduleref.ModulePin) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil && b != nil {
		return -1
	}
	if a != nil && b == nil {
		return 1
	}
	if a.Remote() < b.Remote() {
		return -1
	}
	if a.Remote() > b.Remote() {
		return 1
	}
	if a.Owner() < b.Owner() {
		return -1
	}
	if a.Owner() > b.Owner() {
		return 1
	}
	if a.Repository() < b.Repository() {
		return -1
	}
	if a.Repository() > b.Repository() {
		return 1
	}
	if a.Commit() < b.Commit() {
		return -1
	}
	if a.Commit() > b.Commit() {
		return 1
	}
	return 0
}
