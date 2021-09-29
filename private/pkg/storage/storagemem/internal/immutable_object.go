// Package internal splits out ImmutableObject into a separate package from storagemem
// to make it impossible to modify ImmutableObject via direct field access.
package internal

import (
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
)

// ImmutableObject is an object that contains a path, external path,
// and data that is never modified.
//
// We make this a struct so there is no weirdness with returning a nil interface.
type ImmutableObject struct {
	storageutil.ObjectInfo

	data []byte
}

// NewImmutableObject returns a new ImmutableObject.
//
// path is expected to always be non-empty.
// If externalPath is empty, normalpath.Unnormalize(path) is used.
func NewImmutableObject(
	path string,
	externalPath string,
	data []byte,
) *ImmutableObject {
	if externalPath == "" {
		externalPath = normalpath.Unnormalize(path)
	}
	return &ImmutableObject{
		ObjectInfo: storageutil.NewObjectInfo(path, externalPath),
		data:       data,
	}
}

// Data returns the data.
//
// DO NOT MODIFY.
func (i *ImmutableObject) Data() []byte {
	return i.data
}
