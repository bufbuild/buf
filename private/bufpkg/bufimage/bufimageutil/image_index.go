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
	// NameToExtensions maps fully qualified type names to all known
	// extension definitions for a type name.
	NameToExtensions map[protoreflect.FullName][]*descriptorpb.FieldDescriptorProto
	// NameToOptions maps `google.protobuf.*Options` type names to their
	// known extensions by field tag.
	NameToOptions map[protoreflect.FullName]map[int32]*descriptorpb.FieldDescriptorProto
	// Packages maps package names to package contents.
	Packages map[string]*packageInfo
	// FileTypes maps file names to the fully qualified type names defined
	// in the file.
	FileTypes map[string][]protoreflect.FullName
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
	fullName protoreflect.FullName
	file     bufimage.ImageFile
	parent   namedDescriptor
	element  namedDescriptor
}

type packageInfo struct {
	fullName    protoreflect.FullName
	files       []bufimage.ImageFile
	subPackages []*packageInfo
}

// newImageIndexForImage builds an imageIndex for a given image.
func newImageIndexForImage(image bufimage.Image, options *imageFilterOptions) (*imageIndex, error) {
	index := &imageIndex{
		ByName:       make(map[protoreflect.FullName]elementInfo),
		ByDescriptor: make(map[namedDescriptor]elementInfo),
		Packages:     make(map[string]*packageInfo),
		FileTypes:    make(map[string][]protoreflect.FullName),
	}
	if options.includeCustomOptions {
		index.NameToOptions = make(map[protoreflect.FullName]map[int32]*descriptorpb.FieldDescriptorProto)
	}
	if options.includeKnownExtensions {
		index.NameToExtensions = make(map[protoreflect.FullName][]*descriptorpb.FieldDescriptorProto)
	}

	for _, imageFile := range image.Files() {
		pkg := addPackageToIndex(imageFile.FileDescriptorProto().GetPackage(), index)
		pkg.files = append(pkg.files, imageFile)
		fileName := imageFile.Path()
		fileDescriptorProto := imageFile.FileDescriptorProto()
		index.ByDescriptor[fileDescriptorProto] = elementInfo{
			fullName: pkg.fullName,
			file:     imageFile,
			element:  fileDescriptorProto,
			parent:   nil,
		}
		err := walk.DescriptorProtos(fileDescriptorProto, func(name protoreflect.FullName, msg proto.Message) error {
			if _, existing := index.ByName[name]; existing {
				return fmt.Errorf("duplicate for %q", name)
			}
			descriptor, ok := msg.(namedDescriptor)
			if !ok {
				return fmt.Errorf("unexpected descriptor type %T", msg)
			}
			var parent namedDescriptor = fileDescriptorProto
			if pos := strings.LastIndexByte(string(name), '.'); pos != -1 {
				parent = index.ByName[name[:pos]].element
				if parent == nil {
					// parent name was a package name, not an element name
					parent = fileDescriptorProto
				}
			}

			// certain descriptor types don't need to be indexed:
			//  enum values, normal (non-extension) fields, and oneofs
			switch d := descriptor.(type) {
			case *descriptorpb.EnumValueDescriptorProto, *descriptorpb.OneofDescriptorProto:
				// do not add to package elements; these elements are implicitly included by their enclosing type
				return nil
			case *descriptorpb.FieldDescriptorProto:
				// only add to elements if an extension (regular fields implicitly included by containing message)
				if d.Extendee == nil {
					return nil
				}
				extendeeName := protoreflect.FullName(strings.TrimPrefix(d.GetExtendee(), "."))
				if options.includeCustomOptions && isOptionsTypeName(extendeeName) {
					if _, ok := index.NameToOptions[extendeeName]; !ok {
						index.NameToOptions[extendeeName] = make(map[int32]*descriptorpb.FieldDescriptorProto)
					}
					index.NameToOptions[extendeeName][d.GetNumber()] = d
				}
				if options.includeKnownExtensions {
					index.NameToExtensions[extendeeName] = append(index.NameToExtensions[extendeeName], d)
				}
			}

			info := elementInfo{
				fullName: name,
				file:     imageFile,
				parent:   parent,
				element:  descriptor,
			}
			index.ByName[name] = info
			index.ByDescriptor[descriptor] = info
			index.FileTypes[fileName] = append(index.FileTypes[fileName], name)

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

func isOptionsTypeName(typeName protoreflect.FullName) bool {
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
