package protoencoding

import (
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func newResolver(fileDescriptorProtos ...*descriptorpb.FileDescriptorProto) (Resolver, error) {
	if len(fileDescriptorProtos) == 0 {
		return nil, nil
	}
	files, err := protodesc.NewFiles(
		&descriptorpb.FileDescriptorSet{
			File: fileDescriptorProtos,
		},
	)
	if err != nil {
		return nil, err
	}
	types := &protoregistry.Types{}
	var rangeErr error
	files.RangeFiles(func(fileDescriptor protoreflect.FileDescriptor) bool {
		if err := addMessagesToTypes(types, fileDescriptor.Messages()); err != nil {
			rangeErr = err
			return false
		}
		if err := addExtensionsToTypes(types, fileDescriptor.Extensions()); err != nil {
			rangeErr = err
			return false
		}
		// There is no way to do register enum, and it is not used
		// https://github.com/golang/protobuf/issues/1065
		// https://godoc.org/google.golang.org/protobuf/types/dynamicpb does not have NewEnumType
		return true
	})
	if rangeErr != nil {
		return nil, rangeErr
	}
	return types, nil
}

func addMessagesToTypes(types *protoregistry.Types, messageDescriptors protoreflect.MessageDescriptors) error {
	messagesLen := messageDescriptors.Len()
	for i := 0; i < messagesLen; i++ {
		messageDescriptor := messageDescriptors.Get(i)
		if err := types.RegisterMessage(dynamicpb.NewMessageType(messageDescriptor)); err != nil {
			return err
		}
		if err := addMessagesToTypes(types, messageDescriptor.Messages()); err != nil {
			return err
		}
		if err := addExtensionsToTypes(types, messageDescriptor.Extensions()); err != nil {
			return err
		}
	}
	return nil
}

func addExtensionsToTypes(types *protoregistry.Types, extensionDescriptors protoreflect.ExtensionDescriptors) error {
	extensionsLen := extensionDescriptors.Len()
	for i := 0; i < extensionsLen; i++ {
		extensionDescriptor := extensionDescriptors.Get(i)
		if err := types.RegisterExtension(dynamicpb.NewExtensionType(extensionDescriptor)); err != nil {
			return err
		}
	}
	return nil
}
