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

package bufcli

import (
	"context"
	"errors"
	"io/fs"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
)

// GetBufYAMLFileForDirPathOrOverride get the buf.yaml file for either the usually-flag-based
// override, or if the override is not set, the directory path.
func GetBufYAMLFileForDirPathOrOverride(
	ctx context.Context,
	dirPath string,
	override string,
) (bufconfig.BufYAMLFile, error) {
	bucket, err := newOSReadWriteBucketWithSymlinks(dirPath)
	if err != nil {
		return nil, err
	}
	return bufconfig.GetBufYAMLFileForPrefixOrOverride(ctx, bucket, ".", override)
}

// GetBufYAMLFileForDirPath gets the buf.yaml file for the directory path.
func GetBufYAMLFileForDirPath(
	ctx context.Context,
	dirPath string,
) (bufconfig.BufYAMLFile, error) {
	bucket, err := newOSReadWriteBucketWithSymlinks(dirPath)
	if err != nil {
		return nil, err
	}
	return bufconfig.GetBufYAMLFileForPrefix(ctx, bucket, ".")
}

// PutBufYAMLFileForDirPath write the buf.yaml file to the directory path.
func PutBufYAMLFileForDirPath(
	ctx context.Context,
	dirPath string,
	bufYAMLFile bufconfig.BufYAMLFile,
) error {
	bucket, err := newOSReadWriteBucketWithSymlinks(dirPath)
	if err != nil {
		return err
	}
	return bufconfig.PutBufYAMLFileForPrefix(ctx, bucket, ".", bufYAMLFile)
}

// BufYAMLFileExistsForDirPath returns true if the buf.yaml file exists at the dir path.
func BufYAMLFileExistsForDirPath(
	ctx context.Context,
	dirPath string,
) (bool, error) {
	bucket, err := newOSReadWriteBucketWithSymlinks(dirPath)
	if err != nil {
		return false, err
	}
	if _, err := bufconfig.GetBufYAMLFileVersionForPrefix(ctx, bucket, "."); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func newOSReadWriteBucketWithSymlinks(dirPath string) (storage.ReadWriteBucket, error) {
	return storageos.NewProvider(
		storageos.ProviderWithSymlinks(),
	).NewReadWriteBucket(
		dirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
}
