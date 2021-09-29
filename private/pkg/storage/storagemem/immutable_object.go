package storagemem

import (
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
)

// immutableObject is an object that contains a path, external path,
// and data that is never modified.
//
// Do not use this as a builder.
type immutableObject struct {
	storageutil.ObjectInfo

	data []byte
}

// newImmutableObject returns a new Object.
//
// path is expected to always be non-empty.
// If externalPath is empty, normalpath.Unnormalize(path) is used.
func newImmutableObject(
	path string,
	externalPath string,
	data []byte,
) *immutableObject {
	if externalPath == "" {
		externalPath = normalpath.Unnormalize(path)
	}
	return &immutableObject{
		ObjectInfo: storageutil.NewObjectInfo(
			path,
			externalPath,
		),
		data: data,
	}
}
