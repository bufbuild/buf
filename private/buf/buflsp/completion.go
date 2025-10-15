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

// This file implements code completion support for the LSP.

package buflsp

import (
	"context"

	"go.lsp.dev/protocol"
)

// getCompletionItems returns completion items for the given position in the file.
//
// This function is called by the Completion handler in server.go.
func getCompletionItems(
	ctx context.Context,
	file *file,
	position protocol.Position,
) []protocol.CompletionItem {
	// TODO: Implement completion logic.
	// For now, return an empty list.
	return nil
}

// resolveCompletionItem resolves additional details for a completion item.
//
// This function is called by the CompletionResolve handler in server.go.
func resolveCompletionItem(
	ctx context.Context,
	item *protocol.CompletionItem,
) (*protocol.CompletionItem, error) {
	// TODO: Implement completion resolution logic.
	// For now, just return the item unchanged.
	return item, nil
}
