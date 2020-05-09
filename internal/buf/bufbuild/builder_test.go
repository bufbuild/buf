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
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/bufbuild/buf/internal/pkg/util/utilgithub/utilgithubtesting"
	"github.com/bufbuild/buf/internal/pkg/util/utilproto/utilprototesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/runtime/protoimpl"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	testGoogleapisCommit = "37c923effe8b002884466074f84bc4e78e6ade62"
)

var (
	testEBaz = &protoimpl.ExtensionInfo{
		ExtendedType:  (*descriptorpb.FieldOptions)(nil),
		ExtensionType: (*int32)(nil),
		Field:         50007,
		Name:          "baz",
		Tag:           "varint,50007,opt,name=baz",
		Filename:      "a.proto",
	}
	testGoogleapisDirPath = filepath.Join("cache", "googleapis")
	testLock              sync.Mutex
)

//func init() {
//proto.RegisterExtension(testEBaz)
//}

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
				assertFileDescriptorSetsEqual(t, fileDescriptorSet, protocFileDescriptorSet)
			},
		)
	}
}

func TestCustomOptions(t *testing.T) {
	// https://github.com/bufbuild/buf/issues/52
	t.Skip()

	t.Parallel()
	image, fileAnnotations := testBuild(t, false, false, filepath.Join("testdata", "customoptions1"))
	require.Equal(t, 0, len(fileAnnotations), fileAnnotations)

	require.NotNil(t, image)
	require.Equal(t, 1, len(image.File))
	fileDescriptorProto := image.File[0]
	require.Equal(t, 1, len(fileDescriptorProto.MessageType))
	messageDescriptorProto := fileDescriptorProto.MessageType[0]
	require.Equal(t, 1, len(messageDescriptorProto.Field))
	fieldDescriptorProto := messageDescriptorProto.Field[0]
	fieldOptions := fieldDescriptorProto.Options
	require.NotNil(t, fieldOptions)
	require.True(t, proto.HasExtension(fieldOptions, testEBaz))
	value := proto.GetExtension(fieldOptions, testEBaz)
	valueInt32, ok := value.(*int32)
	require.True(t, ok)
	require.NotNil(t, valueInt32)
	require.Equal(t, int32(42), *valueInt32)
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
	fileDescriptorSet := testBuildProtoc(t, true, includeSourceInfo, testGoogleapisDirPath)
	assert.Equal(t, 1585, len(fileDescriptorSet.GetFile()))
	return fileDescriptorSet
}

func testBuild(t *testing.T, includeImports bool, includeSourceInfo bool, dirPath string) (*imagev1beta1.Image, []*filev1beta1.FileAnnotation) {
	readBucketCloser := testGetReadBucketCloser(t, dirPath)
	protoFileSet := testGetProtoFileSet(t, readBucketCloser)
	image, fileAnnotations, err := newBuilder(zap.NewNop(), runtime.NumCPU()).Build(
		context.Background(),
		readBucketCloser,
		protoFileSet.Roots(),
		protoFileSet.RootFilePaths(),
		includeImports,
		includeSourceInfo,
	)
	require.NoError(t, err)
	require.NoError(t, readBucketCloser.Close())
	return image, fileAnnotations
}

func testBuildProtoc(t *testing.T, includeImports bool, includeSourceInfo bool, dirPath string) *descriptorpb.FileDescriptorSet {
	readBucketCloser := testGetReadBucketCloser(t, dirPath)
	protoFileSet := testGetProtoFileSet(t, readBucketCloser)
	realFilePaths := protoFileSet.RealFilePaths()
	realFilePathsCopy := make([]string, len(realFilePaths))
	for i, realFilePath := range realFilePaths {
		realFilePathsCopy[i] = storagepath.Unnormalize(storagepath.Join(dirPath, realFilePath))
	}
	fileDescriptorSet, err := utilprototesting.GetProtocFileDescriptorSet(
		context.Background(),
		[]string{testGoogleapisDirPath},
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

func assertFileDescriptorSetsEqual(t *testing.T, one *descriptorpb.FileDescriptorSet, two *descriptorpb.FileDescriptorSet) {
	// This also has the effect of verifying output order
	diffOne, err := utilprototesting.DiffMessagesJSON(one, two, "protoparse-protoc")
	assert.NoError(t, err)
	assert.Equal(t, "", diffOne, "JSON diff:\n%s", diffOne)
	// Cannot compare others due to unknown field issue
	//diffTwo, err := utilprototesting.DiffMessagesText(one, two, "protoparse-protoc")
	//assert.NoError(t, err)
	//assert.Equal(t, "", diffTwo, "Text diff:\n%s", diffTwo)
	//equal, err := proto.Equal(one, two)
	//assert.NoError(t, err)
	//assert.True(t, equal, "proto.Equal returned false")
}
