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

package bufmoduleapi

import (
	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
)

type universalProtoFile struct {
	Path    string
	Content []byte
}

func newUniversalProtoFileForV1(v1File *modulev1.File) *universalProtoFile {
	return &universalProtoFile{
		Path:    v1File.Path,
		Content: v1File.Content,
	}
}

func newUniversalProtoFileForV1Beta1(v1beta1File *modulev1beta1.File) *universalProtoFile {
	return &universalProtoFile{
		Path:    v1beta1File.Path,
		Content: v1beta1File.Content,
	}
}

func universalProtoFilesToBucket(universalProtoFiles []*universalProtoFile) (storage.ReadBucket, error) {
	pathToData := make(map[string][]byte, len(universalProtoFiles))
	for _, universalProtoFile := range universalProtoFiles {
		pathToData[universalProtoFile.Path] = universalProtoFile.Content
	}
	return storagemem.NewReadBucket(pathToData)
}

func universalProtoFileToObjectData(universalProtoFile *universalProtoFile) (bufmodule.ObjectData, error) {
	if universalProtoFile == nil {
		return nil, nil
	}
	return bufmodule.NewObjectData(normalpath.Base(universalProtoFile.Path), universalProtoFile.Content)
}
