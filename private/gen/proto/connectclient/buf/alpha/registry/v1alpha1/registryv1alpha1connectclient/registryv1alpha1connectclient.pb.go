// Code generated by protoc-gen-go-connectclient. DO NOT EDIT.

package registryv1alpha1connectclient

import (
	registryv1alpha1api "github.com/bufbuild/buf/private/gen/proto/api/buf/alpha/registry/v1alpha1/registryv1alpha1api"
	connect_go "github.com/bufbuild/connect-go"
)

func NewAdminServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.AdminService {
	return newAdminServiceClient(client, address, options...)
}

func NewAuthnServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.AuthnService {
	return newAuthnServiceClient(client, address, options...)
}

func NewAuthzServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.AuthzService {
	return newAuthzServiceClient(client, address, options...)
}

func NewConvertServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.ConvertService {
	return newConvertServiceClient(client, address, options...)
}

func NewDisplayServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.DisplayService {
	return newDisplayServiceClient(client, address, options...)
}

func NewDocServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.DocService {
	return newDocServiceClient(client, address, options...)
}

func NewDownloadServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.DownloadService {
	return newDownloadServiceClient(client, address, options...)
}

func NewGenerateServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.GenerateService {
	return newGenerateServiceClient(client, address, options...)
}

func NewGithubServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.GithubService {
	return newGithubServiceClient(client, address, options...)
}

func NewImageServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.ImageService {
	return newImageServiceClient(client, address, options...)
}

func NewJSONSchemaServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.JSONSchemaService {
	return newJSONSchemaServiceClient(client, address, options...)
}

func NewLocalResolveServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.LocalResolveService {
	return newLocalResolveServiceClient(client, address, options...)
}

func NewOrganizationServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.OrganizationService {
	return newOrganizationServiceClient(client, address, options...)
}

func NewOwnerServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.OwnerService {
	return newOwnerServiceClient(client, address, options...)
}

func NewPluginServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.PluginService {
	return newPluginServiceClient(client, address, options...)
}

func NewPushServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.PushService {
	return newPushServiceClient(client, address, options...)
}

func NewRecommendationServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.RecommendationService {
	return newRecommendationServiceClient(client, address, options...)
}

func NewReferenceServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.ReferenceService {
	return newReferenceServiceClient(client, address, options...)
}

func NewRepositoryBranchServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.RepositoryBranchService {
	return newRepositoryBranchServiceClient(client, address, options...)
}

func NewRepositoryCommitServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.RepositoryCommitService {
	return newRepositoryCommitServiceClient(client, address, options...)
}

func NewRepositoryServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.RepositoryService {
	return newRepositoryServiceClient(client, address, options...)
}

func NewRepositoryTagServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.RepositoryTagService {
	return newRepositoryTagServiceClient(client, address, options...)
}

func NewRepositoryTrackCommitServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.RepositoryTrackCommitService {
	return newRepositoryTrackCommitServiceClient(client, address, options...)
}

func NewRepositoryTrackServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.RepositoryTrackService {
	return newRepositoryTrackServiceClient(client, address, options...)
}

func NewResolveServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.ResolveService {
	return newResolveServiceClient(client, address, options...)
}

func NewSearchServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.SearchService {
	return newSearchServiceClient(client, address, options...)
}

func NewTokenServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.TokenService {
	return newTokenServiceClient(client, address, options...)
}

func NewUserServiceClient(
	client connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) registryv1alpha1api.UserService {
	return newUserServiceClient(client, address, options...)
}
