// Copyright 2020-2021 Buf Technologies, Inc.
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

// Code generated by protoc-gen-go-api. DO NOT EDIT.

package registryv1alpha1api

import (
	context "context"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

// OrganizationService is the Organization service.
type OrganizationService interface {
	// GetOrganization gets a organization by ID.
	GetOrganization(ctx context.Context, id string) (organization *v1alpha1.Organization, err error)
	// GetOrganizationByName gets a organization by name.
	GetOrganizationByName(ctx context.Context, name string) (organization *v1alpha1.Organization, err error)
	// ListOrganizations lists all organizations.
	ListOrganizations(
		ctx context.Context,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (organizations []*v1alpha1.Organization, nextPageToken string, err error)
	// ListUserOrganizations lists all organizations a user is member of.
	ListUserOrganizations(
		ctx context.Context,
		userId string,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (organizations []*v1alpha1.OrganizationMembership, nextPageToken string, err error)
	// CreateOrganization creates a new organization.
	CreateOrganization(ctx context.Context, name string) (organization *v1alpha1.Organization, err error)
	// DeleteOrganization deletes a organization.
	DeleteOrganization(ctx context.Context, id string) (err error)
	// DeleteOrganizationByName deletes a organization by name.
	DeleteOrganizationByName(ctx context.Context, name string) (err error)
	// AddOrganizationMember add a role to an user in the organization.
	AddOrganizationMember(
		ctx context.Context,
		organizationId string,
		userId string,
		organizationRole v1alpha1.OrganizationRole,
	) (err error)
	// UpdateOrganizationMember update the user's membership information in the organization.
	UpdateOrganizationMember(
		ctx context.Context,
		organizationId string,
		userId string,
		organizationRole v1alpha1.OrganizationRole,
	) (err error)
	// RemoveOrganizationMember remove the role of an user in the organization.
	RemoveOrganizationMember(
		ctx context.Context,
		organizationId string,
		userId string,
	) (err error)
	// GetOrganizationSettings gets the settings of a organization, including organization base roles.
	GetOrganizationSettings(
		ctx context.Context,
		organizationId string,
	) (repositoryBaseRole v1alpha1.RepositoryRole, pluginBaseRole v1alpha1.PluginRole, templateBaseRole v1alpha1.TemplateRole, err error)
	// UpdateOrganizationSettings update the organization settings including base roles.
	UpdateOrganizationSettings(
		ctx context.Context,
		organizationId string,
		repositoryBaseRole *v1alpha1.RepositoryRoleValue,
		pluginBaseRole *v1alpha1.PluginRoleValue,
		templateBaseRole *v1alpha1.TemplateRoleValue,
	) (err error)
}
