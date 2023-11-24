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

func TestGetReadBucketCloserForOSNoTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	readBucketCloser, err := getReadBucketCloserForOS(
		ctx,
		storageos.NewProvider(),
		"testdata/bufyaml/one/two/three/four/five",
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, ".", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "buf.yaml")
	require.NoError(t, err)
	require.Equal(t, "testdata/bufyaml/one/two/three/four/five/buf.yaml", fileInfo.ExternalPath())
	require.NoError(t, readBucketCloser.Close())
}

func TestGetReadBucketCloserForOSTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	readBucketCloser, err := getReadBucketCloserForOS(
		ctx,
		storageos.NewProvider(),
		"testdata/bufyaml/one/two/three/four/five",
		NewTerminateAtFileNamesFunc("buf.work.yaml"),
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

func TestGetReadBucketCloserForOSParentPwd(t *testing.T) {
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
	readBucketCloser, err := getReadBucketCloserForOS(
		ctx,
		storageos.NewProvider(),
		"five",
		NewTerminateAtFileNamesFunc("buf.work.yaml"),
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

func TestGetReadBucketCloserForOSAbsPwd(t *testing.T) {
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
	readBucketCloser, err := getReadBucketCloserForOS(
		ctx,
		storageos.NewProvider(),
		normalpath.Join(absDirPath, "testdata/bufyaml/one/two/three/four/five"),
		NewTerminateAtFileNamesFunc("buf.work.yaml"),
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

func TestGetReadBucketCloserForOSProtoFileNoBufYAMLNoTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	readBucketCloser, err := getReadBucketCloserForOSProtoFile(
		ctx,
		storageos.NewProvider(),
		"testdata/nobufyaml/one/two/three/four/five/proto/foo.proto",
		nil,
		NewTerminateAtFileNamesFunc("buf.yaml"),
	)
	require.NoError(t, err)
	require.Equal(t, ".", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "testdata/nobufyaml/one/two/three/four/five/proto/foo.proto")
	require.NoError(t, err)
	require.Equal(t, "testdata/nobufyaml/one/two/three/four/five/proto/foo.proto", fileInfo.ExternalPath())
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
