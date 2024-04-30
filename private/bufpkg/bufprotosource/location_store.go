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
	sourceCodeInfoLocations []*descriptorpb.SourceCodeInfo_Location

	initLocations  sync.Once
	pathToLocation map[string]Location
}

func newLocationStore(sourceCodeInfoLocations []*descriptorpb.SourceCodeInfo_Location) *locationStore {
	return &locationStore{
		sourceCodeInfoLocations: sourceCodeInfoLocations,
		pathToLocation:          make(map[string]Location),
	}
}

func (l *locationStore) isEmpty() bool {
	return len(l.sourceCodeInfoLocations) == 0
}

func (l *locationStore) getLocation(path []int32) Location {
	return l.getLocationByPathKey(getPathKey(path))
}

func (l *locationStore) getLocationByPathKey(pathKey string) Location {
	l.initLocations.Do(func() {
		pathToLocation := make(map[string]Location)
		for _, sourceCodeInfoLocation := range l.sourceCodeInfoLocations {
			pathKey := getPathKey(sourceCodeInfoLocation.Path)
			// - Multiple locations may have the same path.  This happens when a single
			//   logical declaration is spread out across multiple places.  The most
			//   obvious example is the "extend" block again -- there may be multiple
			//   extend blocks in the same scope, each of which will have the same path.
			if _, ok := pathToLocation[pathKey]; !ok {
				pathToLocation[pathKey] = newLocation(sourceCodeInfoLocation)
			}
		}
		l.pathToLocation = pathToLocation
	})

	return l.pathToLocation[pathKey]
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
