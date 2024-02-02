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

package bufcli

import (
	"context"

	"github.com/bufbuild/buf/private/buf/buflsp"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// NewLSPServer returns a new buflsp.Server.
func NewLSPServer(
	ctx context.Context,
	container appext.Container,
	jsonrpc2Conn jsonrpc2.Conn,
) (protocol.Server, error) {
	controller, err := NewController(container)
	if err != nil {
		return nil, err
	}
	return buflsp.NewServer(
		ctx,
		container.Logger(),
		tracing.NewTracer(container.Tracer()),
		jsonrpc2Conn,
		controller,
		normalpath.Join(container.CacheDirPath(), v3CacheLSPRelDirPath),
	)
}
