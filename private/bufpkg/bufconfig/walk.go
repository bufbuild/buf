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

// Generated. DO NOT EDIT.

package bufconfig

import (
	"context"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// WalkFileInfos walks all the FileInfos in the ReadBucket.
//
// The path will be normalized.
func WalkFileInfos(
	ctx context.Context,
	readBucket storage.ReadBucket,
	f func(path string, fileInfo FileInfo) error,
) error {
	return readBucket.Walk(
		ctx,
		"",
		func(objectInfo storage.ObjectInfo) error {
			path := objectInfo.Path()
			fileType, ok := fileNameToFileType[normalpath.Base(path)]
			if !ok {
				return nil
			}
			defaultFileVersion, ok := fileTypeToDefaultFileVersion[fileType]
			if !ok {
				return syserror.Newf("did not set a default file version for FileType %v", fileType)
			}
			fileNameToSupportedFileVersions, ok := fileTypeToSupportedFileVersions[fileType]
			if !ok {
				return syserror.Newf("did not set a supported file versions map for FileType %v", fileType)
			}
			data, err := storage.ReadPath(ctx, readBucket, path)
			if err != nil {
				return err
			}
			fileVersion, err := getFileVersionForData(data, false, false, fileNameToSupportedFileVersions, 0, defaultFileVersion)
			if err != nil {
				return err
			}
			return f(path, newFileInfo(fileVersion, fileType))
		},
	)
}
