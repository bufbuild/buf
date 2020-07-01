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

package bufsrc

type descriptor struct {
	file          File
	locationStore *locationStore
}

func newDescriptor(
	file File,
	locationStore *locationStore,
) descriptor {
	return descriptor{
		file:          file,
		locationStore: locationStore,
	}
}

func (d *descriptor) File() File {
	return d.file
}

func (d *descriptor) getLocation(path []int32) Location {
	return d.locationStore.getLocation(path)
}

func (d *descriptor) getLocationByPathKey(pathKey string) Location {
	return d.locationStore.getLocationByPathKey(pathKey)
}
