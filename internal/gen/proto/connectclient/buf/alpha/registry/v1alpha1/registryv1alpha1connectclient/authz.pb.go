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

// Code generated by protoc-gen-go-connectclient. DO NOT EDIT.

package registryv1alpha1connectclient

import (
	context "context"
	registryv1alpha1connect "github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	connect_go "github.com/bufbuild/connect-go"
)

type authzServiceClient struct {
	client registryv1alpha1connect.AuthzServiceClient
}

func newAuthzServiceClient(
	httpClient connect_go.HTTPClient,
	address string,
	options ...connect_go.ClientOption,
) *authzServiceClient {
	return &authzServiceClient{
		client: registryv1alpha1connect.NewAuthzServiceClient(
			httpClient,
			address,
			options...,
		),
	}
}

// UserCanCreateOrganizationRepository returns whether the user is authorized
// to create repositories in an organization.
func (s *authzServiceClient) UserCanCreateOrganizationRepository(ctx context.Context, organizationId string) (authorized bool, _ error) {
	response, err := s.client.UserCanCreateOrganizationRepository(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanCreateOrganizationRepositoryRequest{
				OrganizationId: organizationId,
			}),
	)
	if err != nil {
		return false, err
	}
	return response.Msg.Authorized, nil
}

// UserCanSeeRepositorySettings returns whether the user is authorized
// to see repository settings.
func (s *authzServiceClient) UserCanSeeRepositorySettings(ctx context.Context, repositoryId string) (authorized bool, _ error) {
	response, err := s.client.UserCanSeeRepositorySettings(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanSeeRepositorySettingsRequest{
				RepositoryId: repositoryId,
			}),
	)
	if err != nil {
		return false, err
	}
	return response.Msg.Authorized, nil
}

// UserCanSeeOrganizationSettings returns whether the user is authorized
// to see organization settings.
func (s *authzServiceClient) UserCanSeeOrganizationSettings(ctx context.Context, organizationId string) (authorized bool, _ error) {
	response, err := s.client.UserCanSeeOrganizationSettings(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanSeeOrganizationSettingsRequest{
				OrganizationId: organizationId,
			}),
	)
	if err != nil {
		return false, err
	}
	return response.Msg.Authorized, nil
}

// UserCanReadPlugin returns whether the user has read access to the specified plugin.
func (s *authzServiceClient) UserCanReadPlugin(
	ctx context.Context,
	owner string,
	name string,
) (authorized bool, _ error) {
	response, err := s.client.UserCanReadPlugin(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanReadPluginRequest{
				Owner: owner,
				Name:  name,
			}),
	)
	if err != nil {
		return false, err
	}
	return response.Msg.Authorized, nil
}

// UserCanCreatePluginVersion returns whether the user is authorized
// to create a plugin version under the specified plugin.
func (s *authzServiceClient) UserCanCreatePluginVersion(
	ctx context.Context,
	owner string,
	name string,
) (authorized bool, _ error) {
	response, err := s.client.UserCanCreatePluginVersion(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanCreatePluginVersionRequest{
				Owner: owner,
				Name:  name,
			}),
	)
	if err != nil {
		return false, err
	}
	return response.Msg.Authorized, nil
}

// UserCanCreateTemplateVersion returns whether the user is authorized
// to create a template version under the specified template.
func (s *authzServiceClient) UserCanCreateTemplateVersion(
	ctx context.Context,
	owner string,
	name string,
) (authorized bool, _ error) {
	response, err := s.client.UserCanCreateTemplateVersion(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanCreateTemplateVersionRequest{
				Owner: owner,
				Name:  name,
			}),
	)
	if err != nil {
		return false, err
	}
	return response.Msg.Authorized, nil
}

// UserCanCreateOrganizationPlugin returns whether the user is authorized to create
// a plugin in an organization.
func (s *authzServiceClient) UserCanCreateOrganizationPlugin(ctx context.Context, organizationId string) (authorized bool, _ error) {
	response, err := s.client.UserCanCreateOrganizationPlugin(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanCreateOrganizationPluginRequest{
				OrganizationId: organizationId,
			}),
	)
	if err != nil {
		return false, err
	}
	return response.Msg.Authorized, nil
}

// UserCanCreateOrganizationPlugin returns whether the user is authorized to create
// a template in an organization.
func (s *authzServiceClient) UserCanCreateOrganizationTemplate(ctx context.Context, organizationId string) (authorized bool, _ error) {
	response, err := s.client.UserCanCreateOrganizationTemplate(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanCreateOrganizationTemplateRequest{
				OrganizationId: organizationId,
			}),
	)
	if err != nil {
		return false, err
	}
	return response.Msg.Authorized, nil
}

// UserCanSeePluginSettings returns whether the user is authorized
// to see plugin settings.
func (s *authzServiceClient) UserCanSeePluginSettings(
	ctx context.Context,
	owner string,
	name string,
) (authorized bool, _ error) {
	response, err := s.client.UserCanSeePluginSettings(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanSeePluginSettingsRequest{
				Owner: owner,
				Name:  name,
			}),
	)
	if err != nil {
		return false, err
	}
	return response.Msg.Authorized, nil
}

// UserCanSeeTemplateSettings returns whether the user is authorized
// to see template settings.
func (s *authzServiceClient) UserCanSeeTemplateSettings(
	ctx context.Context,
	owner string,
	name string,
) (authorized bool, _ error) {
	response, err := s.client.UserCanSeeTemplateSettings(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanSeeTemplateSettingsRequest{
				Owner: owner,
				Name:  name,
			}),
	)
	if err != nil {
		return false, err
	}
	return response.Msg.Authorized, nil
}

