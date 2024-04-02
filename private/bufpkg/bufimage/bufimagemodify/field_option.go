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

package bufimagemodify

import (
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/protocompile/walk"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// jsTypeSubPath is the SourceCodeInfo sub path for the jstype field option.
// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L215
// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L567
var jsTypeSubPath = []int32{8, 6}

func modifyJsType(
	sweeper internal.MarkSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	modifyOptions := newModifyOptions()
	for _, option := range options {
		option(modifyOptions)
	}
	overrideRules := slicesext.Filter(
		config.Overrides(),
		func(override bufconfig.ManagedOverrideRule) bool {
			return override.FieldOption() == bufconfig.FieldOptionJSType
		},
	)
	// Unless specified, js type is not modified.
	if len(overrideRules) == 0 {
		return nil
	}
	// If the entire file is disabled, skip.
	if isFieldOptionDisabledForFile(imageFile, bufconfig.FieldOptionJSType, config, "") {
		return nil
	}
	if datawkt.Exists(imageFile.Path()) {
		return nil
	}
	return walk.DescriptorProtosWithPath(
		imageFile.FileDescriptorProto(),
		func(
			fullName protoreflect.FullName,
			path protoreflect.SourcePath,
			message proto.Message,
		) error {
			fieldDescriptor, ok := message.(*descriptorpb.FieldDescriptorProto)
			if !ok {
				return nil
			}
			// If the field is disabled, skip.
			if isFieldOptionDisabledForFile(imageFile, bufconfig.FieldOptionJSType, config, fullName) {
				return nil
			}
			var jsType *descriptorpb.FieldOptions_JSType
			for _, override := range config.Overrides() {
				if fileMatchConfig(imageFile, override.Path(), override.ModuleFullName()) &&
					(override.FieldName() == "" || override.FieldName() == string(fullName)) &&
					override.FieldOption() == bufconfig.FieldOptionJSType {
					jsTypeValue, ok := override.Value().(descriptorpb.FieldOptions_JSType)
					if !ok {
						return fmt.Errorf("invalid js_type override value of type %T", override.Value())
					}
					jsType = &jsTypeValue
				}
			}
			if jsType == nil {
				return nil
			}
			if modifyOptions.preserveExisting && fieldDescriptor.Options != nil && fieldDescriptor.Options.Jstype != nil {
				return nil
			}
			if fieldDescriptor.Type == nil || !isJsTypePermittedForType(*fieldDescriptor.Type) {
				return nil
			}
			if options := fieldDescriptor.Options; options != nil {
				if existingJSTYpe := options.Jstype; existingJSTYpe != nil && *existingJSTYpe == *jsType {
					return nil
				}
			}
			if fieldDescriptor.Options == nil {
				fieldDescriptor.Options = &descriptorpb.FieldOptions{}
			}
			fieldDescriptor.Options.Jstype = jsType
			if len(path) > 0 {
				jsTypeOptionPath := append(path, jsTypeSubPath...)
				sweeper.Mark(imageFile, jsTypeOptionPath)
			}
			return nil
		},
	)
}

// *** PRIVATE ***

func isJsTypePermittedForType(fieldType descriptorpb.FieldDescriptorProto_Type) bool {
	// https://github.com/protocolbuffers/protobuf/blob/d4db41d395dcbb2c79b7fb1f109086fa04afd8aa/src/google/protobuf/descriptor.proto#L622
	return fieldType == descriptorpb.FieldDescriptorProto_TYPE_INT64 ||
		fieldType == descriptorpb.FieldDescriptorProto_TYPE_UINT64 ||
		fieldType == descriptorpb.FieldDescriptorProto_TYPE_SINT64 ||
		fieldType == descriptorpb.FieldDescriptorProto_TYPE_FIXED64 ||
		fieldType == descriptorpb.FieldDescriptorProto_TYPE_SFIXED64
}

func isFieldOptionDisabledForFile(
	imageFile bufimage.ImageFile,
	fieldOption bufconfig.FieldOption,
	config bufconfig.GenerateManagedConfig,
	fullName protoreflect.FullName,
) (isDisabled bool) {
	for _, disableRule := range config.Disables() {
		if disableRule.FileOption() != bufconfig.FileOptionUnspecified {
			continue // FileOption specified, not a matching rule.
		}
		if disableRule.FieldOption() != bufconfig.FieldOptionUnspecified && disableRule.FieldOption() != fieldOption {
			continue // FieldOption specified, does not match option.
		}
		if disableRule.FieldName() != "" && disableRule.FieldName() != string(fullName) {
			continue // FieldName specified, does not match field.
		}
		if !fileMatchConfig(imageFile, disableRule.Path(), disableRule.ModuleFullName()) {
			continue
		}
		return true
	}
	return false
}
