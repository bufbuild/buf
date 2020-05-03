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

package utilgithubtesting

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/bufbuild/buf/internal/pkg/storage/storageutil"
	"go.uber.org/multierr"
)

var testHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

// GetGithubArchive gets the GitHub archive and untars it to the output directory path.
//
// The root directory within the tarball is stripped.
// If the directory already exists, this is a no-op.
//
// Only use for testing.
func GetGithubArchive(
	ctx context.Context,
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

	request, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://github.com/%s/%s/archive/%s.tar.gz", owner, repository, ref), nil)
	if err != nil {
		return err
	}
	response, err := testHTTPClient.Do(request)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, response.Body.Close())
	}()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("expected HTTP status code %d to be %d", response.StatusCode, http.StatusOK)
	}

	if err := os.MkdirAll(outputDirPath, 0755); err != nil {
		return err
	}
	// only re-add this if this starts to be a problem
	// this is dangerous
	//defer func() {
	//if retErr != nil {
	//retErr = os.RemoveAll(outputDirPath)
	//}
	//}()

	readWriteBucketCloser, err := storageos.NewReadWriteBucketCloser(outputDirPath)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, readWriteBucketCloser.Close())
	}()
	return storageutil.Untargz(
		ctx,
		response.Body,
		readWriteBucketCloser,
		storagepath.WithStripComponents(1),
	)
}
