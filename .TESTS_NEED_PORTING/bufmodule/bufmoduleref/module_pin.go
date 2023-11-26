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

package bufmoduleref

import (
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
)

type modulePin struct {
	remote     string
	owner      string
	repository string
	commit     string
	digest     string
}

func newModulePin(
	remote string,
	owner string,
	repository string,
	commit string,
	digest string,
) (*modulePin, error) {
	return newModulePinForProto(
		&modulev1alpha1.ModulePin{
			Remote:         remote,
			Owner:          owner,
			Repository:     repository,
			Commit:         commit,
			ManifestDigest: digest,
		},
	)
}

func newModulePinForProto(
	protoModulePin *modulev1alpha1.ModulePin,
) (*modulePin, error) {
	if err := ValidateProtoModulePin(protoModulePin); err != nil {
		return nil, err
	}
	return &modulePin{
		remote:     protoModulePin.Remote,
		owner:      protoModulePin.Owner,
		repository: protoModulePin.Repository,
		commit:     protoModulePin.Commit,
		digest:     protoModulePin.ManifestDigest,
	}, nil
}

func newProtoModulePinForModulePin(
	modulePin ModulePin,
) *modulev1alpha1.ModulePin {
	return &modulev1alpha1.ModulePin{
		Remote:         modulePin.Remote(),
		Owner:          modulePin.Owner(),
		Repository:     modulePin.Repository(),
		Commit:         modulePin.Commit(),
		ManifestDigest: modulePin.Digest(),
	}
}

func (m *modulePin) Remote() string {
	return m.remote
}

func (m *modulePin) Owner() string {
	return m.owner
}

func (m *modulePin) Repository() string {
	return m.repository
}

func (m *modulePin) Commit() string {
	return m.commit
}

func (m *modulePin) Digest() string {
	return m.digest
}

func (m *modulePin) String() string {
	return m.remote + "/" + m.owner + "/" + m.repository + ":" + m.commit
}

func (m *modulePin) IdentityString() string {
	return m.remote + "/" + m.owner + "/" + m.repository
}

func (*modulePin) isModuleOwner()    {}
func (*modulePin) isModuleIdentity() {}
func (*modulePin) isModulePin()      {}
