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

package bufapi

import (
	"buf.build/gen/go/bufbuild/registry/connectrpc/go/buf/registry/module/v1beta1/modulev1beta1connect"
	"buf.build/gen/go/bufbuild/registry/connectrpc/go/buf/registry/owner/v1beta1/ownerv1beta1connect"
)

// TODO: It'd be great if we could detect if we do immutable get by ID queries, and then have a LRU
// cache for those entities. Thinking owners.

type ClientProvider interface {
	BranchServiceClient(registryHostname string) modulev1beta1connect.BranchServiceClient
	CommitServiceClient(registryHostname string) modulev1beta1connect.CommitServiceClient
	ModuleServiceClient(registryHostname string) modulev1beta1connect.ModuleServiceClient
	OrganizationServiceClient(registryHostname string) ownerv1beta1connect.OrganizationServiceClient
	OwnerServiceClient(registryHostname string) ownerv1beta1connect.OwnerServiceClient
	TagServiceClient(registryHostname string) modulev1beta1connect.TagServiceClient
	UserServiceClient(registryHostname string) ownerv1beta1connect.UserServiceClient
	VCSCommitServiceClient(registryHostname string) modulev1beta1connect.VCSCommitServiceClient
}
