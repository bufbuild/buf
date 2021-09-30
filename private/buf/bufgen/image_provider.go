package bufgen

import (
	"fmt"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
)

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
