// Copyright 2020 Buf Technologies, Inc.
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

package appprotogen

import (
	"context"
	"fmt"
	"path"
	"sort"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/appproto"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

// NewHandler returns a new Handler for the protogen Plugin function.
func NewHandler(f func(*protogen.Plugin) error) appproto.Handler {
	return appproto.HandlerFunc(
		func(
			ctx context.Context,
			container app.EnvStderrContainer,
			responseWriter appproto.ResponseWriter,
			request *pluginpb.CodeGeneratorRequest,
		) error {
			plugin, err := protogen.Options{}.New(request)
			if err != nil {
				return err
			}
			if err := f(plugin); err != nil {
				plugin.Error(err)
			}
			response := plugin.Response()
			for _, file := range response.File {
				if err := responseWriter.AddFile(file); err != nil {
					return err
				}
			}
			if response.Error != nil {
				responseWriter.AddError(response.GetError())
			}
			responseWriter.SetFeatureProto3Optional()
			return nil
		},
	)
}

// NewPerFileHandler returns a newHandler for the protogen per-file function.
//
// This will invoke f for every file marked for generation.
func NewPerFileHandler(f func(*protogen.Plugin, *protogen.File) error) appproto.Handler {
	return NewHandler(
		func(plugin *protogen.Plugin) error {
			for _, file := range plugin.Files {
				if file.Generate {
					if err := f(plugin, file); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

// NewPerGoPackageHandler returns a newHandler for the protogen per-package function.
//
// This validates that all files marked for generation that would be generated to
// the same directory also have the same go package and go import path.
//
// This will invoke f for every set of files in a package marked for generation.
func NewPerGoPackageHandler(f func(*protogen.Plugin, *GoPackageFileSet) error) appproto.Handler {
	return NewHandler(
		func(plugin *protogen.Plugin) error {
			generatedDirToGoPackageFileSet := make(map[string]*GoPackageFileSet)
			for _, file := range plugin.Files {
				if file.Generate {
					generatedDir := path.Dir(file.GeneratedFilenamePrefix)
					goPackageFileSet, ok := generatedDirToGoPackageFileSet[generatedDir]
					if !ok {
						generatedDirToGoPackageFileSet[generatedDir] = &GoPackageFileSet{
							GeneratedDir:  generatedDir,
							GoImportPath:  file.GoImportPath,
							GoPackageName: file.GoPackageName,
							Files:         []*protogen.File{file},
						}
					} else {
						if goPackageFileSet.GoImportPath != file.GoImportPath {
							return fmt.Errorf(
								"mismatched go import paths for generated directory %q: %q %q",
								generatedDir,
								string(goPackageFileSet.GoImportPath),
								string(file.GoImportPath),
							)
						}
						if goPackageFileSet.GoPackageName != file.GoPackageName {
							return fmt.Errorf(
								"mismatched go package names for generated directory %q: %q %q",
								generatedDir,
								string(goPackageFileSet.GoPackageName),
								string(file.GoPackageName),
							)
						}
						goPackageFileSet.Files = append(goPackageFileSet.Files, file)
					}
				}
			}
			for _, goPackageFileSet := range generatedDirToGoPackageFileSet {
				if err := f(plugin, goPackageFileSet); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

// GoPackageFileSet are files within a single Go package.
type GoPackageFileSet struct {
	// The directory the golang/protobuf files would be generated to.
	GeneratedDir string
	// The Go import path the golang/protobuf files would be generated to.
	GoImportPath protogen.GoImportPath
	// The Go package name the golang/protobuf files would be generated to.
	GoPackageName protogen.GoPackageName
	// The files within this package.
	Files []*protogen.File
}

// Services returns all the services in this Go package sorted by Go name.
func (g *GoPackageFileSet) Services() []*protogen.Service {
	var services []*protogen.Service
	for _, file := range g.Files {
		services = append(services, file.Services...)
	}
	sort.Slice(
		services,
		func(i int, j int) bool {
			return services[i].GoName < services[j].GoName
		},
	)
	return services
}
