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

// This file provides helpers for bridging protocompile and LSP diagnostics.

package buflsp

import (
	"github.com/bufbuild/protocompile/reporter"
	"go.lsp.dev/protocol"
)

// report is a reporter.Reporter that captures diagnostic events as
// protocol.Diagnostic values.
type report []protocol.Diagnostic

// Error implements reporter.Handler for *diagnostics.
func (r *report) Error(err reporter.ErrorWithPos) error {
	*r = append(*r, newDiagnostic(err, false))
	return nil
}

// Error implements reporter.Handler for *diagnostics.
func (r *report) Warning(err reporter.ErrorWithPos) {
	*r = append(*r, newDiagnostic(err, true))
}

// error2diagnostic converts a protocompile error into a diagnostic.
//
// Unfortunately, protocompile's errors are currently too meagre to provide full code
// spans; that will require a fix in the compiler.
func newDiagnostic(err reporter.ErrorWithPos, isWarning bool) protocol.Diagnostic {
	pos := protocol.Position{
		Line:      uint32(err.GetPosition().Line - 1),
		Character: uint32(err.GetPosition().Col - 1),
	}

	diagnostic := protocol.Diagnostic{
		// TODO: The compiler currently does not record spans for diagnostics. This is
		// essentially a bug that will result in worse diagnostics until fixed.
		Range:    protocol.Range{Start: pos, End: pos},
		Severity: protocol.DiagnosticSeverityError,
		Message:  err.Unwrap().Error(),
	}

	if isWarning {
		diagnostic.Severity = protocol.DiagnosticSeverityWarning
	}

	return diagnostic
}
