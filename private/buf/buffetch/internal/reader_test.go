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

package internal

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/buf/buftarget"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/osext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetReadBucketCloserForBucketNoTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	inputBucket, err := storageos.NewProvider().NewReadWriteBucket("testdata/bufyaml/one/two")
	require.NoError(t, err)
	readBucketCloser, err := getReadBucketCloserForBucket(
		ctx,
		zap.NewNop(),
		storage.NopReadBucketCloser(inputBucket),
		"three/four/five",
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, ".", readBucketCloser.SubDirPath())
	_, err = readBucketCloser.Stat(ctx, "buf.yaml")
	require.NoError(t, err)
	require.NoError(t, readBucketCloser.Close())
}

func TestGetReadBucketCloserTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	inputBucket, err := storageos.NewProvider().NewReadWriteBucket("testdata/bufyaml/one/two")
	require.NoError(t, err)
	readBucketCloser, err := getReadBucketCloserForBucket(
		ctx,
		zap.NewNop(),
		storage.NopReadBucketCloser(inputBucket),
		"three/four/five",
		testNewTerminateAtFileNamesFunc("buf.work.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readBucketCloser.SubDirPath())
	_, err = readBucketCloser.Stat(ctx, "four/five/buf.yaml")
	require.NoError(t, err)
	require.NoError(t, readBucketCloser.Close())
}

func TestGetReadBucketCloserForBucketNoSubDirPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	inputBucket, err := storageos.NewProvider().NewReadWriteBucket("testdata/bufyaml/one/two/three/four/five")
	require.NoError(t, err)
	readBucketCloser, err := getReadBucketCloserForBucket(
		ctx,
		zap.NewNop(),
		storage.NopReadBucketCloser(inputBucket),
		".",
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, ".", readBucketCloser.SubDirPath())
	_, err = readBucketCloser.Stat(ctx, "buf.yaml")
	require.NoError(t, err)
	require.NoError(t, readBucketCloser.Close())
}

func TestGetReadBucketCloserForBucketAbs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	absDirPath, err := filepath.Abs(".")
	require.NoError(t, err)
	inputBucket, err := storageos.NewProvider().NewReadWriteBucket(normalpath.Join(absDirPath, "testdata/bufyaml/one/two"))
	require.NoError(t, err)
	readBucketCloser, err := getReadBucketCloserForBucket(
		ctx,
		zap.NewNop(),
		storage.NopReadBucketCloser(inputBucket),
		"three/four/five",
		testNewTerminateAtFileNamesFunc("buf.work.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readBucketCloser.SubDirPath())
	_, err = readBucketCloser.Stat(ctx, "four/five/buf.yaml")
	require.NoError(t, err)
	require.NoError(t, readBucketCloser.Close())
}

func TestGetReadWriteBucketForOSNoTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	readWriteBucket, err := getReadWriteBucketForOS(
		ctx,
		zap.NewNop(),
		storageos.NewProvider(),
		"testdata/bufyaml/one/two/three/four/five",
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, ".", readWriteBucket.SubDirPath())
	fileInfo, err := readWriteBucket.Stat(ctx, "buf.yaml")
	require.NoError(t, err)
	require.Equal(t, "testdata/bufyaml/one/two/three/four/five/buf.yaml", filepath.ToSlash(fileInfo.ExternalPath()))
}

func TestGetReadWriteBucketForOSTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	readWriteBucket, err := getReadWriteBucketForOS(
		ctx,
		zap.NewNop(),
		storageos.NewProvider(),
		"testdata/bufyaml/one/two/three/four/five",
		testNewTerminateAtFileNamesFunc("buf.work.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readWriteBucket.SubDirPath())
	fileInfo, err := readWriteBucket.Stat(ctx, "four/five/buf.yaml")
	require.NoError(t, err)
	require.Equal(t, "testdata/bufyaml/one/two/three/four/five/buf.yaml", filepath.ToSlash(fileInfo.ExternalPath()))
	fileInfo, err = readWriteBucket.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, "testdata/bufyaml/one/two/three/buf.work.yaml", filepath.ToSlash(fileInfo.ExternalPath()))
}

