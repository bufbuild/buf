// Copyright 2020-2021 Buf Technologies, Inc.
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

	modulev1alpha1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/alpha/module/v1alpha1"
)

type moduleReference struct {
	remote     string
	owner      string
	repository string
	branch     string
	commit     string
}

func newModuleReference(
	remote string,
	owner string,
	repository string,
	branch string,
	commit string,
) (*moduleReference, error) {
	protoModuleReference := &modulev1alpha1.ModuleReference{
		Remote:     remote,
		Owner:      owner,
		Repository: repository,
	}
	switch {
	case branch != "" && commit == "":
		protoModuleReference.Reference = &modulev1alpha1.ModuleReference_Branch{
			Branch: branch,
		}
	case branch == "" && commit != "":
		protoModuleReference.Reference = &modulev1alpha1.ModuleReference_Commit{
			Commit: commit,
		}
	case branch != "" && commit != "":
		return nil, fmt.Errorf("module reference cannot have both a branch and commit")
	}
	// validates that exactly one of branch or commit is set
	return newModuleReferenceForProto(protoModuleReference)
}

func newModuleReferenceForProto(
	protoModuleReference *modulev1alpha1.ModuleReference,
) (*moduleReference, error) {
	// validates that exactly one of branch or commit is set
	if err := ValidateProtoModuleReference(protoModuleReference); err != nil {
		return nil, err
	}
	return &moduleReference{
		remote:     protoModuleReference.Remote,
		owner:      protoModuleReference.Owner,
		repository: protoModuleReference.Repository,
		branch:     protoModuleReference.GetBranch(),
		commit:     protoModuleReference.GetCommit(),
	}, nil
}

func newProtoModuleReferenceForModuleReference(
	moduleReference ModuleReference,
) *modulev1alpha1.ModuleReference {
	// no need to validate as we know we have a valid ModuleReference constructed
	// by this package due to the private interface
	protoModuleReference := &modulev1alpha1.ModuleReference{
		Remote:     moduleReference.Remote(),
		Owner:      moduleReference.Owner(),
		Repository: moduleReference.Repository(),
	}
	branch := moduleReference.Branch()
	commit := moduleReference.Commit()
	switch {
	case branch != "" && commit == "":
		protoModuleReference.Reference = &modulev1alpha1.ModuleReference_Branch{
			Branch: branch,
		}
	case branch == "" && commit != "":
		protoModuleReference.Reference = &modulev1alpha1.ModuleReference_Commit{
			Commit: commit,
		}
	}
	return protoModuleReference
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

func (m *moduleReference) Branch() string {
	return m.branch
}

func (m *moduleReference) Commit() string {
	return m.commit
}

func (m *moduleReference) String() string {
	ref := m.branch
	if ref == "" {
		ref = m.commit
	}
	return m.remote + "/" + m.owner + "/" + m.repository + ":" + ref
}

func (m *moduleReference) IdentityString() string {
	return m.remote + "/" + m.owner + "/" + m.repository
}

func (*moduleReference) isModuleOwner()     {}
func (*moduleReference) isModuleIdentity()  {}
func (*moduleReference) isModuleReference() {}
