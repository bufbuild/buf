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

package bufprotoplugin

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/thread"
	"github.com/bufbuild/protoplugin"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

type generator struct {
	logger  *zap.Logger
	handler protoplugin.Handler
}

func newGenerator(
	logger *zap.Logger,
	handler protoplugin.Handler,
) *generator {
	return &generator{
		logger:  logger,
		handler: handler,
	}
}

func (g *generator) Generate(
	ctx context.Context,
	container app.EnvStderrContainer,
	codeGeneratorRequests []*pluginpb.CodeGeneratorRequest,
) (*pluginpb.CodeGeneratorResponse, error) {
	protopluginResponseWriter := protoplugin.NewResponseWriter(
		protoplugin.ResponseWriterWithLenientValidation(
			func(err error) {
				_, _ = fmt.Fprintln(container.Stderr(), err.Error())
			},
		),
	)
	jobs := make([]func(context.Context) error, len(codeGeneratorRequests))
	for i, codeGeneratorRequest := range codeGeneratorRequests {
		codeGeneratorRequest := codeGeneratorRequest
		jobs[i] = func(ctx context.Context) error {
			protopluginRequest, err := protoplugin.NewRequest(codeGeneratorRequest)
			if err != nil {
				return err
			}
			return g.handler.Handle(
				ctx,
				protoplugin.PluginEnv{
					Environ: app.Environ(container),
					Stderr:  container.Stderr(),
				},
				protopluginResponseWriter,
				protopluginRequest,
			)
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := thread.Parallelize(ctx, jobs, thread.ParallelizeWithCancel(cancel)); err != nil {
		return nil, err
	}
	codeGeneratorResponse, err := protopluginResponseWriter.ToCodeGeneratorResponse()
	if err != nil {
		return nil, err
	}
	if errString := codeGeneratorResponse.GetError(); errString != "" {
		return nil, errors.New(errString)
	}
	return codeGeneratorResponse, nil
}
