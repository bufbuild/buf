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
	"buf.build/go/bufplugin/check/checkutil"
	"buf.build/go/bufplugin/descriptor"
)

func main() {
	check.Main(
		&check.Spec{
			Rules: []*check.RuleSpec{
				{
					ID:             "LINT_PANIC",
					Default:        true,
					Purpose:        `This rule panics.`,
					Type:           check.RuleTypeLint,
					ReplacementIDs: nil,
					Handler:        checkutil.NewFileRuleHandler(checkPanic),
				},
			},
		},
	)
}

func checkPanic(context.Context, check.ResponseWriter, check.Request, descriptor.FileDescriptor) error {
	panic("this panic is intentional")
}
