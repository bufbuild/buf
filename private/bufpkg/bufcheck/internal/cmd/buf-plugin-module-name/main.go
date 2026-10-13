// Copyright 2020-2026 Buf Technologies, Inc.
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
					ID:      "MODULE_NAME",
					Default: true,
					Purpose: "Reports the module name of each file.",
					Type:    check.RuleTypeLint,
					Handler: checkutil.NewFileRuleHandler(
						checkModuleName,
						checkutil.WithoutImports(),
					),
				},
			},
		},
	)
}

func checkModuleName(
	_ context.Context,
	responseWriter check.ResponseWriter,
	_ check.Request,
	fileDescriptor descriptor.FileDescriptor,
) error {
	message := "<none>"
	if moduleName := fileDescriptor.ModuleName(); moduleName != nil {
		message = moduleName.String()
	}
	responseWriter.AddAnnotation(
		check.WithMessage(message),
		check.WithFileName(fileDescriptor.ProtoreflectFileDescriptor().Path()),
	)
	return nil
}
