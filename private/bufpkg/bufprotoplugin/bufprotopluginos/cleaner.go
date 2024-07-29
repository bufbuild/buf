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

package bufprotopluginos

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/filepathext"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/osext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

type cleaner struct {
	storageosProvider storageos.Provider
}

func newCleaner(
	storageosProvider storageos.Provider,
) *cleaner {
	return &cleaner{
		storageosProvider: storageosProvider,
	}
}

func (c *cleaner) DeleteOuts(
	ctx context.Context,
	pluginOuts []string,
) error {
	pwd, err := osext.Getwd()
	if err != nil {
		return err
	}
	pwd, err = reallyCleanPath(pwd)
	if err != nil {
		return err
	}
	for _, pluginOut := range pluginOuts {
		if err := validatePluginOut(pwd, pluginOut); err != nil {
			return err
		}
	}
	for _, pluginOut := range pluginOuts {
		if err := c.deleteOut(ctx, pluginOut); err != nil {
			return err
		}
	}
	return nil
}

func (c *cleaner) deleteOut(
	ctx context.Context,
	pluginOut string,
) error {
	dirPath := pluginOut
	removePath := "."
	switch filepath.Ext(pluginOut) {
	case ".jar", ".zip":
		dirPath = normalpath.Dir(pluginOut)
		removePath = normalpath.Base(pluginOut)
	default:
		// Assume output is a directory.
	}
	bucket, err := c.storageosProvider.NewReadWriteBucket(
		dirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	return bucket.DeleteAll(ctx, removePath)
}

func validatePluginOut(pwd string, pluginOut string) error {
	if pluginOut == "" {
		// This is just triple-making sure.
		return syserror.New("got empty pluginOut in bufprotopluginos.Cleaner")
	}
	if pluginOut == "." {
		// This is just a really defensive safety check. We can't see a reason you'd want to delete
		// your current working directory other than something like a (cd proto && buf generate), so
		// until and unless someone complains, we're just going to outlaw this.
		return errors.New("cannot use --clean if your plugin will output to the current directory")
	}
	cleanedPluginOut, err := reallyCleanPath(pluginOut)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if cleanedPluginOut == pwd {
		// Same thing, more defense for now.
		return errors.New("cannot use --clean if your plugin will output to the current directory")
	}
	return nil
}

func reallyCleanPath(path string) (string, error) {
	path, err := filepathext.RealClean(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(path)
}
