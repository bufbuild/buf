package buftesting

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/internal/buf/buferrs"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/bufbuild/buf/internal/pkg/storage/storageutil"
	"go.uber.org/multierr"
)

func getGithubArchive(
	ctx context.Context,
	httpClient *http.Client,
	outputDirPath string,
	owner string,
	repository string,
	ref string,
) (retErr error) {
	outputDirPath = filepath.Clean(outputDirPath)
	if outputDirPath == "" || outputDirPath == "." || outputDirPath == "/" {
		return buferrs.NewSystemErrorf("bad output dir path: %s", outputDirPath)
	}
	// check if already exists
	if fileInfo, err := os.Stat(outputDirPath); err == nil {
		if !fileInfo.IsDir() {
			return buferrs.NewSystemErrorf("expected %s to be a directory", outputDirPath)
		}
		return nil
	}

	request, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://github.com/%s/%s/archive/%s.tar.gz", owner, repository, ref), nil)
	if err != nil {
		return err
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, response.Body.Close())
	}()
	if response.StatusCode != http.StatusOK {
		return buferrs.NewSystemErrorf("expected HTTP status code %d to be %d", response.StatusCode, http.StatusOK)
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

	bucket, err := storageos.NewBucket(outputDirPath)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, bucket.Close())
	}()
	return storageutil.Untargz(
		ctx,
		response.Body,
		bucket,
		storagepath.WithStripComponents(1),
	)
}
