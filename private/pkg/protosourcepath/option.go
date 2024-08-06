package protosourcepath

import (
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	uninterpretedOptionTypeTag = int32(999)
)

func options(token int32, sourcePath protoreflect.SourcePath, i int) (state, []protoreflect.SourcePath, error) {
	// All option paths are considered terminal, validate and terminate
	if token == uninterpretedOptionTypeTag {
		return nil, nil, newInvalidSourcePathError(sourcePath, "uninterpreted option path provided")
	}
	return nil, []protoreflect.SourcePath{slicesext.Copy(sourcePath)}, nil
}
