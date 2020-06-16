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

package github

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/fetch"
	"github.com/bufbuild/buf/internal/pkg/httpauth"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type archiveReader struct {
	logger      *zap.Logger
	fetchReader fetch.Reader
}

func newArchiveReader(
	logger *zap.Logger,
	httpClient *http.Client,
	httpAuthenticator httpauth.Authenticator,
) *archiveReader {
	return &archiveReader{
		logger: logger.Named("github"),
		fetchReader: fetch.NewReader(
			logger,
			fetch.WithReaderHTTP(
				httpClient,
				httpAuthenticator,
			),
		),
	}
}

func (a *archiveReader) GetArchive(
	ctx context.Context,
	container app.EnvStdinContainer,
	outputDirPath string,
	owner string,
	repository string,
	ref string,
) (retErr error) {
	outputDirPath = filepath.Clean(outputDirPath)
	if outputDirPath == "" || outputDirPath == "." || outputDirPath == "/" {
		return fmt.Errorf("bad output dir path: %s", outputDirPath)
	}
	// check if already exists
	if fileInfo, err := os.Stat(outputDirPath); err == nil {
		if !fileInfo.IsDir() {
			return fmt.Errorf("expected %s to be a directory", outputDirPath)
		}
		return nil
	}
	archiveRef, err := fetch.NewArchiveRef(
		fmt.Sprintf("https://github.com/%s/%s/archive/%s.tar.gz", owner, repository, ref),
		fetch.ArchiveTypeTar,
		fetch.CompressionTypeGzip,
		1,
	)
	if err != nil {
		return err
	}
	readBucketCloser, err := a.fetchReader.GetBucket(ctx, container, archiveRef)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, readBucketCloser.Close())
	}()
	if err := os.MkdirAll(outputDirPath, 0755); err != nil {
		return err
	}
	readWriteBucketCloser, err := storageos.NewReadWriteBucketCloser(outputDirPath)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, readWriteBucketCloser.Close())
	}()
	_, err = storage.Copy(ctx, readBucketCloser, readWriteBucketCloser)
	return err
}
