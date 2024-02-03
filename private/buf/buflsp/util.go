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

package buflsp

import (
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func uriToPath(uri uri.URI) string {
	return normalpath.Normalize(uri.Filename())
}

func annotationToDiagnostic(
	annotation bufanalysis.FileAnnotation,
	severity protocol.DiagnosticSeverity,
) protocol.Diagnostic {
	return protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(annotation.StartLine() - 1),
				Character: uint32(annotation.StartColumn() - 1),
			},
			End: protocol.Position{
				Line:      uint32(annotation.EndLine() - 1),
				Character: uint32(annotation.EndColumn() - 1),
			},
		},
		Severity: severity,
		Message:  annotation.Message(),
	}
}
