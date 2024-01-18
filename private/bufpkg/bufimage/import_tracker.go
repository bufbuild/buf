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

package bufimage

import (
	"strings"

	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

var anyMessageName = (*anypb.Any)(nil).ProtoReflect().Descriptor().FullName()

type importTracker struct {
	resolver protoencoding.Resolver
	used     map[string]map[string]struct{}
}

func (t *importTracker) markUsed(importer *imagev1.ImageFile, element string) {
	desc, err := t.resolver.FindDescriptorByName(protoreflect.FullName(strings.TrimPrefix(element, ".")))
	if err != nil {
		// TODO: Shouldn't be possible. If this happens, element is not in
		//       the resolved files, so nothing to mark anyway...
		return
	}
	importedFile := desc.ParentFile().Path()

	fileImports := t.used[importer.GetName()]
	if fileImports == nil {
		fileImports = map[string]struct{}{}
		t.used[importer.GetName()] = fileImports
	}
	for _, depPath := range importer.Dependency {
		if importedFile == depPath {
			// Found it!
			fileImports[depPath] = struct{}{}
			return
		}
	}
	// Not in any imports. So see if it is publicly imported.
	for _, depPath := range importer.Dependency {
		depFile, err := t.resolver.FindFileByPath(depPath)
		if err != nil {
			// Shouldn't be possible... bail.
			continue
		}
		if t.publiclyImports(depFile, importedFile) {
			// Found it!
			fileImports[depPath] = struct{}{}
			return
		}
	}
}

func (t *importTracker) publiclyImports(file protoreflect.FileDescriptor, importedFile string) bool {
	deps := file.Imports()
	for i, depsLen := 0, deps.Len(); i < depsLen; i++ {
		dep := deps.Get(i)
		if !dep.IsPublic {
			continue
		}
		if dep.Path() == importedFile {
			return true
		}
		if t.publiclyImports(dep, importedFile) {
			return true
		}
	}
	return false
}

func (t *importTracker) findUsedImports(protoImage *imagev1.Image) {
	for _, file := range protoImage.File {
		if len(file.Dependency) == 0 {
			// no imports so nothing to do
			continue
		}
		t.findUsedImportsInOptions(file, file.Options)
		for _, msg := range file.MessageType {
			t.findUsedImportsInMessage(file, msg)
		}
		for _, enum := range file.EnumType {
			t.findUsedImportsInEnum(file, enum)
		}
		for _, ext := range file.Extension {
			t.findUsedImportsInField(file, ext)
		}
		for _, svc := range file.Service {
			t.findUsedImportsInOptions(file, svc.Options)
			for _, method := range svc.Method {
				t.findUsedImportsInOptions(file, method.Options)
				t.markUsed(file, method.GetInputType())
				t.markUsed(file, method.GetOutputType())
			}
		}
	}
}

func (t *importTracker) findUsedImportsInMessage(file *imagev1.ImageFile, msg *descriptorpb.DescriptorProto) {
	t.findUsedImportsInOptions(file, msg.Options)
	for _, field := range msg.Field {
		t.findUsedImportsInField(file, field)
	}
	for _, oneof := range msg.OneofDecl {
		t.findUsedImportsInOptions(file, oneof.Options)
	}
	for _, extRange := range msg.ExtensionRange {
		t.findUsedImportsInOptions(file, extRange.Options)
	}

	for _, nestedMsg := range msg.NestedType {
		t.findUsedImportsInMessage(file, nestedMsg)
	}
	for _, enum := range msg.EnumType {
		t.findUsedImportsInEnum(file, enum)
	}
	for _, ext := range msg.Extension {
		t.findUsedImportsInField(file, ext)
	}
}

func (t *importTracker) findUsedImportsInEnum(file *imagev1.ImageFile, enum *descriptorpb.EnumDescriptorProto) {
	t.findUsedImportsInOptions(file, enum.Options)
	for _, value := range enum.Value {
		t.findUsedImportsInOptions(file, value.Options)
	}
}

func (t *importTracker) findUsedImportsInField(file *imagev1.ImageFile, field *descriptorpb.FieldDescriptorProto) {
	t.findUsedImportsInOptions(file, field.Options)
	switch field.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		descriptorpb.FieldDescriptorProto_TYPE_GROUP,
		descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		t.markUsed(file, field.GetTypeName())
	}
	extendee := field.GetExtendee()
	if extendee != "" {
		t.markUsed(file, extendee)
	}
}

func (t *importTracker) findUsedImportsInOptions(file *imagev1.ImageFile, optionMessage proto.Message) {
	optionMessage.ProtoReflect().Range(func(field protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		t.findUsedImportsInOptionValue(file, field, val)
		return true
	})
}

func (t *importTracker) findUsedImportsInOptionValue(file *imagev1.ImageFile, optionField protoreflect.FieldDescriptor, val protoreflect.Value) {
	if optionField.IsExtension() {
		t.markUsed(file, string(optionField.FullName()))
	}
	switch {
	case optionField.IsMap():
		if optionField.MapValue().Message() == nil {
			return // no messages to examine
		}
		val.Map().Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
			t.findUsedImportsInMessageValue(file, v.Message())
			return true
		})
	case optionField.IsList():
		if optionField.Message() == nil {
			return // no messages to examine
		}
		list := val.List()
		for i, l := 0, list.Len(); i < l; i++ {
			t.findUsedImportsInMessageValue(file, list.Get(i).Message())
		}
	case optionField.Message() != nil:
		t.findUsedImportsInMessageValue(file, val.Message())
	}
}

func (t *importTracker) findUsedImportsInMessageValue(file *imagev1.ImageFile, msg protoreflect.Message) {
	if msg.Descriptor().FullName() == anyMessageName {
		typeURLField := msg.Descriptor().Fields().ByNumber(1)
		if typeURLField == nil || typeURLField.Kind() != protoreflect.StringKind || typeURLField.IsList() {
			// ruh-roh... this should not happen
			return
		}
		valueField := msg.Descriptor().Fields().ByNumber(2)
		if valueField == nil || valueField.Kind() != protoreflect.BytesKind || valueField.IsList() {
			// oof, this should not happen
			return
		}

		typeURL := msg.Get(typeURLField).String()
		msgType, err := t.resolver.FindMessageByURL(typeURL)
		if err != nil {
			// message is not present in the image
			return
		}
		t.markUsed(file, string(msgType.Descriptor().FullName()))
		// process Any messages that might be nested inside this one
		value := msg.Get(valueField).Bytes()
		nestedMessage := msgType.New()
		err = proto.UnmarshalOptions{Resolver: t.resolver}.Unmarshal(value, nestedMessage.Interface())
		if err != nil {
			// bytes are not valid; skip it
			return
		}
		msg = nestedMessage // fall-through to recurse into this message
	}
	msg.Range(func(field protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		t.findUsedImportsInOptionValue(file, field, val)
		return true
	})
}
