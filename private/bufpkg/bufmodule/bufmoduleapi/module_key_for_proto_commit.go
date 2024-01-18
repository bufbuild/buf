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

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
)

func getModuleKeyForProtoCommit(
	ctx context.Context,
	protoModuleProvider *protoModuleProvider,
	protoOwnerProvider *protoOwnerProvider,
	registry string,
	protoCommit *modulev1beta1.Commit,
) (bufmodule.ModuleKey, error) {
	protoModule, err := protoModuleProvider.getProtoModuleForModuleID(
		ctx,
		registry,
		protoCommit.ModuleId,
	)
	if err != nil {
		return nil, err
	}
	protoOwner, err := protoOwnerProvider.getProtoOwnerForOwnerID(
		ctx,
		registry,
		protoCommit.OwnerId,
	)
	if err != nil {
		return nil, err
	}
	var ownerName string
	switch {
	case protoOwner.GetUser() != nil:
		ownerName = protoOwner.GetUser().Name
	case protoOwner.GetOrganization() != nil:
		ownerName = protoOwner.GetOrganization().Name
	default:
		return nil, fmt.Errorf("proto Owner did not have a User or Organization: %v", protoOwner)
	}
	moduleFullName, err := bufmodule.NewModuleFullName(
		registry,
		ownerName,
		protoModule.Name,
	)
	if err != nil {
		return nil, err
	}
	commitID, err := ProtoToCommitID(protoCommit.Id)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleKey(
		moduleFullName,
		commitID,
		func() (bufmodule.Digest, error) {
			return ProtoToDigest(protoCommit.Digest)
		},
	)
}
