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

	"go.lsp.dev/protocol"
)

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

		// Skip file-wide lints (e.g., PACKAGE_DEFINED, FILE_LOWER_SNAKE_CASE)
		// These are reported at position 0:0 or 1:0 and apply to the entire file.
		// They should be suppressed in buf.yaml with ignore_only, not with inline comments.
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
	// Calculate the line where we'll insert the ignore comment
	// We insert AT the diagnostic line (at character 0), which pushes it down
	// This makes the comment appear on the line before the error in the final output
	diagnosticLine := diagnostic.Range.Start.Line
	insertLine := diagnosticLine

	// Get the file text and split into lines
	text := file.file.Text()
	lines := strings.Split(text, "\n")

	// Check if we're trying to insert beyond the file bounds
	if int(insertLine) >= len(lines) {
		s.logger.WarnContext(ctx, "lint_ignore: insert line out of bounds",
			slog.Uint64("insert_line", uint64(insertLine)),
			slog.Int("total_lines", len(lines)),
		)
		return nil
	}

	// Check if an ignore comment already exists on the line before the diagnostic
	// We check for the specific rule ID to allow multiple ignore comments for different rules
	if insertLine > 0 {
		prevLine := insertLine - 1
		if int(prevLine) < len(lines) {
			prevLineText := lines[prevLine]
			// Check if this specific rule is already ignored (case-insensitive)
			if strings.Contains(strings.ToLower(prevLineText), "buf:lint:ignore") &&
				strings.Contains(prevLineText, ruleID) {
				s.logger.DebugContext(ctx, "lint_ignore: ignore comment already exists for this rule",
					slog.Uint64("line", uint64(prevLine)),
					slog.String("rule_id", ruleID),
				)
				return nil
			}
		}
	}

	// Extract indentation from the diagnostic line
	diagnosticLineText := lines[diagnosticLine]
	indentation := diagnosticLineText[:len(diagnosticLineText)-len(strings.TrimLeft(diagnosticLineText, " \t"))]

	// Generate the ignore comment
	commentText := fmt.Sprintf("// buf:lint:ignore %s\n", ruleID)

	// Create the text edit
	edit := protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: insertLine, Character: 0},
			End:   protocol.Position{Line: insertLine, Character: 0},
		},
		NewText: indentation + commentText,
	}

	// Create the code action
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

// isFileWideLint returns true if the diagnostic is a file-wide lint.
// File-wide lints apply to the entire file and are typically reported at position 0:0.
// These should be suppressed in buf.yaml using ignore_only, not with inline comments.
// Examples include PACKAGE_DEFINED and FILE_LOWER_SNAKE_CASE.
// See: https://github.com/bufbuild/intellij-buf/issues/215
func isFileWideLint(r protocol.Range) bool {
	// File-wide lints are reported at line 0 (0-indexed) with character 0
	return r.Start.Line == 0 && r.Start.Character == 0
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
