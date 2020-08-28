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
	server     string
	owner      string
	repository string
	version    string
	digest     string
}

func newModuleName(
	server string,
	owner string,
	repository string,
	version string,
	digest string,
) (*moduleName, error) {
	// we get all the validation logic from protoc-gen-validate
	// that we also want to apply here, so we just do this
	// this is a little hacky but better than replicating the logic for now
	return newModuleNameForProto(
		&modulev1.ModuleName{
			Server:     server,
			Owner:      owner,
			Repository: repository,
			Version:    version,
			Digest:     digest,
		},
	)
}

func newModuleNameForProto(
	protoModuleName *modulev1.ModuleName,
) (*moduleName, error) {
	if err := validateProtoModuleName(protoModuleName); err != nil {
		return nil, err
	}
	return &moduleName{
		server:     protoModuleName.Server,
		owner:      protoModuleName.Owner,
		repository: protoModuleName.Repository,
		version:    protoModuleName.Version,
		digest:     protoModuleName.Digest,
	}, nil
}

func (m *moduleName) Server() string {
	return m.server
}

func (m *moduleName) Owner() string {
	return m.owner
}

func (m *moduleName) Repository() string {
	return m.repository
}

func (m *moduleName) Version() string {
	return m.version
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
) (*modulev1.ModuleName, error) {
	protoModuleName := &modulev1.ModuleName{
		Server:     moduleName.Server(),
		Owner:      moduleName.Owner(),
		Repository: moduleName.Repository(),
		Version:    moduleName.Version(),
		Digest:     moduleName.Digest(),
	}
	if err := validateProtoModuleName(protoModuleName); err != nil {
		return nil, err
	}
	return protoModuleName, nil
}

// moduleNameIdentity returns the given module name's identity. This is
// the string representation, minus the digest.
func moduleNameIdentity(moduleName ModuleName) string {
	return fmt.Sprintf("%s/%s/%s/%s", moduleName.Server(), moduleName.Owner(), moduleName.Repository(), moduleName.Version())
}
