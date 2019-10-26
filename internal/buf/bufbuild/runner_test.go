package bufbuild_test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufpb"
	"github.com/bufbuild/buf/internal/buf/buftesting"
	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/errs"
	"github.com/bufbuild/buf/internal/pkg/protodesc"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testGoogleapisCommit = "37c923effe8b002884466074f84bc4e78e6ade62"
)

var (
	testGoogleapisDirPath = filepath.Join("cache", "googleapis")
	testLock              sync.Mutex
)

func TestGoogleapis(t *testing.T) {
	t.Parallel()
	//for _, includeSourceInfo := range []bool{true, false} {
	for _, includeSourceInfo := range []bool{false} {
		t.Run(
			fmt.Sprintf("includeSourceInfo:%v", includeSourceInfo),
			func(t *testing.T) {
				t.Parallel()
				testBuildGoogleapis(t, includeSourceInfo)
			},
		)
	}
}

func TestProtocGoogleapis(t *testing.T) {
	t.Parallel()
	//for _, includeSourceInfo := range []bool{true, false} {
	for _, includeSourceInfo := range []bool{false} {
		t.Run(
			fmt.Sprintf("includeSourceInfo:%v", includeSourceInfo),
			func(t *testing.T) {
				t.Parallel()
				testBuildProtocGoogleapis(t, includeSourceInfo)
			},
		)
	}
}

func TestCompareGoogleapis(t *testing.T) {
	t.Parallel()
	//for _, includeSourceInfo := range []bool{true, false} {
	for _, includeSourceInfo := range []bool{false} {
		t.Run(
			fmt.Sprintf("includeSourceInfo:%v", includeSourceInfo),
			func(t *testing.T) {
				t.Parallel()
				image := testBuildGoogleapis(t, includeSourceInfo)
				protocImage := testBuildProtocGoogleapis(t, includeSourceInfo)
				assertImagesEqual(t, image, protocImage)
			},
		)
	}
}

func testBuildGoogleapis(t *testing.T, includeSourceInfo bool) bufpb.Image {
	bucket := testGetBucketGoogleapis(t)
	protoFileSet := testGetProtoFileSetGoogleapis(t, bucket)
	image, annotations := testBuild(t, includeSourceInfo, bucket, protoFileSet)
	assert.NoError(t, bucket.Close())

	assert.Equal(t, 0, len(annotations), annotations)
	assert.Equal(t, 1585, len(image.GetFile()))
	importNames, err := image.ImportNames()
	assert.NoError(t, err)
	assert.Equal(
		t,
		[]string{
			"google/protobuf/any.proto",
			"google/protobuf/api.proto",
			"google/protobuf/descriptor.proto",
			"google/protobuf/duration.proto",
			"google/protobuf/empty.proto",
			"google/protobuf/field_mask.proto",
			"google/protobuf/source_context.proto",
			"google/protobuf/struct.proto",
			"google/protobuf/timestamp.proto",
			"google/protobuf/type.proto",
			"google/protobuf/wrappers.proto",
		},
		importNames,
	)

	imageWithoutImports, err := image.WithoutImports()
	assert.NoError(t, err)
	assert.Equal(t, 1574, len(imageWithoutImports.GetFile()))
	importNames, err = imageWithoutImports.ImportNames()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(importNames))

	imageWithoutImports, err = imageWithoutImports.WithoutImports()
	assert.NoError(t, err)
	assert.Equal(t, 1574, len(imageWithoutImports.GetFile()))
	importNames, err = imageWithoutImports.ImportNames()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(importNames))

	imageWithSpecificNames, err := image.WithSpecificNames(
		true,
		"google/protobuf/descriptor.proto",
		"google/protobuf/api.proto",
		"google/../google/type/date.proto",
		"google/foo/nonsense.proto",
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(imageWithSpecificNames.GetFile()))
	_, err = image.WithSpecificNames(
		false,
		"google/protobuf/descriptor.proto",
		"google/protobuf/api.proto",
		"google/../google/type/date.proto",
		"google/foo/nonsense.proto",
	)
	assert.Equal(t, errs.NewInvalidArgument("google/foo/nonsense.proto is not present in the Image"), err)
	importNames, err = imageWithSpecificNames.ImportNames()
	assert.NoError(t, err)
	assert.Equal(
		t,
		[]string{
			"google/protobuf/api.proto",
			"google/protobuf/descriptor.proto",
		},
		importNames,
	)
	imageWithoutImports, err = imageWithSpecificNames.WithoutImports()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(imageWithoutImports.GetFile()))
	importNames, err = imageWithoutImports.ImportNames()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(importNames))

	assert.Equal(t, 1585, len(image.GetFile()))
	// basic check to make sure there is no error at this scale
	_, err = protodesc.NewFiles(image.GetFile()...)
	assert.NoError(t, err)
	return image
}

