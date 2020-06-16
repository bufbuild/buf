// Copyright 2020 Buf Technologies, Inc.
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

package buffetch

import (
	"context"
	"io"
	"net/http"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/fetch"
	"github.com/bufbuild/buf/internal/pkg/git"
	"github.com/bufbuild/buf/internal/pkg/httpauth"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

type reader struct {
	fetchReader fetch.Reader
}

func newReader(
	logger *zap.Logger,
	httpClient *http.Client,
	httpAuthenticator httpauth.Authenticator,
	gitCloner git.Cloner,
) *reader {
	return &reader{
		fetchReader: fetch.NewReader(
			logger,
			fetch.WithReaderHTTP(
				httpClient,
				httpAuthenticator,
			),
			fetch.WithReaderGit(
				gitCloner,
			),
			fetch.WithReaderLocal(),
			fetch.WithReaderStdio(),
		),
	}
}

func (a *reader) GetImageFile(
	ctx context.Context,
	container app.EnvStdinContainer,
	imageRef ImageRef,
) (io.ReadCloser, error) {
	return a.fetchReader.GetFile(ctx, container, imageRef.fetchFileRef())
}

func (a *reader) GetSourceBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	sourceRef SourceRef,
) (storage.ReadBucketCloser, error) {
	return a.fetchReader.GetBucket(ctx, container, sourceRef.fetchBucketRef())
}
