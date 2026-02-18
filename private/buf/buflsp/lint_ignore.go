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

// This file implements the buf:lint:ignore code action for suppressing lint diagnostics.
// Reference implementation: https://github.com/bufbuild/intellij-buf/pull/212

package buflsp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bufbuild/protocompile/experimental/token"
	"go.lsp.dev/protocol"
)

// isFileWideLint returns true if the diagnostic is a file-wide lint.
// File-wide lints apply to the entire file and are reported at position 0:0.
// These should be suppressed in buf.yaml using ignore_only, not with inline comments.
// Examples include PACKAGE_DEFINED and FILE_LOWER_SNAKE_CASE.
// See: https://github.com/bufbuild/intellij-buf/issues/215
func isFileWideLint(r protocol.Range) bool {
	return r.Start.Line == 0 && r.Start.Character == 0
}

// getLintIgnoreCodeActions generates code actions for suppressing lint diagnostics
// at the given range. Returns a list of code actions, one for each applicable lint
// diagnostic that overlaps with the provided range.
func (s *server) getLintIgnoreCodeActions(
	ctx context.Context,
	file *file,
	params *protocol.CodeActionParams,
) []protocol.CodeAction {
	s.logger.DebugContext(ctx, "lint_ignore: checking code actions",
		slog.String("uri", string(params.TextDocument.URI)),
		slog.Uint64("line", uint64(params.Range.Start.Line)),
		slog.Uint64("char", uint64(params.Range.Start.Character)),
	)

	var actions []protocol.CodeAction

	// Filter diagnostics to find lint errors that overlap with the requested range
	for _, diagnostic := range file.diagnostics {
		// Only handle lint diagnostics (not IR diagnostics)
		if diagnostic.Source != "buf lint" {
			continue
		}

		// Diagnostic must have a code (the rule ID)
		ruleID, ok := diagnostic.Code.(string)
		if !ok || ruleID == "" {
			continue
		}

		// Skip file-wide lints (e.g., PACKAGE_DEFINED, FILE_LOWER_SNAKE_CASE).
		if isFileWideLint(diagnostic.Range) {
			s.logger.DebugContext(ctx, "lint_ignore: skipping file-wide lint",
				slog.String("rule_id", ruleID),
			)
			continue
		}

		// Check if diagnostic overlaps with the requested range
		if !rangesOverlap(diagnostic.Range, params.Range) {
			continue
		}

		// Generate the code action for this diagnostic
		if action := s.generateLintIgnoreAction(ctx, file, diagnostic, ruleID); action != nil {
			actions = append(actions, *action)
		}
	}

	// If there's only one lint ignore action, mark it as preferred, since this likely corresponds
	// closely to a user's intent while requesting code actions on this line.
	if len(actions) == 1 {
		actions[0].IsPreferred = true
	}

	s.logger.DebugContext(ctx, "lint_ignore: generated actions",
		slog.Int("count", len(actions)),
	)

	return actions
}

// generateLintIgnoreAction creates a single code action for suppressing a lint diagnostic.
func (s *server) generateLintIgnoreAction(
	ctx context.Context,
	file *file,
	diagnostic protocol.Diagnostic,
	ruleID string,
) *protocol.CodeAction {
	// We insert AT the diagnostic line (at character 0), which pushes it down,
	// making the comment appear on the line before the error in the final output.
	diagnosticLine := diagnostic.Range.Start.Line

	// file.file.LineOffsets is 1-indexed; diagnosticLine is 0-indexed LSP.
	diagLineStart, _ := file.file.LineOffsets(int(diagnosticLine) + 1)

	// Check if a standalone ignore comment already exists on the line before the diagnostic.
	// Walk back from the start of the diagnostic line through the token stream.
	if diagnosticLine > 0 && file.ir != nil && file.ir.AST() != nil && file.ir.AST().Stream() != nil {
		before, _ := file.ir.AST().Stream().Around(diagLineStart)
		if !before.IsZero() && before.Kind() == token.Comment &&
			strings.Contains(strings.ToLower(before.Text()), "buf:lint:ignore") &&
			strings.Contains(before.Text(), ruleID) {
			// Confirm the comment is standalone (not a trailing comment after code on the same line).
			// Walk back past indentation spaces; if what precedes is a newline or start of
			// block, the comment occupies its own line.
			cursor := token.NewCursorAt(before)
			prevTok := cursor.PrevSkippable()
			for !prevTok.IsZero() && isTokenSpace(prevTok) {
				prevTok = cursor.PrevSkippable()
			}
			if prevTok.IsZero() || strings.ContainsRune(prevTok.Text(), '\n') {
				s.logger.DebugContext(ctx, "lint_ignore: ignore comment already exists for this rule",
					slog.Uint64("line", uint64(diagnosticLine-1)),
					slog.String("rule_id", ruleID),
				)
				return nil
			}
		}
	}

	// Extract indentation from the diagnostic line.
	indentation := file.file.Indentation(diagLineStart)

	// Generate the ignore comment.
	commentText := fmt.Sprintf("// buf:lint:ignore %s\n", ruleID)

	// Create the text edit.
	edit := protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: diagnosticLine, Character: 0},
			End:   protocol.Position{Line: diagnosticLine, Character: 0},
		},
		NewText: indentation + commentText,
	}

	// Create the code action.
	action := protocol.CodeAction{
		Title: fmt.Sprintf("Suppress %s with buf:lint:ignore", ruleID),
		Kind:  protocol.QuickFix,
		Edit: &protocol.WorkspaceEdit{
			Changes: map[protocol.DocumentURI][]protocol.TextEdit{
				file.uri: {edit},
			},
		},
		Diagnostics: []protocol.Diagnostic{diagnostic},
	}

	return &action
}

// rangesOverlap returns true if two ranges overlap or touch each other.
func rangesOverlap(r1, r2 protocol.Range) bool {
	// Check if the ranges are on the same line (or overlapping lines)
	// For code actions, we consider a diagnostic to overlap if it's on the same line
	// as the cursor, even if the cursor is not directly on the diagnostic text.
	if r1.Start.Line > r2.End.Line || r2.Start.Line > r1.End.Line {
		return false
	}

	// If they're on the same line or overlapping lines, consider them overlapping
	return true
}