func testBuildProtocGoogleapis(t *testing.T, includeSourceInfo bool) bufpb.Image {
	bucket := testGetBucketGoogleapis(t)
	protoFileSet := testGetProtoFileSetGoogleapis(t, bucket)
	image := testBuildProtoc(t, includeSourceInfo, testGoogleapisDirPath, protoFileSet)
	assert.NoError(t, bucket.Close())
	assert.Equal(t, 1585, len(image.GetFile()))
	return image
}

func testGetBucketGoogleapis(t *testing.T) storage.ReadBucket {
	testGetGoogleapis(t)
	bucket, err := storageos.NewReadBucket(testGoogleapisDirPath)
	require.NoError(t, err)
	return bucket
}

func testGetProtoFileSetGoogleapis(t *testing.T, bucket storage.ReadBucket) bufbuild.ProtoFileSet {
	config, err := bufbuild.ConfigBuilder{}.NewConfig()
	require.NoError(t, err)
	protoFileSet, err := bufbuild.NewProvider(zap.NewNop()).GetProtoFileSetForBucket(
		context.Background(),
		bucket,
		config,
	)
	require.NoError(t, err)
	return protoFileSet
}

func testBuild(t *testing.T, includeSourceInfo bool, bucket storage.ReadBucket, protoFileSet bufbuild.ProtoFileSet) (bufpb.Image, []*analysis.Annotation) {
	var runOptions []bufbuild.RunOption
	runOptions = append(runOptions, bufbuild.RunWithIncludeImports())
	if includeSourceInfo {
		runOptions = append(runOptions, bufbuild.RunWithIncludeSourceInfo())
	}
	image, annotations, err := bufbuild.NewRunner(zap.NewNop()).Run(
		context.Background(),
		bucket,
		protoFileSet,
		runOptions...,
	)
	require.NoError(t, err)
	return image, annotations
}

func testBuildProtoc(t *testing.T, includeSourceInfo bool, baseDirPath string, protoFileSet bufbuild.ProtoFileSet) bufpb.Image {
	realFilePaths := protoFileSet.RealFilePaths()
	realFilePathsCopy := make([]string, len(realFilePaths))
	for i, realFilePath := range realFilePaths {
		realFilePathsCopy[i] = storagepath.Unnormalize(storagepath.Join(baseDirPath, realFilePath))
	}
	image, err := buftesting.GetProtocImage(
		context.Background(),
		[]string{testGoogleapisDirPath},
		realFilePathsCopy,
		true,
		includeSourceInfo,
	)
	require.NoError(t, err)
	return image
}

func testGetGoogleapis(t *testing.T) {
	testLock.Lock()
	defer func() {
		if r := recover(); r != nil {
			testLock.Unlock()
			panic(r)
		}
	}()
	defer testLock.Unlock()

	require.NoError(
		t,
		buftesting.GetGithubArchive(
			context.Background(),
			testGoogleapisDirPath,
			"googleapis",
			"googleapis",
			testGoogleapisCommit,
		),
	)
}

func assertImagesEqual(t *testing.T, one bufpb.Image, two bufpb.Image) {
	// This also has the effect of verifying output order
	diffOne, err := buftesting.DiffImagesJSON(one, two, "protoparse-protoc")
	assert.NoError(t, err)
	assert.Equal(t, "", diffOne, "JSON diff:\n%s", diffOne)
	// Cannot compare others due to unknown field issue
	//diffTwo, err := buftesting.DiffImagesText(one, two, "protoparse-protoc")
	//assert.NoError(t, err)
	//assert.Equal(t, "", diffTwo, "Text diff:\n%s", diffTwo)
	//equal, err := buftesting.ImagesEqual(one, two)
	//assert.NoError(t, err)
	//assert.True(t, equal, "proto.Equal returned false")
}
