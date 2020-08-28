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

package githubtesting

import (
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type archiveReader struct {
	logger     *zap.Logger
	httpClient *http.Client
}

func newArchiveReader(
	logger *zap.Logger,
	httpClient *http.Client,
) *archiveReader {
	return &archiveReader{
		logger:     logger.Named("githubtesting"),
		httpClient: httpClient,
	}
}

func (a *archiveReader) GetArchive(
	ctx context.Context,
	outputDirPath string,
	owner string,
	repository string,
	ref string,
) (retErr error) {
	outputDirPath = normalpath.Normalize(outputDirPath)
	// check if already exists, if so, do nothing
	if fileInfo, err := os.Stat(outputDirPath); err == nil {
		if !fileInfo.IsDir() {
			return fmt.Errorf("expected %s to be a directory", outputDirPath)
		}
		return nil
	}
	request, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("https://github.com/%s/%s/archive/%s.tar.gz", owner, repository, ref),
		nil,
	)
	if err != nil {
		return err
	}
	response, err := a.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, response.Body.Close())
	}()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("expected HTTP status code %d to be %d", response.StatusCode, http.StatusOK)
	}
	gzipReader, err := gzip.NewReader(response.Body)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, gzipReader.Close())
	}()
	if err := os.MkdirAll(outputDirPath, 0755); err != nil {
		return err
	}
	readWriteBucket, err := storageos.NewReadWriteBucket(outputDirPath)
	if err != nil {
		return err
	}
	return storagearchive.Untar(
		ctx,
		gzipReader,
		readWriteBucket,
		nil,
		1,
	)
}
