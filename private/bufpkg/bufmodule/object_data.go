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

package bufmodule

import (
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/normalpath"
)

// ObjectData is individual file data.
//
// It matches bufconfig.ObjectData, but is also defined here to avoid circular dependencies.
// As opposed to most of our interfaces, it does not have a private method limiting its implementation
// to this package.
//
// registry-proto Files can be converted into ObjectDatas.
type ObjectData interface {
	// Name returns the file name.
	//
	// Always non-empty.
	Name() string
	// Data returns the file data.
	Data() []byte
}

func NewObjectData(name string, data []byte) (ObjectData, error) {
	return newObjectData(name, data)
}

/// *** PRIVATE ***

type objectData struct {
	name string
	data []byte
}

func newObjectData(name string, data []byte) (*objectData, error) {
	if name == "" {
		return nil, errors.New("name is empty when constructing an ObjectData")
	}
	if normalpath.Base(name) != name {
		return nil, fmt.Errorf("expected file name but got file path %q", name)
	}
	return &objectData{
		name: name,
		data: data,
	}, nil
}

func (o *objectData) Name() string {
	return o.name
}

func (o *objectData) Data() []byte {
	return o.data
}
