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

package bufreflect

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// NewMessage returns a new dynamic proto.Message for the fully qualified typeName
// in the bufimage.Image.
func NewMessage(
	ctx context.Context,
	image bufimage.Image,
	typeName string,
) (proto.Message, error) {
	if err := ValidateTypeName(typeName); err != nil {
		return nil, err
	}
	messageType, err := image.Resolver().FindMessageByName(protoreflect.FullName(typeName))
	if err != nil {
		return nil, err
	}
	return messageType.New().Interface(), nil
}

// ValidateTypeName validates that the typeName is well-formed, such that it has one or more
// '.'-delimited package components and no '/' elements.
func ValidateTypeName(typeName string) error {
	if fullName := protoreflect.FullName(typeName); !fullName.IsValid() {
		return fmt.Errorf("%q is not a valid fully qualified type name", fullName)
	}
	return nil
}
