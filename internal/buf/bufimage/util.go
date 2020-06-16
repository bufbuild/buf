// Copyright 2020 Buf Technologies, Inc.
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

package bufimage

import (
	"errors"

	imagev1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1"
)

func getImportFileIndexes(protoImage *imagev1.Image) (map[int]struct{}, error) {
	imageImportRefs := protoImage.GetBufbuildImageExtension().GetImageImportRefs()
	importFileIndexes := make(map[int]struct{}, len(imageImportRefs))
	for _, imageImportRef := range imageImportRefs {
		if imageImportRef.FileIndex == nil {
			// this should have been caught in validation but just in case
			return nil, errors.New("nil fileIndex")
		}
		importFileIndexes[int(imageImportRef.GetFileIndex())] = struct{}{}
	}
	return importFileIndexes, nil
}