// UserCanAddOrganizationMember returns whether the user is authorized to add
// any members to the organization and the list of roles they can add.
func (s *authzServiceClient) UserCanAddOrganizationMember(ctx context.Context, organizationId string) (authorizedRoles []v1alpha1.OrganizationRole, _ error) {
	response, err := s.client.UserCanAddOrganizationMember(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanAddOrganizationMemberRequest{
				OrganizationId: organizationId,
			}),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.AuthorizedRoles, nil
}

// UserCanUpdateOrganizationMember returns whether the user is authorized to update
// any members' membership information in the organization and the list of roles they can update.
func (s *authzServiceClient) UserCanUpdateOrganizationMember(ctx context.Context, organizationId string) (authorizedRoles []v1alpha1.OrganizationRole, _ error) {
	response, err := s.client.UserCanUpdateOrganizationMember(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanUpdateOrganizationMemberRequest{
				OrganizationId: organizationId,
			}),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.AuthorizedRoles, nil
}

// UserCanRemoveOrganizationMember returns whether the user is authorized to remove
// any members from the organization and the list of roles they can remove.
func (s *authzServiceClient) UserCanRemoveOrganizationMember(ctx context.Context, organizationId string) (authorizedRoles []v1alpha1.OrganizationRole, _ error) {
	response, err := s.client.UserCanRemoveOrganizationMember(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanRemoveOrganizationMemberRequest{
				OrganizationId: organizationId,
			}),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.AuthorizedRoles, nil
}

// UserCanDeleteOrganization returns whether the user is authorized
// to delete an organization.
func (s *authzServiceClient) UserCanDeleteOrganization(ctx context.Context, organizationId string) (authorized bool, _ error) {
	response, err := s.client.UserCanDeleteOrganization(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanDeleteOrganizationRequest{
				OrganizationId: organizationId,
			}),
	)
	if err != nil {
		return false, err
	}
	return response.Msg.Authorized, nil
}

// UserCanDeleteRepository returns whether the user is authorized
// to delete a repository.
func (s *authzServiceClient) UserCanDeleteRepository(ctx context.Context, repositoryId string) (authorized bool, _ error) {
	response, err := s.client.UserCanDeleteRepository(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanDeleteRepositoryRequest{
				RepositoryId: repositoryId,
			}),
	)
	if err != nil {
		return false, err
	}
	return response.Msg.Authorized, nil
}

// UserCanDeleteTemplate returns whether the user is authorized
// to delete a template.
func (s *authzServiceClient) UserCanDeleteTemplate(ctx context.Context, templateId string) (authorized bool, _ error) {
	response, err := s.client.UserCanDeleteTemplate(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanDeleteTemplateRequest{
				TemplateId: templateId,
			}),
	)
	if err != nil {
		return false, err
	}
	return response.Msg.Authorized, nil
}

// UserCanDeletePlugin returns whether the user is authorized
// to delete a plugin.
func (s *authzServiceClient) UserCanDeletePlugin(ctx context.Context, pluginId string) (authorized bool, _ error) {
	response, err := s.client.UserCanDeletePlugin(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanDeletePluginRequest{
				PluginId: pluginId,
			}),
	)
	if err != nil {
		return false, err
	}
	return response.Msg.Authorized, nil
}

// UserCanDeleteUser returns whether the user is authorized
// to delete a user.
func (s *authzServiceClient) UserCanDeleteUser(ctx context.Context) (authorized bool, _ error) {
	response, err := s.client.UserCanDeleteUser(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanDeleteUserRequest{}),
	)
	if err != nil {
		return false, err
	}
	return response.Msg.Authorized, nil
}

// UserCanSeeServerAdminPanel returns whether the user is authorized
// to see server admin panel.
func (s *authzServiceClient) UserCanSeeServerAdminPanel(ctx context.Context) (authorized bool, _ error) {
	response, err := s.client.UserCanSeeServerAdminPanel(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanSeeServerAdminPanelRequest{}),
	)
	if err != nil {
		return false, err
	}
	return response.Msg.Authorized, nil
}

// UserCanManageRepositoryContributors returns whether the user is authorized to manage
// any contributors to the repository and the list of roles they can manage.
func (s *authzServiceClient) UserCanManageRepositoryContributors(ctx context.Context, repositoryId string) (authorizedRoles []v1alpha1.RepositoryRole, _ error) {
	response, err := s.client.UserCanManageRepositoryContributors(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanManageRepositoryContributorsRequest{
				RepositoryId: repositoryId,
			}),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.AuthorizedRoles, nil
}

// UserCanManagePluginContributors returns whether the user is authorized to manage
// any contributors to the plugin and the list of roles they can manage.
func (s *authzServiceClient) UserCanManagePluginContributors(ctx context.Context, pluginId string) (authorizedRoles []v1alpha1.PluginRole, _ error) {
	response, err := s.client.UserCanManagePluginContributors(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanManagePluginContributorsRequest{
				PluginId: pluginId,
			}),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.AuthorizedRoles, nil
}

// UserCanManageTemplateContributors returns whether the user is authorized to manage
// any contributors to the template and the list of roles they can manage.
func (s *authzServiceClient) UserCanManageTemplateContributors(ctx context.Context, templateId string) (authorizedRoles []v1alpha1.TemplateRole, _ error) {
	response, err := s.client.UserCanManageTemplateContributors(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.UserCanManageTemplateContributorsRequest{
				TemplateId: templateId,
			}),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.AuthorizedRoles, nil
}
