package bufimageutil

import (
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/protosource"
)

type imageIndex struct {
	NameToDescriptor map[string]protosource.NamedDescriptor
	NameToExtensions map[string][]protosource.Field
	NameToOptions    map[string]map[int32]protosource.Field
}

func newImageIndexForImage(image bufimage.Image) (*imageIndex, error) {
	index := &imageIndex{
		NameToDescriptor: make(map[string]protosource.NamedDescriptor),
		NameToExtensions: make(map[string][]protosource.Field),
		NameToOptions:    make(map[string]map[int32]protosource.Field),
	}
	for _, file := range image.Files() {
		protosourceFile, err := protosource.NewFile(newInputFile(file))
		if err != nil {
			return nil, err
		}
		for _, field := range protosourceFile.Extensions() {
			index.NameToDescriptor[field.FullName()] = field
			extendeeName := strings.TrimPrefix(field.Extendee(), ".")
			if isOptionsTypeName(extendeeName) {
				if _, ok := index.NameToOptions[extendeeName]; !ok {
					index.NameToOptions[extendeeName] = make(map[int32]protosource.Field)
				}
				index.NameToOptions[extendeeName][int32(field.Number())] = field
			} else {
				index.NameToExtensions[extendeeName] = append(index.NameToExtensions[extendeeName], field)
			}
		}
		if err := protosource.ForEachMessage(func(message protosource.Message) error {
			if storedDescriptor, ok := index.NameToDescriptor[message.FullName()]; ok && storedDescriptor != message {
				return fmt.Errorf("duplicate for %q: %#v != %#v", message.FullName(), storedDescriptor, message)
			}
			index.NameToDescriptor[message.FullName()] = message

			for _, field := range message.Extensions() {
				index.NameToDescriptor[field.FullName()] = field
				extendeeName := strings.TrimPrefix(field.Extendee(), ".")
				if isOptionsTypeName(extendeeName) {
					if _, ok := index.NameToOptions[extendeeName]; !ok {
						index.NameToOptions[extendeeName] = make(map[int32]protosource.Field)
					}
					index.NameToOptions[extendeeName][int32(field.Number())] = field
				} else {
					index.NameToExtensions[extendeeName] = append(index.NameToExtensions[extendeeName], field)
				}
			}
			return nil
		}, protosourceFile); err != nil {
			return nil, err
		}
		if err = protosource.ForEachEnum(func(enum protosource.Enum) error {
			if storedDescriptor, ok := index.NameToDescriptor[enum.FullName()]; ok {
				return fmt.Errorf("duplicate for %q: %#v != %#v", enum.FullName(), storedDescriptor, enum)
			}
			index.NameToDescriptor[enum.FullName()] = enum
			return nil
		}, protosourceFile); err != nil {
			return nil, err
		}
		for _, service := range protosourceFile.Services() {
			if storedDescriptor, ok := index.NameToDescriptor[service.FullName()]; ok {
				return nil, fmt.Errorf("duplicate for %q: %#v != %#v", service.FullName(), storedDescriptor, service)
			}
			index.NameToDescriptor[service.FullName()] = service
		}
	}
	return index, nil
}

func isOptionsTypeName(typeName string) bool {
	switch typeName {
	case "google.protobuf.FileOptions",
		"google.protobuf.MessageOptions",
		"google.protobuf.FieldOptions",
		"google.protobuf.OneofOptions",
		"google.protobuf.EnumOptions",
		"google.protobuf.EnumValueOptions",
		"google.protobuf.ServiceOptions",
		"google.protobuf.MethodOptions":
		return true
	default:
		return false
	}
}
