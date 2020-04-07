package bufbuild

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewProtoFileSet1(t *testing.T) {
	testNewProtoFileSet(
		t,
		"testdata/1",
		[]string{
			"proto",
		},
		[]string{
			"proto/b",
		},
		[]string{
			"proto/a/1.proto",
			"proto/a/2.proto",
			"proto/a/3.proto",
			"proto/a/c/1.proto",
			"proto/a/c/2.proto",
			"proto/a/c/3.proto",
			"proto/d/1.proto",
			"proto/d/2.proto",
			"proto/d/3.proto",
		},
	)
}

func TestNewProtoFileSet2(t *testing.T) {
	testNewProtoFileSet(
		t,
		"testdata/1",
		[]string{
			"proto",
		},
		[]string{
			"proto/b",
		},
		[]string{
			"proto/a/1.proto",
			"proto/a/2.proto",
			"proto/a/3.proto",
			"proto/a/c/1.proto",
			"proto/a/c/2.proto",
			"proto/a/c/3.proto",
			"proto/d/1.proto",
			"proto/d/2.proto",
			"proto/d/3.proto",
		},
	)
}

func TestNewProtoFileSet3(t *testing.T) {
	testNewProtoFileSet(
		t,
		"testdata/1",
		[]string{
			"proto",
		},
		[]string{
			"proto/a",
		},
		[]string{
			"proto/b/1.proto",
			"proto/b/2.proto",
			"proto/b/3.proto",
			"proto/d/1.proto",
			"proto/d/2.proto",
			"proto/d/3.proto",
		},
	)
}

func TestNewProtoFileSet4(t *testing.T) {
	testNewProtoFileSet(
		t,
		"testdata/1",
		[]string{
			"proto",
		},
		[]string{
			"proto/a/c",
		},
		[]string{
			"proto/a/1.proto",
			"proto/a/2.proto",
			"proto/a/3.proto",
			"proto/b/1.proto",
			"proto/b/2.proto",
			"proto/b/3.proto",
			"proto/d/1.proto",
			"proto/d/2.proto",
			"proto/d/3.proto",
		},
	)
}

func TestNewProtoFileSet5(t *testing.T) {
	t.Parallel()
	testNewProtoFileSetErrorReadBucket(
		t,
		"testdata/1",
		[]string{
			"proto",
		},
		[]string{
			"proto/a/c",
			// error
			"proto/d/1.proto",
		},
	)
}
func TestNewProtoFileSet6(t *testing.T) {
	testNewProtoFileSet(
		t,
		"testdata/1",
		[]string{
			"proto",
		},
		[]string{
			"proto/a/c",
			"proto/d",
		},
		[]string{
			"proto/a/1.proto",
			"proto/a/2.proto",
			"proto/a/3.proto",
			"proto/b/1.proto",
			"proto/b/2.proto",
			"proto/b/3.proto",
		},
	)
}

func TestNewProtoFileSetError1(t *testing.T) {
	testNewProtoFileSetError(
		t,
		"testdata/2",
		[]string{
			"a",
			"b",
		},
		[]string{},
		[]string{
			"a/1.proto",
			"a/2.proto",
			"a/3.proto",
			"b/1.proto",
			"b/4.proto",
			"b/5.proto",
		},
	)
}

func testNewProtoFileSet(
	t *testing.T,
	relDir string,
	relRoots []string,
	relExcludes []string,
	expectedRelFiles []string,
) {
	t.Parallel()
	readWriteBucketCloser, err := storageos.NewReadWriteBucketCloser(relDir)
	require.NoError(t, err)

	protoFileSet, err := newProtoFileSetProvider(zap.NewNop()).GetProtoFileSetForReadBucket(
		context.Background(),
		readWriteBucketCloser,
		relRoots,
		relExcludes,
	)
	assert.NoError(t, err)
	assert.NotNil(t, protoFileSet)
	if protoFileSet != nil {
		assert.Equal(
			t,
			expectedRelFiles,
			protoFileSet.RealFilePaths(),
		)
	}
	if len(expectedRelFiles) > 1 {
		expectedRelFiles = expectedRelFiles[:len(expectedRelFiles)-1]
		protoFileSet, err := newProtoFileSetProvider(zap.NewNop()).GetProtoFileSetForRealFilePaths(
			context.Background(),
			readWriteBucketCloser,
			relRoots,
			expectedRelFiles,
			false,
		)
		assert.NoError(t, err)
		assert.NotNil(t, protoFileSet)
		if protoFileSet != nil {
			assert.Equal(
				t,
				expectedRelFiles,
				protoFileSet.RealFilePaths(),
			)
		}
	}
	assert.NoError(t, readWriteBucketCloser.Close())
}

func testNewProtoFileSetError(
	t *testing.T,
	relDir string,
	relRoots []string,
	relExcludes []string,
	allRelFiles []string,
) {
	t.Parallel()
	testNewProtoFileSetErrorReadBucket(t, relDir, relRoots, relExcludes)
	if len(allRelFiles) > 1 {
		testNewProtoFileSetErrorRealFilePaths(t, relDir, relRoots, relExcludes, allRelFiles)
	}
}

func testNewProtoFileSetErrorReadBucket(
	t *testing.T,
	relDir string,
	relRoots []string,
	relExcludes []string,
) {
	readWriteBucketCloser, err := storageos.NewReadWriteBucketCloser(relDir)
	require.NoError(t, err)
	_, err = newProtoFileSetProvider(zap.NewNop()).GetProtoFileSetForReadBucket(
		context.Background(),
		readWriteBucketCloser,
		relRoots,
		relExcludes,
	)
	assert.Error(t, err)
	assert.NoError(t, readWriteBucketCloser.Close())
}

func testNewProtoFileSetErrorRealFilePaths(
	t *testing.T,
	relDir string,
	relRoots []string,
	relExcludes []string,
	allRelFiles []string,
) {
	readWriteBucketCloser, err := storageos.NewReadWriteBucketCloser(relDir)
	require.NoError(t, err)
	allRelFiles = allRelFiles[:len(allRelFiles)-1]
	_, err = newProtoFileSetProvider(zap.NewNop()).GetProtoFileSetForRealFilePaths(
		context.Background(),
		readWriteBucketCloser,
		relRoots,
		allRelFiles,
		false,
	)
	assert.Error(t, err)
	assert.NoError(t, readWriteBucketCloser.Close())
}