func TestGetReadWriteBucketForOSParentPwd(t *testing.T) {
	// Cannot be parallel since we chdir.

	ctx := context.Background()
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	require.NoError(t, osext.Chdir(normalpath.Unnormalize(normalpath.Join(pwd, "testdata/bufyaml/one/two/three/four"))))
	defer func() {
		r := recover()
		assert.NoError(t, osext.Chdir(pwd))
		if r != nil {
			panic(r)
		}
	}()
	readWriteBucket, err := getReadWriteBucketForOS(
		ctx,
		zap.NewNop(),
		storageos.NewProvider(),
		"five",
		testNewTerminateAtFileNamesFunc("buf.work.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readWriteBucket.SubDirPath())
	fileInfo, err := readWriteBucket.Stat(ctx, "four/five/buf.yaml")
	require.NoError(t, err)
	require.Equal(t, "five/buf.yaml", filepath.ToSlash(fileInfo.ExternalPath()))
	fileInfo, err = readWriteBucket.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, "../buf.work.yaml", filepath.ToSlash(fileInfo.ExternalPath()))
}

func TestGetReadWriteBucketForOSAbsPwd(t *testing.T) {
	// Cannot be parallel since we chdir.

	ctx := context.Background()
	absDirPath, err := filepath.Abs(".")
	require.NoError(t, err)
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	require.NoError(t, osext.Chdir(normalpath.Unnormalize(normalpath.Join(pwd, "testdata/bufyaml/one/two/three/four"))))
	defer func() {
		r := recover()
		assert.NoError(t, osext.Chdir(pwd))
		if r != nil {
			panic(r)
		}
	}()
	readWriteBucket, err := getReadWriteBucketForOS(
		ctx,
		zap.NewNop(),
		storageos.NewProvider(),
		normalpath.Join(absDirPath, "testdata/bufyaml/one/two/three/four/five"),
		testNewTerminateAtFileNamesFunc("buf.work.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readWriteBucket.SubDirPath())
	fileInfo, err := readWriteBucket.Stat(ctx, "four/five/buf.yaml")
	require.NoError(t, err)
	require.Equal(t, normalpath.Join(absDirPath, "testdata/bufyaml/one/two/three/four/five/buf.yaml"), filepath.ToSlash(fileInfo.ExternalPath()))
	fileInfo, err = readWriteBucket.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, normalpath.Join(absDirPath, "testdata/bufyaml/one/two/three/buf.work.yaml"), filepath.ToSlash(fileInfo.ExternalPath()))
}

func TestGetReadBucketCloserForOSProtoFileNoWorkspaceTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	readBucketCloser, err := getReadBucketCloserForOSProtoFile(
		ctx,
		zap.NewNop(),
		storageos.NewProvider(),
		"testdata/bufyaml/one/two/three/four/five/proto/foo.proto",
		nil,
		testNewTerminateAtFileNamesFunc("buf.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, ".", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "buf.yaml")
	require.NoError(t, err)
	require.Equal(t, "testdata/bufyaml/one/two/three/four/five/buf.yaml", filepath.ToSlash(fileInfo.ExternalPath()))
	require.NoError(t, readBucketCloser.Close())
}

func TestGetReadBucketCloserForOSProtoFileTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	readBucketCloser, err := getReadBucketCloserForOSProtoFile(
		ctx,
		zap.NewNop(),
		storageos.NewProvider(),
		"testdata/bufyaml/one/two/three/four/five/proto/foo.proto",
		testNewTerminateAtFileNamesFunc("buf.work.yaml"),
		testNewTerminateAtFileNamesFunc("buf.yaml", "buf.work.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "four/five/buf.yaml")
	require.NoError(t, err)
	require.Equal(t, "testdata/bufyaml/one/two/three/four/five/buf.yaml", filepath.ToSlash(fileInfo.ExternalPath()))
	fileInfo, err = readBucketCloser.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, "testdata/bufyaml/one/two/three/buf.work.yaml", filepath.ToSlash(fileInfo.ExternalPath()))
	require.NoError(t, readBucketCloser.Close())
}

func TestGetReadBucketCloserForOSProtoFileParentPwd(t *testing.T) {
	// Cannot be parallel since we chdir.

	ctx := context.Background()
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	require.NoError(t, osext.Chdir(normalpath.Unnormalize(normalpath.Join(pwd, "testdata/bufyaml/one/two/three/four"))))
	defer func() {
		r := recover()
		assert.NoError(t, osext.Chdir(pwd))
		if r != nil {
			panic(r)
		}
	}()
	readBucketCloser, err := getReadBucketCloserForOSProtoFile(
		ctx,
		zap.NewNop(),
		storageos.NewProvider(),
		"five/proto/foo.proto",
		testNewTerminateAtFileNamesFunc("buf.work.yaml"),
		testNewTerminateAtFileNamesFunc("buf.yaml", "buf.work.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "four/five/buf.yaml")
	require.NoError(t, err)
	require.Equal(t, "five/buf.yaml", filepath.ToSlash(fileInfo.ExternalPath()))
	fileInfo, err = readBucketCloser.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, "../buf.work.yaml", filepath.ToSlash(fileInfo.ExternalPath()))
	require.NoError(t, readBucketCloser.Close())
}

func TestGetReadBucketCloserForOSProtoFileAbsPwd(t *testing.T) {
	// Cannot be parallel since we chdir.

	ctx := context.Background()
	absDirPath, err := filepath.Abs(".")
	require.NoError(t, err)
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	require.NoError(t, osext.Chdir(normalpath.Unnormalize(normalpath.Join(pwd, "testdata/bufyaml/one/two/three/four"))))
	defer func() {
		r := recover()
		assert.NoError(t, osext.Chdir(pwd))
		if r != nil {
			panic(r)
		}
	}()
	readBucketCloser, err := getReadBucketCloserForOSProtoFile(
		ctx,
		zap.NewNop(),
		storageos.NewProvider(),
		normalpath.Join(absDirPath, "testdata/bufyaml/one/two/three/four/five/proto/foo.proto"),
		testNewTerminateAtFileNamesFunc("buf.work.yaml"),
		testNewTerminateAtFileNamesFunc("buf.yaml", "buf.work.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "four/five/buf.yaml")
	require.NoError(t, err)
	require.Equal(t, normalpath.Join(absDirPath, "testdata/bufyaml/one/two/three/four/five/buf.yaml"), filepath.ToSlash(fileInfo.ExternalPath()))
	fileInfo, err = readBucketCloser.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, normalpath.Join(absDirPath, "testdata/bufyaml/one/two/three/buf.work.yaml"), filepath.ToSlash(fileInfo.ExternalPath()))
	require.NoError(t, readBucketCloser.Close())
}

func TestGetReadBucketCloserForOSProtoFileNoBufYAMLTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	readBucketCloser, err := getReadBucketCloserForOSProtoFile(
		ctx,
		zap.NewNop(),
		storageos.NewProvider(),
		"testdata/nobufyaml/one/two/three/four/five/proto/foo.proto",
		testNewTerminateAtFileNamesFunc("buf.work.yaml"),
		testNewTerminateAtFileNamesFunc("buf.yaml", "buf.work.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, ".", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "four/five/proto/foo.proto")
	require.NoError(t, err)
	require.Equal(t, "testdata/nobufyaml/one/two/three/four/five/proto/foo.proto", filepath.ToSlash(fileInfo.ExternalPath()))
	require.NoError(t, readBucketCloser.Close())
}

func TestGetReadBucketCloserForOSProtoFileNoBufYAMLParentPwd(t *testing.T) {
	// Cannot be parallel since we chdir.

	ctx := context.Background()
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	require.NoError(t, osext.Chdir(normalpath.Unnormalize(normalpath.Join(pwd, "testdata/nobufyaml/one/two/three/four"))))
	defer func() {
		r := recover()
		assert.NoError(t, osext.Chdir(pwd))
		if r != nil {
			panic(r)
		}
	}()
	readBucketCloser, err := getReadBucketCloserForOSProtoFile(
		ctx,
		zap.NewNop(),
		storageos.NewProvider(),
		"five/proto/foo.proto",
		testNewTerminateAtFileNamesFunc("buf.work.yaml"),
		testNewTerminateAtFileNamesFunc("buf.yaml", "buf.work.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, ".", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "four/five/proto/foo.proto")
	require.NoError(t, err)
	require.Equal(t, "five/proto/foo.proto", filepath.ToSlash(fileInfo.ExternalPath()))
	fileInfo, err = readBucketCloser.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, "../buf.work.yaml", filepath.ToSlash(fileInfo.ExternalPath()))
	require.NoError(t, readBucketCloser.Close())
}

func TestGetReadBucketCloserForOSProtoFileNoBufYAMLAbsPwd(t *testing.T) {
	// Cannot be parallel since we chdir.
	t.Skip()

	ctx := context.Background()
	absDirPath, err := filepath.Abs(".")
	require.NoError(t, err)
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	require.NoError(t, osext.Chdir(normalpath.Unnormalize(normalpath.Join(pwd, "testdata/nobufyaml/one/two/three/four"))))
	defer func() {
		r := recover()
		assert.NoError(t, osext.Chdir(pwd))
		if r != nil {
			panic(r)
		}
	}()
	readBucketCloser, err := getReadBucketCloserForOSProtoFile(
		ctx,
		zap.NewNop(),
		storageos.NewProvider(),
		normalpath.Join(absDirPath, "testdata/nobufyaml/one/two/three/four/five/proto/foo.proto"),
		testNewTerminateAtFileNamesFunc("buf.work.yaml"),
		testNewTerminateAtFileNamesFunc("buf.yaml", "buf.work.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "four/five/proto/foo.proto")
	require.NoError(t, err)
	require.Equal(t, normalpath.Join(absDirPath, "testdata/nobufyaml/one/two/three/four/five/proto/foo.proto"), fileInfo.ExternalPath())
	fileInfo, err = readBucketCloser.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, normalpath.Join(absDirPath, "testdata/nobufyaml/one/two/three/buf.work.yaml"), fileInfo.ExternalPath())
	require.NoError(t, readBucketCloser.Close())
}

func testNewTerminateAtFileNamesFunc(terminateFileNames ...string) buftarget.TerminateFunc {
	return buftarget.TerminateFunc(
		func(
			ctx context.Context,
			bucket storage.ReadBucket,
			prefix string,
			originalSubDirPath string,
		) (bool, error) {
			for _, terminateFileName := range terminateFileNames {
				if _, err := bucket.Stat(ctx, normalpath.Join(prefix, terminateFileName)); err == nil {
					return true, nil
				}
			}
			return false, nil
		},
	)
}
