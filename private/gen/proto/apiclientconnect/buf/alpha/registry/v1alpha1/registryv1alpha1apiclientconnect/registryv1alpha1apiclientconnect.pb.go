// Copyright 2020-2022 Buf Technologies, Inc.
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

// Code generated by protoc-gen-go-apiclientconnect. DO NOT EDIT.

package registryv1alpha1apiclientconnect

import (
	context "context"
	registryv1alpha1api "github.com/bufbuild/buf/private/gen/proto/api/buf/alpha/registry/v1alpha1/registryv1alpha1api"
	registryv1alpha1apiclient "github.com/bufbuild/buf/private/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	registryv1alpha1connect "github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	connect_go "github.com/bufbuild/connect-go"
	zap "go.uber.org/zap"
)

// NewProvider returns a new Provider.
func NewProvider(
	logger *zap.Logger,
	httpClient connect_go.HTTPClient,
	options ...ProviderOption,
) registryv1alpha1apiclient.Provider {
	provider := &provider{
		logger:     logger,
		httpClient: httpClient,
	}
	for _, option := range options {
		option(provider)
	}
	return provider
}

type provider struct {
	logger                  *zap.Logger
	httpClient              connect_go.HTTPClient
	addressMapper           func(string) string
	interceptors            []connect_go.Interceptor
	authInterceptorProvider func(string) connect_go.UnaryInterceptorFunc
}

// ProviderOption is an option for a new Provider.
type ProviderOption func(*provider)

// WithAddressMapper maps the address with the given function.
func WithAddressMapper(addressMapper func(string) string) ProviderOption {
	return func(provider *provider) {
		provider.addressMapper = addressMapper
	}
}

// WithInterceptors adds the slice of interceptors to all clients returned from this provider.
func WithInterceptors(interceptors []connect_go.Interceptor) ProviderOption {
	return func(provider *provider) {
		provider.interceptors = interceptors
	}
}

// WithAuthInterceptorProvider configures a provider that, when invoked, returns an interceptor that can be added
// to a client for setting the auth token
func WithAuthInterceptorProvider(authInterceptorProvider func(string) connect_go.UnaryInterceptorFunc) ProviderOption {
	return func(provider *provider) {
		provider.authInterceptorProvider = authInterceptorProvider
	}
}

