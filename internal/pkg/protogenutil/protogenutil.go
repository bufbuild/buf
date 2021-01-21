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

// Package protogenutil provides support for protoc plugin development with the
// appproto and protogen packages.
package protogenutil

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/appproto"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/pluginpb"
)

// NewHandler returns a new appproto.Handler for the protogen.Plugin function.
func NewHandler(f func(*protogen.Plugin) error, options ...HandlerOption) appproto.Handler {
	handlerOptions := newHandlerOptions()
	for _, option := range options {
		option(handlerOptions)
	}
	return appproto.HandlerFunc(
		func(
			ctx context.Context,
			container app.EnvStderrContainer,
			responseWriter appproto.ResponseWriter,
			request *pluginpb.CodeGeneratorRequest,
		) error {
			plugin, err := protogen.Options{
				ParamFunc: handlerOptions.optionHandler,
			}.New(request)
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
func NewPerFileHandler(f func(*protogen.Plugin, *protogen.File) error, options ...HandlerOption) appproto.Handler {
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
		options...,
	)
}

// NewPerGoPackageHandler returns a newHandler for the protogen per-package function.
//
// This validates that all files marked for generation that would be generated to
// the same directory also have the same go package and go import path.
//
// This will invoke f for every set of files in a package marked for generation.
func NewPerGoPackageHandler(f func(*protogen.Plugin, *GoPackageFileSet) error, options ...HandlerOption) appproto.Handler {
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
		options...,
	)
}

// HandlerOption is an option for a new Handler.
type HandlerOption func(*handlerOptions)

// HandlerWithOptionHandler returns a new HandlerOption that sets the given param function.
//
// This parses options given on the command line.
func HandlerWithOptionHandler(optionHandler func(string, string) error) HandlerOption {
	return func(handlerOptions *handlerOptions) {
		handlerOptions.optionHandler = optionHandler
	}
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

// NamedHelper is a helper to deal with named golang plugins.
//
// Named plugins should be named in the form protoc-gen-go-foobar, where the plugin
// name is consiered to be "foobar". The plugin name must be lowercase.
type NamedHelper interface {
	// NewGoPackageName gets the helper GoPackageName for the pluginName.
	NewGoPackageName(
		baseGoPackageName protogen.GoPackageName,
		pluginName string,
	) protogen.GoPackageName
	// NewGoImportPath gets the helper GoImportPath for the pluginName.
	NewGoImportPath(
		file *protogen.File,
		pluginName string,
	) (protogen.GoImportPath, error)
	// NewPackageGoImportPath gets the helper GoImportPath for the pluginName.
	NewPackageGoImportPath(
		goPackageFileSet *GoPackageFileSet,
		pluginName string,
	) (protogen.GoImportPath, error)
	// NewGeneratedFile returns a new individual GeneratedFile for a named plugin.
	//
	// This should be used for named plugins that have a 1-1 mapping between Protobuf files
	// and generated files.
	//
	// This also prints the file header and package.
	NewGeneratedFile(
		plugin *protogen.Plugin,
		file *protogen.File,
		pluginName string,
	) (*protogen.GeneratedFile, error)
	// NewPackageGeneratedFile returns a new individual GeneratedFile for a named plugin.
	//
	// This should be used for named plugins that have a 1-1 mapping between Protobuf files
	// and generated files. The generated file name will not overlap with the base name
	// of any .proto file in the package.
	//
	// This also prints the file header and package.
	NewPackageGeneratedFile(
		plugin *protogen.Plugin,
		goPackageFileSet *GoPackageFileSet,
		pluginName string,
	) (*protogen.GeneratedFile, error)
}

// NewNamedPerFileHandler returns a new per-file handler for a named plugin.
func NewNamedPerFileHandler(f func(NamedHelper, *protogen.Plugin, *protogen.File) error) appproto.Handler {
	namedHelper := newNamedHelper()
	return NewPerFileHandler(
		func(plugin *protogen.Plugin, file *protogen.File) error {
			return f(namedHelper, plugin, file)
		},
		HandlerWithOptionHandler(
			namedHelper.handleOption,
		),
	)
}

// NewNamedPerGoPackageHandler returns a new per-go-package handler for a named plugin.
func NewNamedPerGoPackageHandler(f func(NamedHelper, *protogen.Plugin, *GoPackageFileSet) error) appproto.Handler {
	namedHelper := newNamedHelper()
	return NewPerGoPackageHandler(
		func(plugin *protogen.Plugin, goPackageFileSet *GoPackageFileSet) error {
			return f(namedHelper, plugin, goPackageFileSet)
		},
		HandlerWithOptionHandler(
			namedHelper.handleOption,
		),
	)
}

// ValidateMethodUnary validates that the method is unary.
func ValidateMethodUnary(method *protogen.Method) error {
	if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
		return fmt.Errorf("plugin does not allow streaming methods: %v", method.GoName)
	}
	return nil
}

// ValidateFieldNotOneof validates that the field is not a oneof.
func ValidateFieldNotOneof(field *protogen.Field) error {
	if oneof := field.Oneof; oneof != nil && !oneof.Desc.IsSynthetic() {
		return fmt.Errorf("plugin does not allow oneofs for request fields: %v", field.GoName)
	}
	return nil
}

// ValidateFieldNotMap validates that the field is not a map.
func ValidateFieldNotMap(field *protogen.Field) error {
	if field.Desc.IsMap() {
		return fmt.Errorf("plugin does not allow maps for request fields: %v", field.GoName)
	}
	return nil
}

// GetUnexportGoName returns a new unexported type for the go name.
//
// This makes the first character lowercase.
// If the goName is empty, this returns empty.
func GetUnexportGoName(goName string) string {
	if goName == "" {
		return ""
	}
	return strings.ToLower(goName[:1]) + goName[1:]
}

// GetParameterStrings gets the parameters for the given fields.
func GetParameterStrings(
	generatedFile *protogen.GeneratedFile,
	fields []*protogen.Field,
) ([]string, error) {
	if len(fields) == 0 {
		return nil, nil
	}
	parameterStrings := make([]string, len(fields))
	for i, field := range fields {
		if err := ValidateFieldNotOneof(field); err != nil {
			return nil, err
		}
		fieldGoType, err := GetFieldGoType(generatedFile, field)
		if err != nil {
			return nil, err
		}
		parameterStrings[i] = GetUnexportGoName(field.GoName) + ` ` + fieldGoType
	}
	return parameterStrings, nil
}

// GetParameterErrorReturnString gets the return string for an error for a method.
func GetParameterErrorReturnString(
	generatedFile *protogen.GeneratedFile,
	fields []*protogen.Field,
	errorVarName string,
) (string, error) {
	varStrings := make([]string, len(fields)+1)
	for i, field := range fields {
		if err := ValidateFieldNotOneof(field); err != nil {
			return "", err
		}
		fieldGoZeroValue, err := GetFieldGoZeroValue(generatedFile, field)
		if err != nil {
			return "", err
		}
		varStrings[i] = fieldGoZeroValue
	}
	varStrings[len(varStrings)-1] = errorVarName
	return "return " + strings.Join(varStrings, ", "), nil
}

// GetFieldGoType returns the Go type used for a field.
//
// Adapted from https://github.com/protocolbuffers/protobuf-go/blob/81d297c66c9b1e0606eee19a9ee718dcf149276d/cmd/protoc-gen-go/internal_gengo/main.go#L640
// See https://github.com/protocolbuffers/protobuf-go/blob/81d297c66c9b1e0606eee19a9ee718dcf149276d/LICENSE for the license.
func GetFieldGoType(
	generatedFile *protogen.GeneratedFile,
	field *protogen.Field,
) (string, error) {
	if field.Desc.IsWeak() {
		return "struct{}", nil
	}
	var goType string
	pointer := field.Desc.HasPresence()
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		goType = "bool"
	case protoreflect.EnumKind:
		goType = generatedFile.QualifiedGoIdent(field.Enum.GoIdent)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		goType = "int32"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		goType = "uint32"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		goType = "int64"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		goType = "uint64"
	case protoreflect.FloatKind:
		goType = "float32"
	case protoreflect.DoubleKind:
		goType = "float64"
	case protoreflect.StringKind:
		goType = "string"
	case protoreflect.BytesKind:
		goType = "[]byte"
		pointer = false // rely on nullability of slices for presence
	case protoreflect.MessageKind, protoreflect.GroupKind:
		goType = "*" + generatedFile.QualifiedGoIdent(field.Message.GoIdent)
		pointer = false // pointer captured as part of the type
	default:
		return "", fmt.Errorf("unknown Kind: %T", field.Desc.Kind())
	}
	switch {
	case field.Desc.IsList():
		return "[]" + goType, nil
	case field.Desc.IsMap():
		keyType, err := GetFieldGoType(generatedFile, field.Message.Fields[0])
		if err != nil {
			return "", err
		}
		valType, err := GetFieldGoType(generatedFile, field.Message.Fields[1])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("map[%v]%v", keyType, valType), nil
	}
	if pointer {
		goType = "*" + goType
	}
	return goType, nil
}

// GetFieldGoZeroValue returns the go zero value for a field.
func GetFieldGoZeroValue(
	generatedFile *protogen.GeneratedFile,
	field *protogen.Field,
) (string, error) {
	if field.Desc.IsWeak() {
		return "struct{}", nil
	}
	if field.Desc.HasPresence() {
		return "nil", nil
	}
	if field.Desc.IsList() {
		return "nil", nil
	}
	if field.Desc.IsMap() {
		return "nil", nil
	}
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		return "false", nil
	case protoreflect.EnumKind:
		return generatedFile.QualifiedGoIdent(field.Enum.GoIdent) + "(0)", nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return "0", nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "0", nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "0", nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "0", nil
	case protoreflect.FloatKind:
		return "0", nil
	case protoreflect.DoubleKind:
		return "0", nil
	case protoreflect.StringKind:
		return `""`, nil
	case protoreflect.BytesKind:
		return "nil", nil
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return "nil", nil
	default:
		return "", fmt.Errorf("unknown Kind: %T", field.Desc.Kind())
	}
}

type handlerOptions struct {
	optionHandler func(string, string) error
}

func newHandlerOptions() *handlerOptions {
	return &handlerOptions{}
}
