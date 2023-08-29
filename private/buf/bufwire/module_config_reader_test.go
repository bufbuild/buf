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

package bufwire

import (
	"context"
	"io"
	"testing"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestExcludePathsForModule(t *testing.T) {
	t.Parallel()
	moduleConfigReader := NewModuleConfigReader(
		zap.NewNop(),
		storageos.NewProvider(),
		&fakeModuleFetcher{},
		nil,
	)
	someModuleRef, err := buffetch.NewModuleRefParser(zap.NewNop()).GetModuleRef(context.Background(), "buf.build/foo/bar")
	require.NoError(t, err)
	moduleConfigSet, err := moduleConfigReader.GetModuleConfigSet(
		context.Background(),
		nil,
		someModuleRef, // the fake module fetcher doesn't care what ref this is
		"",
		nil,
		[]string{"dir/foo.proto"},
		true,
	)
	require.NoError(t, err)
	require.Len(t, moduleConfigSet.ModuleConfigs(), 1)
	moduleConfig := moduleConfigSet.ModuleConfigs()[0]
	require.NotNil(t, moduleConfig)
	module := moduleConfig.Module()
	require.NotNil(t, module)
	targetFileInfos, err := module.TargetFileInfos(context.Background())
	require.NoError(t, err)
	require.Len(t, targetFileInfos, 1)
	require.Equal(t, "dir/bar.proto", targetFileInfos[0].Path())
}

func TestIncludePathsForModule(t *testing.T) {
	t.Parallel()
	moduleConfigReader := NewModuleConfigReader(
		zap.NewNop(),
		storageos.NewProvider(),
		&fakeModuleFetcher{},
		nil,
	)
	someModuleRef, err := buffetch.NewModuleRefParser(zap.NewNop()).GetModuleRef(context.Background(), "buf.build/foo/bar")
	require.NoError(t, err)
	moduleConfigSet, err := moduleConfigReader.GetModuleConfigSet(
		context.Background(),
		nil,
		someModuleRef, // the fake module fetcher doesn't care what ref this is
		"",
		[]string{"dir/foo.proto"},
		nil,
		true,
	)
	require.NoError(t, err)
	require.Len(t, moduleConfigSet.ModuleConfigs(), 1)
	moduleConfig := moduleConfigSet.ModuleConfigs()[0]
	require.NotNil(t, moduleConfig)
	module := moduleConfig.Module()
	require.NotNil(t, module)
	targetFileInfos, err := module.TargetFileInfos(context.Background())
	require.NoError(t, err)
	require.Len(t, targetFileInfos, 1)
	require.Equal(t, "dir/foo.proto", targetFileInfos[0].Path())
}

func TestIncludeAndExcludePathsForModule(t *testing.T) {
	t.Parallel()
	moduleConfigReader := NewModuleConfigReader(
		zap.NewNop(),
		storageos.NewProvider(),
		&fakeModuleFetcher{},
		nil,
	)
	someModuleRef, err := buffetch.NewModuleRefParser(zap.NewNop()).GetModuleRef(context.Background(), "buf.build/foo/bar")
	require.NoError(t, err)
	moduleConfigSet, err := moduleConfigReader.GetModuleConfigSet(
		context.Background(),
		nil,
		someModuleRef, // the fake module fetcher doesn't care what ref this is
		"",
		[]string{"dir"},
		[]string{"dir/bar.proto"},
		true,
	)
	require.NoError(t, err)
	require.Len(t, moduleConfigSet.ModuleConfigs(), 1)
	moduleConfig := moduleConfigSet.ModuleConfigs()[0]
	require.NotNil(t, moduleConfig)
	module := moduleConfig.Module()
	require.NotNil(t, module)
	targetFileInfos, err := module.TargetFileInfos(context.Background())
	require.NoError(t, err)
	require.Len(t, targetFileInfos, 1)
	require.Equal(t, "dir/foo.proto", targetFileInfos[0].Path())
}

type fakeModuleFetcher struct{}

func (r *fakeModuleFetcher) GetModule(
	ctx context.Context,
	container app.EnvStdinContainer,
	moduleRef buffetch.ModuleRef,
) (bufmodule.Module, error) {
	fileFoo := `message Foo {}
`
	fileBar := `message Foo {}
`
	moduleBucket, err := storagemem.NewReadBucket(
		map[string][]byte{
			"dir/foo.proto": []byte(fileFoo),
			"dir/bar.proto": []byte(fileBar),
		},
	)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleForBucket(
		context.Background(),
		moduleBucket,
	)
}

func (r *fakeModuleFetcher) GetImageFile(
	ctx context.Context,
	container app.EnvStdinContainer,
	imageRef buffetch.ImageRef,
) (io.ReadCloser, error) {
	return nil, nil
}

func (r *fakeModuleFetcher) GetSourceBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	sourceRef buffetch.SourceRef,
	options ...buffetch.GetSourceBucketOption,
) (buffetch.ReadBucketCloserWithTerminateFileProvider, error) {
	return nil, nil
}
