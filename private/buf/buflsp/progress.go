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

package buflsp

import (
	"context"
	"fmt"
	"math/rand/v2"

	"go.lsp.dev/protocol"
)

// progress is a client-side progress bar.
//
// This type manages all the necessary state with the client to show the
// progress bar.
type progress struct {
	lsp   *lsp
	token string
}

// Creates new server-initiated progress.
func newProgress(lsp *lsp) *progress {
	return &progress{
		lsp:   lsp,
		token: fmt.Sprintf("%016x", rand.Uint64()),
	}
}

// Creates progress to track client-initiated progress.
//
// If params is nil (i.e., the client doesn't want progress) this returns a nil progress
// that will do nothing when notified.
func newProgressFromClient(lsp *lsp, params *protocol.WorkDoneProgressParams) *progress {
	if params == nil || params.WorkDoneToken == nil {
		return nil
	}

	return &progress{
		lsp:   lsp,
		token: params.WorkDoneToken.String(),
	}
}

func (p *progress) Begin(ctx context.Context, title string) {
	if p == nil {
		return
	}

	// NOTE: The error is automatically logged by the client binding.
	_ = p.lsp.client.Progress(ctx, &protocol.ProgressParams{
		Token: *protocol.NewProgressToken(p.token),
		Value: &protocol.WorkDoneProgressBegin{
			Kind:  protocol.WorkDoneProgressKindBegin,
			Title: title,
		},
	})
}

func (p *progress) Report(ctx context.Context, message string, percent float64) {
	if p == nil {
		return
	}

	// NOTE: The error is automatically logged by the client binding.
	_ = p.lsp.client.Progress(ctx, &protocol.ProgressParams{
		Token: *protocol.NewProgressToken(p.token),
		Value: &protocol.WorkDoneProgressReport{
			Kind:       protocol.WorkDoneProgressKindReport,
			Message:    message,
			Percentage: uint32(percent * 100),
		},
	})
}

func (p *progress) Done(ctx context.Context) {
	if p == nil {
		return
	}

	// NOTE: The error is automatically logged by the client binding.
	_ = p.lsp.client.Progress(ctx, &protocol.ProgressParams{
		Token: *protocol.NewProgressToken(p.token),
		Value: &protocol.WorkDoneProgressEnd{
			Kind: protocol.WorkDoneProgressKindEnd,
		},
	})
}
