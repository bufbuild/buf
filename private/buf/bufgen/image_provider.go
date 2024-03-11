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

package bufgen

import (
	"fmt"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
)

// imageProvider is used to provide the images used
// when generating with a local plugin. Each plugin is
// in control of its own Strategy - we cache the
// imagesByDir so that we only have to build it once for
// all of the plugins that configure the Directory
// strategy.
type imageProvider struct {
	image       bufimage.Image
	imagesByDir []bufimage.Image
	lock        sync.Mutex
}

func newImageProvider(image bufimage.Image) *imageProvider {
	return &imageProvider{
		image: image,
	}
}

func (p *imageProvider) GetImages(strategy Strategy) ([]bufimage.Image, error) {
	switch strategy {
	case StrategyAll:
		return []bufimage.Image{p.image}, nil
	case StrategyDirectory:
		p.lock.Lock()
		defer p.lock.Unlock()
		if p.imagesByDir == nil {
			var err error
			p.imagesByDir, err = bufimage.ImageByDir(p.image)
			if err != nil {
				return nil, err
			}
		}
		return p.imagesByDir, nil
	default:
		return nil, fmt.Errorf("unknown strategy: %v", strategy)
	}
}
