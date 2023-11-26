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

package bufcli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/private/buf/bufconfig"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
)

// GetBufYAMLFileForOverrideOrPrefix get the buf.yaml file for either the usually-flag-based
// override, or if the override is not set, the directory path.
//
//   - If the override is set and ends in .json, .yaml, or .yml, the override is treated as a
//     **direct file path on disk** and read (ie not via buckets).
//   - If the override is otherwise non-empty, it is treated as raw data.
//   - Otherwise, the prefix is read.
//
// This function is the result of the endlessly annoying and shortsighted design decision that the
// original author of this repository made to allow overriding configuration files on the command line.
// Of course, the original author never envisioned buf.work.yamls, merging buf.work.yamls into buf.yamls,
// buf.gen.yamls, or anything of the like, and was very concentrated on "because Bazel."
func GetBufYAMLFileForOverrideOrDirPath(
	ctx context.Context,
	dirPath string,
	override string,
) (bufconfig.BufYAMLFile, error) {
	if override != "" {
		var data []byte
		var err error
		switch filepath.Ext(override) {
		case ".json", ".yaml", ".yml":
			data, err = os.ReadFile(override)
			if err != nil {
				return nil, fmt.Errorf("could not read file: %v", err)
			}
		default:
			data = []byte(override)
		}
		return bufconfig.ReadBufYAMLFile(bytes.NewReader(data))
	}
	return GetBufYAMLFileForDirPath(ctx, dirPath)
}

// GetBufYAMLFileForDirPath gets the buf.yaml file for the directory path.
func GetBufYAMLFileForDirPath(
	ctx context.Context,
	dirPath string,
) (bufconfig.BufYAMLFile, error) {
	bucket, err := storageos.NewProvider(
		storageos.ProviderWithSymlinks(),
	).NewReadWriteBucket(
		dirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
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
	bucket, err := storageos.NewProvider(
		storageos.ProviderWithSymlinks(),
	).NewReadWriteBucket(
		dirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
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
	bucket, err := storageos.NewProvider(
		storageos.ProviderWithSymlinks(),
	).NewReadWriteBucket(
		dirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
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
