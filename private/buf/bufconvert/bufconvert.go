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

package bufconvert

import (
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// ImageWithoutMessageSetWireFormatResolution returns an image with the
// same contents as the given image, but whose resolver refuses to
// resolve elements that refer to messages that use the message-set wire
// format.
//
// We do this because the protobuf-go runtime does not support message-set
// wire format (unless built with a "protolegacy" build tag). So if such a
// message were used, we would not be able to correctly serialize or
// de-serialize it.
func ImageWithoutMessageSetWireFormatResolution(image bufimage.Image) bufimage.Image {
	return &noResolveMessageSetWireFormatImage{image}
}

type noResolveMessageSetWireFormatImage struct {
	bufimage.Image
}

func (n *noResolveMessageSetWireFormatImage) Resolver() protoencoding.Resolver {
	return n
}

func (n *noResolveMessageSetWireFormatImage) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	return n.Image.Resolver().FindFileByPath(path)
}

func (n *noResolveMessageSetWireFormatImage) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	descriptor, err := n.Image.Resolver().FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	switch descriptor := descriptor.(type) {
	case protoreflect.MessageDescriptor:
		if err := checkNoMessageSetWireFormat(descriptor, n.Image, nil); err != nil {
			return nil, err
		}
	case protoreflect.ExtensionDescriptor:
		if descriptor.Message() != nil {
			if err := checkNoMessageSetWireFormat(descriptor.Message(), n.Image, nil); err != nil {
				return nil, err
			}
		}
	}
	return descriptor, nil
}

func (n *noResolveMessageSetWireFormatImage) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	extension, err := n.Image.Resolver().FindExtensionByName(field)
	if err != nil {
		return nil, err
	}
	if msgDescriptor := extension.TypeDescriptor().Message(); msgDescriptor != nil {
		if err := checkNoMessageSetWireFormat(msgDescriptor, n.Image, nil); err != nil {
			return nil, err
		}
	}
	return extension, nil
}

func (n *noResolveMessageSetWireFormatImage) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	extension, err := n.Image.Resolver().FindExtensionByNumber(message, field)
	if err != nil {
		return nil, err
	}
	if msgDescriptor := extension.TypeDescriptor().Message(); msgDescriptor != nil {
		if err := checkNoMessageSetWireFormat(msgDescriptor, n.Image, nil); err != nil {
			return nil, err
		}
	}
	return extension, nil
}

func (n *noResolveMessageSetWireFormatImage) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	messageType, err := n.Image.Resolver().FindMessageByName(message)
	if err != nil {
		return nil, err
	}
	if msgDescriptor := messageType.Descriptor(); msgDescriptor != nil {
		if err := checkNoMessageSetWireFormat(msgDescriptor, n.Image, nil); err != nil {
			return nil, err
		}
	}
	return messageType, nil
}

func (n *noResolveMessageSetWireFormatImage) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	messageType, err := n.Image.Resolver().FindMessageByURL(url)
	if err != nil {
		return nil, err
	}
	if msgDescriptor := messageType.Descriptor(); msgDescriptor != nil {
		if err := checkNoMessageSetWireFormat(msgDescriptor, n.Image, nil); err != nil {
			return nil, err
		}
	}
	return messageType, nil
}

func (n *noResolveMessageSetWireFormatImage) FindEnumByName(enum protoreflect.FullName) (protoreflect.EnumType, error) {
	return n.Image.Resolver().FindEnumByName(enum)
}

type messageSetNotSupportedError struct {
	typeName protoreflect.FullName
}

func (e *messageSetNotSupportedError) Error() string {
	return fmt.Sprintf("message type %q uses message-set wire format, which is not supported", e.typeName)
}

func checkNoMessageSetWireFormat(descriptor protoreflect.MessageDescriptor, image bufimage.Image, checked []protoreflect.FullName) error {
	// In order to create a protoreflect.MessageDescriptor, we may have already stripped
	// any message_set_wire_format option. So we must examine the file descriptor proto
	// in the image to see if the option is set. If it is, we cannot support this message.
	name := descriptor.FullName()
	for _, alreadyChecked := range checked {
		if name == alreadyChecked {
			return nil
		}
	}
	checked = append(checked, name)

	path := descriptor.ParentFile().Path()
	imageFile := image.GetFile(path)
	if imageFile == nil {
		// shouldn't actually be possible
		return fmt.Errorf("message type %q should be in file %q but that file was not found", name, path)
	}
	descriptorProto := findMessageInFile(name, imageFile.FileDescriptorProto())
	if descriptorProto == nil {
		// shouldn't actually be possible
		return fmt.Errorf("message type %q should be in file %q but it was not found", name, path)
	}
	if descriptorProto.GetOptions().GetMessageSetWireFormat() {
		return &messageSetNotSupportedError{name}
	}
	// Also check all message fields
	fields := descriptor.Fields()
	for i, length := 0, fields.Len(); i < length; i++ {
		field := fields.Get(i)
		if field.Message() == nil {
			continue
		}
		if err := checkNoMessageSetWireFormat(field.Message(), image, checked); err != nil {
			return err
		}
	}
	return nil
}

func findMessageInFile(name protoreflect.FullName, file *descriptorpb.FileDescriptorProto) *descriptorpb.DescriptorProto {
	if pkg := file.GetPackage(); pkg != "" {
		prefix := pkg + "."
		if !strings.HasPrefix(string(name), prefix) {
			// name has wrong package prefix
			return nil
		}
	}
	return findMessageInFileRec(name, file)
}

func findMessageInFileRec(name protoreflect.FullName, file *descriptorpb.FileDescriptorProto) *descriptorpb.DescriptorProto {
	simpleName := string(name.Name())
	parentName := name.Parent()

	if parentName == protoreflect.FullName(file.GetPackage()) {
		// This is a top-level message
		for _, descriptor := range file.MessageType {
			if descriptor.GetName() == simpleName {
				return descriptor
			}
		}
		return nil
	}

	// This is a nested message.
	parentMessage := findMessageInFileRec(parentName, file)
	if parentMessage == nil {
		return nil
	}
	for _, descriptor := range parentMessage.NestedType {
		if descriptor.GetName() == simpleName {
			return descriptor
		}
	}
	return nil
}
