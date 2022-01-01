// Copyright 2020-2022 Buf Technologies, Inc.
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

package appproto

import (
	"context"
	"errors"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/bufbuild/buf/private/pkg/thread"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

type generator struct {
	logger  *zap.Logger
	handler Handler
}

func newGenerator(
	logger *zap.Logger,
	handler Handler,
) *generator {
	return &generator{
		logger:  logger,
		handler: handler,
	}
}

func (g *generator) Generate(
	ctx context.Context,
	container app.EnvStderrContainer,
	requests []*pluginpb.CodeGeneratorRequest,
) (*pluginpb.CodeGeneratorResponse, error) {
	responseBuilder := newResponseBuilder(container)
	jobs := make([]func(context.Context) error, len(requests))
	for i, request := range requests {
		request := request
		jobs[i] = func(ctx context.Context) error {
			if err := protodescriptor.ValidateCodeGeneratorRequest(request); err != nil {
				return err
			}
			return g.handler.Handle(ctx, container, responseBuilder, request)
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := thread.Parallelize(ctx, jobs, thread.ParallelizeWithCancel(cancel)); err != nil {
		return nil, err
	}
	response := responseBuilder.toResponse()
	if err := protodescriptor.ValidateCodeGeneratorResponse(response); err != nil {
		return nil, err
	}
	if errString := response.GetError(); errString != "" {
		return nil, errors.New(errString)
	}
	return response, nil
}
