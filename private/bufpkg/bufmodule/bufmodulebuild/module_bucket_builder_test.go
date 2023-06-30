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

package bufmodulebuild

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/bufbreakingconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/buflintconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBucketGetFileInfos1(t *testing.T) {
	config, err := bufmoduleconfig.NewConfigV1(
		bufmoduleconfig.ExternalConfigV1{
			Excludes: []string{"proto/b"},
		},
	)
	require.NoError(t, err)
	testBucketGetFileInfos(
		t,
		"testdata/1",
		config,
		bufmoduletesting.NewFileInfo(t, "proto/a/1.proto", "testdata/1/proto/a/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/a/2.proto", "testdata/1/proto/a/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/a/3.proto", "testdata/1/proto/a/3.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/a/c/1.proto", "testdata/1/proto/a/c/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/a/c/2.proto", "testdata/1/proto/a/c/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/a/c/3.proto", "testdata/1/proto/a/c/3.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/d/1.proto", "testdata/1/proto/d/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/d/2.proto", "testdata/1/proto/d/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/d/3.proto", "testdata/1/proto/d/3.proto", false, nil, ""),
	)
}

func TestBucketGetFileInfos2(t *testing.T) {
	config, err := bufmoduleconfig.NewConfigV1(
		bufmoduleconfig.ExternalConfigV1{
			Excludes: []string{"proto/a"},
		},
	)
	require.NoError(t, err)
	testBucketGetFileInfos(
		t,
		"testdata/1",
		config,
		bufmoduletesting.NewFileInfo(t, "proto/b/1.proto", "testdata/1/proto/b/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/b/2.proto", "testdata/1/proto/b/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/b/3.proto", "testdata/1/proto/b/3.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/d/1.proto", "testdata/1/proto/d/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/d/2.proto", "testdata/1/proto/d/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/d/3.proto", "testdata/1/proto/d/3.proto", false, nil, ""),
	)
}

func TestBucketGetFileInfo3(t *testing.T) {
	config, err := bufmoduleconfig.NewConfigV1(
		bufmoduleconfig.ExternalConfigV1{
			Excludes: []string{"proto/a/c"},
		},
	)
	require.NoError(t, err)
	testBucketGetFileInfos(
		t,
		"testdata/1",
		config,
		bufmoduletesting.NewFileInfo(t, "proto/a/1.proto", "testdata/1/proto/a/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/a/2.proto", "testdata/1/proto/a/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/a/3.proto", "testdata/1/proto/a/3.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/b/1.proto", "testdata/1/proto/b/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/b/2.proto", "testdata/1/proto/b/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/b/3.proto", "testdata/1/proto/b/3.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/d/1.proto", "testdata/1/proto/d/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/d/2.proto", "testdata/1/proto/d/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/d/3.proto", "testdata/1/proto/d/3.proto", false, nil, ""),
	)
}

func TestBucketGetFileInfos4(t *testing.T) {
	config, err := bufmoduleconfig.NewConfigV1(
		bufmoduleconfig.ExternalConfigV1{
			Excludes: []string{
				"proto/a/c",
				"proto/d",
			},
		},
	)
	require.NoError(t, err)
	testBucketGetFileInfos(
		t,
		"testdata/1",
		config,
		bufmoduletesting.NewFileInfo(t, "proto/a/1.proto", "testdata/1/proto/a/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/a/2.proto", "testdata/1/proto/a/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/a/3.proto", "testdata/1/proto/a/3.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/b/1.proto", "testdata/1/proto/b/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/b/2.proto", "testdata/1/proto/b/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/b/3.proto", "testdata/1/proto/b/3.proto", false, nil, ""),
	)
}

func TestBucketGetAllFileInfos5(t *testing.T) {
	config, err := bufmoduleconfig.NewConfigV1(
		bufmoduleconfig.ExternalConfigV1{},
	)
	require.NoError(t, err)
	testBucketGetFileInfos(
		t,
		"testdata/3",
		config,
	)
}

