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
	"fmt"
	"strings"
	"time"

	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/prototime"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type modulePin struct {
	remote     string
	owner      string
	repository string
	branch     string
	commit     string
	digest     string
	createTime time.Time
}

func newModulePin(
	remote string,
	owner string,
	repository string,
	branch string,
	commit string,
	digest string,
	createTime time.Time,
) (*modulePin, error) {
	protoCreateTime, err := prototime.NewTimestamp(createTime)
	if err != nil {
		return nil, err
	}
	return newModulePinForProto(
		&modulev1alpha1.ModulePin{
			Remote:         remote,
			Owner:          owner,
			Repository:     repository,
			Branch:         branch,
			Commit:         commit,
			ManifestDigest: digest,
			CreateTime:     protoCreateTime,
		},
	)
}

type newModulePinForStringOptions struct {
	branch     string
	digest     string
	createTime time.Time
}

func newModulePinForString(
	pin string,
	opts ...NewModulePinForStringOption,
) (*modulePin, error) {
	config := newModulePinForStringOptions{branch: Main}
	for _, option := range opts {
		option(&config)
	}
	// Pin string representation: remote/owner/repository:commit
	pinParts := strings.Split(pin, "/")
	if len(pinParts) != 3 {
		return nil, fmt.Errorf("invalid pin, expected 3 parts when splitting remote, owner and repository")
	}
	remote, owner, repoWithCommit := pinParts[0], pinParts[1], pinParts[2]
	repoWithCommitParts := strings.Split(repoWithCommit, ":")
	if len(repoWithCommitParts) != 2 {
		return nil, fmt.Errorf("invalid pin, expected 2 parts when splitting repository and commit")
	}
	repository, commit := repoWithCommitParts[0], repoWithCommitParts[1]
	if err := validateRemote(remote); err != nil {
		return nil, err
	}
	if err := ValidateOwner(owner, "owner"); err != nil {
		return nil, err
	}
	if err := ValidateRepository(repository); err != nil {
		return nil, err
	}
	if err := ValidateCommit(commit); err != nil {
		return nil, err
	}
	return &modulePin{
		remote:     remote,
		owner:      owner,
		repository: repository,
		commit:     commit,
		branch:     config.branch,
		digest:     config.digest,
		createTime: config.createTime,
	}, nil
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
		branch:     protoModulePin.Branch,
		commit:     protoModulePin.Commit,
		digest:     protoModulePin.ManifestDigest,
		createTime: protoModulePin.CreateTime.AsTime(),
	}, nil
}

func newProtoModulePinForModulePin(
	modulePin ModulePin,
) *modulev1alpha1.ModulePin {
	return &modulev1alpha1.ModulePin{
		Remote:         modulePin.Remote(),
		Owner:          modulePin.Owner(),
		Repository:     modulePin.Repository(),
		Branch:         modulePin.Branch(),
		Commit:         modulePin.Commit(),
		ManifestDigest: modulePin.Digest(),
		// no need to validate as we already know this is valid
		CreateTime: timestamppb.New(modulePin.CreateTime()),
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

func (m *modulePin) Branch() string {
	return m.branch
}

func (m *modulePin) Commit() string {
	return m.commit
}

func (m *modulePin) Digest() string {
	return m.digest
}

func (m *modulePin) CreateTime() time.Time {
	return m.createTime
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
