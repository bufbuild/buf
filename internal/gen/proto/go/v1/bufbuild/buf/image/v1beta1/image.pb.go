// Code generated by protoc-gen-go. DO NOT EDIT.
// source: bufbuild/buf/image/v1beta1/image.proto

package imagev1beta1

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

// Image is analogous to a FileDescriptorSet.
type Image struct {
	// file matches the file field of a FileDescriptorSet.
	File []*descriptor.FileDescriptorProto `protobuf:"bytes,1,rep,name=file" json:"file,omitempty"`
	// bufbuild_image_extension is the ImageExtension for this image.
	//
	// The prefixed name and high tag value is used to all but guarantee there
	// will never be any conflict with Google's FileDescriptorSet definition.
	// The definition of a FileDescriptorSet has not changed in 11 years, so
	// we're not too worried about a conflict here.
	BufbuildImageExtension *ImageExtension `protobuf:"bytes,8042,opt,name=bufbuild_image_extension,json=bufbuildImageExtension" json:"bufbuild_image_extension,omitempty"`
	XXX_NoUnkeyedLiteral   struct{}        `json:"-"`
	XXX_unrecognized       []byte          `json:"-"`
	XXX_sizecache          int32           `json:"-"`
}

func (m *Image) Reset()         { *m = Image{} }
func (m *Image) String() string { return proto.CompactTextString(m) }
func (*Image) ProtoMessage()    {}
func (*Image) Descriptor() ([]byte, []int) {
	return fileDescriptor_9e3606ec0a0627fd, []int{0}
}

func (m *Image) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Image.Unmarshal(m, b)
}
func (m *Image) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Image.Marshal(b, m, deterministic)
}
func (m *Image) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Image.Merge(m, src)
}
func (m *Image) XXX_Size() int {
	return xxx_messageInfo_Image.Size(m)
}
func (m *Image) XXX_DiscardUnknown() {
	xxx_messageInfo_Image.DiscardUnknown(m)
}

var xxx_messageInfo_Image proto.InternalMessageInfo

func (m *Image) GetFile() []*descriptor.FileDescriptorProto {
	if m != nil {
		return m.File
	}
	return nil
}

func (m *Image) GetBufbuildImageExtension() *ImageExtension {
	if m != nil {
		return m.BufbuildImageExtension
	}
	return nil
}

// ImageExtension contains extensions to Images.
//
// The fields are not included directly on the Image so that we can both
// detect if extensions exist, which signifies this was created by buf
// and not by protoc, and so that we can add fields in a freeform manner
// without worrying about conflicts with google.protobuf.FileDescriptorSet.
type ImageExtension struct {
	// image_import_refs are the image import references for this specific Image.
	//
	// A given FileDescriptorProto may or may not be an import depending on
	// the image context, so this information is not stored on each FileDescriptorProto.
	ImageImportRefs      []*ImageImportRef `protobuf:"bytes,1,rep,name=image_import_refs,json=imageImportRefs" json:"image_import_refs,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *ImageExtension) Reset()         { *m = ImageExtension{} }
func (m *ImageExtension) String() string { return proto.CompactTextString(m) }
func (*ImageExtension) ProtoMessage()    {}
func (*ImageExtension) Descriptor() ([]byte, []int) {
	return fileDescriptor_9e3606ec0a0627fd, []int{1}
}

func (m *ImageExtension) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ImageExtension.Unmarshal(m, b)
}
func (m *ImageExtension) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ImageExtension.Marshal(b, m, deterministic)
}
func (m *ImageExtension) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ImageExtension.Merge(m, src)
}
func (m *ImageExtension) XXX_Size() int {
	return xxx_messageInfo_ImageExtension.Size(m)
}
func (m *ImageExtension) XXX_DiscardUnknown() {
	xxx_messageInfo_ImageExtension.DiscardUnknown(m)
}

var xxx_messageInfo_ImageExtension proto.InternalMessageInfo

func (m *ImageExtension) GetImageImportRefs() []*ImageImportRef {
	if m != nil {
		return m.ImageImportRefs
	}
	return nil
}

// ImageImportRef is a reference to an image import.
//
// This is a message type instead of a scalar type so that we can add
// additional information about an import reference in the future, such as
// the external location of the import.
type ImageImportRef struct {
	// file_index is the index within the Image file array of the import.
	//
	// This signifies that file[file_index] is an import.
	// This field must be set.
	FileIndex            *uint32  `protobuf:"varint,1,opt,name=file_index,json=fileIndex" json:"file_index,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ImageImportRef) Reset()         { *m = ImageImportRef{} }
func (m *ImageImportRef) String() string { return proto.CompactTextString(m) }
func (*ImageImportRef) ProtoMessage()    {}
func (*ImageImportRef) Descriptor() ([]byte, []int) {
	return fileDescriptor_9e3606ec0a0627fd, []int{2}
}

func (m *ImageImportRef) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ImageImportRef.Unmarshal(m, b)
}
func (m *ImageImportRef) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ImageImportRef.Marshal(b, m, deterministic)
}
func (m *ImageImportRef) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ImageImportRef.Merge(m, src)
}
func (m *ImageImportRef) XXX_Size() int {
	return xxx_messageInfo_ImageImportRef.Size(m)
}
func (m *ImageImportRef) XXX_DiscardUnknown() {
	xxx_messageInfo_ImageImportRef.DiscardUnknown(m)
}

