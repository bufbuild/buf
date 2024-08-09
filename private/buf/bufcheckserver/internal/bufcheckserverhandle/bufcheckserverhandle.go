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

package bufcheckserverhandle

import (
	"github.com/bufbuild/buf/private/buf/bufcheckserver/internal/bufcheckserverutil"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/pkg/stringutil"
)

// HandleLintServicePascalCase is a handle function.
var HandleLintServicePascalCase = bufcheckserverutil.NewLintServiceRuleHandler(handleLintServicePascalCase)

func handleLintServicePascalCase(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	service bufprotosource.Service,
) error {
	name := service.Name()
	expectedName := stringutil.ToPascalCase(name)
	if name != expectedName {
		responseWriter.AddProtosourceAnnotation(
			service.NameLocation(),
			nil,
			"Service name %q should be PascalCase, such as %q.",
			name,
			expectedName,
		)
	}
	return nil
}
