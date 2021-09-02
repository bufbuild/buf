// Copyright 2020-2021 Buf Technologies, Inc.
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

package appprotoos

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/app/appproto/appprotoexec"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

var (
	manifestPath    = normalpath.Join("META-INF", "MANIFEST.MF")
	manifestContent = []byte(`Manifest-Version: 1.0
Created-By: 1.6.0 (protoc)

`)
)

type generator struct {
	logger            *zap.Logger
	storageosProvider storageos.Provider
}

func newGenerator(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
) *generator {
	return &generator{
		logger:            logger,
		storageosProvider: storageosProvider,
	}
}

func (g *generator) Generate(
	ctx context.Context,
	container app.EnvStderrContainer,
	pluginName string,
	pluginOut string,
	requests []*pluginpb.CodeGeneratorRequest,
	options ...GenerateOption,
) (retErr error) {
	generateOptions := newGenerateOptions()
	for _, option := range options {
		option(generateOptions)
	}
	handler, err := appprotoexec.NewHandler(
		g.logger,
		g.storageosProvider,
		pluginName,
		appprotoexec.HandlerWithPluginPath(generateOptions.pluginPath),
	)
	if err != nil {
		return err
	}
	appprotoGenerator := appproto.NewGenerator(g.logger, handler)
	readBucketBuilder := storagemem.NewReadBucketBuilder()
	if err := appprotoGenerator.Generate(ctx, container, readBucketBuilder, requests); err != nil {
		return err
	}
	readBucket, err := readBucketBuilder.ToReadBucket()
	if err != nil {
		return err
	}
	return writePluginOutput(
		ctx,
		readBucket,
		pluginOut,
		generateOptions.createOutDirIfNotExists,
		g.storageosProvider,
	)
}

func writePluginOutput(
	ctx context.Context,
	readBucket storage.ReadBucket,
	pluginOut string,
	createOutDirIfNotExists bool,
	storageosProvider storageos.Provider,
) error {
	switch filepath.Ext(pluginOut) {
	case ".jar":
		return generateZip(
			ctx,
			readBucket,
			pluginOut,
			true,
			createOutDirIfNotExists,
		)
	case ".zip":
		return generateZip(
			ctx,
			readBucket,
			pluginOut,
			false,
			createOutDirIfNotExists,
		)
	default:
		return generateDirectory(
			ctx,
			readBucket,
			pluginOut,
			createOutDirIfNotExists,
			storageosProvider,
		)
	}
}

func generateZip(
	ctx context.Context,
	readBucket storage.ReadBucket,
	outFilePath string,
	includeManifest bool,
	createOutDirIfNotExists bool,
) (retErr error) {
	outDirPath := filepath.Dir(outFilePath)
	// OK to use os.Stat instead of os.Lstat here
	fileInfo, err := os.Stat(outDirPath)
	if err != nil {
		if os.IsNotExist(err) {
			if createOutDirIfNotExists {
				if err := os.MkdirAll(outDirPath, 0755); err != nil {
					return err
				}
			} else {
				return err
			}
		}
		return err
	} else if !fileInfo.IsDir() {
		return fmt.Errorf("not a directory: %s", outDirPath)
	}
	if includeManifest {
		readBucketBuilder := storagemem.NewReadBucketBuilder()
		if err := storage.PutPath(ctx, readBucketBuilder, manifestPath, manifestContent); err != nil {
			return err
		}
		manifestReadBucket, err := readBucketBuilder.ToReadBucket()
		if err != nil {
			return err
		}
		readBucket = storage.MultiReadBucket(readBucket, manifestReadBucket)
	}
	file, err := os.Create(outFilePath)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, file.Close())
	}()
	// protoc does not compress
	return storagearchive.Zip(ctx, readBucket, file, false)
}

func generateDirectory(
	ctx context.Context,
	readBucket storage.ReadBucket,
	outDirPath string,
	createOutDirIfNotExists bool,
	storageosProvider storageos.Provider,
) error {
	if createOutDirIfNotExists {
		if err := os.MkdirAll(outDirPath, 0755); err != nil {
			return err
		}
	}
	// this checks that the directory exists
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		outDirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	if _, err := storage.Copy(ctx, readBucket, readWriteBucket); err != nil {
		return err
	}
	return nil
}

type generateOptions struct {
	pluginPath              string
	createOutDirIfNotExists bool
}

func newGenerateOptions() *generateOptions {
	return &generateOptions{}
}
