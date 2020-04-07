package storagetesting

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagegit"
	"github.com/bufbuild/buf/internal/pkg/storage/storagegit/storagegitplumbing"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/bufbuild/buf/internal/pkg/storage/storageutil"
	"github.com/bufbuild/buf/internal/pkg/util/utilstring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testProtoContent = `syntax = "proto3";

package foo;
`
	testTxtContent = `foo
`
)

func TestBasic1(t *testing.T) {
	testBasic(
		t,
		"testdata",
		"",
		map[string]string{
			"one/a/b/1.proto": testProtoContent,
			"one/a/b/2.proto": testProtoContent,
			"one/a/b/2.txt":   "",
			"one/ab/1.proto":  testProtoContent,
			"one/ab/2.proto":  testProtoContent,
			"one/ab/2.txt":    "",
			"one/a/1.proto":   "",
			"one/a/1.txt":     testTxtContent,
			"one/a/bar.yaml":  "",
			"one/c/1.proto":   testProtoContent,
			"one/1.proto":     testProtoContent,
			"one/foo.yaml":    "",
		},
	)
}

func TestBasic2(t *testing.T) {
	testBasic(
		t,
		"testdata",
		".",
		map[string]string{
			"one/a/b/1.proto": testProtoContent,
			"one/a/b/2.proto": testProtoContent,
			"one/a/b/2.txt":   "",
			"one/ab/1.proto":  testProtoContent,
			"one/ab/2.proto":  testProtoContent,
			"one/ab/2.txt":    "",
			"one/a/1.proto":   "",
			"one/a/1.txt":     testTxtContent,
			"one/a/bar.yaml":  "",
			"one/c/1.proto":   testProtoContent,
			"one/1.proto":     testProtoContent,
			"one/foo.yaml":    "",
		},
	)
}

func TestBasic3(t *testing.T) {
	testBasic(
		t,
		"testdata",
		"./",
		map[string]string{
			"one/a/b/1.proto": testProtoContent,
			"one/a/b/2.proto": testProtoContent,
			"one/a/b/2.txt":   "",
			"one/ab/1.proto":  testProtoContent,
			"one/ab/2.proto":  testProtoContent,
			"one/ab/2.txt":    "",
			"one/a/1.proto":   "",
			"one/a/bar.yaml":  "",
			"one/a/1.txt":     testTxtContent,
			"one/c/1.proto":   testProtoContent,
			"one/1.proto":     testProtoContent,
			"one/foo.yaml":    "",
		},
	)
}

func TestBasic4(t *testing.T) {
	testBasic(
		t,
		"testdata",
		"",
		map[string]string{
			"one/a/b/1.proto": testProtoContent,
			"one/a/b/2.proto": testProtoContent,
			"one/ab/1.proto":  testProtoContent,
			"one/ab/2.proto":  testProtoContent,
			"one/a/1.proto":   "",
			"one/c/1.proto":   testProtoContent,
			"one/1.proto":     testProtoContent,
			"one/foo.yaml":    "",
		},
		storagepath.WithExt(".proto"),
		storagepath.WithExactPath("one/foo.yaml"),
	)
}

func TestBasic5(t *testing.T) {
	testBasic(
		t,
		"testdata",
		"one/a",
		map[string]string{
			"one/a/b/1.proto": testProtoContent,
			"one/a/b/2.proto": testProtoContent,
			"one/a/1.proto":   "",
		},
		storagepath.WithExt(".proto"),
		storagepath.WithExactPath("foo.yaml"),
	)
}

func TestBasic6(t *testing.T) {
	testBasic(
		t,
		"testdata",
		"./one/a",
		map[string]string{
			"one/a/b/1.proto": testProtoContent,
			"one/a/b/2.proto": testProtoContent,
			"one/a/1.proto":   "",
		},
		storagepath.WithExt(".proto"),
		storagepath.WithExactPath("foo.yaml"),
	)
}

func TestBasic7(t *testing.T) {
	testBasic(
		t,
		"testdata",
		"",
		map[string]string{
			"a/b/1.proto": testProtoContent,
			"a/b/2.proto": testProtoContent,
			"ab/1.proto":  testProtoContent,
			"ab/2.proto":  testProtoContent,
			"a/1.proto":   "",
			"c/1.proto":   testProtoContent,
			"1.proto":     testProtoContent,
			"a/bar.yaml":  "",
		},
		storagepath.WithExt(".proto"),
		storagepath.WithExactPath("a/bar.yaml"),
		storagepath.WithStripComponents(1),
	)
}

func TestGitClone(t *testing.T) {
	testGitClone(t, false)
}

func TestGitCloneExperimental(t *testing.T) {
	testGitClone(t, true)
}

func testGitClone(t *testing.T, experimental bool) {
	t.Parallel()
	absGitPath, err := filepath.Abs("../../../../../.git")
	require.NoError(t, err)
	_, err = os.Stat(absGitPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("no .git repository")
			return
		}
		require.NoError(t, err)
	}

	absFilePathSuccess1, err := filepath.Abs("storagetesting.go")
	require.NoError(t, err)
	relFilePathSuccess1, err := filepath.Rel(filepath.Dir(absGitPath), absFilePathSuccess1)
	require.NoError(t, err)
	absFilePathSuccess2, err := filepath.Abs("testdata/one/1.proto")
	require.NoError(t, err)
	relFilePathSuccess2, err := filepath.Rel(filepath.Dir(absGitPath), absFilePathSuccess2)
	require.NoError(t, err)
	relFilePathError1 := "Makefile"

	readWriteBucketCloser := storagemem.NewReadWriteBucketCloser()
	if experimental {
		err = storagegit.ExperimentalClone(
			context.Background(),
			zap.NewNop(),
			nil,
			absGitPath,
			"master",
			"",
			false,
			readWriteBucketCloser,
			storagepath.WithExt(".proto"),
			storagepath.WithExt(".go"),
		)
	} else {
		err = storagegit.Clone(
			context.Background(),
			zap.NewNop(),
			nil,
			"",
			absGitPath,
			storagegitplumbing.NewBranchRefName("master"),
			false,
			"",
			"",
			"",
			"",
			"",
			readWriteBucketCloser,
			storagepath.WithExt(".proto"),
			storagepath.WithExt(".go"),
		)
	}
	assert.NoError(t, err)

	_, err = readWriteBucketCloser.Stat(context.Background(), relFilePathSuccess1)
	assert.NoError(t, err)
	_, err = readWriteBucketCloser.Stat(context.Background(), relFilePathSuccess2)
	assert.NoError(t, err)
	_, err = readWriteBucketCloser.Stat(context.Background(), relFilePathError1)
	assert.True(t, storage.IsNotExist(err))

	assert.NoError(t, readWriteBucketCloser.Close())
}

func testBasic(
	t *testing.T,
	dirPath string,
	walkPrefix string,
	expectedPathToContent map[string]string,
	transformerOptions ...storagepath.TransformerOption,
) {
	t.Parallel()
	t.Run("static", func(t *testing.T) {
		testBasicStatic(
			t,
			walkPrefix,
			expectedPathToContent,
			transformerOptions...,
		)
	})
	t.Run("mem", func(t *testing.T) {
		t.Parallel()
		testBasicMem(
			t,
			dirPath,
			false,
			walkPrefix,
			expectedPathToContent,
			transformerOptions...,
		)
	})
	t.Run("os", func(t *testing.T) {
		t.Parallel()
		testBasicOS(
			t,
			dirPath,
			false,
			walkPrefix,
			expectedPathToContent,
			transformerOptions...,
		)
	})
	t.Run("mem-tar", func(t *testing.T) {
		t.Parallel()
		testBasicMem(
			t,
			dirPath,
			true,
			walkPrefix,
			expectedPathToContent,
			transformerOptions...,
		)
	})
	t.Run("os-tar", func(t *testing.T) {
		t.Parallel()
		testBasicOS(
			t,
			dirPath,
			true,
			walkPrefix,
			expectedPathToContent,
			transformerOptions...,
		)
	})
}

func testBasicStatic(
	t *testing.T,
	walkPrefix string,
	expectedPathToContent map[string]string,
	transformerOptions ...storagepath.TransformerOption,
) {
	pathToData := make(map[string][]byte)
	for path, content := range expectedPathToContent {
		pathToData[path] = []byte(content)
	}
	readBucket, err := storagemem.NewImmutableReadBucket(pathToData)
	require.NoError(t, err)
	assertExpectedPathToContent(t, readBucket, walkPrefix, expectedPathToContent)
}

func testBasicMem(
	t *testing.T,
	dirPath string,
	doAsTar bool,
	walkPrefix string,
	expectedPathToContent map[string]string,
	transformerOptions ...storagepath.TransformerOption,
) {
	readWriteBucketCloser := storagemem.NewReadWriteBucketCloser()
	testBasicBucket(
		t,
		readWriteBucketCloser,
		dirPath,
		doAsTar,
		walkPrefix,
		expectedPathToContent,
		transformerOptions...,
	)
}

func testBasicOS(
	t *testing.T,
	dirPath string,
	doAsTar bool,
	walkPrefix string,
	expectedPathToContent map[string]string,
	transformerOptions ...storagepath.TransformerOption,
) {
	tempDirPath, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	require.NotEmpty(t, tempDirPath)
	defer func() {
		// won't work with requires but just temporary directory
		require.NoError(t, os.RemoveAll(tempDirPath))
	}()
	readWriteBucketCloser, err := storageos.NewReadWriteBucketCloser(tempDirPath)
	require.NoError(t, err)
	testBasicBucket(
		t,
		readWriteBucketCloser,
		dirPath,
		doAsTar,
		walkPrefix,
		expectedPathToContent,
		transformerOptions...,
	)
}

func testBasicBucket(
	t *testing.T,
	readWriteBucketCloser storage.ReadWriteBucketCloser,
	dirPath string,
	doAsTar bool,
	walkPrefix string,
	expectedPathToContent map[string]string,
	transformerOptions ...storagepath.TransformerOption,
) {
	inputReadWriteBucketCloser, err := storageos.NewReadWriteBucketCloser(dirPath)
	require.NoError(t, err)
	if doAsTar {
		buffer := bytes.NewBuffer(nil)
		require.NoError(t, storageutil.Targz(
			context.Background(),
			buffer,
			inputReadWriteBucketCloser,
			"",
		))
		require.NoError(t, err)
		require.NoError(t, storageutil.Untargz(
			context.Background(),
			buffer,
			readWriteBucketCloser,
			transformerOptions...,
		))
	} else {
		_, err := storageutil.Copy(
			context.Background(),
			inputReadWriteBucketCloser,
			readWriteBucketCloser,
			"",
			transformerOptions...,
		)
		require.NoError(t, err)
	}
	require.NoError(t, inputReadWriteBucketCloser.Close())
	assertExpectedPathToContent(t, readWriteBucketCloser, walkPrefix, expectedPathToContent)
	assert.NoError(t, readWriteBucketCloser.Close())
}

func assertExpectedPathToContent(
	t *testing.T,
	readBucket storage.ReadBucket,
	walkPrefix string,
	expectedPathToContent map[string]string,
) {
	var paths []string
	require.NoError(t, readBucket.Walk(
		context.Background(),
		walkPrefix,
		func(path string) error {
			paths = append(paths, path)
			return nil
		},
	))
	require.Equal(t, len(paths), len(utilstring.SliceToUniqueSortedSlice(paths)))
	assert.Equal(t, len(expectedPathToContent), len(paths), paths)
	for _, path := range paths {
		expectedContent, ok := expectedPathToContent[path]
		assert.True(t, ok, path)
		expectedSize := len(expectedContent)
		objectInfo, err := readBucket.Stat(context.Background(), path)
		assert.NoError(t, err, path)
		// weird issue with int vs uint64
		if expectedSize == 0 {
			assert.Equal(t, 0, int(objectInfo.Size()), path)
		} else {
			assert.Equal(t, expectedSize, int(objectInfo.Size()), path)
		}
		readerCloser, err := readBucket.Get(context.Background(), path)
		assert.NoError(t, err, path)
		data, err := ioutil.ReadAll(readerCloser)
		assert.NoError(t, err, path)
		assert.NoError(t, readerCloser.Close())
		assert.Equal(t, expectedContent, string(data))
	}
}
