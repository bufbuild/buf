// Copyright 2023 Buf Technologies, Inc.
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
	"io"

	"github.com/bufbuild/protocompile/ast"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"go.lsp.dev/protocol"
)

type parserDiagnosticReporter struct {
	diags []protocol.Diagnostic
}

func (r *parserDiagnosticReporter) Error(diag reporter.ErrorWithPos) error {
	pos := protocol.Position{
		Line:      uint32(diag.GetPosition().Line - 1),
		Character: uint32(diag.GetPosition().Col - 1),
	}
	r.diags = append(r.diags, protocol.Diagnostic{
		Range: protocol.Range{
			Start: pos,
			End:   pos,
		},
		Severity: protocol.DiagnosticSeverityError,
		Message:  diag.Unwrap().Error(),
	})
	return nil
}

func (r *parserDiagnosticReporter) Warning(diag reporter.ErrorWithPos) {
	pos := protocol.Position{
		Line:      uint32(diag.GetPosition().Line - 1),
		Character: uint32(diag.GetPosition().Col - 1),
	}
	r.diags = append(r.diags, protocol.Diagnostic{
		Range: protocol.Range{
			Start: pos,
			End:   pos,
		},
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  diag.Unwrap().Error(),
	})
}

func parseFile(fileName string, dataReader io.Reader) (*ast.FileNode, []protocol.Diagnostic, error) {
	diagReporter := &parserDiagnosticReporter{}
	handler := reporter.NewHandler(diagReporter)

	// Create a reader for the data.
	fileNode, err := parser.Parse(fileName, dataReader, handler)
	if err == nil {
		_, err = parser.ResultFromAST(fileNode, true, handler)
	}
	return fileNode, diagReporter.diags, err
}
