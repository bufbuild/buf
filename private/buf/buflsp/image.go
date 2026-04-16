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
	"context"
	"errors"
	"log/slog"

	"buf.build/go/standard/xlog/xslog"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/protocompile/experimental/report"
	"github.com/bufbuild/protocompile/experimental/source"
	"go.lsp.dev/protocol"
)

// buildImage builds a Buf Image for the given path using the new experimental compiler.
// This does not use the controller to build the image, because we need delicate control
// over the input files: namely, for the case when we depend on a file that has been
// opened and modified in the editor.
//
// The opener should contain all files in the current workspace, including any unsaved
// modifications. Files not present in the opener are resolved via [source.WKTs].
func buildImage(
	ctx context.Context,
	path string,
	logger *slog.Logger,
	opener source.Opener,
) (bufimage.Image, []protocol.Diagnostic) {
	image, rpt, err := bufimage.BuildImageFromOpener(ctx, logger, opener, []string{path})

	var diagnostics []protocol.Diagnostic
	var hasErrors bool
	for _, diagnostic := range rpt.Diagnostics {
		if diagnostic.Primary().IsZero() || diagnostic.Level() > report.Error {
			continue
		}
		hasErrors = true
		diagnostics = append(diagnostics, reportDiagnosticToProtocolDiagnostic(diagnostic))
	}

	if err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.WarnContext(ctx, "error building image", slog.String("path", path), xslog.ErrorAttr(err))
		}
		return nil, diagnostics
	}

	if hasErrors {
		// Don't return an image when there are compile errors: the image may be
		// incomplete, and lint checks on a broken image produce misleading results.
		return nil, diagnostics
	}

	return image, diagnostics
}
