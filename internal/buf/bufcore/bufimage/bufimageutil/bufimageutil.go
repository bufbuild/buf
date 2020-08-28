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

package bufimageutil

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/pkg/protosource"
)

// NewInputFiles gets around Go's lack of generics.
func NewInputFiles(imageFiles []bufimage.ImageFile) []protosource.InputFile {
	inputFiles := make([]protosource.InputFile, len(imageFiles))
	for i, imageFile := range imageFiles {
		inputFiles[i] = imageFile
	}
	return inputFiles
}

// FreeMessageRangeStrings gets the free MessageRange strings for the target files.
//
// TODO: this should not depend on bufmodule.
//
// Recursive.
func FreeMessageRangeStrings(
	ctx context.Context,
	moduleFileSet bufmodule.ModuleFileSet,
	image bufimage.Image,
) ([]string, error) {
	fileInfos, err := moduleFileSet.TargetFileInfos(ctx)
	if err != nil {
		return nil, err
	}
	var s []string
	for _, fileInfo := range fileInfos {
		imageFile := image.GetFile(fileInfo.Path())
		if imageFile == nil {
			return nil, fmt.Errorf("unexpected nil image file: %q", fileInfo.Path())
		}
		file, err := protosource.NewFile(imageFile)
		if err != nil {
			return nil, err
		}
		for _, message := range file.Messages() {
			s = freeMessageRangeStringsRec(s, message)
		}
	}
	return s, nil
}

func freeMessageRangeStringsRec(
	s []string,
	message protosource.Message,
) []string {
	for _, nestedMessage := range message.Messages() {
		s = freeMessageRangeStringsRec(s, nestedMessage)
	}
	if e := protosource.FreeMessageRangeString(message); e != "" {
		return append(s, e)
	}
	return s
}
