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
	"io/ioutil"
	"sort"

	"github.com/bufbuild/buf/internal/buf/bufcore"
	modulev1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/module/v1"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/multierr"
)

func sortFileInfos(fileInfos []bufcore.FileInfo) {
	sort.Slice(
		fileInfos,
		func(i int, j int) bool {
			return fileInfos[i].Path() < fileInfos[j].Path()
		},
	)
}

func sortResolvedModuleNames(resolvedModuleNames []ResolvedModuleName) {
	sort.Slice(resolvedModuleNames, func(i, j int) bool {
		return resolvedModuleNameLess(resolvedModuleNames[i], resolvedModuleNames[j])
	})
}

func resolvedModuleNameLess(a ResolvedModuleName, b ResolvedModuleName) bool {
	return resolvedModuleNameCompareTo(a, b) < 0
}

// return -1 if less
// return 1 if greater
// return 0 if equal
func resolvedModuleNameCompareTo(a ResolvedModuleName, b ResolvedModuleName) int {
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
	if a.Version() < b.Version() {
		return -1
	}
	if a.Version() > b.Version() {
		return 1
	}
	if a.Digest() < b.Digest() {
		return -1
	}
	if a.Digest() > b.Digest() {
		return 1
	}
	return 0
}

func newInvalidModuleNameStringError(path string, reason string) error {
	return fmt.Errorf("invalid module name: %s: %s", reason, path)
}

func moduleFileToBucket(ctx context.Context, module Module, path string, writeBucket storage.WriteBucket) (retErr error) {
	readCloser, err := module.GetFile(ctx, path)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, readCloser.Close())
	}()
	return storage.CopyReader(ctx, writeBucket, readCloser, path)
}

func moduleFileToProto(ctx context.Context, module Module, path string) (_ *modulev1.ModuleFile, retErr error) {
	protoModuleFile := &modulev1.ModuleFile{
		Path: path,
	}
	readCloser, err := module.GetFile(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readCloser.Close())
	}()
	protoModuleFile.Content, err = ioutil.ReadAll(readCloser)
	if err != nil {
		return nil, err
	}
	return protoModuleFile, nil
}
