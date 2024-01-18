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

package bufconfig

// ObjectData is the data of the underlying storage.ReadObject that was used to create a Object.
//
// This is present on Files if they were created from storage.ReadBuckets. It is not present
// if the File was created via a New constructor or Read method.
type ObjectData interface {
	// Name returns the name of the underlying storage.ReadObject.
	//
	// This will be normalpath.Base(readObject.Path()).
	Name() string
	// Data returns the data from the underlying storage.ReadObject.
	Data() []byte

	isObjectData()
}

// *** PRIVATE ***

type objectData struct {
	name string
	data []byte
}

func newObjectData(name string, data []byte) *objectData {
	return &objectData{
		name: name,
		data: data,
	}
}

func (f *objectData) Name() string {
	return f.name
}

func (f *objectData) Data() []byte {
	return f.data
}

func (*objectData) isObjectData() {}
