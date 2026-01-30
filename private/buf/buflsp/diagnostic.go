// Copyright 2020-2025 Buf Technologies, Inc.
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

// This file provides helpers for bridging protocompile and LSP diagnostics.

package buflsp

import (
	"github.com/bufbuild/protocompile/experimental/report"
	"github.com/bufbuild/protocompile/experimental/report/tags"
	"github.com/bufbuild/protocompile/experimental/source/length"
	"go.lsp.dev/protocol"
)

// UTF-16 is the default per LSP spec. Position encoding negotiation is not yet
// supported by the go.lsp.dev/protocol library.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#position
const positionalEncoding = length.UTF16

// reportLevelToDiagnosticSeverity is a mapping of [report.Level] to [protocol.DiagnosticSeverity].
var reportLevelToDiagnosticSeverity = map[report.Level]protocol.DiagnosticSeverity{
	report.ICE:     protocol.DiagnosticSeverityError,
	report.Error:   protocol.DiagnosticSeverityError,
	report.Warning: protocol.DiagnosticSeverityWarning,
	report.Remark:  protocol.DiagnosticSeverityInformation,
}

// reportDiagnosticToProtocolDiagnostic takes a [report.Diagnostic] and returns the
// corresponding [protocol.Diagnostic].
func reportDiagnosticToProtocolDiagnostic(
	reportDiagnostic report.Diagnostic,
) (protocol.Diagnostic, error) {
	diagnostic := protocol.Diagnostic{
		Source:   serverName,
		Severity: reportLevelToDiagnosticSeverity[reportDiagnostic.Level()],
		Message:  reportDiagnostic.Message(),
	}
	if primary := reportDiagnostic.Primary(); !primary.IsZero() {
		startLocation := primary.Location(primary.Start, positionalEncoding)
		endLocation := primary.Location(primary.End, positionalEncoding)
		diagnostic.Range = protocol.Range{
			Start: protocol.Position{
				Line:      uint32(startLocation.Line - 1),
				Character: uint32(startLocation.Column - 1),
			},
			End: protocol.Position{
				Line:      uint32(endLocation.Line - 1),
				Character: uint32(endLocation.Column - 1),
			},
		}
	}
	switch reportDiagnostic.Tag() {
	case tags.UnusedImport:
		diagnostic.Tags = []protocol.DiagnosticTag{
			protocol.DiagnosticTagUnnecessary,
		}
	case tags.Deprecated:
		diagnostic.Tags = []protocol.DiagnosticTag{
			protocol.DiagnosticTagDeprecated,
		}
	}
	return diagnostic, nil
}
