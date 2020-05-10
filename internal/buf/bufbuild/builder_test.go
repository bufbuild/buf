// Copyright 2020 Buf Technologies Inc.
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

package bufbuild

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/bufbuild/buf/internal/buf/ext/extimage"
	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/protodesc"
	"github.com/bufbuild/buf/internal/pkg/prototesting"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/bufbuild/buf/internal/pkg/util/utilgithub/utilgithubtesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
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
				fileDescriptorSet, err := extimage.ImageToFileDescriptorSet(image)
				assert.NoError(t, err)
				protocFileDescriptorSet := testBuildProtocGoogleapis(t, includeSourceInfo)
				assertFileDescriptorSetsEqualJSON(t, fileDescriptorSet, protocFileDescriptorSet)
				// Cannot compare due to unknown field issue
				//assertFileDescriptorSetsEqualProto(t, fileDescriptorSet, protocFileDescriptorSet)
			},
		)
	}
}

func TestCompareCustomOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "customoptions1")
	image, fileAnnotations := testBuild(t, false, false, dirPath)
	require.Equal(t, 0, len(fileAnnotations), fileAnnotations)
	fileDescriptorSet, err := extimage.ImageToFileDescriptorSet(image)
	require.NoError(t, err)
	protocFileDescriptorSet := testBuildProtoc(t, false, false, dirPath, dirPath)
	assertFileDescriptorSetsEqualWire(t, fileDescriptorSet, protocFileDescriptorSet)
	assertFileDescriptorSetsEqualJSON(t, fileDescriptorSet, protocFileDescriptorSet)
	assertFileDescriptorSetsEqualProto(t, fileDescriptorSet, protocFileDescriptorSet)
}

func testBuildGoogleapis(t *testing.T, includeSourceInfo bool) *imagev1beta1.Image {
	testGetGoogleapis(t)
	image, fileAnnotations := testBuild(t, true, includeSourceInfo, testGoogleapisDirPath)

	assert.Equal(t, 0, len(fileAnnotations), fileAnnotations)
	assert.Equal(t, 1585, len(image.GetFile()))
	importNames, err := extimage.ImageImportNames(image)
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

	imageWithoutImports, err := extimage.ImageWithoutImports(image)
	assert.NoError(t, err)
	assert.Equal(t, 1574, len(imageWithoutImports.GetFile()))
	importNames, err = extimage.ImageImportNames(imageWithoutImports)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(importNames))

	imageWithoutImports, err = extimage.ImageWithoutImports(imageWithoutImports)
	assert.NoError(t, err)
	assert.Equal(t, 1574, len(imageWithoutImports.GetFile()))
	importNames, err = extimage.ImageImportNames(imageWithoutImports)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(importNames))

	imageWithSpecificNames, err := extimage.ImageWithSpecificNames(
		image,
		true,
		"google/protobuf/descriptor.proto",
		"google/protobuf/api.proto",
		"google/../google/type/date.proto",
		"google/foo/nonsense.proto",
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(imageWithSpecificNames.GetFile()))
	_, err = extimage.ImageWithSpecificNames(
		image,
		false,
		"google/protobuf/descriptor.proto",
		"google/protobuf/api.proto",
		"google/../google/type/date.proto",
		"google/foo/nonsense.proto",
	)
	assert.Equal(t, errors.New("google/foo/nonsense.proto is not present in the Image"), err)
	importNames, err = extimage.ImageImportNames(imageWithSpecificNames)
	assert.NoError(t, err)
	assert.Equal(
		t,
		[]string{
			"google/protobuf/api.proto",
			"google/protobuf/descriptor.proto",
		},
		importNames,
	)
	imageWithoutImports, err = extimage.ImageWithoutImports(imageWithSpecificNames)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(imageWithoutImports.GetFile()))
	importNames, err = extimage.ImageImportNames(imageWithoutImports)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(importNames))

	assert.Equal(t, 1585, len(image.GetFile()))
	// basic check to make sure there is no error at this scale
	_, err = protodesc.NewFilesUnstable(context.Background(), image.GetFile()...)
	assert.NoError(t, err)
	return image
}

