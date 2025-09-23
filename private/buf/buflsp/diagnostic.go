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
	"encoding/json"
	"strings"

	"github.com/bufbuild/protocompile/experimental/report"
	"go.lsp.dev/protocol"
)

// diagnosticData is a structure to hold the [report.Diagnostic] notes, help, and debug
// messages, to marshal into JSON for the [protocol.Diagnostic].Data field.
type diagnosticData struct {
	Notes string `json:"notes,omitempty"`
	Help  string `json:"help,omitempty"`
	Debug string `json:"debug,omitempty"`
}

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
	if !reportDiagnostic.Primary().IsZero() {
		diagnostic.Range = protocol.Range{
			Start: protocol.Position{
				Line:      uint32(reportDiagnostic.Primary().StartLoc().Line - 1),
				Character: uint32(reportDiagnostic.Primary().StartLoc().Column - 1),
			},
			End: protocol.Position{
				Line:      uint32(reportDiagnostic.Primary().EndLoc().Line - 1),
				Character: uint32(reportDiagnostic.Primary().EndLoc().Column - 1),
			},
		}
	}
	data := diagnosticData{
		Notes: strings.Join(reportDiagnostic.Notes(), "\n"),
		Help:  strings.Join(reportDiagnostic.Help(), "\n"),
		Debug: strings.Join(reportDiagnostic.Debug(), "\n"),
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return protocol.Diagnostic{}, err
	}
	if bytes != nil {
		// We serialize the bytes into a string before providing the structure to diagnostic.Data
		// because diagnostic.Data is an interface{}, which is treated as a JSON "any", which
		// will not cleanly deserialize.
		diagnostic.Data = string(bytes)
	}
	return diagnostic, nil
}
