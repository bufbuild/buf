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
	v1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	pluginpb "google.golang.org/protobuf/types/pluginpb"
)

// GenerateService manages remote generation requests.
type GenerateService interface {
	// GeneratePlugins generates an array of files given the provided
	// module reference and plugin version and option tuples. No attempt
	// is made at merging insertion points.
	GeneratePlugins(
		ctx context.Context,
		image *v1.Image,
		plugins []*v1alpha1.PluginReference,
		includeImports bool,
	) (responses []*pluginpb.CodeGeneratorResponse, runtimeLibraries []*v1alpha1.RuntimeLibrary, err error)
	// GenerateTemplate generates an array of files given the provided
	// module reference and template version.
	GenerateTemplate(
		ctx context.Context,
		image *v1.Image,
		templateOwner string,
		templateName string,
		templateVersion string,
		includeImports bool,
	) (files []*v1alpha1.File, runtimeLibraries []*v1alpha1.RuntimeLibrary, err error)
}
