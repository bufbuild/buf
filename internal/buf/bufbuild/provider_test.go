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
		"proto",
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
		"proto",
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
		"proto",
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
		"proto",
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
	testNewProtoFileSet(
		t,
		"testdata/1",
		"proto",
		[]string{
			"proto/a/c",
			// will not result in anything excluded as we do storagepath.Dir on the input file
			"proto/d/1.proto",
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
func TestNewProtoFileSet6(t *testing.T) {
	testNewProtoFileSet(
		t,
		"testdata/1",
		"proto",
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
	relRoot string,
	relExcludes []string,
	expectedRelFiles []string,
) {
	t.Parallel()
	bucket, err := storageos.NewReadBucket(relDir)
	require.NoError(t, err)

	set, err := NewProvider(zap.NewNop()).GetProtoFileSetForBucket(
		context.Background(),
		bucket,
		testNewConfig(
			t,
			[]string{relRoot},
			relExcludes,
		),
	)
	assert.NoError(t, err)
	assert.NotNil(t, set)
	assert.Equal(
		t,
		expectedRelFiles,
		set.RealFilePaths(),
	)
	if len(expectedRelFiles) > 1 {
		expectedRelFiles = expectedRelFiles[:len(expectedRelFiles)-1]
		set, err := NewProvider(zap.NewNop()).GetProtoFileSetForRealFilePaths(
			context.Background(),
			bucket,
			testNewConfig(
				t,
				[]string{relRoot},
				relExcludes,
			),
			expectedRelFiles,
			false,
		)
		assert.NoError(t, err)
		assert.NotNil(t, set)
		assert.Equal(
			t,
			expectedRelFiles,
			set.RealFilePaths(),
		)
	}
	assert.NoError(t, bucket.Close())
}

func testNewProtoFileSetError(
	t *testing.T,
	relDir string,
	relRoots []string,
	relExcludes []string,
	allRelFiles []string,
) {
	t.Parallel()
	bucket, err := storageos.NewReadBucket(relDir)
	require.NoError(t, err)

	_, err = NewProvider(zap.NewNop()).GetProtoFileSetForBucket(
		context.Background(),
		bucket,
		testNewConfig(
			t,
			relRoots,
			relExcludes,
		),
	)
	assert.Error(t, err)
	if len(allRelFiles) > 1 {
		allRelFiles = allRelFiles[:len(allRelFiles)-1]
		_, err = NewProvider(zap.NewNop()).GetProtoFileSetForRealFilePaths(
			context.Background(),
			bucket,
			testNewConfig(
				t,
				relRoots,
				relExcludes,
			),
			allRelFiles,
			false,
		)
		assert.Error(t, err)
	}
	assert.NoError(t, bucket.Close())
}
