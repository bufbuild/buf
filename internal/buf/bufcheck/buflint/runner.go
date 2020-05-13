// Copyright 2020 Buf Technologies Inc.
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

package buflint

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufcheck/internal"
	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/proto/protosrc"
	"go.uber.org/zap"
)

type runner struct {
	delegate *internal.Runner
}

func newRunner(logger *zap.Logger) *runner {
	return &runner{
		delegate: internal.NewRunner(logger.Named("lint")),
	}
}

func (r *runner) Check(ctx context.Context, config *Config, files []protosrc.File) ([]*filev1beta1.FileAnnotation, error) {
	return r.delegate.Check(ctx, configToInternalConfig(config), nil, files)
}
