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

package bufimagemodifyv2

import (
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
)

type markSweeper struct {
	image bufimage.Image
	// Filepath -> SourceCodeInfo_Location.Path keys.
	sourceCodeInfoPaths map[string]map[string]struct{}
}

func newMarkSweeper(image bufimage.Image) *markSweeper {
	return &markSweeper{
		image:               image,
		sourceCodeInfoPaths: make(map[string]map[string]struct{}),
	}
}

func (s *markSweeper) Mark(imageFile bufimage.ImageFile, path []int32) {
	paths, ok := s.sourceCodeInfoPaths[imageFile.Path()]
	if !ok {
		paths = make(map[string]struct{})
		s.sourceCodeInfoPaths[imageFile.Path()] = paths
	}
	paths[internal.GetPathKey(path)] = struct{}{}
}

func (s *markSweeper) Sweep() error {
	for _, imageFile := range s.image.Files() {
		descriptor := imageFile.Proto()
		if descriptor.SourceCodeInfo == nil {
			continue
		}
		paths, ok := s.sourceCodeInfoPaths[imageFile.Path()]
		if !ok {
			continue
		}
		err := internal.RemoveLocationsFromSourceCodeInfo(descriptor.SourceCodeInfo, paths)
		if err != nil {
			return err
		}
	}
	return nil
}