func testBuildProtocGoogleapis(t *testing.T, includeSourceInfo bool) *descriptorpb.FileDescriptorSet {
	testGetGoogleapis(t)
	fileDescriptorSet := testBuildProtoc(t, true, includeSourceInfo, testGoogleapisDirPath, testGoogleapisDirPath)
	assert.Equal(t, 1585, len(fileDescriptorSet.GetFile()))
	return fileDescriptorSet
}

func testBuild(t *testing.T, includeImports bool, includeSourceInfo bool, dirPath string) (*imagev1beta1.Image, []*filev1beta1.FileAnnotation) {
	readBucketCloser := testGetReadBucketCloser(t, dirPath)
	protoFileSet := testGetProtoFileSet(t, readBucketCloser)
	buildResult, fileAnnotations, err := newBuilder(zap.NewNop(), runtime.NumCPU()).Build(
		context.Background(),
		readBucketCloser,
		protoFileSet.Roots(),
		protoFileSet.RootFilePaths(),
		includeImports,
		includeSourceInfo,
	)
	require.NoError(t, err)
	require.NoError(t, readBucketCloser.Close())
	// this drops the ImageWithImports as we are not using it
	return buildResult.Image, fileAnnotations
}

func testBuildProtoc(t *testing.T, includeImports bool, includeSourceInfo bool, includeDirPath string, dirPath string) *descriptorpb.FileDescriptorSet {
	readBucketCloser := testGetReadBucketCloser(t, dirPath)
	protoFileSet := testGetProtoFileSet(t, readBucketCloser)
	realFilePaths := protoFileSet.RealFilePaths()
	realFilePathsCopy := make([]string, len(realFilePaths))
	for i, realFilePath := range realFilePaths {
		realFilePathsCopy[i] = storagepath.Unnormalize(storagepath.Join(dirPath, realFilePath))
	}
	fileDescriptorSet, err := prototesting.GetProtocFileDescriptorSet(
		context.Background(),
		[]string{includeDirPath},
		realFilePathsCopy,
		includeImports,
		includeSourceInfo,
	)
	require.NoError(t, err)
	require.NoError(t, readBucketCloser.Close())
	return fileDescriptorSet
}

func testGetReadBucketCloser(t *testing.T, dirPath string) storage.ReadBucketCloser {
	readWriteBucketCloser, err := storageos.NewReadWriteBucketCloser(dirPath)
	require.NoError(t, err)
	return readWriteBucketCloser
}

func testGetProtoFileSet(t *testing.T, readBucket storage.ReadBucket) ProtoFileSet {
	protoFileSet, err := newProtoFileSetProvider(zap.NewNop()).GetProtoFileSetForReadBucket(
		context.Background(),
		readBucket,
		nil,
		nil,
	)
	require.NoError(t, err)
	return protoFileSet
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
		utilgithubtesting.GetGithubArchive(
			context.Background(),
			testGoogleapisDirPath,
			"googleapis",
			"googleapis",
			testGoogleapisCommit,
		),
	)
}

func assertFileDescriptorSetsEqualWire(t *testing.T, one *descriptorpb.FileDescriptorSet, two *descriptorpb.FileDescriptorSet) {
	diffTwo, err := prototesting.DiffFileDescriptorSetsWire(one, two, "protoparse-protoc")
	assert.NoError(t, err)
	assert.Equal(t, "", diffTwo, "Wire diff:\n%s", diffTwo)
}

func assertFileDescriptorSetsEqualJSON(t *testing.T, one *descriptorpb.FileDescriptorSet, two *descriptorpb.FileDescriptorSet) {
	// TODO: test with resolver?
	// This also has the effect of verifying output order
	diffOne, err := prototesting.DiffFileDescriptorSetsJSON(one, two, "protoparse-protoc")
	assert.NoError(t, err)
	assert.Equal(t, "", diffOne, "JSON diff:\n%s", diffOne)
}

func assertFileDescriptorSetsEqualProto(t *testing.T, one *descriptorpb.FileDescriptorSet, two *descriptorpb.FileDescriptorSet) {
	equal := proto.Equal(one, two)
	assert.True(t, equal, "proto.Equal returned false")
}
