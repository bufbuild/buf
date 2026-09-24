// Copyright 2020-2026 Buf Technologies, Inc.
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

package protosourcepath

import (
	"google.golang.org/protobuf/reflect/protoreflect"
)

// DescriptorForSourcePath returns the descriptor of the Protobuf declaration that the given
// source path points to, or nil if the source path does not point to a declaration.
//
// A source path to a declaration alternates a type tag and an index, descending from the
// FileDescriptorProto through the declarations that contain it. For example, the source path
// [4, 0, 3, 1, 2, 0] is .message_type(0).nested_type(1).field(0), the first field of the
// second nested message of the first message in the file. The empty source path points to the
// file itself, so the given file descriptor is returned for it.
//
// A source path that points to an attribute of a declaration rather than to the declaration
// itself, such as [4, 0, 1] for .message_type(0).name, returns nil. GetAssociatedSourcePaths
// can be used to resolve such a source path to the source paths of the declarations it
// belongs to.
func DescriptorForSourcePath(
	fileDescriptor protoreflect.FileDescriptor,
	sourcePath protoreflect.SourcePath,
) protoreflect.Descriptor {
	if len(sourcePath)%2 != 0 {
		// Each declaration is addressed by a type tag and an index, so a source path with an
		// odd number of elements cannot point to a declaration.
		return nil
	}
	descriptor := protoreflect.Descriptor(fileDescriptor)
	for ; len(sourcePath) > 0; sourcePath = sourcePath[2:] {
		typeTag, index := sourcePath[0], int(sourcePath[1])
		// The same type tag means different things depending on the descriptor it is read
		// against, so the descriptor we have descended to so far selects the tags to check.
		switch typedDescriptor := descriptor.(type) {
		case protoreflect.FileDescriptor:
			switch typeTag {
			case messagesTypeTag:
				descriptor = descriptorAtIndex(typedDescriptor.Messages(), index)
			case enumsTypeTag:
				descriptor = descriptorAtIndex(typedDescriptor.Enums(), index)
			case servicesTypeTag:
				descriptor = descriptorAtIndex(typedDescriptor.Services(), index)
			case extensionsTypeTag:
				descriptor = descriptorAtIndex(typedDescriptor.Extensions(), index)
			default:
				return nil
			}
		case protoreflect.MessageDescriptor:
			switch typeTag {
			case messageFieldsTypeTag:
				descriptor = descriptorAtIndex(typedDescriptor.Fields(), index)
			case nestedMessagesTypeTag:
				descriptor = descriptorAtIndex(typedDescriptor.Messages(), index)
			case nestedEnumsTypeTag:
				descriptor = descriptorAtIndex(typedDescriptor.Enums(), index)
			case messageExtensionsTypeTag:
				descriptor = descriptorAtIndex(typedDescriptor.Extensions(), index)
			case messageOneOfsTypeTag:
				descriptor = descriptorAtIndex(typedDescriptor.Oneofs(), index)
			default:
				return nil
			}
		case protoreflect.EnumDescriptor:
			if typeTag != enumValuesTypeTag {
				return nil
			}
			descriptor = descriptorAtIndex(typedDescriptor.Values(), index)
		case protoreflect.ServiceDescriptor:
			if typeTag != serviceMethodsTypeTag {
				return nil
			}
			descriptor = descriptorAtIndex(typedDescriptor.Methods(), index)
		default:
			// Fields, oneofs, enum values, and methods do not contain declarations.
			return nil
		}
		if descriptor == nil {
			return nil
		}
	}
	return descriptor
}

// *** PRIVATE ***

// descriptorList is the shape shared by protoreflect's descriptor list types, such as
// protoreflect.MessageDescriptors and protoreflect.FieldDescriptors.
type descriptorList[D protoreflect.Descriptor] interface {
	Len() int
	Get(i int) D
}

// descriptorAtIndex returns the descriptor at the given index, or nil if the index is out
// of range.
func descriptorAtIndex[D protoreflect.Descriptor, L descriptorList[D]](
	descriptors L,
	index int,
) protoreflect.Descriptor {
	if index < 0 || index >= descriptors.Len() {
		return nil
	}
	return descriptors.Get(index)
}
