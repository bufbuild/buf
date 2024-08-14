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

// TODO: This entire package should be deleted once bufcheckserver is operational.
package bufplugin

import (
	"context"
	"io"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/pluginrpc-go"
	"go.uber.org/zap"
)

type CheckClient interface {
	// Check checks the image.
	//
	// The image should have source code info for this to work properly.
	//
	// Images should *not* be filtered with regards to imports before passing to this function.
	// TODO: reconcile with bufbreaking.
	//
	// An error of type bufanalysis.FileAnnotationSet will be returned lint failure.
	Check(ctx context.Context, image bufimage.Image, againstImage bufimage.Image) error
}

func NewCheckClient(
	logger *zap.Logger,
	stderr io.Writer,
	runner command.Runner,
	programName string,
) CheckClient {
	return newCheckClient(
		logger,
		pluginrpc.NewClient(
			newRunner(runner, programName),
			pluginrpc.ClientWithStderr(stderr),
		),
	)
}
