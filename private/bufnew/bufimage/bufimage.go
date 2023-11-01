package bufimage

import (
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/bufbuild/buf/private/pkg/storage"
	"google.golang.org/protobuf/types/descriptorpb"
)

type ImageFile interface {
	storage.ObjectInfo
	bufmodule.ModuleInfo

	FileDescriptorProto() *descriptorpb.FileDescriptorProto
	FileDescriptor() protodescriptor.FileDescriptor
	IsImport() bool
	IsSyntaxUnspecified() bool
	UnusedDependencyIndexes() int32

	isImageFile()
}

type Image interface {
	Files() []ImageFile
	GetFile(path string) ImageFile

	isImage()
}

type ImageSet interface {
	Images() []Image
}

type ImageWorkspace interface {
	ImageSet() ImageSet
	//Config() Config
}
