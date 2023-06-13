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
	"github.com/bufbuild/buf/private/bufpkg/bufconnect"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
)

type moduleReference struct {
	remote     string
	owner      string
	repository string
	reference  string
}

func newModuleReference(
	remote string,
	owner string,
	repository string,
	reference string,
) (*moduleReference, error) {
	protoModuleReference := &modulev1alpha1.ModuleReference{
		Remote:     remote,
		Owner:      owner,
		Repository: repository,
		Reference:  reference,
	}
	return newModuleReferenceForProto(protoModuleReference)
}

func newModuleReferenceForProto(
	protoModuleReference *modulev1alpha1.ModuleReference,
) (*moduleReference, error) {
	if err := ValidateProtoModuleReference(protoModuleReference); err != nil {
		return nil, err
	}
	return &moduleReference{
		remote:     protoModuleReference.Remote,
		owner:      protoModuleReference.Owner,
		repository: protoModuleReference.Repository,
		reference:  protoModuleReference.Reference,
	}, nil
}

func newProtoModuleReferenceForModuleReference(
	moduleReference ModuleReference,
) *modulev1alpha1.ModuleReference {
	// no need to validate as we know we have a valid ModuleReference constructed
	// by this package due to the private interface
	return &modulev1alpha1.ModuleReference{
		Remote:     moduleReference.Remote(),
		Owner:      moduleReference.Owner(),
		Repository: moduleReference.Repository(),
		Reference:  moduleReference.Reference(),
	}
}

func (m *moduleReference) Remote() string {
	return m.remote
}

func (m *moduleReference) Owner() string {
	return m.owner
}

func (m *moduleReference) Repository() string {
	return m.repository
}

func (m *moduleReference) Reference() string {
	return m.reference
}

func (m *moduleReference) String() string {
	if m.reference == Main {
		return m.remote + "/" + m.owner + "/" + m.repository
	}
	return m.remote + "/" + m.owner + "/" + m.repository + ":" + m.reference
}

func (m *moduleReference) IdentityString() string {
	return m.remote + "/" + m.owner + "/" + m.repository
}

// DiscoverRemote returns the first remote in a list of dependencies that is not "buf.build", if
// any. If not, it defaults to "buf.build". It skips any nil reference. It's useful for targeting
// single-tenant BSR addresses.
func DiscoverRemote(refs []ModuleReference) string {
	for _, ref := range refs {
		if ref == nil {
			continue
		}
		if ref.Remote() != bufconnect.DefaultRemote {
			return ref.Remote()
		}
	}
	return bufconnect.DefaultRemote
}

func (*moduleReference) isModuleOwner()     {}
func (*moduleReference) isModuleIdentity()  {}
func (*moduleReference) isModuleReference() {}
