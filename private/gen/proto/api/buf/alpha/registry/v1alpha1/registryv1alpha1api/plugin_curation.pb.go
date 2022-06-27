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

// Code generated by protoc-gen-go-api. DO NOT EDIT.

package registryv1alpha1api

import (
	context "context"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

// PluginCurationService manages curated plugins.
type PluginCurationService interface {
	// ListCuratedPlugins returns all the curated plugins available.
	ListCuratedPlugins(
		ctx context.Context,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (plugins []*v1alpha1.CuratedPlugin, nextPageToken string, err error)
	// CreateCuratedPlugin creates a new curated plugin.
	CreateCuratedPlugin(
		ctx context.Context,
		owner string,
		name string,
		language v1alpha1.PluginLanguage,
		version string,
		containerImageDigest string,
		options []string,
		dependencies []string,
		sourceUrl string,
		description string,
		runtimeConfig *v1alpha1.RuntimeConfig,
	) (configuration *v1alpha1.CuratedPlugin, err error)
}
