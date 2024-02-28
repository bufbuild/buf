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

package bufimage

import (
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/gofrs/uuid/v5"
)

type moduleImageFileInfo struct {
	bufmodule.FileInfo
}

func newModuleImageFileInfo(fileInfo bufmodule.FileInfo) *moduleImageFileInfo {
	return &moduleImageFileInfo{
		FileInfo: fileInfo,
	}
}

func (p *moduleImageFileInfo) ModuleFullName() bufmodule.ModuleFullName {
	return p.FileInfo.Module().ModuleFullName()
}

func (p *moduleImageFileInfo) CommitID() uuid.UUID {
	return p.FileInfo.Module().CommitID()
}

func (p *moduleImageFileInfo) Imports() ([]string, error) {
	return p.FileInfo.ProtoFileImports()
}

func (p *moduleImageFileInfo) IsImport() bool {
	return !p.FileInfo.IsTargetFile()
}

func (*moduleImageFileInfo) isImageFileInfo() {}
