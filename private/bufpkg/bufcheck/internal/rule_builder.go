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

package internal

import (
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
)

// RuleBuilder is a rule builder.
type RuleBuilder struct {
	id             string
	newPurpose     func(ConfigBuilder) (string, error)
	deprecated     bool
	replacementIDs []string
	newCheck       func(ConfigBuilder) (CheckFunc, error)
}

// NewRuleBuilder returns a new undeprecated RuleBuilder.
func NewRuleBuilder(
	id string,
	newPurpose func(ConfigBuilder) (string, error),
	newCheck func(ConfigBuilder) (CheckFunc, error),
) *RuleBuilder {
	return &RuleBuilder{
		id:             id,
		newPurpose:     newPurpose,
		deprecated:     false,
		replacementIDs: nil,
		newCheck:       newCheck,
	}
}

// NewNopRuleBuilder returns a new undeprecated RuleBuilder for the direct
// purpose and CheckFunc.
func NewNopRuleBuilder(
	id string,
	purpose string,
	checkFunc CheckFunc,
) *RuleBuilder {
	return NewRuleBuilder(
		id,
		newNopPurpose(purpose),
		newNopCheckFunc(checkFunc),
	)
}

// NewDeprecatedRuleBuilder returns a new RuleBuilder for a deprecated Rule.
//
// replacementIDs may be nil or empty.
func NewDeprecatedRuleBuilder(
	id string,
	purpose string,
	replacementIDs []string,
) *RuleBuilder {
	return &RuleBuilder{
		id:             id,
		newPurpose:     newNopPurpose(purpose),
		deprecated:     true,
		replacementIDs: replacementIDs,
		newCheck:       newNopCheckFunc(newNopCheck),
	}
}

// NewRule returns a new Rule.
//
// Categories will be sorted and Purpose will be prepended with "Checks that "
// and appended with ".".
//
// Categories is an actual copy from the ruleBuilder.
func (c *RuleBuilder) NewRule(configBuilder ConfigBuilder, categories []string) (*Rule, error) {
	purpose, err := c.newPurpose(configBuilder)
	if err != nil {
		return nil, err
	}
	check, err := c.newCheck(configBuilder)
	if err != nil {
		return nil, err
	}
	return newRule(
		c.id,
		categories,
		purpose,
		c.deprecated,
		c.replacementIDs,
		check,
	), nil
}

// ID returns the id.
func (c *RuleBuilder) ID() string {
	return c.id
}

// Deprecated returns whether or not the Rule was deprecated.
func (c *RuleBuilder) Deprecated() bool {
	return c.deprecated
}

// ReplacementIDs returns the replacement IDs.
func (c *RuleBuilder) ReplacementIDs() []string {
	return c.replacementIDs
}

func newNopPurpose(purpose string) func(ConfigBuilder) (string, error) {
	return func(ConfigBuilder) (string, error) {
		return purpose, nil
	}
}

func newNopCheckFunc(
	f func(string, IgnoreFunc, []bufprotosource.File, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error),
) func(ConfigBuilder) (CheckFunc, error) {
	return func(ConfigBuilder) (CheckFunc, error) {
		return f, nil
	}
}

func newNopCheck(string, IgnoreFunc, []bufprotosource.File, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return nil, nil
}
