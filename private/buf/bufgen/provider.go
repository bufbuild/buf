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

package bufgen

import (
	"context"
	"io"

	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type provider struct {
	logger *zap.Logger
	tracer trace.Tracer
}

func newProvider(logger *zap.Logger) *provider {
	return &provider{
		logger: logger,
		tracer: otel.GetTracerProvider().Tracer("bufbuild/buf"),
	}
}

func (p *provider) GetConfig(ctx context.Context, readBucket storage.ReadBucket) (_ *Config, retErr error) {
	ctx, span := p.tracer.Start(ctx, "get_config")
	defer span.End()
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()

	readObjectCloser, err := readBucket.Get(ctx, ExternalConfigFilePath)
	if err != nil {
		// There is no default generate template, so we propagate all errors, including
		// storage.ErrNotExist.
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readObjectCloser.Close())
	}()
	data, err := io.ReadAll(readObjectCloser)
	if err != nil {
		return nil, err
	}
	return getConfig(
		p.logger,
		encoding.UnmarshalYAMLNonStrict,
		encoding.UnmarshalYAMLStrict,
		data,
		`File "`+readObjectCloser.ExternalPath()+`"`,
	)
}
