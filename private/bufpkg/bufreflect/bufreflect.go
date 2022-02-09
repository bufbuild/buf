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

package bufreflect

import (
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// NewMessage returns a new dynamic proto.Message for the fully qualified typeName
// in the bufimage.Image.
func NewMessage(
	ctx context.Context,
	image bufimage.Image,
	typeName string,
) (proto.Message, error) {
	if err := validateTypeName(typeName); err != nil {
		return nil, err
	}
	files, err := protodesc.NewFiles(bufimage.ImageToFileDescriptorSet(image))
	if err != nil {
		return nil, err
	}
	descriptor, err := files.FindDescriptorByName(protoreflect.FullName(typeName))
	if err != nil {
		return nil, err
	}
	switch typedDescriptor := descriptor.(type) {
	case protoreflect.MessageDescriptor:
		return dynamicpb.NewMessage(typedDescriptor), nil
	default:
		return nil, fmt.Errorf("%q must be a message but is a %T", typeName, typedDescriptor)
	}
}

func ParseFullyQualifiedPath(
	moduleTypeName string,
) (moduleName string, typeName string, _ error) {
	if moduleTypeName == "" {
		return "", "", appcmd.NewInvalidArgumentError("you must specify a module type name")
	}
	components := strings.Split(moduleTypeName, "/")
	if len(components) != 4 {
		return "", "", appcmd.NewInvalidArgumentErrorf("%q is not a valid fully qualified path", moduleTypeName)
	}
	if err := validateTypeName(components[3]); err != nil {
		return "", "", err
	}
	return moduleTypeName[:strings.LastIndex(moduleTypeName, "/")], components[3], nil
}

// validateTypeName validates that the typeName is well-formed, such that it has one or more
// '.'-delimited package components and no '/' elements.
func validateTypeName(typeName string) error {
	if fullName := protoreflect.FullName(typeName); !fullName.IsValid() {
		return fmt.Errorf("%q is not a valid fully qualified type name", fullName)
	}
	return nil
}
