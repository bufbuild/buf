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
