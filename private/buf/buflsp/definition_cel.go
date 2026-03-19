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

package buflsp

import (
	protocol "github.com/bufbuild/buf/private/pkg/lspprotocol"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
)

// getCELDefinition returns the definition location for a CEL token at the given position.
// If the cursor is over `this` in a CEL expression, it navigates to the field or message
// that `this` refers to:
//   - For field-level rules, `this` refers to the field being validated.
//   - For message-level rules, `this` refers to the message being validated.
func getCELDefinition(
	file *file,
	position protocol.Position,
	celEnv *cel.Env,
) *protocol.Location {
	// Convert position to byte offset.
	lineColumn := file.file.InverseLocation(int(position.Line)+1, int(position.Character)+1, positionalEncoding)
	byteOffset := lineColumn.Offset

	for _, sym := range file.symbols {
		celExpressions := extractCELExpressions(file, sym)
		for _, exprInfo := range celExpressions {
			// Check if cursor is within this CEL expression span.
			if byteOffset < exprInfo.span.Start || byteOffset >= exprInfo.span.End {
				continue
			}

			// Calculate offset within the CEL expression.
			celOffset := fileByteOffsetToCELOffset(byteOffset, exprInfo.span)
			if celOffset < 0 || celOffset >= len(exprInfo.expression) {
				continue
			}

			// Parse the CEL expression.
			parsedAST, parseIssues := celEnv.Parse(exprInfo.expression)
			if parseIssues.Err() != nil {
				continue
			}

			// Default to the parsed AST, but prefer the checked AST for full type information.
			astForTypeInfo := parsedAST
			var typeMap map[int64]*types.Type
			if checkedAST, compileIssues := celEnv.Compile(exprInfo.expression); compileIssues.Err() == nil {
				astForTypeInfo = checkedAST
				typeMap = checkedAST.NativeRep().TypeMap()
			}

			// Find the token at the cursor position.
			hoverInfo := findCELTokenAtOffset(astForTypeInfo, parsedAST, celOffset, exprInfo, typeMap)
			if hoverInfo == nil {
				continue
			}

			// Determine what the token refers to and find the corresponding symbol.
			var fullName ir.FullName
			switch {
			case hoverInfo.kind == celHoverKeyword && hoverInfo.text == "this":
				if !exprInfo.irMember.IsZero() {
					// Field-level CEL rule: `this` refers to the field being validated.
					fullName = exprInfo.irMember.FullName()
				} else if !exprInfo.thisIRType.IsZero() {
					// Message-level CEL rule: `this` refers to the message being validated.
					fullName = exprInfo.thisIRType.FullName()
				}
			case hoverInfo.kind == celHoverField && !hoverInfo.protoMember.IsZero():
				// Field access (e.g. `this.name`): navigate to the proto field declaration.
				fullName = hoverInfo.protoMember.FullName()
			}
			if fullName == "" {
				continue
			}

			// Look up the symbol in the current file and workspace.
			defSym := file.findSymbolByFullName(fullName)
			if defSym == nil {
				continue
			}
			loc := defSym.Definition()
			return &loc
		}
	}
	return nil
}