var xxx_messageInfo_ImageImportRef proto.InternalMessageInfo

func (m *ImageImportRef) GetFileIndex() uint32 {
	if m != nil && m.FileIndex != nil {
		return *m.FileIndex
	}
	return 0
}

func init() {
	proto.RegisterType((*Image)(nil), "bufbuild.buf.image.v1beta1.Image")
	proto.RegisterType((*ImageExtension)(nil), "bufbuild.buf.image.v1beta1.ImageExtension")
	proto.RegisterType((*ImageImportRef)(nil), "bufbuild.buf.image.v1beta1.ImageImportRef")
}

func init() {
	proto.RegisterFile("bufbuild/buf/image/v1beta1/image.proto", fileDescriptor_9e3606ec0a0627fd)
}

var fileDescriptor_9e3606ec0a0627fd = []byte{
	// 267 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x8f, 0x3f, 0x4b, 0xc4, 0x30,
	0x18, 0xc6, 0x09, 0xea, 0x60, 0xce, 0x3f, 0x18, 0x41, 0xca, 0x81, 0x50, 0x8a, 0x48, 0x71, 0x48,
	0xb8, 0x9b, 0x9c, 0x1c, 0x44, 0xc5, 0x6e, 0x92, 0xc1, 0xc1, 0xa5, 0x5c, 0xec, 0x9b, 0x1a, 0xe8,
	0x35, 0x25, 0x4d, 0xe5, 0x3e, 0x92, 0x9f, 0xcd, 0x4f, 0xe0, 0x28, 0x49, 0x9a, 0x42, 0x07, 0x71,
	0x7c, 0x9f, 0x3c, 0xbf, 0xf7, 0xf7, 0x06, 0x5f, 0x8b, 0x41, 0x8a, 0x41, 0x35, 0x15, 0x13, 0x83,
	0x64, 0x6a, 0xbb, 0xa9, 0x81, 0x7d, 0xae, 0x04, 0xd8, 0xcd, 0x2a, 0x4c, 0xb4, 0x33, 0xda, 0x6a,
	0xb2, 0x8c, 0x3d, 0x2a, 0x06, 0x49, 0xc3, 0xcb, 0xd8, 0x5b, 0xa6, 0xb5, 0xd6, 0x75, 0x03, 0xcc,
	0x37, 0xdd, 0x9a, 0x0a, 0xfa, 0x77, 0xa3, 0x3a, 0xab, 0x4d, 0xa0, 0xb3, 0x2f, 0x84, 0x0f, 0x0a,
	0xc7, 0x90, 0x5b, 0xbc, 0x2f, 0x55, 0x03, 0x09, 0x4a, 0xf7, 0xf2, 0xc5, 0xfa, 0x8a, 0x06, 0x94,
	0x46, 0x94, 0x3e, 0xa9, 0x06, 0x1e, 0x26, 0xfc, 0xc5, 0xc5, 0xdc, 0x13, 0x04, 0x70, 0x12, 0x6f,
	0x28, 0xbd, 0xbf, 0x84, 0x9d, 0x85, 0xb6, 0x57, 0xba, 0x4d, 0xbe, 0xef, 0x52, 0x94, 0x2f, 0xd6,
	0x37, 0xf4, 0xef, 0x2b, 0xa9, 0xf7, 0x3f, 0x46, 0x84, 0x5f, 0xc4, 0xea, 0x3c, 0xcf, 0x3e, 0xf0,
	0xc9, 0x3c, 0x21, 0xaf, 0xf8, 0x2c, 0xf8, 0xd4, 0xb6, 0xd3, 0xc6, 0x96, 0x06, 0x64, 0x3f, 0xde,
	0xff, 0xbf, 0xb0, 0xf0, 0x0c, 0x07, 0xc9, 0x4f, 0xd5, 0x6c, 0xee, 0x33, 0x36, 0x9a, 0xa6, 0x88,
	0x5c, 0x62, 0xec, 0xbe, 0x5a, 0xaa, 0xb6, 0x82, 0x5d, 0x82, 0x52, 0x94, 0x1f, 0xf3, 0x43, 0x97,
	0x14, 0x2e, 0xb8, 0x3f, 0x7f, 0x46, 0x6f, 0x47, 0x7e, 0xcb, 0xa8, 0xf8, 0x41, 0xe8, 0x37, 0x00,
	0x00, 0xff, 0xff, 0x2e, 0xf1, 0xc1, 0x5e, 0xc1, 0x01, 0x00, 0x00,
}
