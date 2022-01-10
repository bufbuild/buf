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
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	zap "go.uber.org/zap"
)

type organizationService struct {
	logger          *zap.Logger
	client          v1alpha1.OrganizationServiceClient
	contextModifier func(context.Context) context.Context
}

// GetOrganization gets a organization by ID.
func (s *organizationService) GetOrganization(ctx context.Context, id string) (organization *v1alpha1.Organization, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetOrganization(
		ctx,
		&v1alpha1.GetOrganizationRequest{
			Id: id,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.Organization, nil
}

// GetOrganizationByName gets a organization by name.
func (s *organizationService) GetOrganizationByName(ctx context.Context, name string) (organization *v1alpha1.Organization, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetOrganizationByName(
		ctx,
		&v1alpha1.GetOrganizationByNameRequest{
			Name: name,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.Organization, nil
}

// ListOrganizations lists all organizations.
func (s *organizationService) ListOrganizations(
	ctx context.Context,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (organizations []*v1alpha1.Organization, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListOrganizations(
		ctx,
		&v1alpha1.ListOrganizationsRequest{
			PageSize:  pageSize,
			PageToken: pageToken,
			Reverse:   reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.Organizations, response.NextPageToken, nil
}

// ListUserOrganizations lists all organizations a user is member of.
func (s *organizationService) ListUserOrganizations(
	ctx context.Context,
	userId string,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (organizations []*v1alpha1.OrganizationMembership, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListUserOrganizations(
		ctx,
		&v1alpha1.ListUserOrganizationsRequest{
			UserId:    userId,
			PageSize:  pageSize,
			PageToken: pageToken,
			Reverse:   reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.Organizations, response.NextPageToken, nil
}

// CreateOrganization creates a new organization.
func (s *organizationService) CreateOrganization(ctx context.Context, name string) (organization *v1alpha1.Organization, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.CreateOrganization(
		ctx,
		&v1alpha1.CreateOrganizationRequest{
			Name: name,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.Organization, nil
}

// DeleteOrganization deletes a organization.
func (s *organizationService) DeleteOrganization(ctx context.Context, id string) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.DeleteOrganization(
		ctx,
		&v1alpha1.DeleteOrganizationRequest{
			Id: id,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// DeleteOrganizationByName deletes a organization by name.
func (s *organizationService) DeleteOrganizationByName(ctx context.Context, name string) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.DeleteOrganizationByName(
		ctx,
		&v1alpha1.DeleteOrganizationByNameRequest{
			Name: name,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// AddOrganizationMember add a role to an user in the organization.
func (s *organizationService) AddOrganizationMember(
	ctx context.Context,
	organizationId string,
	userId string,
	organizationRole v1alpha1.OrganizationRole,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.AddOrganizationMember(
		ctx,
		&v1alpha1.AddOrganizationMemberRequest{
			OrganizationId:   organizationId,
			UserId:           userId,
			OrganizationRole: organizationRole,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// UpdateOrganizationMember update the user's membership information in the organization.
func (s *organizationService) UpdateOrganizationMember(
	ctx context.Context,
	organizationId string,
	userId string,
	organizationRole v1alpha1.OrganizationRole,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.UpdateOrganizationMember(
		ctx,
		&v1alpha1.UpdateOrganizationMemberRequest{
			OrganizationId:   organizationId,
			UserId:           userId,
			OrganizationRole: organizationRole,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// RemoveOrganizationMember remove the role of an user in the organization.
func (s *organizationService) RemoveOrganizationMember(
	ctx context.Context,
	organizationId string,
	userId string,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.RemoveOrganizationMember(
		ctx,
		&v1alpha1.RemoveOrganizationMemberRequest{
			OrganizationId: organizationId,
			UserId:         userId,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// SetOrganizationMember sets the role of a user in the organization.
func (s *organizationService) SetOrganizationMember(
	ctx context.Context,
	organizationId string,
	userId string,
	organizationRole v1alpha1.OrganizationRole,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.SetOrganizationMember(
		ctx,
		&v1alpha1.SetOrganizationMemberRequest{
			OrganizationId:   organizationId,
			UserId:           userId,
			OrganizationRole: organizationRole,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// GetOrganizationSettings gets the settings of an organization, including organization base roles.
func (s *organizationService) GetOrganizationSettings(
	ctx context.Context,
	organizationId string,
) (repositoryBaseRole v1alpha1.RepositoryRole, pluginBaseRole v1alpha1.PluginRole, templateBaseRole v1alpha1.TemplateRole, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetOrganizationSettings(
		ctx,
		&v1alpha1.GetOrganizationSettingsRequest{
			OrganizationId: organizationId,
		},
	)
	if err != nil {
		return v1alpha1.RepositoryRole(0), v1alpha1.PluginRole(0), v1alpha1.TemplateRole(0), err
	}
	return response.RepositoryBaseRole, response.PluginBaseRole, response.TemplateBaseRole, nil
}

// UpdateOrganizationSettings update the organization settings including base roles.
func (s *organizationService) UpdateOrganizationSettings(
	ctx context.Context,
	organizationId string,
	repositoryBaseRole v1alpha1.RepositoryRole,
	pluginBaseRole v1alpha1.PluginRole,
	templateBaseRole v1alpha1.TemplateRole,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.UpdateOrganizationSettings(
		ctx,
		&v1alpha1.UpdateOrganizationSettingsRequest{
			OrganizationId:     organizationId,
			RepositoryBaseRole: repositoryBaseRole,
			PluginBaseRole:     pluginBaseRole,
			TemplateBaseRole:   templateBaseRole,
		},
	)
	if err != nil {
		return err
	}
	return nil
}
