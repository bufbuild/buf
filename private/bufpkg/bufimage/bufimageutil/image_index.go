// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufimageutil

import (
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/protocompile/walk"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// imageIndex holds an index that allows for easily navigating a descriptor
// hierarchy and its relationships.
type imageIndex struct {
	// ByDescriptor maps descriptor proto pointers to information about the
	// element. The info includes the actual descriptor proto, its parent
	// element (if it has one), and the file in which it is defined.
	ByDescriptor map[namedDescriptor]elementInfo
	// ByName maps fully qualified type names to information about the named
	// element.
	ByName map[protoreflect.FullName]elementInfo
	// Files maps file names to the file descriptor protos.
	Files map[string]*descriptorpb.FileDescriptorProto

	// NameToExtensions maps fully qualified type names to all known
	// extension definitions for a type name.
	NameToExtensions map[string][]*descriptorpb.FieldDescriptorProto

	// NameToOptions maps `google.protobuf.*Options` type names to their
	// known extensions by field tag.
	NameToOptions map[string]map[int32]*descriptorpb.FieldDescriptorProto

	// Packages maps package names to package contents.
	Packages map[string]*packageInfo
}

type namedDescriptor interface {
	proto.Message
	GetName() string
}

var _ namedDescriptor = (*descriptorpb.FileDescriptorProto)(nil)
var _ namedDescriptor = (*descriptorpb.DescriptorProto)(nil)
var _ namedDescriptor = (*descriptorpb.FieldDescriptorProto)(nil)
var _ namedDescriptor = (*descriptorpb.OneofDescriptorProto)(nil)
var _ namedDescriptor = (*descriptorpb.EnumDescriptorProto)(nil)
var _ namedDescriptor = (*descriptorpb.EnumValueDescriptorProto)(nil)
var _ namedDescriptor = (*descriptorpb.ServiceDescriptorProto)(nil)
var _ namedDescriptor = (*descriptorpb.MethodDescriptorProto)(nil)

type elementInfo struct {
	fullName   protoreflect.FullName
	imageFile  bufimage.ImageFile    //string // TODO: maybe bufimage.ImageFile?
	parentName protoreflect.FullName //namedDescriptor
	element    namedDescriptor
}

type packageInfo struct {
	fullName    protoreflect.FullName
	files       []bufimage.ImageFile
	types       []protoreflect.FullName
	subPackages []*packageInfo
}

// newImageIndexForImage builds an imageIndex for a given image.
func newImageIndexForImage(image bufimage.Image, options *imageFilterOptions) (*imageIndex, error) {
	index := &imageIndex{
		ByName:       make(map[protoreflect.FullName]elementInfo),
		ByDescriptor: make(map[namedDescriptor]elementInfo),
		Files:        make(map[string]*descriptorpb.FileDescriptorProto),
		Packages:     make(map[string]*packageInfo),
	}
	if options.includeCustomOptions {
		index.NameToOptions = make(map[string]map[int32]*descriptorpb.FieldDescriptorProto)
	}
	if options.includeKnownExtensions {
		index.NameToExtensions = make(map[string][]*descriptorpb.FieldDescriptorProto)
	}

	for _, imageFile := range image.Files() {
		pkg := addPackageToIndex(imageFile.FileDescriptorProto().GetPackage(), index)
		pkg.files = append(pkg.files, imageFile)
		fileName := imageFile.Path()
		fileDescriptorProto := imageFile.FileDescriptorProto()
		index.Files[fileName] = fileDescriptorProto
		err := walk.DescriptorProtos(fileDescriptorProto, func(name protoreflect.FullName, msg proto.Message) error {
			if _, existing := index.ByName[name]; existing {
				return fmt.Errorf("duplicate for %q", name)
			}
			descriptor, ok := msg.(namedDescriptor)
			if !ok {
				return fmt.Errorf("unexpected descriptor type %T", msg)
			}
			var parentName protoreflect.FullName
			if pos := strings.LastIndexByte(string(name), '.'); pos != -1 {
				parentName = name[:pos]
			}

			// certain descriptor types don't need to be indexed:
			//  enum values, normal (non-extension) fields, and oneofs
			var includeInIndex bool
			switch d := descriptor.(type) {
			case *descriptorpb.EnumValueDescriptorProto, *descriptorpb.OneofDescriptorProto:
				// do not add to package elements; these elements are implicitly included by their enclosing type
			case *descriptorpb.FieldDescriptorProto:
				// only add to elements if an extension (regular fields implicitly included by containing message)
				includeInIndex = d.Extendee != nil
			default:
				includeInIndex = true
			}

			if includeInIndex {
				info := elementInfo{
					fullName:   name,
					imageFile:  imageFile,
					parentName: parentName,
					element:    descriptor,
				}
				index.ByName[name] = info
				index.ByDescriptor[descriptor] = info
				pkg.types = append(pkg.types, name)
			}

			ext, ok := descriptor.(*descriptorpb.FieldDescriptorProto)
			if !ok || ext.Extendee == nil {
				// not an extension, so the rest does not apply
				return nil
			}

			extendeeName := strings.TrimPrefix(ext.GetExtendee(), ".")
			if options.includeCustomOptions && isOptionsTypeName(extendeeName) {
				if _, ok := index.NameToOptions[extendeeName]; !ok {
					index.NameToOptions[extendeeName] = make(map[int32]*descriptorpb.FieldDescriptorProto)
				}
				index.NameToOptions[extendeeName][ext.GetNumber()] = ext
			}
			if options.includeKnownExtensions {
				index.NameToExtensions[extendeeName] = append(index.NameToExtensions[extendeeName], ext)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return index, nil
}

func addPackageToIndex(pkgName string, index *imageIndex) *packageInfo {
	pkg := index.Packages[pkgName]
	if pkg != nil {
		return pkg
	}
	pkg = &packageInfo{
		fullName: protoreflect.FullName(pkgName),
	}
	index.Packages[pkgName] = pkg
	if pkgName == "" {
		return pkg
	}
	var parentPkgName string
	if pos := strings.LastIndexByte(pkgName, '.'); pos != -1 {
		parentPkgName = pkgName[:pos]
	}
	parentPkg := addPackageToIndex(parentPkgName, index)
	parentPkg.subPackages = append(parentPkg.subPackages, pkg)
	return pkg
}

func isOptionsTypeName(typeName string) bool {
	switch typeName {
	case "google.protobuf.FileOptions",
		"google.protobuf.MessageOptions",
		"google.protobuf.FieldOptions",
		"google.protobuf.OneofOptions",
		"google.protobuf.ExtensionRangeOptions",
		"google.protobuf.EnumOptions",
		"google.protobuf.EnumValueOptions",
		"google.protobuf.ServiceOptions",
		"google.protobuf.MethodOptions":
		return true
	default:
		return false
	}
}

func getFullName(parentName protoreflect.FullName, descriptor namedDescriptor) protoreflect.FullName {
	fullName := protoreflect.FullName(descriptor.GetName())
	if parentName == "" {
		return fullName
	}
	return parentName + "." + fullName
}
