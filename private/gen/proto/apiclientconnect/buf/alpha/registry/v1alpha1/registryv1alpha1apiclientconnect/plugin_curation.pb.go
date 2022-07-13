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
	registryv1alpha1connect "github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	v1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	connect_go "github.com/bufbuild/connect-go"
	zap "go.uber.org/zap"
)

type pluginCurationServiceClient struct {
	logger *zap.Logger
	client registryv1alpha1connect.PluginCurationServiceClient
}

// ListCuratedPlugins returns all the curated plugins available.
func (s *pluginCurationServiceClient) ListCuratedPlugins(
	ctx context.Context,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (plugins []*v1alpha1.CuratedPlugin, nextPageToken string, _ error) {
	response, err := s.client.ListCuratedPlugins(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.ListCuratedPluginsRequest{
				PageSize:  pageSize,
				PageToken: pageToken,
				Reverse:   reverse,
			}),
	)
	if err != nil {
		return nil, "", err
	}
	return response.Msg.Plugins, response.Msg.NextPageToken, nil
}

// CreateCuratedPlugin creates a new curated plugin.
func (s *pluginCurationServiceClient) CreateCuratedPlugin(
	ctx context.Context,
	owner string,
	name string,
	language v1alpha1.PluginLanguage,
	version string,
	containerImageDigest string,
	options []string,
	dependencies []*v1alpha1.CuratedPluginReference,
	sourceUrl string,
	description string,
	runtimeConfig *v1alpha1.RuntimeConfig,
	revision uint32,
) (configuration *v1alpha1.CuratedPlugin, _ error) {
	response, err := s.client.CreateCuratedPlugin(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.CreateCuratedPluginRequest{
				Owner:                owner,
				Name:                 name,
				Language:             language,
				Version:              version,
				ContainerImageDigest: containerImageDigest,
				Options:              options,
				Dependencies:         dependencies,
				SourceUrl:            sourceUrl,
				Description:          description,
				RuntimeConfig:        runtimeConfig,
				Revision:             revision,
			}),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.Configuration, nil
}

// GetLatestCuratedPlugin returns the latest version of a plugin matching given parameters.
func (s *pluginCurationServiceClient) GetLatestCuratedPlugin(
	ctx context.Context,
	owner string,
	name string,
	version string,
) (plugin *v1alpha1.CuratedPlugin, _ error) {
	response, err := s.client.GetLatestCuratedPlugin(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.GetLatestCuratedPluginRequest{
				Owner:   owner,
				Name:    name,
				Version: version,
			}),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.Plugin, nil
}

type codeGenerationServiceClient struct {
	logger *zap.Logger
	client registryv1alpha1connect.CodeGenerationServiceClient
}

// GenerateCode generates code using the specified remote plugins.
func (s *codeGenerationServiceClient) GenerateCode(
	ctx context.Context,
	image *v1.Image,
	requests []*v1alpha1.PluginGenerationRequest,
	includeImports bool,
	includeWellKnownTypes bool,
) (responses []*v1alpha1.PluginGenerationResponse, _ error) {
	response, err := s.client.GenerateCode(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.GenerateCodeRequest{
				Image:                 image,
				Requests:              requests,
				IncludeImports:        includeImports,
				IncludeWellKnownTypes: includeWellKnownTypes,
			}),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.Responses, nil
}
