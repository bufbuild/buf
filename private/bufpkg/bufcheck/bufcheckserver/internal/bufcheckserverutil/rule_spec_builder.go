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

package bufcheckserverutil

import (
	"buf.build/go/bufplugin/check"
)

// RuleSpecBuilder matches check.RuleSpec but without categories.
//
// We have very similar RuleSpecs across our versions of our lint rules, however their categories do change
// across versions. This allows us to share the basic RuleSpec shape across versions.
type RuleSpecBuilder struct {
	// Required.
	ID string
	// Required.
	Purpose string
	// Required.
	Type           check.RuleType
	Deprecated     bool
	ReplacementIDs []string
	// Required.
	Handler check.RuleHandler
}

// Build builds the RuleSpec for the categories.
//
// Not making categories variadic in case we want to add extra parameters later easily.
func (b *RuleSpecBuilder) Build(isDefault bool, categoryIDs []string) *check.RuleSpec {
	return &check.RuleSpec{
		ID:             b.ID,
		CategoryIDs:    categoryIDs,
		Default:        isDefault,
		Purpose:        b.Purpose,
		Type:           b.Type,
		Deprecated:     b.Deprecated,
		ReplacementIDs: b.ReplacementIDs,
		Handler:        b.Handler,
	}
}
