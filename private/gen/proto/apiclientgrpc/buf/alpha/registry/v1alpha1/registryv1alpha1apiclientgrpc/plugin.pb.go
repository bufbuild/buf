// Code generated by protoc-gen-go-apiclientgrpc. DO NOT EDIT.

package registryv1alpha1apiclientgrpc

import (
	context "context"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	zap "go.uber.org/zap"
)

type pluginService struct {
	logger          *zap.Logger
	client          v1alpha1.PluginServiceClient
	contextModifier func(context.Context) context.Context
}

// ListPlugins returns all the plugins available to the user. This includes
// public plugins, those uploaded to organizations the user is part of,
// and any plugins uploaded directly by the user.
func (s *pluginService) ListPlugins(
	ctx context.Context,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (plugins []*v1alpha1.Plugin, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListPlugins(
		ctx,
		&v1alpha1.ListPluginsRequest{
			PageSize:  pageSize,
			PageToken: pageToken,
			Reverse:   reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.Plugins, response.NextPageToken, nil
}

// ListUserPlugins lists all plugins belonging to a user.
func (s *pluginService) ListUserPlugins(
	ctx context.Context,
	owner string,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (plugins []*v1alpha1.Plugin, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListUserPlugins(
		ctx,
		&v1alpha1.ListUserPluginsRequest{
			Owner:     owner,
			PageSize:  pageSize,
			PageToken: pageToken,
			Reverse:   reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.Plugins, response.NextPageToken, nil
}

// ListOrganizationPlugins lists all plugins for an organization.
func (s *pluginService) ListOrganizationPlugins(
	ctx context.Context,
	organization string,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (plugins []*v1alpha1.Plugin, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListOrganizationPlugins(
		ctx,
		&v1alpha1.ListOrganizationPluginsRequest{
			Organization: organization,
			PageSize:     pageSize,
			PageToken:    pageToken,
			Reverse:      reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.Plugins, response.NextPageToken, nil
}

// GetPluginVersion returns the plugin version, if found.
func (s *pluginService) GetPluginVersion(
	ctx context.Context,
	owner string,
	name string,
	version string,
) (pluginVersion *v1alpha1.PluginVersion, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetPluginVersion(
		ctx,
		&v1alpha1.GetPluginVersionRequest{
			Owner:   owner,
			Name:    name,
			Version: version,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.PluginVersion, nil
}

// ListPluginVersions lists all the versions available for the specified plugin.
func (s *pluginService) ListPluginVersions(
	ctx context.Context,
	owner string,
	name string,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (pluginVersions []*v1alpha1.PluginVersion, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListPluginVersions(
		ctx,
		&v1alpha1.ListPluginVersionsRequest{
			Owner:     owner,
			Name:      name,
			PageSize:  pageSize,
			PageToken: pageToken,
			Reverse:   reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.PluginVersions, response.NextPageToken, nil
}

// CreatePlugin creates a new plugin.
func (s *pluginService) CreatePlugin(
	ctx context.Context,
	owner string,
	name string,
	visibility v1alpha1.PluginVisibility,
) (plugin *v1alpha1.Plugin, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.CreatePlugin(
		ctx,
		&v1alpha1.CreatePluginRequest{
			Owner:      owner,
			Name:       name,
			Visibility: visibility,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.Plugin, nil
}

// GetPlugin returns the plugin, if found.
func (s *pluginService) GetPlugin(
	ctx context.Context,
	owner string,
	name string,
) (plugin *v1alpha1.Plugin, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetPlugin(
		ctx,
		&v1alpha1.GetPluginRequest{
			Owner: owner,
			Name:  name,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.Plugin, nil
}

// DeletePlugin deletes the plugin, if it exists. Note that deleting
// a plugin may cause breaking changes for templates using that plugin,
// and should be done with extreme care.
func (s *pluginService) DeletePlugin(
	ctx context.Context,
	owner string,
	name string,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.DeletePlugin(
		ctx,
		&v1alpha1.DeletePluginRequest{
			Owner: owner,
			Name:  name,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// SetPluginContributor sets the role of a user in the plugin.
func (s *pluginService) SetPluginContributor(
	ctx context.Context,
	pluginId string,
	userId string,
	pluginRole v1alpha1.PluginRole,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.SetPluginContributor(
		ctx,
		&v1alpha1.SetPluginContributorRequest{
			PluginId:   pluginId,
			UserId:     userId,
			PluginRole: pluginRole,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// ListPluginContributors returns the list of contributors that has an explicit role against the plugin.
// This does not include users who have implicit roles against the plugin, unless they have also been
// assigned a role explicitly.
func (s *pluginService) ListPluginContributors(
	ctx context.Context,
	pluginId string,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (users []*v1alpha1.PluginContributor, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListPluginContributors(
		ctx,
		&v1alpha1.ListPluginContributorsRequest{
			PluginId:  pluginId,
			PageSize:  pageSize,
			PageToken: pageToken,
			Reverse:   reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.Users, response.NextPageToken, nil
}

// DeprecatePlugin deprecates the plugin, if found.
func (s *pluginService) DeprecatePlugin(
	ctx context.Context,
	owner string,
	name string,
	message string,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.DeprecatePlugin(
		ctx,
		&v1alpha1.DeprecatePluginRequest{
			Owner:   owner,
			Name:    name,
			Message: message,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// UndeprecatePlugin makes the plugin not deprecated and removes any deprecation_message.
func (s *pluginService) UndeprecatePlugin(
	ctx context.Context,
	owner string,
	name string,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.UndeprecatePlugin(
		ctx,
		&v1alpha1.UndeprecatePluginRequest{
			Owner: owner,
			Name:  name,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// GetTemplate returns the template, if found.
func (s *pluginService) GetTemplate(
	ctx context.Context,
	owner string,
	name string,
) (template *v1alpha1.Template, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetTemplate(
		ctx,
		&v1alpha1.GetTemplateRequest{
			Owner: owner,
			Name:  name,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.Template, nil
}

// ListTemplates returns all the templates available to the user. This includes
// public templates, those owned by organizations the user is part of,
// and any created directly by the user.
func (s *pluginService) ListTemplates(
	ctx context.Context,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (templates []*v1alpha1.Template, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListTemplates(
		ctx,
		&v1alpha1.ListTemplatesRequest{
			PageSize:  pageSize,
			PageToken: pageToken,
			Reverse:   reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.Templates, response.NextPageToken, nil
}

// ListTemplatesUserCanAccess is like ListTemplates, but does not return
// public templates.
func (s *pluginService) ListTemplatesUserCanAccess(
	ctx context.Context,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (templates []*v1alpha1.Template, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListTemplatesUserCanAccess(
		ctx,
		&v1alpha1.ListTemplatesUserCanAccessRequest{
			PageSize:  pageSize,
			PageToken: pageToken,
			Reverse:   reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.Templates, response.NextPageToken, nil
}

// ListUserPlugins lists all templates belonging to a user.
func (s *pluginService) ListUserTemplates(
	ctx context.Context,
	owner string,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (templates []*v1alpha1.Template, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListUserTemplates(
		ctx,
		&v1alpha1.ListUserTemplatesRequest{
			Owner:     owner,
			PageSize:  pageSize,
			PageToken: pageToken,
			Reverse:   reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.Templates, response.NextPageToken, nil
}

// ListOrganizationTemplates lists all templates for an organization.
func (s *pluginService) ListOrganizationTemplates(
	ctx context.Context,
	organization string,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (templates []*v1alpha1.Template, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListOrganizationTemplates(
		ctx,
		&v1alpha1.ListOrganizationTemplatesRequest{
			Organization: organization,
			PageSize:     pageSize,
			PageToken:    pageToken,
			Reverse:      reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.Templates, response.NextPageToken, nil
}

// GetTemplateVersion returns the template version, if found.
func (s *pluginService) GetTemplateVersion(
	ctx context.Context,
	owner string,
	name string,
	version string,
) (templateVersion *v1alpha1.TemplateVersion, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetTemplateVersion(
		ctx,
		&v1alpha1.GetTemplateVersionRequest{
			Owner:   owner,
			Name:    name,
			Version: version,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.TemplateVersion, nil
}

// ListTemplateVersions lists all the template versions available for the specified template.
func (s *pluginService) ListTemplateVersions(
	ctx context.Context,
	owner string,
	name string,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (templateVersions []*v1alpha1.TemplateVersion, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListTemplateVersions(
		ctx,
		&v1alpha1.ListTemplateVersionsRequest{
			Owner:     owner,
			Name:      name,
			PageSize:  pageSize,
			PageToken: pageToken,
			Reverse:   reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.TemplateVersions, response.NextPageToken, nil
}

// CreateTemplate creates a new template.
func (s *pluginService) CreateTemplate(
	ctx context.Context,
	owner string,
	name string,
	visibility v1alpha1.PluginVisibility,
	pluginConfigs []*v1alpha1.PluginConfig,
) (template *v1alpha1.Template, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.CreateTemplate(
		ctx,
		&v1alpha1.CreateTemplateRequest{
			Owner:         owner,
			Name:          name,
			Visibility:    visibility,
			PluginConfigs: pluginConfigs,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.Template, nil
}

// DeleteTemplate deletes the template, if it exists.
func (s *pluginService) DeleteTemplate(
	ctx context.Context,
	owner string,
	name string,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.DeleteTemplate(
		ctx,
		&v1alpha1.DeleteTemplateRequest{
			Owner: owner,
			Name:  name,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// CreateTemplateVersion creates a new template version.
func (s *pluginService) CreateTemplateVersion(
	ctx context.Context,
	name string,
	templateOwner string,
	templateName string,
	pluginVersions []*v1alpha1.PluginVersionMapping,
) (templateVersion *v1alpha1.TemplateVersion, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.CreateTemplateVersion(
		ctx,
		&v1alpha1.CreateTemplateVersionRequest{
			Name:           name,
			TemplateOwner:  templateOwner,
			TemplateName:   templateName,
			PluginVersions: pluginVersions,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.TemplateVersion, nil
}

// SetTemplateContributor sets the role of a user in the template.
func (s *pluginService) SetTemplateContributor(
	ctx context.Context,
	templateId string,
	userId string,
	templateRole v1alpha1.TemplateRole,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.SetTemplateContributor(
		ctx,
		&v1alpha1.SetTemplateContributorRequest{
			TemplateId:   templateId,
			UserId:       userId,
			TemplateRole: templateRole,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// ListTemplateContributors returns the list of contributors that has an explicit role against the template.
// This does not include users who have implicit roles against the template, unless they have also been
// assigned a role explicitly.
func (s *pluginService) ListTemplateContributors(
	ctx context.Context,
	templateId string,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (users []*v1alpha1.TemplateContributor, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListTemplateContributors(
		ctx,
		&v1alpha1.ListTemplateContributorsRequest{
			TemplateId: templateId,
			PageSize:   pageSize,
			PageToken:  pageToken,
			Reverse:    reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.Users, response.NextPageToken, nil
}

// DeprecateTemplate deprecates the template, if found.
func (s *pluginService) DeprecateTemplate(
	ctx context.Context,
	owner string,
	name string,
	message string,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.DeprecateTemplate(
		ctx,
		&v1alpha1.DeprecateTemplateRequest{
			Owner:   owner,
			Name:    name,
			Message: message,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// UndeprecateTemplate makes the template not deprecated and removes any deprecation_message.
func (s *pluginService) UndeprecateTemplate(
	ctx context.Context,
	owner string,
	name string,
) (_ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	_, err := s.client.UndeprecateTemplate(
		ctx,
		&v1alpha1.UndeprecateTemplateRequest{
			Owner: owner,
			Name:  name,
		},
	)
	if err != nil {
		return err
	}
	return nil
}
