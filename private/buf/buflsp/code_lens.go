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

package buflsp

import (
	"fmt"

	"github.com/bufbuild/protocompile/experimental/ir"
	"go.lsp.dev/protocol"
)

// getCodeLenses returns all code lenses for a file.
// Code lenses are shown on referenceable symbol definitions (messages, enums, services, extensions).
// Fields and oneofs are excluded to reduce visual clutter.
func getCodeLenses(file *file) []protocol.CodeLens {
	if file == nil {
		return nil
	}

	var lenses []protocol.CodeLens

	// Iterate through all referenceable symbols in the file
	// referenceableSymbols is a map of full names to symbols that can be referenced
	for _, symbol := range file.referenceableSymbols {
		// Skip fields and oneofs - only show code lens on top-level types (messages, enums, services)
		kind := symbol.ir.Kind()
		if kind == ir.SymbolKindField || kind == ir.SymbolKindOneof {
			continue
		}

		// Create unresolved code lens (command will be added during resolve)
		lens := createCodeLensForSymbol(symbol)
		if lens != nil {
			lenses = append(lenses, *lens)
		}
	}

	return lenses
}

// createCodeLensForSymbol creates an informational code lens for a symbol.
// The code lens displays the reference count but is not clickable.
// Users can still use "Find References" from the context menu or keyboard shortcut.
func createCodeLensForSymbol(symbol *symbol) *protocol.CodeLens {
	if symbol == nil {
		return nil
	}

	// Get the symbol's position
	symbolRange := symbol.Range()

	// Get references (excluding the declaration itself)
	references := symbol.References(false)
	referenceCount := len(references)

	// Format the title with proper pluralization
	var title string
	if referenceCount == 1 {
		title = "1 reference"
	} else {
		title = fmt.Sprintf("%d references", referenceCount)
	}

	// Create an informational code lens (not clickable)
	return &protocol.CodeLens{
		Range: symbolRange,
		Command: &protocol.Command{
			Title: title,
		},
	}
}

// resolveCodeLens resolves a code lens (currently a no-op since we resolve eagerly).
func resolveCodeLens(lens *protocol.CodeLens) (*protocol.CodeLens, error) {
	// Code lenses are already fully resolved in createCodeLensForSymbol
	// Just return the lens as-is
	return lens, nil
}
