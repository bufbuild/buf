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
				ID:      "PACKAGE_DIRECTORY_MATCH", // duplicate of a built-in lint rule
				Purpose: "Checks that all files are in a directory that matches their package name.",
				Type:    check.RuleTypeLint,
				Handler: check.RuleHandlerFunc(func(context.Context, check.ResponseWriter, check.Request) error { return nil }),
			},
			{
				ID:      "ENUM_NO_DELETE", // duplicate of a built-in breaking rule
				Purpose: "Checks that enums are not deleted from a given file.",
				Type:    check.RuleTypeBreaking,
				Handler: check.RuleHandlerFunc(func(context.Context, check.ResponseWriter, check.Request) error { return nil }),
			},
		},
	})
}
