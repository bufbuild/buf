// Copyright 2020 Buf Technologies Inc.
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

package storagegitplumbing

import "github.com/go-git/go-git/v5/plumbing"

// RefName is a git reference name.
type RefName interface {
	// ReferenceName returns the go-git ReferenceName.
	ReferenceName() plumbing.ReferenceName
	String() string
}

// NewBranchRefName returns a new branch RefName.
func NewBranchRefName(branch string) RefName {
	return newRefName(plumbing.NewBranchReferenceName(branch))
}

// NewTagRefName returns a new tag RefName.
func NewTagRefName(tag string) RefName {
	return newRefName(plumbing.NewTagReferenceName(tag))
}

type refName struct {
	referenceName plumbing.ReferenceName
}

func newRefName(referenceName plumbing.ReferenceName) *refName {
	return &refName{
		referenceName: referenceName,
	}
}

func (r *refName) ReferenceName() plumbing.ReferenceName {
	return r.referenceName
}

func (r *refName) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r.String() + `"`), nil
}

func (r *refName) String() string {
	if r == nil {
		return ""
	}
	return r.referenceName.String()
}
