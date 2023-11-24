// Copyright 2020-2023 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/private/buf/buffetch/internal"
	"github.com/bufbuild/buf/private/bufnew/bufconfig"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/httpauth"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/zap"
)

type reader struct {
	internalReader internal.Reader
}

func newReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	httpClient *http.Client,
	httpAuthenticator httpauth.Authenticator,
	gitCloner git.Cloner,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
) *reader {
	return &reader{
		internalReader: internal.NewReader(
			logger,
			storageosProvider,
			internal.WithReaderHTTP(
				httpClient,
				httpAuthenticator,
			),
			internal.WithReaderGit(
				gitCloner,
			),
			internal.WithReaderLocal(),
			internal.WithReaderStdio(),
			internal.WithReaderModule(
				moduleKeyProvider,
			),
		),
	}
}

func newMessageReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	httpClient *http.Client,
	httpAuthenticator httpauth.Authenticator,
	gitCloner git.Cloner,
) *reader {
	return &reader{
		internalReader: internal.NewReader(
			logger,
			storageosProvider,
			internal.WithReaderHTTP(
				httpClient,
				httpAuthenticator,
			),
			internal.WithReaderLocal(),
			internal.WithReaderStdio(),
		),
	}
}

func newSourceReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	httpClient *http.Client,
	httpAuthenticator httpauth.Authenticator,
	gitCloner git.Cloner,
) *reader {
	return &reader{
		internalReader: internal.NewReader(
			logger,
			storageosProvider,
			internal.WithReaderHTTP(
				httpClient,
				httpAuthenticator,
			),
			internal.WithReaderGit(
				gitCloner,
			),
			internal.WithReaderLocal(),
			internal.WithReaderStdio(),
		),
	}
}

func newModuleFetcher(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
) *reader {
	return &reader{
		internalReader: internal.NewReader(
			logger,
			storageosProvider,
			internal.WithReaderModule(
				moduleKeyProvider,
			),
		),
	}
}

func (a *reader) GetMessageFile(
	ctx context.Context,
	container app.EnvStdinContainer,
	messageRef MessageRef,
) (io.ReadCloser, error) {
	return a.internalReader.GetFile(ctx, container, messageRef.internalSingleRef())
}

func (a *reader) GetSourceBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	sourceRef SourceRef,
) (ReadBucketCloser, error) {
	return a.internalReader.GetBucket(
		ctx,
		container,
		sourceRef.internalBucketRef(),
		internal.WithGetBucketTerminateFileNames(bufconfig.AllBufWorkYAMLFileNames),
	)
}

func (a *reader) GetModuleKey(
	ctx context.Context,
	container app.EnvStdinContainer,
	moduleRef ModuleRef,
) (bufmodule.ModuleKey, error) {
	return a.internalReader.GetModuleKey(ctx, container, moduleRef.internalModuleRef())
}
