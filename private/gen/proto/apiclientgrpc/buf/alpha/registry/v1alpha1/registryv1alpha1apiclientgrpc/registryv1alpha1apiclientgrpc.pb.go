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

// Code generated by protoc-gen-go-apiclientgrpc. DO NOT EDIT.

package registryv1alpha1apiclientgrpc

import (
	context "context"
	registryv1alpha1api "github.com/bufbuild/buf/private/gen/proto/api/buf/alpha/registry/v1alpha1/registryv1alpha1api"
	registryv1alpha1apiclient "github.com/bufbuild/buf/private/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	grpcclient "github.com/bufbuild/buf/private/pkg/transport/grpc/grpcclient"
	zap "go.uber.org/zap"
)

// NewProvider returns a new Provider.
func NewProvider(
	logger *zap.Logger,
	clientConnProvider grpcclient.ClientConnProvider,
	options ...ProviderOption,
) registryv1alpha1apiclient.Provider {
	provider := &provider{
		logger:             logger,
		clientConnProvider: clientConnProvider,
	}
	for _, option := range options {
		option(provider)
	}
	return provider
}

type provider struct {
	logger                  *zap.Logger
	clientConnProvider      grpcclient.ClientConnProvider
	addressMapper           func(string) string
	contextModifierProvider func(string) (func(context.Context) context.Context, error)
}

// ProviderOption is an option for a new Provider.
type ProviderOption func(*provider)

// WithAddressMapper maps the address with the given function.
func WithAddressMapper(addressMapper func(string) string) ProviderOption {
	return func(provider *provider) {
		provider.addressMapper = addressMapper
	}
}

// WithContextModifierProvider provides a function that  modifies the context before every RPC invocation.
// Applied before the address mapper.
func WithContextModifierProvider(contextModifierProvider func(address string) (func(context.Context) context.Context, error)) ProviderOption {
	return func(provider *provider) {
		provider.contextModifierProvider = contextModifierProvider
	}
}

func (p *provider) NewAuditLogsService(ctx context.Context, address string) (registryv1alpha1api.AuditLogsService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &auditLogsService{
		logger:          p.logger,
		client:          v1alpha1.NewAuditLogsServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewAuthnService(ctx context.Context, address string) (registryv1alpha1api.AuthnService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &authnService{
		logger:          p.logger,
		client:          v1alpha1.NewAuthnServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewAuthzService(ctx context.Context, address string) (registryv1alpha1api.AuthzService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &authzService{
		logger:          p.logger,
		client:          v1alpha1.NewAuthzServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewDisplayService(ctx context.Context, address string) (registryv1alpha1api.DisplayService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &displayService{
		logger:          p.logger,
		client:          v1alpha1.NewDisplayServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewDocService(ctx context.Context, address string) (registryv1alpha1api.DocService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &docService{
		logger:          p.logger,
		client:          v1alpha1.NewDocServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewDownloadService(ctx context.Context, address string) (registryv1alpha1api.DownloadService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &downloadService{
		logger:          p.logger,
		client:          v1alpha1.NewDownloadServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewGenerateService(ctx context.Context, address string) (registryv1alpha1api.GenerateService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &generateService{
		logger:          p.logger,
		client:          v1alpha1.NewGenerateServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewImageService(ctx context.Context, address string) (registryv1alpha1api.ImageService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &imageService{
		logger:          p.logger,
		client:          v1alpha1.NewImageServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewJSONSchemaService(ctx context.Context, address string) (registryv1alpha1api.JSONSchemaService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &jSONSchemaService{
		logger:          p.logger,
		client:          v1alpha1.NewJSONSchemaServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewLocalResolveService(ctx context.Context, address string) (registryv1alpha1api.LocalResolveService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &localResolveService{
		logger:          p.logger,
		client:          v1alpha1.NewLocalResolveServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewOrganizationService(ctx context.Context, address string) (registryv1alpha1api.OrganizationService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &organizationService{
		logger:          p.logger,
		client:          v1alpha1.NewOrganizationServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewOwnerService(ctx context.Context, address string) (registryv1alpha1api.OwnerService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &ownerService{
		logger:          p.logger,
		client:          v1alpha1.NewOwnerServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewPluginService(ctx context.Context, address string) (registryv1alpha1api.PluginService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &pluginService{
		logger:          p.logger,
		client:          v1alpha1.NewPluginServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewPushService(ctx context.Context, address string) (registryv1alpha1api.PushService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &pushService{
		logger:          p.logger,
		client:          v1alpha1.NewPushServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewRecommendationService(ctx context.Context, address string) (registryv1alpha1api.RecommendationService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &recommendationService{
		logger:          p.logger,
		client:          v1alpha1.NewRecommendationServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewReferenceService(ctx context.Context, address string) (registryv1alpha1api.ReferenceService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &referenceService{
		logger:          p.logger,
		client:          v1alpha1.NewReferenceServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewRepositoryBranchService(ctx context.Context, address string) (registryv1alpha1api.RepositoryBranchService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &repositoryBranchService{
		logger:          p.logger,
		client:          v1alpha1.NewRepositoryBranchServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewRepositoryCommitService(ctx context.Context, address string) (registryv1alpha1api.RepositoryCommitService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &repositoryCommitService{
		logger:          p.logger,
		client:          v1alpha1.NewRepositoryCommitServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewRepositoryService(ctx context.Context, address string) (registryv1alpha1api.RepositoryService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &repositoryService{
		logger:          p.logger,
		client:          v1alpha1.NewRepositoryServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewRepositoryTagService(ctx context.Context, address string) (registryv1alpha1api.RepositoryTagService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &repositoryTagService{
		logger:          p.logger,
		client:          v1alpha1.NewRepositoryTagServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewRepositoryTrackService(ctx context.Context, address string) (registryv1alpha1api.RepositoryTrackService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &repositoryTrackService{
		logger:          p.logger,
		client:          v1alpha1.NewRepositoryTrackServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewResolveService(ctx context.Context, address string) (registryv1alpha1api.ResolveService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &resolveService{
		logger:          p.logger,
		client:          v1alpha1.NewResolveServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewSearchService(ctx context.Context, address string) (registryv1alpha1api.SearchService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &searchService{
		logger:          p.logger,
		client:          v1alpha1.NewSearchServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewTokenService(ctx context.Context, address string) (registryv1alpha1api.TokenService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &tokenService{
		logger:          p.logger,
		client:          v1alpha1.NewTokenServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewUserService(ctx context.Context, address string) (registryv1alpha1api.UserService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(address)
		if err != nil {
			return nil, err
		}
	}
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	clientConn, err := p.clientConnProvider.NewClientConn(ctx, address)
	if err != nil {
		return nil, err
	}
	return &userService{
		logger:          p.logger,
		client:          v1alpha1.NewUserServiceClient(clientConn),
		contextModifier: contextModifier,
	}, nil
}
