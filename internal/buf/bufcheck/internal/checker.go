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

package internal

import (
	"encoding/json"
	"sort"

	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/protodesc"
)

// CheckFunc is a check function.
type CheckFunc func(id string, previousFiles []protodesc.File, files []protodesc.File) ([]*filev1beta1.FileAnnotation, error)

// Checker provides a base embeddable checker.
type Checker struct {
	id         string
	categories []string
	purpose    string
	checkFunc  CheckFunc
}

// newChecker returns a new Checker.
//
// Categories will be sorted and purpose will have "Checks that "
// prepended and "." appended.
func newChecker(
	id string,
	categories []string,
	purpose string,
	checkFunc CheckFunc,
) *Checker {
	c := make([]string, len(categories))
	copy(c, categories)
	sort.Slice(
		c,
		func(i int, j int) bool {
			return categoryCompare(c[i], c[j]) < 0
		},
	)
	return &Checker{
		id:         id,
		categories: c,
		purpose:    "Checks that " + purpose + ".",
		checkFunc:  checkFunc,
	}
}

// ID implements Checker.
func (c *Checker) ID() string {
	return c.id
}

// Categories implements Checker.
func (c *Checker) Categories() []string {
	return c.categories
}

// Purpose implements Checker.
func (c *Checker) Purpose() string {
	return c.purpose
}

// MarshalJSON implements Checker.
func (c *Checker) MarshalJSON() ([]byte, error) {
	return json.Marshal(checkerJSON{ID: c.id, Categories: c.categories, Purpose: c.purpose})
}

func (c *Checker) check(previousFiles []protodesc.File, files []protodesc.File) ([]*filev1beta1.FileAnnotation, error) {
	return c.checkFunc(c.ID(), previousFiles, files)
}

type checkerJSON struct {
	ID         string   `json:"id" yaml:"id"`
	Categories []string `json:"categories" yaml:"categories"`
	Purpose    string   `json:"purpose" yaml:"purpose"`
}
