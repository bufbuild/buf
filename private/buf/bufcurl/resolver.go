// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufcurl

import (
	"context"
	"log/slog"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Resolver is used to resolve descriptors, types, extensions, etc. Unlike
// the base protoencoding.Resolver interface, it can also enumerate services.
type Resolver interface {
	protoencoding.Resolver
	ListServices() ([]protoreflect.FullName, error)
}

// ResolverForImage returns a Resolver backed by the given image.
func ResolverForImage(image bufimage.Image) Resolver {
	return &imageResolver{
		Resolver: image.Resolver(),
		image:    image,
	}
}

// CombineResolvers returns a Resolver backed by the given underlying
// resolvers. For any given query, each underlying resolver is checked,
// starting with the first resolver provided. If the first cannot answer
// a query, the second one is checked, and so on. The service names
// returned by the ListServices methods is the union of service names
// for all given resolvers.
func CombineResolvers(resolvers ...Resolver) Resolver {
	encodingResolvers := make([]protoencoding.Resolver, len(resolvers))
	for i := range resolvers {
		encodingResolvers[i] = resolvers[i]
	}
	return &combinedResolver{
		Resolver:  protoencoding.CombineResolvers(encodingResolvers...),
		resolvers: resolvers,
	}
}

type imageResolver struct {
	protoencoding.Resolver
	image bufimage.Image
}

func (i *imageResolver) ListServices() ([]protoreflect.FullName, error) {
	var names []protoreflect.FullName
	for _, file := range i.image.Files() {
		fileDescriptor := file.FileDescriptorProto()
		for _, service := range fileDescriptor.Service {
			var serviceName string
			if fileDescriptor.Package != nil {
				serviceName = fileDescriptor.GetPackage() + "." + service.GetName()
			} else {
				serviceName = service.GetName()
			}
			names = append(names, protoreflect.FullName(serviceName))
		}
	}
	return names, nil
}

type combinedResolver struct {
	protoencoding.Resolver // underlying resolver is already combined
	resolvers              []Resolver
}

func (c *combinedResolver) ListServices() ([]protoreflect.FullName, error) {
	names := map[protoreflect.FullName]struct{}{}
	for _, resolver := range c.resolvers {
		serviceNames, err := resolver.ListServices()
		if err != nil {
			return nil, err
		}
		for _, serviceName := range serviceNames {
			names[serviceName] = struct{}{}
		}
	}
	serviceNames := make([]protoreflect.FullName, 0, len(names))
	for serviceName := range names {
		serviceNames = append(serviceNames, serviceName)
	}
	return serviceNames, nil
}

// NewWKTResolver returns a Resolver that can resolve all well-known types.
func NewWKTResolver(ctx context.Context, logger *slog.Logger) (Resolver, error) {
	moduleSet, err := bufmodule.NewModuleSetBuilder(
		ctx,
		logger,
		bufmodule.NopModuleDataProvider,
		bufmodule.NopCommitProvider,
	).AddLocalModule(
		datawkt.ReadBucket,
		".",
		true,
	).Build()
	if err != nil {
		return nil, err
	}
	module := bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet)
	image, err := bufimage.BuildImage(
		ctx,
		logger,
		module,
		bufimage.WithExcludeSourceCodeInfo(),
	)
	if err != nil {
		return nil, err
	}
	return ResolverForImage(image), nil
}
