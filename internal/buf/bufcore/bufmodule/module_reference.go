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

	modulev1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/module/v1"
)

type moduleReference struct {
	remote     string
	owner      string
	repository string
	track      string
	commit     string
}

func newModuleReference(
	remote string,
	owner string,
	repository string,
	track string,
	commit string,
) (*moduleReference, error) {
	protoModuleReference := &modulev1.ModuleReference{
		Remote:     remote,
		Owner:      owner,
		Repository: repository,
	}
	switch {
	case track != "" && commit == "":
		protoModuleReference.Reference = &modulev1.ModuleReference_Track{
			Track: track,
		}
	case track == "" && commit != "":
		protoModuleReference.Reference = &modulev1.ModuleReference_Commit{
			Commit: commit,
		}
	case track != "" && commit != "":
		return nil, fmt.Errorf("module reference cannot have both a track and commit")
	}
	// validates that exactly one of track or commit is set
	return newModuleReferenceForProto(protoModuleReference)
}

func newModuleReferenceForProto(
	protoModuleReference *modulev1.ModuleReference,
) (*moduleReference, error) {
	// validates that exactly one of track or commit is set
	if err := ValidateProtoModuleReference(protoModuleReference); err != nil {
		return nil, err
	}
	return &moduleReference{
		remote:     protoModuleReference.Remote,
		owner:      protoModuleReference.Owner,
		repository: protoModuleReference.Repository,
		track:      protoModuleReference.GetTrack(),
		commit:     protoModuleReference.GetCommit(),
	}, nil
}

func newProtoModuleReferenceForModuleReference(
	moduleReference ModuleReference,
) *modulev1.ModuleReference {
	// no need to validate as we know we have a valid ModuleReference constructed
	// by this package due to the private interface
	protoModuleReference := &modulev1.ModuleReference{
		Remote:     moduleReference.Remote(),
		Owner:      moduleReference.Owner(),
		Repository: moduleReference.Repository(),
	}
	track := moduleReference.Track()
	commit := moduleReference.Commit()
	switch {
	case track != "" && commit == "":
		protoModuleReference.Reference = &modulev1.ModuleReference_Track{
			Track: track,
		}
	case track == "" && commit != "":
		protoModuleReference.Reference = &modulev1.ModuleReference_Commit{
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

func (m *moduleReference) Track() string {
	return m.track
}

func (m *moduleReference) Commit() string {
	return m.commit
}

func (m *moduleReference) String() string {
	if m.track != "" {
		return m.remote + "/" + m.owner + "/" + m.repository + "/" + m.track
	}
	return m.remote + "/" + m.owner + "/" + m.repository + "@" + m.commit
}

func (m *moduleReference) identity() string {
	return m.remote + "/" + m.owner + "/" + m.repository
}

func (*moduleReference) isModuleIdentity()  {}
func (*moduleReference) isModuleReference() {}
