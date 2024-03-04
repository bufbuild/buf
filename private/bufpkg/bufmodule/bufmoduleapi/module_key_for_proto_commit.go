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

package bufmoduleapi

import (
	"context"
	"fmt"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/gofrs/uuid/v5"
)

func getModuleKeyForV1ProtoCommit(
	ctx context.Context,
	v1ProtoModuleProvider *v1ProtoModuleProvider,
	v1ProtoOwnerProvider *v1ProtoOwnerProvider,
	registry string,
	v1ProtoCommit *modulev1.Commit,
) (bufmodule.ModuleKey, error) {
	moduleFullName, err := getModuleFullNameForRegistryProtoOwnerIdProtoModuleId(
		ctx,
		v1ProtoModuleProvider,
		v1ProtoOwnerProvider,
		v1Beta1ProtoCommit.OwnerId,
		v1ProtoCommit.ModuleId,
	)
	if err != nil {
		return nil, err
	}
	commitID, err := uuid.FromString(v1ProtoCommit.Id)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleKey(
		moduleFullName,
		commitID,
		func() (bufmodule.Digest, error) {
			return V1ProtoToDigest(v1ProtoCommit.Digest)
		},
	)
}

func getModuleKeyForV1Beta1ProtoCommit(
	ctx context.Context,
	v1ProtoModuleProvider *v1ProtoModuleProvider,
	v1ProtoOwnerProvider *v1ProtoOwnerProvider,
	registry string,
	v1beta1ProtoCommit *modulev1beta1.Commit,
) (bufmodule.ModuleKey, error) {
	moduleFullName, err := getModuleFullNameForRegistryProtoOwnerIdProtoModuleId(
		ctx,
		v1ProtoModuleProvider,
		v1ProtoOwnerProvider,
		v1Beta1ProtoCommit.OwnerId,
		v1beta1ProtoCommit.ModuleId,
	)
	if err != nil {
		return nil, err
	}
	commitID, err := uuid.FromString(v1beta1ProtoCommit.Id)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleKey(
		moduleFullName,
		commitID,
		func() (bufmodule.Digest, error) {
			return V1Beta1ProtoToDigest(v1beta1ProtoCommit.Digest)
		},
	)
}

func getModuleFullNameForRegistryProtoOwnerIdProtoModuleId(
	ctx context.Context,
	v1ProtoModuleProvider *v1ProtoModuleProvider,
	v1ProtoOwnerProvider *v1ProtoOwnerProvider,
	registry string,
	protoOwnerID string,
	protoModuleID string,
) (bufmodule.ModuleFullName, error) {
	v1ProtoModule, err := v1ProtoModuleProvider.getV1ProtoModuleForModuleID(
		ctx,
		registry,
		protoModuleID,
	)
	if err != nil {
		return nil, err
	}
	v1ProtoOwner, err := v1ProtoOwnerProvider.getV1ProtoOwnerForOwnerID(
		ctx,
		registry,
		protoOwnerID,
	)
	if err != nil {
		return nil, err
	}
	var ownerName string
	switch {
	case v1ProtoOwner.GetUser() != nil:
		ownerName = v1ProtoOwner.GetUser().Name
	case v1ProtoOwner.GetOrganization() != nil:
		ownerName = v1ProtoOwner.GetOrganization().Name
	default:
		return nil, fmt.Errorf("proto Owner did not have a User or Organization: %v", v1ProtoOwner)
	}
	return bufmodule.NewModuleFullName(
		registry,
		ownerName,
		v1ProtoModule.Name,
	)
}
