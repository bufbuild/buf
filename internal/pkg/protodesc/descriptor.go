package protodesc

import (
	"errors"

	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
)

type descriptor struct {
	filePath      string
	pkg           string
	locationStore *locationStore
}

func newDescriptor(
	filePath string,
	pkg string,
	locationStore *locationStore,
) (descriptor, error) {
	if filePath == "" {
		return descriptor{}, errors.New("no filePath")
	}
	filePath, err := storagepath.NormalizeAndValidate(filePath)
	if err != nil {
		return descriptor{}, err
	}
	return descriptor{
		filePath:      filePath,
		pkg:           pkg,
		locationStore: locationStore,
	}, nil
}

func (d *descriptor) FilePath() string {
	return d.filePath
}

func (d *descriptor) Package() string {
	return d.pkg
}

func (d *descriptor) getLocation(path []int32) Location {
	return d.locationStore.getLocation(path)
}

func (d *descriptor) getLocationByPathKey(pathKey string) Location {
	return d.locationStore.getLocationByPathKey(pathKey)
}
