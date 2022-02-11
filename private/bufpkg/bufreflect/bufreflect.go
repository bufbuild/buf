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
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
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
	typedDescriptor, ok := descriptor.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("%q must be a message but is a %T", typeName, descriptor)
	}
	return dynamicpb.NewMessage(typedDescriptor), nil
}

// ParseSourceAndType returns the moduleReference and typeName from the source and type provided by the user.
// When source is not provided, we assume the type is a fully-qualified path to the type and try to parse it.
// Otherwise, if both source and type are provided, the type must be a valid Protobuf identifier (e.g. weather.v1.Units).
func ParseSourceAndType(
	ctx context.Context,
	flagSource string,
	flagType string,
) (moduleReference string, typeName string, _ error) {
	if flagSource != "" && flagType != "" {
		if err := validateTypeName(flagType); err != nil {
			return "", "", err
		}
		return flagSource, flagType, nil
	}
	if flagType == "" {
		return "", "", appcmd.NewInvalidArgumentError("type is required")
	}
	moduleReference, typeName, err := parseFullyQualifiedPath(flagType)
	if err != nil {
		return "", "", appcmd.NewInvalidArgumentErrorf("if source is not provided, the type need to be a fully-qualified path that includes the module reference, failed to parse the type: %v", err)
	}
	return moduleReference, typeName, nil
}

// parseFullyQualifiedPath parse a string in <buf.build/owner/repository#fully-qualified-type> or
// <buf.build/owner/repository:reference#fully-qualified-type> format into a module reference and a type name
func parseFullyQualifiedPath(
	fullyQualifiedPath string,
) (moduleRef string, typeName string, _ error) {
	if fullyQualifiedPath == "" {
		return "", "", appcmd.NewInvalidArgumentError("you must specify a fully qualified path")
	}
	components := strings.Split(fullyQualifiedPath, "#")
	if len(components) != 2 {
		return "", "", appcmd.NewInvalidArgumentErrorf("%q is not a valid fully qualified path", fullyQualifiedPath)
	}
	moduleReference, err := bufmoduleref.ModuleReferenceForString(components[0])
	if err != nil {
		return "", "", err
	}
	if err := validateTypeName(components[1]); err != nil {
		return "", "", err
	}
	return moduleReference.String(), components[1], nil
}

// validateTypeName validates that the typeName is well-formed, such that it has one or more
// '.'-delimited package components and no '/' elements.
func validateTypeName(typeName string) error {
	if fullName := protoreflect.FullName(typeName); !fullName.IsValid() {
		return fmt.Errorf("%q is not a valid fully qualified type name", fullName)
	}
	return nil
}
