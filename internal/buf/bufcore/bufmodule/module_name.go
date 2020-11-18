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
	"fmt"

	modulev1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/module/v1"
)

// moduleName implements the ModuleName interface.
type moduleName struct {
	remote     string
	owner      string
	repository string
	track      string
	digest     string
}

func newModuleName(
	remote string,
	owner string,
	repository string,
	track string,
	digest string,
) (*moduleName, error) {
	return newModuleNameForProto(
		&modulev1.ModuleName{
			Remote:     remote,
			Owner:      owner,
			Repository: repository,
			Track:      track,
			Digest:     digest,
		},
	)
}

func newModuleNameForProto(
	protoModuleName *modulev1.ModuleName,
) (*moduleName, error) {
	if err := ValidateProtoModuleName(protoModuleName); err != nil {
		return nil, err
	}
	return &moduleName{
		remote:     protoModuleName.Remote,
		owner:      protoModuleName.Owner,
		repository: protoModuleName.Repository,
		track:      protoModuleName.Track,
		digest:     protoModuleName.Digest,
	}, nil
}

func (m *moduleName) Remote() string {
	return m.remote
}

func (m *moduleName) Owner() string {
	return m.owner
}

func (m *moduleName) Repository() string {
	return m.repository
}

func (m *moduleName) Track() string {
	return m.track
}

func (m *moduleName) Digest() string {
	return m.digest
}

func (m *moduleName) String() string {
	base := moduleNameIdentity(m)
	if m.digest == "" {
		return base
	}
	return fmt.Sprintf("%s:%s", base, m.digest)
}

func (m *moduleName) isModuleName() {}

func newProtoModuleNameForModuleName(
	moduleName ModuleName,
) *modulev1.ModuleName {
	return &modulev1.ModuleName{
		Remote:     moduleName.Remote(),
		Owner:      moduleName.Owner(),
		Repository: moduleName.Repository(),
		Track:      moduleName.Track(),
		Digest:     moduleName.Digest(),
	}
}

// moduleNameIdentity returns the given module name's identity. This is
// the string representation, minus the digest.
func moduleNameIdentity(moduleName ModuleName) string {
	return fmt.Sprintf("%s/%s/%s/%s", moduleName.Remote(), moduleName.Owner(), moduleName.Repository(), moduleName.Track())
}
