// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufimagemodifyv1

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
)

type fileOptionSweeper struct {
	// Filepath -> SourceCodeInfo_Location.Path keys.
	sourceCodeInfoPaths map[string]map[string]struct{}
}

func newFileOptionSweeper() *fileOptionSweeper {
	return &fileOptionSweeper{
		sourceCodeInfoPaths: make(map[string]map[string]struct{}),
	}
}

// mark is used to mark the given SourceCodeInfo_Location indices for
// deletion. This method should be called in each of the file option
// modifiers.
func (s *fileOptionSweeper) mark(imageFilePath string, path []int32) {
	paths, ok := s.sourceCodeInfoPaths[imageFilePath]
	if !ok {
		paths = make(map[string]struct{})
		s.sourceCodeInfoPaths[imageFilePath] = paths
	}
	paths[internal.GetPathKey(path)] = struct{}{}
}

// Sweep applies all of the marks and sweeps the file option SourceCodeInfo_Locations.
func (s *fileOptionSweeper) Sweep(ctx context.Context, image bufimage.Image) error {
	for _, imageFile := range image.Files() {
		descriptor := imageFile.Proto()
		if descriptor.SourceCodeInfo == nil {
			continue
		}
		paths, ok := s.sourceCodeInfoPaths[imageFile.Path()]
		if !ok {
			continue
		}
		if err := internal.RemoveLocationsFromSourceCodeInfo(descriptor.SourceCodeInfo, paths); err != nil {
			return err
		}
	}
	return nil
}
