// Copyright 2020-2022 Buf Technologies, Inc.
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

package protosource

import (
	"fmt"
	"strings"
)

type oneofDescriptor struct {
	locationDescriptor

	name        string
	namePath    []int32
	nestedNames []string
}

func newOneofDescriptor(
	locationDescriptor locationDescriptor,
	name string,
	namePath []int32,
	nestedNames []string,
) (oneofDescriptor, error) {
	if name == "" {
		return oneofDescriptor{}, fmt.Errorf("no name in %q", locationDescriptor.File().Path())
	}
	return oneofDescriptor{
		locationDescriptor: locationDescriptor,
		name:               name,
		namePath:           namePath,
		nestedNames:        nestedNames,
	}, nil
}

func (n *oneofDescriptor) FullName() string {
	if n.File().Package() != "" {
		return n.File().Package() + "." + n.NestedName()
	}
	return n.NestedName()
}

func (n *oneofDescriptor) NestedName() string {
	if len(n.nestedNames) == 0 {
		return n.Name()
	}
	return strings.Join(n.nestedNames, ".") + "." + n.Name()
}

func (n *oneofDescriptor) Name() string {
	return n.name
}

func (n *oneofDescriptor) NameLocation() Location {
	nameLocation := n.getLocation(n.namePath)
	location := n.getLocation(n.path)
	if nameLocation != nil {
		if location != nil {
			return newMergeCommentLocation(nameLocation, location)
		}
		return nameLocation
	}
	return location
}
