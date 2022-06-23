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

	bufconnect "github.com/bufbuild/buf/private/bufpkg/bufconnect"
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
	logger        *zap.Logger
	httpClient    connect_go.HTTPClient
	addressMapper func(string) string
	token         string
	tokenReader   func(string) (string, error)
	interceptors  []connect_go.Interceptor
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

func WithToken(token string) ProviderOption {
	return func(provider *provider) {
		provider.token = token
	}
}

// WithTokenReader invokes a given function to lookup a token based on a given address
// Useful for looking up token during client construction
// If a token is explicitly provided via WithToken, then this option is ignored
func WithTokenReader(tokenReader func(string) (string, error)) ProviderOption {
	return func(provider *provider) {
		provider.tokenReader = tokenReader
	}
}

func (p *provider) NewAdminService(ctx context.Context, address string) (registryv1alpha1api.AdminService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewAuthnService(ctx context.Context, address string) (registryv1alpha1api.AuthnService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewAuthzService(ctx context.Context, address string) (registryv1alpha1api.AuthzService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewConvertService(ctx context.Context, address string) (registryv1alpha1api.ConvertService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewDisplayService(ctx context.Context, address string) (registryv1alpha1api.DisplayService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewDocService(ctx context.Context, address string) (registryv1alpha1api.DocService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewDownloadService(ctx context.Context, address string) (registryv1alpha1api.DownloadService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewGenerateService(ctx context.Context, address string) (registryv1alpha1api.GenerateService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewGithubService(ctx context.Context, address string) (registryv1alpha1api.GithubService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewImageService(ctx context.Context, address string) (registryv1alpha1api.ImageService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewJSONSchemaService(ctx context.Context, address string) (registryv1alpha1api.JSONSchemaService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewLocalResolveService(ctx context.Context, address string) (registryv1alpha1api.LocalResolveService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewOrganizationService(ctx context.Context, address string) (registryv1alpha1api.OrganizationService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewOwnerService(ctx context.Context, address string) (registryv1alpha1api.OwnerService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewPluginService(ctx context.Context, address string) (registryv1alpha1api.PluginService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewPushService(ctx context.Context, address string) (registryv1alpha1api.PushService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewRecommendationService(ctx context.Context, address string) (registryv1alpha1api.RecommendationService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewReferenceService(ctx context.Context, address string) (registryv1alpha1api.ReferenceService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewRepositoryBranchService(ctx context.Context, address string) (registryv1alpha1api.RepositoryBranchService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &repositoryBranchServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewRepositoryBranchServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

func (p *provider) NewRepositoryCommitService(ctx context.Context, address string) (registryv1alpha1api.RepositoryCommitService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewRepositoryService(ctx context.Context, address string) (registryv1alpha1api.RepositoryService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewRepositoryTagService(ctx context.Context, address string) (registryv1alpha1api.RepositoryTagService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewRepositoryTrackCommitService(ctx context.Context, address string) (registryv1alpha1api.RepositoryTrackCommitService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &repositoryTrackCommitServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewRepositoryTrackCommitServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

func (p *provider) NewRepositoryTrackService(ctx context.Context, address string) (registryv1alpha1api.RepositoryTrackService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	return &repositoryTrackServiceClient{
		logger: p.logger,
		client: registryv1alpha1connect.NewRepositoryTrackServiceClient(
			p.httpClient,
			address,
			connect_go.WithInterceptors(interceptors...),
		),
	}, nil
}

func (p *provider) NewResolveService(ctx context.Context, address string) (registryv1alpha1api.ResolveService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewSearchService(ctx context.Context, address string) (registryv1alpha1api.SearchService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewStudioService(ctx context.Context, address string) (registryv1alpha1api.StudioService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewTokenService(ctx context.Context, address string) (registryv1alpha1api.TokenService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewUserService(ctx context.Context, address string) (registryv1alpha1api.UserService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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

func (p *provider) NewWebhookService(ctx context.Context, address string) (registryv1alpha1api.WebhookService, error) {
	var err error
	token := p.token
	if token == "" && p.tokenReader != nil {
		token, err = p.tokenReader(address)
		if err != nil {
			return nil, err
		}
	}
	tokenInterceptor := bufconnect.NewWithTokenInterceptor(token)
	interceptors := p.interceptors
	interceptors = append(interceptors, tokenInterceptor)
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
