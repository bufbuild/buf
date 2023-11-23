package internal

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/osext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

func TestGetReadBucketCloserForBucketNoTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	inputBucket, err := storageos.NewProvider().NewReadWriteBucket("testdata/one/two")
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
	inputBucket, err := storageos.NewProvider().NewReadWriteBucket("testdata/one/two")
	require.NoError(t, err)
	readBucketCloser, err := getReadBucketCloserForBucket(
		ctx,
		storage.NopReadBucketCloser(inputBucket),
		"three/four/five",
		[]string{"buf.work.yaml"},
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
	inputBucket, err := storageos.NewProvider().NewReadWriteBucket("testdata/one/two/three/four/five")
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
	inputBucket, err := storageos.NewProvider().NewReadWriteBucket(normalpath.Join(absDirPath, "testdata/one/two"))
	require.NoError(t, err)
	readBucketCloser, err := getReadBucketCloserForBucket(
		ctx,
		storage.NopReadBucketCloser(inputBucket),
		"three/four/five",
		[]string{"buf.work.yaml"},
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
		"testdata/one/two/three/four/five",
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, ".", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "buf.yaml")
	require.NoError(t, err)
	require.Equal(t, "testdata/one/two/three/four/five/buf.yaml", fileInfo.ExternalPath())
	require.NoError(t, readBucketCloser.Close())
}

func TestGetReadBucketCloserForOSTerminateFileName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	readBucketCloser, err := getReadBucketCloserForOS(
		ctx,
		storageos.NewProvider(),
		"testdata/one/two/three/four/five",
		[]string{"buf.work.yaml"},
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "four/five/buf.yaml")
	require.NoError(t, err)
	require.Equal(t, "testdata/one/two/three/four/five/buf.yaml", fileInfo.ExternalPath())
	fileInfo, err = readBucketCloser.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, "testdata/one/two/three/buf.work.yaml", fileInfo.ExternalPath())
	require.NoError(t, readBucketCloser.Close())
}

func TestGetReadBucketCloserForOSParentPwd(t *testing.T) {
	// Cannot be parallel since we chdir.

	ctx := context.Background()
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	require.NoError(t, osext.Chdir(normalpath.Unnormalize("./testdata/one/two/three/four")))
	readBucketCloser, err := getReadBucketCloserForOS(
		ctx,
		storageos.NewProvider(),
		"five",
		[]string{"buf.work.yaml"},
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
	require.NoError(t, osext.Chdir(pwd))
}

func TestGetReadBucketCloserForOSAbsPwd(t *testing.T) {
	// Cannot be parallel since we chdir.

	ctx := context.Background()
	absDirPath, err := filepath.Abs(".")
	require.NoError(t, err)
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	require.NoError(t, osext.Chdir(normalpath.Unnormalize("./testdata/one/two/three/four")))
	readBucketCloser, err := getReadBucketCloserForOS(
		ctx,
		storageos.NewProvider(),
		normalpath.Join(absDirPath, "testdata/one/two/three/four/five"),
		[]string{"buf.work.yaml"},
	)
	require.NoError(t, err)
	require.Equal(t, "four/five", readBucketCloser.SubDirPath())
	fileInfo, err := readBucketCloser.Stat(ctx, "four/five/buf.yaml")
	require.NoError(t, err)
	require.Equal(t, normalpath.Join(absDirPath, "testdata/one/two/three/four/five/buf.yaml"), fileInfo.ExternalPath())
	fileInfo, err = readBucketCloser.Stat(ctx, "buf.work.yaml")
	require.NoError(t, err)
	require.Equal(t, normalpath.Join(absDirPath, "testdata/one/two/three/buf.work.yaml"), fileInfo.ExternalPath())
	require.NoError(t, readBucketCloser.Close())
	require.NoError(t, osext.Chdir(pwd))
}