func TestConfigV1Beta1BucketGetFileInfos1(t *testing.T) {
	config, err := bufmoduleconfig.NewConfigV1Beta1(
		bufmoduleconfig.ExternalConfigV1Beta1{
			Roots: []string{
				"proto",
			},
			Excludes: []string{
				"proto/b",
			},
		},
	)
	require.NoError(t, err)
	testBucketGetFileInfos(
		t,
		"testdata/1",
		config,
		bufmoduletesting.NewFileInfo(t, "a/1.proto", "testdata/1/proto/a/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "a/2.proto", "testdata/1/proto/a/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "a/3.proto", "testdata/1/proto/a/3.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "a/c/1.proto", "testdata/1/proto/a/c/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "a/c/2.proto", "testdata/1/proto/a/c/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "a/c/3.proto", "testdata/1/proto/a/c/3.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "d/1.proto", "testdata/1/proto/d/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "d/2.proto", "testdata/1/proto/d/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "d/3.proto", "testdata/1/proto/d/3.proto", false, nil, ""),
	)
}

func TestConfigV1Beta1BucketGetFileInfos2(t *testing.T) {
	config, err := bufmoduleconfig.NewConfigV1Beta1(
		bufmoduleconfig.ExternalConfigV1Beta1{
			Roots: []string{
				"proto",
			},
			Excludes: []string{
				"proto/a",
			},
		},
	)
	require.NoError(t, err)
	testBucketGetFileInfos(
		t,
		"testdata/1",
		config,
		bufmoduletesting.NewFileInfo(t, "b/1.proto", "testdata/1/proto/b/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "b/2.proto", "testdata/1/proto/b/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "b/3.proto", "testdata/1/proto/b/3.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "d/1.proto", "testdata/1/proto/d/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "d/2.proto", "testdata/1/proto/d/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "d/3.proto", "testdata/1/proto/d/3.proto", false, nil, ""),
	)
}

func TestConfigV1Beta1BucketGetFileInfo3(t *testing.T) {
	config, err := bufmoduleconfig.NewConfigV1Beta1(
		bufmoduleconfig.ExternalConfigV1Beta1{
			Roots: []string{
				"proto",
			},
			Excludes: []string{
				"proto/a/c",
			},
		},
	)
	require.NoError(t, err)
	testBucketGetFileInfos(
		t,
		"testdata/1",
		config,
		bufmoduletesting.NewFileInfo(t, "a/1.proto", "testdata/1/proto/a/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "a/2.proto", "testdata/1/proto/a/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "a/3.proto", "testdata/1/proto/a/3.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "b/1.proto", "testdata/1/proto/b/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "b/2.proto", "testdata/1/proto/b/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "b/3.proto", "testdata/1/proto/b/3.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "d/1.proto", "testdata/1/proto/d/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "d/2.proto", "testdata/1/proto/d/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "d/3.proto", "testdata/1/proto/d/3.proto", false, nil, ""),
	)
}

func TestConfigV1Beta1BucketGetFileInfos4(t *testing.T) {
	config, err := bufmoduleconfig.NewConfigV1Beta1(
		bufmoduleconfig.ExternalConfigV1Beta1{
			Roots: []string{
				"proto",
			},
			Excludes: []string{
				"proto/a/c",
				"proto/d",
			},
		},
	)
	require.NoError(t, err)
	testBucketGetFileInfos(
		t,
		"testdata/1",
		config,
		bufmoduletesting.NewFileInfo(t, "a/1.proto", "testdata/1/proto/a/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "a/2.proto", "testdata/1/proto/a/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "a/3.proto", "testdata/1/proto/a/3.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "b/1.proto", "testdata/1/proto/b/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "b/2.proto", "testdata/1/proto/b/2.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "b/3.proto", "testdata/1/proto/b/3.proto", false, nil, ""),
	)
}

func TestConfigV1Beta1BucketGetAllFileInfos5(t *testing.T) {
	config, err := bufmoduleconfig.NewConfigV1Beta1(
		bufmoduleconfig.ExternalConfigV1Beta1{
			Roots: []string{
				".",
			},
		},
	)
	require.NoError(t, err)
	testBucketGetFileInfos(
		t,
		"testdata/3",
		config,
	)
}

func TestConfigV1Beta1BucketGetAllFileInfosError1(t *testing.T) {
	config, err := bufmoduleconfig.NewConfigV1Beta1(
		bufmoduleconfig.ExternalConfigV1Beta1{
			Roots: []string{
				"a",
				"b",
			},
		},
	)
	require.NoError(t, err)
	testBucketGetAllFileInfosError(
		t,
		"testdata/2",
		config,
		nil,
	)
}

func TestConfigV1Beta1BucketGetFileInfosForExternalPathsError1(t *testing.T) {
	config, err := bufmoduleconfig.NewConfigV1Beta1(
		bufmoduleconfig.ExternalConfigV1Beta1{
			Roots: []string{
				"a",
				"b",
			},
		},
	)
	require.NoError(t, err)
	testBucketGetFileInfosForExternalPathsError(
		t,
		"testdata/2",
		config,
		[]string{
			"testdata/2/a/1.proto",
			"testdata/2/a/2.proto",
			"testdata/2/a/3.proto",
			"testdata/2/b/1.proto",
			"testdata/2/b/4.proto",
		},
	)
}

func TestDocumentation(t *testing.T) {
	testDocumentationBucket(
		t,
		"testdata/4",
		bufmodule.DefaultDocumentationPath,
		bufmoduletesting.NewFileInfo(t, "proto/1.proto", "testdata/4/proto/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/a/2.proto", "testdata/4/proto/a/2.proto", false, nil, ""),
	)
}

func TestLicense(t *testing.T) {
	t.Parallel()
	testLicenseBucket(
		t,
		"testdata/5",
		"Test Module License",
		bufmoduletesting.NewFileInfo(t, "proto/1.proto", "testdata/5/proto/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/a/2.proto", "testdata/5/proto/a/2.proto", false, nil, ""),
	)
}

func TestConfigInclusion(t *testing.T) {
	t.Parallel()
	t.Run("buf.yaml", func(t *testing.T) {
		t.Parallel()
		testConfigInclusion(t, "buf.yaml")
	})
	t.Run("buf.mod", func(t *testing.T) {
		t.Parallel()
		testConfigInclusion(t, "buf.mod")
	})
}

func testConfigInclusion(t *testing.T, confname string) {
	// bucket creation
	bufyaml := `
version: v1
breaking:
  ignore_unstable_packages: true
lint:
  allow_comment_ignores: true
`
	ctx := context.Background()
	bucket, err := memBucket(ctx,
		confname, bufyaml,
		"a/1.proto", "",
	)
	require.NoError(t, err)

	// build
	config, err := bufmoduleconfig.NewConfigV1(
		bufmoduleconfig.ExternalConfigV1{},
	)
	require.NoError(t, err)
	module, err := NewModuleBucketBuilder().BuildForBucket(
		ctx,
		bucket,
		config,
	)
	require.NoError(t, err)
	require.NotNil(t, module)

	// assert: one proto consumed
	fileInfos, err := module.TargetFileInfos(ctx)
	assert.NoError(t, err)
	assert.Len(t, fileInfos, 1)

	// assert: breaking and lint configuration exists
	zeroBreaking := bufbreakingconfig.NewConfigV1(
		bufbreakingconfig.ExternalConfigV1{},
	)
	assert.NotEqual(t, zeroBreaking, module.BreakingConfig(), "empty BreakingConfig")
	zeroLint := buflintconfig.NewConfigV1(
		buflintconfig.ExternalConfigV1{},
	)
	assert.NotEqual(t, zeroLint, module.LintConfig(), "empty LintConfig")
}

func memBucket(ctx context.Context, pathcontent ...string) (storage.ReadBucket, error) {
	membucket := storagemem.NewReadWriteBucket()
	for i := 0; i < len(pathcontent); i += 2 {
		fh, err := membucket.Put(ctx, pathcontent[i])
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(fh, strings.NewReader(pathcontent[i+1]))
		if err != nil {
			return nil, err
		}
		fh.Close()
	}
	return membucket, nil
}

func testBucketGetFileInfos(
	t *testing.T,
	relDir string,
	config *bufmoduleconfig.Config,
	expectedFileInfos ...bufmoduleref.FileInfo,
) {
	t.Parallel()
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		relDir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	module, err := NewModuleBucketBuilder().BuildForBucket(
		context.Background(),
		readWriteBucket,
		config,
	)
	require.NoError(t, err)
	fileInfos, err := module.SourceFileInfos(context.Background())
	assert.NoError(t, err)
	assert.Equal(
		t,
		expectedFileInfos,
		fileInfos,
	)
	if len(expectedFileInfos) > 1 {
		expectedFileInfos = expectedFileInfos[:len(expectedFileInfos)-1]
		bucketRelPaths := make([]string, len(expectedFileInfos))
		for i, expectedFileInfo := range expectedFileInfos {
			bucketRelExternalPath, err := filepath.Rel(relDir, expectedFileInfo.ExternalPath())
			require.NoError(t, err)
			bucketRelPath, err := normalpath.NormalizeAndValidate(bucketRelExternalPath)
			require.NoError(t, err)
			bucketRelPaths[i] = bucketRelPath
		}
		module, err := NewModuleBucketBuilder().BuildForBucket(
			context.Background(),
			readWriteBucket,
			config,
			WithPaths(bucketRelPaths),
		)
		require.NoError(t, err)
		fileInfos, err := module.TargetFileInfos(context.Background())
		assert.NoError(t, err)
		assert.Equal(
			t,
			expectedFileInfos,
			fileInfos,
		)
	}
}

func testBucketGetAllFileInfosError(
	t *testing.T,
	relDir string,
	config *bufmoduleconfig.Config,
	expectedSpecificError error,
) {
	t.Parallel()
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		relDir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	module, err := NewModuleBucketBuilder().BuildForBucket(
		context.Background(),
		readWriteBucket,
		config,
	)
	require.NoError(t, err)
	_, err = module.SourceFileInfos(context.Background())
	if expectedSpecificError != nil {
		assert.Equal(t, expectedSpecificError, err)
	} else {
		assert.Error(t, err)
	}
}

func testBucketGetFileInfosForExternalPathsError(
	t *testing.T,
	relDir string,
	config *bufmoduleconfig.Config,
	externalPaths []string,
) {
	t.Parallel()
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		relDir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	bucketRelPaths := make([]string, len(externalPaths))
	for i, externalPath := range externalPaths {
		bucketRelExternalPath, err := filepath.Rel(relDir, externalPath)
		require.NoError(t, err)
		bucketRelPath, err := normalpath.NormalizeAndValidate(bucketRelExternalPath)
		require.NoError(t, err)
		bucketRelPaths[i] = bucketRelPath
	}
	_, err = NewModuleBucketBuilder().BuildForBucket(
		context.Background(),
		readWriteBucket,
		config,
		WithPaths(bucketRelPaths),
	)
	assert.Error(t, err)
}

func testDocumentationBucket(
	t *testing.T,
	relDir string,
	expectedDocPath string,
	expectedFileInfos ...bufmoduleref.FileInfo,
) {
	t.Parallel()
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		relDir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	config, err := bufmoduleconfig.NewConfigV1(
		bufmoduleconfig.ExternalConfigV1{},
	)
	require.NoError(t, err)
	module, err := NewModuleBucketBuilder().BuildForBucket(
		context.Background(),
		readWriteBucket,
		config,
	)
	require.NoError(t, err)
	require.NotNil(t, module)
	assert.NotEmpty(t, module.Documentation())
	assert.Equal(t, expectedDocPath, module.DocumentationPath())
	fileInfos, err := module.TargetFileInfos(context.Background())
	assert.NoError(t, err)
	assert.Equal(
		t,
		expectedFileInfos,
		fileInfos,
	)
}

func testLicenseBucket(
	t *testing.T,
	relDir string,
	expectedLicense string,
	expectedFileInfos ...bufmoduleref.FileInfo,
) {
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		relDir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	config, err := bufmoduleconfig.NewConfigV1(
		bufmoduleconfig.ExternalConfigV1{},
	)
	require.NoError(t, err)
	module, err := NewModuleBucketBuilder().BuildForBucket(
		context.Background(),
		readWriteBucket,
		config,
	)
	require.NoError(t, err)
	require.NotNil(t, module)
	assert.NotEmpty(t, module.License())
	assert.Equal(t, expectedLicense, module.License())
	fileInfos, err := module.TargetFileInfos(context.Background())
	assert.NoError(t, err)
	assert.Equal(
		t,
		expectedFileInfos,
		fileInfos,
	)
}
