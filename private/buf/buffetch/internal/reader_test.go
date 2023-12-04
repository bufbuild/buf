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

package internal

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/osext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetReadBucketCloserForBucketNoTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	inputBucket, err := storageos.NewProvider().NewReadWriteBucket("testdata/bufyaml/one/two")
	require.NoError(t, err)
	readBucketCloser, err := getReadBucketCloserForBucket(
		ctx,
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
		storage.NopReadBucketCloser(inputBucket),
		"three/four/five",
		NewTerminateAtFileNamesFunc("buf.work.yaml"),
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
		storage.NopReadBucketCloser(inputBucket),
		"three/four/five",
		NewTerminateAtFileNamesFunc("buf.work.yaml"),
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
		storageos.NewProvider(),
		"testdata/bufyaml/one/two/three/four/five",
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, ".", readWriteBucket.SubDirPath())
	fileInfo, err := readWriteBucket.Stat(ctx, "buf.yaml")
	require.NoError(t, err)
	require.Equal(t, "testdata/bufyaml/one/two/three/four/five/buf.yaml", fileInfo.ExternalPath())
}

func TestGetReadWriteBucketForOSTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	readWriteBucket, err := getReadWriteBucketForOS(
		ctx,
		storageos.NewProvider(),
		"testdata/bufyaml/one/two/three/four/five",
		NewTerminateAtFileNamesFunc("buf.work.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readWriteBucket.SubDirPath())
	fileInfo, err := readWriteBucket.Stat(ctx, "four/five/buf.yaml")
	require.NoError(t, err)
	require.Equal(t, "testdata/bufyaml/one/two/three/four/five/buf.yaml", fileInfo.ExternalPath())
	fileInfo, err = readWriteBucket.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, "testdata/bufyaml/one/two/three/buf.work.yaml", fileInfo.ExternalPath())
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
		storageos.NewProvider(),
		"five",
		NewTerminateAtFileNamesFunc("buf.work.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readWriteBucket.SubDirPath())
	fileInfo, err := readWriteBucket.Stat(ctx, "four/five/buf.yaml")
	require.NoError(t, err)
	require.Equal(t, "five/buf.yaml", fileInfo.ExternalPath())
	fileInfo, err = readWriteBucket.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, "../buf.work.yaml", fileInfo.ExternalPath())
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
		storageos.NewProvider(),
		normalpath.Join(absDirPath, "testdata/bufyaml/one/two/three/four/five"),
		NewTerminateAtFileNamesFunc("buf.work.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readWriteBucket.SubDirPath())
	fileInfo, err := readWriteBucket.Stat(ctx, "four/five/buf.yaml")
	require.NoError(t, err)
	require.Equal(t, normalpath.Join(absDirPath, "testdata/bufyaml/one/two/three/four/five/buf.yaml"), fileInfo.ExternalPath())
	fileInfo, err = readWriteBucket.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, normalpath.Join(absDirPath, "testdata/bufyaml/one/two/three/buf.work.yaml"), fileInfo.ExternalPath())
}

func TestGetReadBucketCloserForOSProtoFileNoTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	readBucketCloser, err := getReadBucketCloserForOSProtoFile(
		ctx,
		storageos.NewProvider(),
		"testdata/bufyaml/one/two/three/four/five/proto/foo.proto",
		nil,
		NewTerminateAtFileNamesFunc("buf.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, ".", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "buf.yaml")
	require.NoError(t, err)
	require.Equal(t, "testdata/bufyaml/one/two/three/four/five/buf.yaml", fileInfo.ExternalPath())
	require.NoError(t, readBucketCloser.Close())
}

func TestGetReadBucketCloserForOSProtoFileTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	readBucketCloser, err := getReadBucketCloserForOSProtoFile(
		ctx,
		storageos.NewProvider(),
		"testdata/bufyaml/one/two/three/four/five/proto/foo.proto",
		NewTerminateAtFileNamesFunc("buf.work.yaml"),
		NewTerminateAtFileNamesFunc("buf.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "four/five/buf.yaml")
	require.NoError(t, err)
	require.Equal(t, "testdata/bufyaml/one/two/three/four/five/buf.yaml", fileInfo.ExternalPath())
	fileInfo, err = readBucketCloser.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, "testdata/bufyaml/one/two/three/buf.work.yaml", fileInfo.ExternalPath())
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
		storageos.NewProvider(),
		"five/proto/foo.proto",
		NewTerminateAtFileNamesFunc("buf.work.yaml"),
		NewTerminateAtFileNamesFunc("buf.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "four/five/buf.yaml")
	require.NoError(t, err)
	require.Equal(t, "five/buf.yaml", fileInfo.ExternalPath())
	fileInfo, err = readBucketCloser.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, "../buf.work.yaml", fileInfo.ExternalPath())
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
		storageos.NewProvider(),
		normalpath.Join(absDirPath, "testdata/bufyaml/one/two/three/four/five/proto/foo.proto"),
		NewTerminateAtFileNamesFunc("buf.work.yaml"),
		NewTerminateAtFileNamesFunc("buf.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "four/five/buf.yaml")
	require.NoError(t, err)
	require.Equal(t, normalpath.Join(absDirPath, "testdata/bufyaml/one/two/three/four/five/buf.yaml"), fileInfo.ExternalPath())
	fileInfo, err = readBucketCloser.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, normalpath.Join(absDirPath, "testdata/bufyaml/one/two/three/buf.work.yaml"), fileInfo.ExternalPath())
	require.NoError(t, readBucketCloser.Close())
}

func TestGetReadBucketCloserForOSProtoFileNoBufYAMLTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	readBucketCloser, err := getReadBucketCloserForOSProtoFile(
		ctx,
		storageos.NewProvider(),
		"testdata/nobufyaml/one/two/three/four/five/proto/foo.proto",
		NewTerminateAtFileNamesFunc("buf.work.yaml"),
		NewTerminateAtFileNamesFunc("buf.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, ".", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "four/five/proto/foo.proto")
	require.NoError(t, err)
	require.Equal(t, "testdata/nobufyaml/one/two/three/four/five/proto/foo.proto", fileInfo.ExternalPath())
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
		storageos.NewProvider(),
		"five/proto/foo.proto",
		NewTerminateAtFileNamesFunc("buf.work.yaml"),
		NewTerminateAtFileNamesFunc("buf.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, ".", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "four/five/proto/foo.proto")
	require.NoError(t, err)
	require.Equal(t, "five/proto/foo.proto", fileInfo.ExternalPath())
	fileInfo, err = readBucketCloser.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, "../buf.work.yaml", fileInfo.ExternalPath())
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
		storageos.NewProvider(),
		normalpath.Join(absDirPath, "testdata/nobufyaml/one/two/three/four/five/proto/foo.proto"),
		NewTerminateAtFileNamesFunc("buf.work.yaml"),
		NewTerminateAtFileNamesFunc("buf.yaml"),
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
