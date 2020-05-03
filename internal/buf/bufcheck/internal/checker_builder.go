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
	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/protodesc"
)

// CheckerBuilder is a checker builder.
type CheckerBuilder struct {
	id         string
	newPurpose func(ConfigBuilder) (string, error)
	newCheck   func(ConfigBuilder) (CheckFunc, error)
}

// NewCheckerBuilder returns a new CheckerBuilder.
func NewCheckerBuilder(
	id string,
	newPurpose func(ConfigBuilder) (string, error),
	newCheck func(ConfigBuilder) (CheckFunc, error),
) *CheckerBuilder {
	return &CheckerBuilder{
		id:         id,
		newPurpose: newPurpose,
		newCheck:   newCheck,
	}
}

// NewNopCheckerBuilder returns a new CheckerBuilder for the direct
// purpose and CheckFunc.
func NewNopCheckerBuilder(
	id string,
	purpose string,
	checkFunc CheckFunc,
) *CheckerBuilder {
	return NewCheckerBuilder(
		id,
		newNopPurpose(purpose),
		newNopCheckFunc(checkFunc),
	)
}

// NewChecker returns a new Checker.
//
// Categories will be sorted and Purpose will be prepended with "Checks that "
// and appended with ".".
//
// Categories is an actual copy from the checkerBuilder.
func (c *CheckerBuilder) NewChecker(configBuilder ConfigBuilder, categories []string) (*Checker, error) {
	purpose, err := c.newPurpose(configBuilder)
	if err != nil {
		return nil, err
	}
	check, err := c.newCheck(configBuilder)
	if err != nil {
		return nil, err
	}
	return newChecker(
		c.id,
		categories,
		purpose,
		check,
	), nil
}

// ID returns the id.
func (c *CheckerBuilder) ID() string {
	return c.id
}

func newNopPurpose(purpose string) func(ConfigBuilder) (string, error) {
	return func(ConfigBuilder) (string, error) {
		return purpose, nil
	}
}

func newNopCheckFunc(
	f func(string, []protodesc.File, []protodesc.File) ([]*filev1beta1.FileAnnotation, error),
) func(ConfigBuilder) (CheckFunc, error) {
	return func(ConfigBuilder) (CheckFunc, error) {
		return f, nil
	}
}
