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
	"path/filepath"
	"sync"
	"time"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/gofrs/flock"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type archiveReader struct {
	logger            *zap.Logger
	storageosProvider storageos.Provider
	httpClient        *http.Client
	lock              sync.Mutex
}

func newArchiveReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	httpClient *http.Client,
) *archiveReader {
	return &archiveReader{
		logger:            logger.Named("githubtesting"),
		storageosProvider: storageosProvider,
		httpClient:        httpClient,
	}
}

func (a *archiveReader) GetArchive(
	ctx context.Context,
	outputDirPath string,
	owner string,
	repository string,
	ref string,
) (retErr error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	outputDirPath = normalpath.Unnormalize(outputDirPath)

	// creates a file in the same parent directory as outputDirPath
	flockPath := outputDirPath + ".flock"
	if err := os.MkdirAll(filepath.Dir(flockPath), 0755); err != nil {
		return err
	}
	flock := flock.New(flockPath)
	locked, err := flock.TryLockContext(ctx, time.Second)
	if err != nil {
		return fmt.Errorf("could not get file lock %q: %v", flockPath, err)
	}
	if !locked {
		return fmt.Errorf("could not lock %q", flockPath)
	}
	defer func() {
		retErr = multierr.Append(retErr, flock.Unlock())
	}()

	// check if already exists, if so, do nothing
	// OK to use os.Stat here
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
	// do NOT want to read in symlinks
	readWriteBucket, err := a.storageosProvider.NewReadWriteBucket(normalpath.Normalize(outputDirPath))
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