// NewAdminService creates a new AdminService
func (p *provider) NewAdminService(ctx context.Context, address string) (registryv1alpha1api.AdminService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &adminServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewAdminServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewAuthnService creates a new AuthnService
func (p *provider) NewAuthnService(ctx context.Context, address string) (registryv1alpha1api.AuthnService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &authnServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewAuthnServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewAuthzService creates a new AuthzService
func (p *provider) NewAuthzService(ctx context.Context, address string) (registryv1alpha1api.AuthzService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &authzServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewAuthzServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewConvertService creates a new ConvertService
func (p *provider) NewConvertService(ctx context.Context, address string) (registryv1alpha1api.ConvertService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &convertServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewConvertServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewDisplayService creates a new DisplayService
func (p *provider) NewDisplayService(ctx context.Context, address string) (registryv1alpha1api.DisplayService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &displayServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewDisplayServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewDocService creates a new DocService
func (p *provider) NewDocService(ctx context.Context, address string) (registryv1alpha1api.DocService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &docServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewDocServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewDownloadService creates a new DownloadService
func (p *provider) NewDownloadService(ctx context.Context, address string) (registryv1alpha1api.DownloadService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &downloadServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewDownloadServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewGenerateService creates a new GenerateService
func (p *provider) NewGenerateService(ctx context.Context, address string) (registryv1alpha1api.GenerateService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &generateServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewGenerateServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewGithubService creates a new GithubService
func (p *provider) NewGithubService(ctx context.Context, address string) (registryv1alpha1api.GithubService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &githubServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewGithubServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewImageService creates a new ImageService
func (p *provider) NewImageService(ctx context.Context, address string) (registryv1alpha1api.ImageService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &imageServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewImageServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewJSONSchemaService creates a new JSONSchemaService
func (p *provider) NewJSONSchemaService(ctx context.Context, address string) (registryv1alpha1api.JSONSchemaService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &jSONSchemaServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewJSONSchemaServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewLocalResolveService creates a new LocalResolveService
func (p *provider) NewLocalResolveService(ctx context.Context, address string) (registryv1alpha1api.LocalResolveService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &localResolveServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewLocalResolveServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewOrganizationService creates a new OrganizationService
func (p *provider) NewOrganizationService(ctx context.Context, address string) (registryv1alpha1api.OrganizationService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &organizationServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewOrganizationServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewOwnerService creates a new OwnerService
func (p *provider) NewOwnerService(ctx context.Context, address string) (registryv1alpha1api.OwnerService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &ownerServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewOwnerServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewPluginCodeGenerationService creates a new PluginCodeGenerationService
func (p *provider) NewPluginCodeGenerationService(ctx context.Context, address string) (registryv1alpha1api.PluginCodeGenerationService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &pluginCodeGenerationServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewPluginCodeGenerationServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewPluginCurationService creates a new PluginCurationService
func (p *provider) NewPluginCurationService(ctx context.Context, address string) (registryv1alpha1api.PluginCurationService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &pluginCurationServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewPluginCurationServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewPluginService creates a new PluginService
func (p *provider) NewPluginService(ctx context.Context, address string) (registryv1alpha1api.PluginService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &pluginServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewPluginServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewPushService creates a new PushService
func (p *provider) NewPushService(ctx context.Context, address string) (registryv1alpha1api.PushService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &pushServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewPushServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewRecommendationService creates a new RecommendationService
func (p *provider) NewRecommendationService(ctx context.Context, address string) (registryv1alpha1api.RecommendationService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &recommendationServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewRecommendationServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewReferenceService creates a new ReferenceService
func (p *provider) NewReferenceService(ctx context.Context, address string) (registryv1alpha1api.ReferenceService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &referenceServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewReferenceServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewRepositoryCommitService creates a new RepositoryCommitService
func (p *provider) NewRepositoryCommitService(ctx context.Context, address string) (registryv1alpha1api.RepositoryCommitService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &repositoryCommitServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewRepositoryCommitServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewRepositoryService creates a new RepositoryService
func (p *provider) NewRepositoryService(ctx context.Context, address string) (registryv1alpha1api.RepositoryService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &repositoryServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewRepositoryServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewRepositoryTagService creates a new RepositoryTagService
func (p *provider) NewRepositoryTagService(ctx context.Context, address string) (registryv1alpha1api.RepositoryTagService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &repositoryTagServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewRepositoryTagServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewResolveService creates a new ResolveService
func (p *provider) NewResolveService(ctx context.Context, address string) (registryv1alpha1api.ResolveService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &resolveServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewResolveServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewSearchService creates a new SearchService
func (p *provider) NewSearchService(ctx context.Context, address string) (registryv1alpha1api.SearchService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &searchServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewSearchServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewStudioService creates a new StudioService
func (p *provider) NewStudioService(ctx context.Context, address string) (registryv1alpha1api.StudioService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &studioServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewStudioServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewTokenService creates a new TokenService
func (p *provider) NewTokenService(ctx context.Context, address string) (registryv1alpha1api.TokenService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &tokenServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewTokenServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewUserService creates a new UserService
func (p *provider) NewUserService(ctx context.Context, address string) (registryv1alpha1api.UserService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &userServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewUserServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

// NewWebhookService creates a new WebhookService
func (p *provider) NewWebhookService(ctx context.Context, address string) (registryv1alpha1api.WebhookService, error) {
	interceptors := p.interceptors
	if p.authInterceptorProvider != nil {
		interceptor := p.authInterceptorProvider(address)
		interceptors = append(interceptors, interceptor)
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &webhookServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewWebhookServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}
