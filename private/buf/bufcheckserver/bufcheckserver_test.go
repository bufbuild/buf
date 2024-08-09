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

package bufcheckserver

import (
	"testing"

	"github.com/bufbuild/bufplugin-go/check"
	"github.com/bufbuild/bufplugin-go/check/checktest"
)

func TestServicePascalCase(t *testing.T) {
	t.Parallel()

	for _, spec := range []*check.Spec{
		v2Spec,
	} {
		checktest.TestCase{
			Spec: spec,
			Files: &checktest.ProtoFileSpec{
				DirPaths:  []string{"testdata/lint/service_pascal_case"},
				FilePaths: []string{"a.proto"},
			},
			RuleIDs: []string{
				servicePascalCaseRuleID,
			},
			ExpectedAnnotations: []checktest.ExpectedAnnotation{
				{
					ID: servicePascalCaseRuleID,
					Location: &checktest.ExpectedLocation{
						FileName:    "a.proto",
						StartLine:   7,
						StartColumn: 8,
						EndLine:     7,
						EndColumn:   12,
					},
				},
				{
					ID: servicePascalCaseRuleID,
					Location: &checktest.ExpectedLocation{
						FileName:    "a.proto",
						StartLine:   8,
						StartColumn: 8,
						EndLine:     8,
						EndColumn:   15,
					},
				},
				{
					ID: servicePascalCaseRuleID,
					Location: &checktest.ExpectedLocation{
						FileName:    "a.proto",
						StartLine:   9,
						StartColumn: 8,
						EndLine:     9,
						EndColumn:   18,
					},
				},
				{
					ID: servicePascalCaseRuleID,
					Location: &checktest.ExpectedLocation{
						FileName:    "a.proto",
						StartLine:   10,
						StartColumn: 8,
						EndLine:     10,
						EndColumn:   17,
					},
				},
			},
		}.Run(t)
	}
}
