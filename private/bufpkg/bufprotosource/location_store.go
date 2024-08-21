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

package bufprotosource

import (
	"sync"

	"google.golang.org/protobuf/types/descriptorpb"
)

type locationStore struct {
	filePath                string
	sourceCodeInfoLocations []*descriptorpb.SourceCodeInfo_Location
	getPathToLocation       func() map[string]Location
}

func newLocationStore(fileDescriptorProto *descriptorpb.FileDescriptorProto) *locationStore {
	locationStore := &locationStore{
		filePath:                fileDescriptorProto.GetName(),
		sourceCodeInfoLocations: fileDescriptorProto.GetSourceCodeInfo().GetLocation(),
	}
	locationStore.getPathToLocation = sync.OnceValue(locationStore.getPathToLocationUncached)
	return locationStore
}

func (l *locationStore) isEmpty() bool {
	return len(l.getPathToLocation()) == 0
}

func (l *locationStore) getLocation(path []int32) Location {
	return l.getLocationByPathKey(getPathKey(path))
}

func (l *locationStore) getLocationByPathKey(pathKey string) Location {
	return l.getPathToLocation()[pathKey]
}

// Expensive - not cached.
//
// This is specific to optionExtensionDescriptor.OptionLocation.
func (l *locationStore) getBestMatchOptionExtensionLocation(path []int32, extensionPathLen int) Location {
	// "Fuzzy" search: find a location whose path is at least extensionPathLen long,
	// preferring the longest matching ancestor path (i.e. as many extraPath elements
	// as can be found). If we find a *sub*path (a descendant path, that points INTO
	// the path we are trying to find), use the first such one encountered.
	var bestMatch *descriptorpb.SourceCodeInfo_Location
	var bestMatchPathLen int
	for _, loc := range l.sourceCodeInfoLocations {
		if len(loc.Path) >= extensionPathLen &&
			isDescendantPath(path, loc.Path) &&
			len(loc.Path) > bestMatchPathLen {
			bestMatch = loc
			bestMatchPathLen = len(loc.Path)
		} else if isDescendantPath(loc.Path, path) {
			return newLocation(l.filePath, loc)
		}
	}
	if bestMatch != nil {
		return newLocation(l.filePath, bestMatch)
	}
	return nil
}

// Do not use outside of locationStore!
func (l *locationStore) getPathToLocationUncached() map[string]Location {
	pathToLocation := make(map[string]Location, len(l.sourceCodeInfoLocations))
	for _, sourceCodeInfoLocation := range l.sourceCodeInfoLocations {
		pathKey := getPathKey(sourceCodeInfoLocation.Path)
		// - Multiple locations may have the same path.  This happens when a single
		//   logical declaration is spread out across multiple places.  The most
		//   obvious example is the "extend" block again -- there may be multiple
		//   extend blocks in the same scope, each of which will have the same path.
		if _, ok := pathToLocation[pathKey]; !ok {
			pathToLocation[pathKey] = newLocation(l.filePath, sourceCodeInfoLocation)
		}
	}
	return pathToLocation
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

func isDescendantPath(descendant, ancestor []int32) bool {
	if len(descendant) < len(ancestor) {
		return false
	}
	for i := range ancestor {
		if descendant[i] != ancestor[i] {
			return false
		}
	}
	return true
}
