package protodescpb

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

var (
	jsonMarshaler       = &jsonpb.Marshaler{}
	jsonIndentMarshaler = &jsonpb.Marshaler{
		Indent: "  ",
	}
)

type fileDescriptorSet struct {
	backing *descriptor.FileDescriptorSet
}

func newFileDescriptorSet(backing *descriptor.FileDescriptorSet) (*fileDescriptorSet, error) {
	fileDescriptorSet := &fileDescriptorSet{
		backing: backing,
	}
	if err := fileDescriptorSet.validate(); err != nil {
		return nil, err
	}
	return fileDescriptorSet, nil
}

func (f *fileDescriptorSet) GetFile() []FileDescriptor {
	fileDescriptors := make([]FileDescriptor, len(f.backing.File))
	for i, file := range f.backing.File {
		fileDescriptors[i] = file
	}
	return fileDescriptors
}

func (f *fileDescriptorSet) MarshalWire() ([]byte, error) {
	return proto.Marshal(f.backing)
}

func (f *fileDescriptorSet) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := jsonMarshaler.Marshal(buffer, f.backing); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (f *fileDescriptorSet) MarshalJSONIndent() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := jsonIndentMarshaler.Marshal(buffer, f.backing); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (f *fileDescriptorSet) MarshalText() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := proto.MarshalText(buffer, f.backing); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (f *fileDescriptorSet) Equal(other FileDescriptorSet) bool {
	otherFileDescriptorSet, ok := other.(*fileDescriptorSet)
	if !ok {
		return false
	}
	return proto.Equal(f.backing, otherFileDescriptorSet.backing)
}

func (f *fileDescriptorSet) validate() error {
	if f.backing == nil {
		return errors.New("validate error: nil FileDescriptorSet")
	}
	if len(f.backing.File) == 0 {
		return errors.New("validate error: empty FileDescriptorSet.File")
	}
	seenNames := make(map[string]struct{}, len(f.backing.File))
	for _, file := range f.backing.File {
		if file == nil {
			return errors.New("validate error: nil FileDescriptorProto")
		}
		if file.Name == nil {
			return errors.New("validate error: nil FileDescriptorProto.Name")
		}
		name := *file.Name
		if name == "" {
			return errors.New("validate error: empty FileDescriptorProto.Name")
		}
		if _, ok := seenNames[name]; ok {
			return fmt.Errorf("validate error: duplicate FileDescriptorProto.Name: %q", name)
		}
		seenNames[name] = struct{}{}
		normalizedName, err := storagepath.NormalizeAndValidate(name)
		if err != nil {
			return fmt.Errorf("validate error: %v", err)
		}
		if name != normalizedName {
			return fmt.Errorf("validate error: FileDescriptorProto.Name %q has normalized name %q", name, normalizedName)
		}
	}
	return nil
}
