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

package bufimage

import (
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
	"google.golang.org/protobuf/types/descriptorpb"
)

type ImageFile interface {
	storage.ObjectInfo

	// ModuleFullName returns the full name of the Module that this ImageFile came from,
	// if the ImageFile came from a Module (as opposed to a serialized Protobuf message),
	// and if the ModuleFullName was known.
	//
	// May be nil. Callers should not rely on this value being present.
	ModuleFullName() bufmodule.ModuleFullName
	// CommitID returns the BSR ID of the Commit of the Module that this ImageFile came from.
	// if the ImageFile came from a Module (as opposed to a serialized Protobuf message), and
	// if the CommitID was known..
	//
	// May be empty. Callers should not rely on this value being present. If
	// ModuleFullName is nil, this will always be empty.
	CommitID() string

	FileDescriptorProto() *descriptorpb.FileDescriptorProto
	IsImport() bool
	IsSyntaxUnspecified() bool
	UnusedDependencyIndexes() int32

	isImageFile()
}

type Image interface {
	Files() []ImageFile
	GetFile(path string) ImageFile

	isImage()
}

type ImageSet interface {
	Images() []Image
}

type ImageWorkspace interface {
	ImageSet() ImageSet
	//Config() Config
}
