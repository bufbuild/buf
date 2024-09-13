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

package main

import (
	"context"

	"buf.build/go/bufplugin/check"
)

func main() {
	check.Main(&check.Spec{
		Rules: []*check.RuleSpec{
			{
				ID:          "FOO",
				Purpose:     "Checks foo.",
				Type:        check.RuleTypeLint,
				CategoryIDs: []string{"STANDARD", "NOT_DUPLICATE"},
				Handler:     check.RuleHandlerFunc(func(context.Context, check.ResponseWriter, check.Request) error { return nil }),
			},
			{
				ID:          "BAR",
				Purpose:     "Checks bar.",
				Type:        check.RuleTypeBreaking,
				CategoryIDs: []string{"STANDARD", "NOT_DUPLICATE"},
				Handler:     check.RuleHandlerFunc(func(context.Context, check.ResponseWriter, check.Request) error { return nil }),
			},
		},
		Categories: []*check.CategorySpec{
			{
				ID:      "STANDARD",
				Purpose: "Duplicate the built-in STANDARD category.",
			},
			{
				ID:      "NOT_DUPLICATE",
				Purpose: "To be a non-duplicate category.",
			},
		},
	})
}
