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

type wellKnownTypeImageFileInfo struct {
	path     string
	imports  []string
	isImport bool
}

func newWellKnownTypeImageFileInfo(
	path string,
	imports []string,
	isImport bool,
) *wellKnownTypeImageFileInfo {
	return &wellKnownTypeImageFileInfo{
		path:     path,
		imports:  imports,
		isImport: isImport,
	}
}

func (p *wellKnownTypeImageFileInfo) Path() string {
	return p.path
}

func (p *wellKnownTypeImageFileInfo) ExternalPath() string {
	return p.path
}

func (p *wellKnownTypeImageFileInfo) LocalPath() string {
	return ""
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
