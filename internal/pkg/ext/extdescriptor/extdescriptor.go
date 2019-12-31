package extdescriptor

import (
	"errors"
	"fmt"

	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin_go "github.com/golang/protobuf/protoc-gen-go/plugin"
)

// TODO: evaluate the normalization of names
// Right now we make sure every input name is normalized and valid.
// This is always the case with buf input, but may not be the case with protoc input.

// ValidateFileDescriptorSet validates a FileDescriptorSet.
func ValidateFileDescriptorSet(fileDescriptorSet *descriptor.FileDescriptorSet) error {
	if fileDescriptorSet == nil {
		return errors.New("validate error: nil FileDescriptorSet")
	}
	return ValidateFileDescriptorProtos(fileDescriptorSet.File)
}

// ValidateFileDescriptorProtos validates multiple FileDescriptorProtos.
func ValidateFileDescriptorProtos(fileDescriptorProtos []*descriptor.FileDescriptorProto) error {
	if len(fileDescriptorProtos) == 0 {
		return errors.New("validate error: empty FileDescriptorProtos")
	}
	seenNames := make(map[string]struct{}, len(fileDescriptorProtos))
	for _, fileDescriptorProto := range fileDescriptorProtos {
		if err := ValidateFileDescriptorProto(fileDescriptorProto); err != nil {
			return err
		}
		name := fileDescriptorProto.GetName()
		if _, ok := seenNames[name]; ok {
			return fmt.Errorf("validate error: duplicate FileDescriptorProto.Name: %q", name)
		}
		seenNames[name] = struct{}{}
	}
	return nil
}

// ValidateFileDescriptorProto validates a FileDescriptorProto.
func ValidateFileDescriptorProto(fileDescriptorProto *descriptor.FileDescriptorProto) error {
	if fileDescriptorProto == nil {
		return errors.New("validate error: nil FileDescriptorProto")
	}
	if fileDescriptorProto.Name == nil {
		return errors.New("validate error: nil FileDescriptorProto.Name")
	}
	name := fileDescriptorProto.GetName()
	if name == "" {
		return errors.New("validate error: empty FileDescriptorProto.Name")
	}
	normalizedName, err := storagepath.NormalizeAndValidate(name)
	if err != nil {
		return fmt.Errorf("validate error: %v", err)
	}
	if name != normalizedName {
		return fmt.Errorf("validate error: FileDescriptorProto.Name %q has normalized name %q", name, normalizedName)
	}
	return nil
}

// FileDescriptorSetWithSpecificNames returns a copy of the FileDescriptorSet with only the Files with the given names.
//
// Names are normalized and validated.
// If allowNotExist is false, the specific names must exist on the input FileDescriptorSet.
// Backing FileDescriptorProtos are not copied, only the references are copied.
//
// Validates the input and output.
func FileDescriptorSetWithSpecificNames(
	fileDescriptorSet *descriptor.FileDescriptorSet,
	allowNotExist bool,
	specificNames ...string,
) (*descriptor.FileDescriptorSet, error) {
	if err := ValidateFileDescriptorSet(fileDescriptorSet); err != nil {
		return nil, err
	}
	// If no modifications would be made, then we return the original
	if len(specificNames) == 0 {
		return fileDescriptorSet, nil
	}

	newFileDescriptorSet := &descriptor.FileDescriptorSet{}

	specificNamesMap := make(map[string]struct{}, len(specificNames))
	for _, specificName := range specificNames {
		normalizedName, err := storagepath.NormalizeAndValidate(specificName)
		if err != nil {
			return nil, err
		}
		specificNamesMap[normalizedName] = struct{}{}
	}

	if !allowNotExist {
		allNamesMap := make(map[string]struct{}, len(fileDescriptorSet.File))
		for _, file := range fileDescriptorSet.File {
			allNamesMap[file.GetName()] = struct{}{}
		}
		for specificName := range specificNamesMap {
			if _, ok := allNamesMap[specificName]; !ok {
				return nil, fmt.Errorf("%s is not present in the FileDescriptorSet", specificName)
			}
		}
	}

	for _, file := range fileDescriptorSet.File {
		// we already know that file.GetName() is normalized and validated from validation
		if _, add := specificNamesMap[file.GetName()]; !add {
			continue
		}
		newFileDescriptorSet.File = append(newFileDescriptorSet.File, file)
	}
	if len(newFileDescriptorSet.File) == 0 {
		return nil, errors.New("no input files match the given names")
	}
	if err := ValidateFileDescriptorSet(newFileDescriptorSet); err != nil {
		return nil, err
	}
	return newFileDescriptorSet, nil
}

// FileDescriptorSetToCodeGeneratorRequest converts the FileDescriptorSet to a CodeGeneratorRequest.
//
// The files to generate must be within the FileDescriptorSet.
// Files to generate are normalized and validated.
//
// Validates the input.
func FileDescriptorSetToCodeGeneratorRequest(
	fileDescriptorSet *descriptor.FileDescriptorSet,
	parameter string,
	fileToGenerate ...string,
) (*plugin_go.CodeGeneratorRequest, error) {
	if err := ValidateFileDescriptorSet(fileDescriptorSet); err != nil {
		return nil, err
	}
	if len(fileToGenerate) == 0 {
		return nil, fmt.Errorf("no file to generate")
	}
	normalizedFileToGenerate := make([]string, len(fileToGenerate))
	for i, elem := range fileToGenerate {
		normalized, err := storagepath.NormalizeAndValidate(elem)
		if err != nil {
			return nil, err
		}
		normalizedFileToGenerate[i] = normalized
	}
	namesMap := make(map[string]struct{}, len(fileDescriptorSet.File))
	for _, file := range fileDescriptorSet.File {
		namesMap[file.GetName()] = struct{}{}
	}
	for _, normalized := range normalizedFileToGenerate {
		if _, ok := namesMap[normalized]; !ok {
			return nil, fmt.Errorf("file to generate %q is not within the fileDescriptorSet", normalized)
		}
	}
	var parameterPtr *string
	if parameter != "" {
		parameterPtr = proto.String(parameter)
	}
	return &plugin_go.CodeGeneratorRequest{
		FileToGenerate: normalizedFileToGenerate,
		Parameter:      parameterPtr,
		ProtoFile:      fileDescriptorSet.File,
	}, nil
}

// CodeGeneratorRequestToFileDescriptorSet converts the CodeGeneratorRequest to an FileDescriptorSet.
//
// Validates the output.
func CodeGeneratorRequestToFileDescriptorSet(request *plugin_go.CodeGeneratorRequest) (*descriptor.FileDescriptorSet, error) {
	fileDescriptorSet := &descriptor.FileDescriptorSet{
		File: request.GetProtoFile(),
	}
	return FileDescriptorSetWithSpecificNames(fileDescriptorSet, false, request.FileToGenerate...)
}
