// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufconfig

import (
	"context"
	"errors"
	"io/fs"

	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// ObjectData is the data of the underlying storage.ReadObject that was used to create a Object.
//
// This is present on Files if they were created from storage.ReadBuckets. It is not present
// if the File was created via a New constructor or Read method.
type ObjectData interface {
	// Name returns the name of the underlying storage.ReadObject.
	//
	// This will be normalpath.Base(readObject.Path()).
	Name() string
	// Data returns the data from the underlying storage.ReadObject.
	Data() []byte

	isObjectData()
}

// GetBufYAMLV1Beta1OrV1ObjectDataForPrefix is a helper function that gets the ObjectData for the buf.yaml file at
// the given bucket prefix, if the buf.yaml file was v1beta1 or v1.
//
// The file is only parsed for its file version. No additional validation is performed.
// If the file does not exist, an error that satisfies fs.ErrNotExist is returned.
//
// This function is used to help optionally get ObjectData where it is needed for digest calculations.
func GetBufYAMLV1Beta1OrV1ObjectDataForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (ObjectData, error) {
	return getV1Beta1OrV1ObjectDataForPrefix(ctx, bucket, prefix, bufYAMLFileNames, bufYAMLFileNameToSupportedFileVersions)
}

// GetBufLockV1Beta1OrV1ObjectDataForPrefix is a helper function that gets the ObjectData for the buf.lock file at
// the given bucket prefix, if the buf.lock file was v1beta1 or v1.
//
// The file is only parsed for its file version. No additional validation is performed.
// If the file does not exist, an error that satisfies fs.ErrNotExist is returned.
//
// This function is used to help optionally get ObjectData where it is needed for digest calculations.
func GetBufLockV1Beta1OrV1ObjectDataForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (ObjectData, error) {
	return getV1Beta1OrV1ObjectDataForPrefix(ctx, bucket, prefix, bufLockFileNames, bufLockFileNameToSupportedFileVersions)
}

// *** PRIVATE ***

type objectData struct {
	name string
	data []byte
}

func newObjectData(name string, data []byte) *objectData {
	return &objectData{
		name: name,
		data: data,
	}
}

func (f *objectData) Name() string {
	return f.name
}

func (f *objectData) Data() []byte {
	return f.data
}

func (*objectData) isObjectData() {}

func getV1Beta1OrV1ObjectDataForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	fileNames []string,
	fileNameToSupportedFileVersions map[string]map[FileVersion]struct{},
) (ObjectData, error) {
	if len(fileNames) == 0 {
		return nil, syserror.New("expected at least one file name for getV1Beta1OrV1ObjectDataForPrefix")
	}
	for _, fileName := range fileNames {
		path := normalpath.Join(prefix, fileName)
		data, err := storage.ReadPath(ctx, bucket, path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}
		// As soon as we find a file with the correct name, we have to roll with that. We don't
		// then cascade to other file names if there was a parse error.

		if len(data) == 0 {
			return nil, nil
		}
		var externalFileVersion externalFileVersion
		if err := encoding.UnmarshalYAMLNonStrict(data, &externalFileVersion); err != nil {
			// This could be a source of bugs in the future - we likely just took a buf.yaml/buf.lock
			// as-is for digest calculations pre-refactor, and didn't require a version.
			return nil, newDecodeError(path, err)
		}
		fileVersion, err := parseFileVersion(externalFileVersion.Version, fileName, false, fileNameToSupportedFileVersions, FileVersionV1Beta1, FileVersionV1Beta1)
		if err != nil {
			// This could be a source of bugs in the future - we likely just took a buf.yaml/buf.lock
			// as-is for digest calculations pre-refactor, and didn't require a version.
			return nil, newDecodeError(path, err)
		}
		switch fileVersion {
		case FileVersionV1Beta1, FileVersionV1:
			return newObjectData(fileName, data), nil
		case FileVersionV2:
			return nil, nil
		default:
			return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
		}
	}
	return nil, &fs.PathError{Op: "read", Path: normalpath.Join(prefix, fileNames[0]), Err: fs.ErrNotExist}
}
