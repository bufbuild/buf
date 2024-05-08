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
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/gofrs/uuid/v5"
)

type wellKnownTypeImageFileInfo struct {
	storage.ObjectInfo
	imports  []string
	isImport bool
}

func newWellKnownTypeImageFileInfo(
	objectInfo storage.ObjectInfo,
	imports []string,
	isImport bool,
) *wellKnownTypeImageFileInfo {
	return &wellKnownTypeImageFileInfo{
		ObjectInfo: objectInfo,
		imports:    imports,
		isImport:   isImport,
	}
}

func (p *wellKnownTypeImageFileInfo) ModuleFullName() bufmodule.ModuleFullName {
	return nil
}

func (p *wellKnownTypeImageFileInfo) CommitID() uuid.UUID {
	return uuid.Nil
}

func (p *wellKnownTypeImageFileInfo) Imports() ([]string, error) {
	return p.imports, nil
}

func (p *wellKnownTypeImageFileInfo) IsImport() bool {
	return p.isImport
}

func (*wellKnownTypeImageFileInfo) isImageFileInfo() {}
