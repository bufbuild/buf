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

// PluginService manages plugins.
type PluginService interface {
	// ListPlugins returns all the plugins available to the user. This includes
	// public plugins, those uploaded to organizations the user is part of,
	// and any plugins uploaded directly by the user.
	ListPlugins(
		ctx context.Context,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (plugins []*v1alpha1.Plugin, nextPageToken string, err error)
	// ListUserPlugins lists all plugins belonging to a user.
	ListUserPlugins(
		ctx context.Context,
		owner string,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (plugins []*v1alpha1.Plugin, nextPageToken string, err error)
	// ListOrganizationPlugins lists all plugins for an organization.
	ListOrganizationPlugins(
		ctx context.Context,
		organization string,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (plugins []*v1alpha1.Plugin, nextPageToken string, err error)
	// ListPluginVersions lists all the versions available for the specified plugin.
	ListPluginVersions(
		ctx context.Context,
		owner string,
		name string,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (pluginVersions []*v1alpha1.PluginVersion, nextPageToken string, err error)
	// CreatePlugin creates a new plugin.
	CreatePlugin(
		ctx context.Context,
		owner string,
		name string,
		visibility v1alpha1.PluginVisibility,
	) (plugin *v1alpha1.Plugin, err error)
	// GetPlugin returns the plugin, if found.
	GetPlugin(
		ctx context.Context,
		owner string,
		name string,
	) (plugin *v1alpha1.Plugin, err error)
	// DeletePlugin deletes the plugin, if it exists. Note that deleting
	// a plugin may cause breaking changes for templates using that plugin,
	// and should be done with extreme care.
	DeletePlugin(
		ctx context.Context,
		owner string,
		name string,
	) (err error)
	// SetPluginContributor sets the role of a user in the plugin.
	SetPluginContributor(
		ctx context.Context,
		pluginId string,
		userId string,
		pluginRole v1alpha1.PluginRole,
	) (err error)
	// GetTemplate returns the template, if found.
	GetTemplate(
		ctx context.Context,
		owner string,
		name string,
	) (template *v1alpha1.Template, err error)
	// ListTemplates returns all the templates available to the user. This includes
	// public templates, those owned by organizations the user is part of,
	// and any created directly by the user.
	ListTemplates(
		ctx context.Context,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (templates []*v1alpha1.Template, nextPageToken string, err error)
	// ListUserPlugins lists all templates belonging to a user.
	ListUserTemplates(
		ctx context.Context,
		owner string,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (templates []*v1alpha1.Template, nextPageToken string, err error)
	// ListOrganizationTemplates lists all templates for an organization.
	ListOrganizationTemplates(
		ctx context.Context,
		organization string,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (templates []*v1alpha1.Template, nextPageToken string, err error)
	// GetTemplateVersion returns the template version, if found.
	GetTemplateVersion(
		ctx context.Context,
		owner string,
		name string,
		version string,
	) (templateVersion *v1alpha1.TemplateVersion, err error)
	// ListTemplateVersions lists all the template versions available for the specified template.
	ListTemplateVersions(
		ctx context.Context,
		owner string,
		name string,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (templateVersions []*v1alpha1.TemplateVersion, nextPageToken string, err error)
	// CreateTemplate creates a new template.
	CreateTemplate(
		ctx context.Context,
		owner string,
		name string,
		visibility v1alpha1.PluginVisibility,
		pluginConfigs []*v1alpha1.PluginConfig,
	) (template *v1alpha1.Template, err error)
	// DeleteTemplate deletes the template, if it exists.
	DeleteTemplate(
		ctx context.Context,
		owner string,
		name string,
	) (err error)
	// CreateTemplateVersion creates a new template version.
	CreateTemplateVersion(
		ctx context.Context,
		name string,
		templateOwner string,
		templateName string,
		pluginVersions []*v1alpha1.PluginVersionMapping,
	) (templateVersion *v1alpha1.TemplateVersion, err error)
	// SetTemplateContributor sets the role of a user in the template.
	SetTemplateContributor(
		ctx context.Context,
		templateId string,
		userId string,
		templateRole v1alpha1.TemplateRole,
	) (err error)
}
