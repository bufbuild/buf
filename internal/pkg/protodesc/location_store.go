package protodesc

import (
	"sync"

	protobufdescriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type locationStore struct {
	sourceCodeInfoLocations []*protobufdescriptor.SourceCodeInfo_Location

	pathToLocation               map[string]Location
	pathToSourceCodeInfoLocation map[string]*protobufdescriptor.SourceCodeInfo_Location
	locationLock                 sync.RWMutex
	sourceCodeInfoLocationLock   sync.RWMutex
}

func newLocationStore(sourceCodeInfoLocations []*protobufdescriptor.SourceCodeInfo_Location) *locationStore {
	return &locationStore{
		sourceCodeInfoLocations: sourceCodeInfoLocations,
		pathToLocation:          make(map[string]Location),
	}
}

func (l *locationStore) getLocation(path []int32) Location {
	return l.getLocationByPathKey(getPathKey(path))
}

// optimization for keys we know ahead of time such as package location, certain file options
func (l *locationStore) getLocationByPathKey(pathKey string) Location {
	// check cache first
	l.locationLock.RLock()
	location, ok := l.pathToLocation[pathKey]
	l.locationLock.RUnlock()
	if ok {
		return location
	}

	// build index and get sourceCodeInfoLocation
	l.sourceCodeInfoLocationLock.RLock()
	pathToSourceCodeInfoLocation := l.pathToSourceCodeInfoLocation
	l.sourceCodeInfoLocationLock.RUnlock()
	if pathToSourceCodeInfoLocation == nil {
		l.sourceCodeInfoLocationLock.Lock()
		pathToSourceCodeInfoLocation = l.pathToSourceCodeInfoLocation
		if pathToSourceCodeInfoLocation == nil {
			pathToSourceCodeInfoLocation = make(map[string]*protobufdescriptor.SourceCodeInfo_Location)
			for _, sourceCodeInfoLocation := range l.sourceCodeInfoLocations {
				pathKey := getPathKey(sourceCodeInfoLocation.Path)
				// - Multiple locations may have the same path.  This happens when a single
				//   logical declaration is spread out across multiple places.  The most
				//   obvious example is the "extend" block again -- there may be multiple
				//   extend blocks in the same scope, each of which will have the same path.
				if _, ok := pathToSourceCodeInfoLocation[pathKey]; !ok {
					pathToSourceCodeInfoLocation[pathKey] = sourceCodeInfoLocation
				}
			}
		}
		l.pathToSourceCodeInfoLocation = pathToSourceCodeInfoLocation
		l.sourceCodeInfoLocationLock.Unlock()
	}
	sourceCodeInfoLocation, ok := pathToSourceCodeInfoLocation[pathKey]
	if !ok {
		return nil
	}

	// populate cache and return
	if sourceCodeInfoLocation == nil {
		location = nil
	} else {
		location = newLocation(sourceCodeInfoLocation)
	}
	l.locationLock.Lock()
	l.pathToLocation[pathKey] = location
	l.locationLock.Unlock()
	return location
}

func getPathKey(path []int32) string {
	key := make([]byte, len(path)*4)
	j := 0
	for _, elem := range path {
		key[j] = byte(elem)
		key[j+1] = byte(elem >> 8)
		key[j+2] = byte(elem >> 16)
		key[j+3] = byte(elem >> 24)
		j += 4
	}
	return string(key)
}
