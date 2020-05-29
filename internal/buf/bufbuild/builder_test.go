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
	"net/http"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufpath"
	"github.com/bufbuild/buf/internal/buf/bufsrc"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/github"
	"github.com/bufbuild/buf/internal/pkg/httpauth"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/proto/protoc"
	"github.com/bufbuild/buf/internal/pkg/proto/protodiff"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
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
	testHTTPClient = &http.Client{
		Timeout: 10 * time.Second,
	}
	testHTTPAuthenticator = httpauth.NewNopAuthenticator()
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
				fileDescriptorSet := bufimage.ImageToFileDescriptorSet(image)
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
	image, fileAnnotations := testBuild(t, false, dirPath)
	require.Equal(t, 0, len(fileAnnotations), fileAnnotations)
	image = bufimage.ImageWithoutImports(image)
	fileDescriptorSet := bufimage.ImageToFileDescriptorSet(image)
	protocFileDescriptorSet := testBuildProtoc(t, false, false, dirPath)
	assertFileDescriptorSetsEqualWire(t, fileDescriptorSet, protocFileDescriptorSet)
	assertFileDescriptorSetsEqualJSON(t, fileDescriptorSet, protocFileDescriptorSet)
	assertFileDescriptorSetsEqualProto(t, fileDescriptorSet, protocFileDescriptorSet)
}

func testBuildGoogleapis(t *testing.T, includeSourceInfo bool) bufimage.Image {
	testGetGoogleapis(t)
	image, fileAnnotations := testBuild(t, includeSourceInfo, testGoogleapisDirPath)

	assert.Equal(t, 0, len(fileAnnotations), fileAnnotations)
	assert.Equal(t, 1585, len(image.Files()))
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
		testGetImageImportNames(image),
	)

	imageWithoutImports := bufimage.ImageWithoutImports(image)
	assert.Equal(t, 1574, len(imageWithoutImports.Files()))
	imageWithoutImports = bufimage.ImageWithoutImports(imageWithoutImports)
	assert.Equal(t, 1574, len(imageWithoutImports.Files()))

	imageWithSpecificNames, err := bufimage.ImageWithOnlyRootRelFilePathsAllowNotExist(
		image,
		[]string{
			"google/protobuf/descriptor.proto",
			"google/protobuf/api.proto",
			"google/type/date.proto",
			"google/foo/nonsense.proto",
		},
	)
	assert.NoError(t, err)
	assert.Equal(
		t,
		[]string{
			"google/protobuf/any.proto",
			"google/protobuf/api.proto",
			"google/protobuf/descriptor.proto",
			"google/protobuf/source_context.proto",
			"google/protobuf/type.proto",
			"google/type/date.proto",
		},
		testGetImageFileNames(imageWithSpecificNames),
	)
	imageWithoutImports = bufimage.ImageWithoutImports(imageWithSpecificNames)
	assert.Equal(
		t,
		[]string{
			"google/protobuf/api.proto",
			"google/protobuf/descriptor.proto",
			"google/type/date.proto",
		},
		testGetImageFileNames(imageWithoutImports),
	)
	_, err = bufimage.ImageWithOnlyRootRelFilePaths(
		image,
		[]string{
			"google/protobuf/descriptor.proto",
			"google/protobuf/api.proto",
			"google/type/date.proto",
			"google/foo/nonsense.proto",
		},
	)
	// TODO
	assert.Equal(t, errors.New("google/foo/nonsense.proto is not present in the Image"), err)

	assert.Equal(t, 1585, len(image.Files()))
	// basic check to make sure there is no error at this scale
	_, err = bufsrc.NewFilesUnstable(context.Background(), image.Files()...)
	assert.NoError(t, err)
	return image
}

func testBuildProtocGoogleapis(t *testing.T, includeSourceInfo bool) *descriptorpb.FileDescriptorSet {
	testGetGoogleapis(t)
	fileDescriptorSet := testBuildProtoc(t, true, includeSourceInfo, testGoogleapisDirPath)
	assert.Equal(t, 1585, len(fileDescriptorSet.GetFile()))
	return fileDescriptorSet
}

func testBuild(t *testing.T, includeSourceInfo bool, dirPath string) (bufimage.Image, []bufanalysis.FileAnnotation) {
	readBucketCloser := testGetReadBucketCloser(t, dirPath)
	pathResolver := bufpath.NewDirPathResolver(dirPath)
	fileRefs := testGetAllFileRefs(t, readBucketCloser, pathResolver)
	var options []BuildOption
	if !includeSourceInfo {
		options = append(options, WithExcludeSourceCodeInfo())
	}
	image, fileAnnotations, err := NewBuilder(zap.NewNop()).Build(
		context.Background(),
		readBucketCloser,
		pathResolver,
		fileRefs,
		options...,
	)
	require.NoError(t, err)
	require.NoError(t, readBucketCloser.Close())
	// this drops the ImageWithImports as we are not using it
	return image, fileAnnotations
}

func testBuildProtoc(
	t *testing.T,
	includeImports bool,
	includeSourceInfo bool,
	dirPath string,
) *descriptorpb.FileDescriptorSet {
	readBucketCloser := testGetReadBucketCloser(t, dirPath)
	pathResolver := bufpath.NewDirPathResolver(dirPath)
	fileRefs := testGetAllFileRefs(t, readBucketCloser, pathResolver)
	realFilePaths := make([]string, len(fileRefs))
	for i, fileRef := range fileRefs {
		realFilePaths[i] = normalpath.Unnormalize(normalpath.Join(dirPath, fileRef.RootDirPath(), fileRef.RootRelFilePath()))
	}
	fileDescriptorSet, err := protoc.GetFileDescriptorSet(
		context.Background(),
		[]string{dirPath},
		realFilePaths,
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

func testGetAllFileRefs(
	t *testing.T,
	readBucket storage.ReadBucket,
	externalPathResolver bufpath.ExternalPathResolver,
) []bufimage.FileRef {
	fileRefs, err := NewFileRefProvider(zap.NewNop()).GetAllFileRefs(
		context.Background(),
		readBucket,
		externalPathResolver,
		nil,
		nil,
	)
	require.NoError(t, err)
	return fileRefs
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

	archiveReader := github.NewArchiveReader(
		zap.NewNop(),
		testHTTPClient,
		testHTTPAuthenticator,
	)
	require.NoError(
		t,
		archiveReader.GetArchive(
			context.Background(),
			app.NewContainer(nil, nil, nil, nil),
			testGoogleapisDirPath,
			"googleapis",
			"googleapis",
			testGoogleapisCommit,
		),
	)
}

func testGetImageFileNames(image bufimage.Image) []string {
	var fileNames []string
	for _, file := range image.Files() {
		fileNames = append(fileNames, file.RootRelFilePath())
	}
	sort.Strings(fileNames)
	return fileNames
}

func testGetImageImportNames(image bufimage.Image) []string {
	var importNames []string
	for _, file := range image.Files() {
		if file.IsImport() {
			importNames = append(importNames, file.RootRelFilePath())
		}
	}
	sort.Strings(importNames)
	return importNames
}

func assertFileDescriptorSetsEqualWire(t *testing.T, one *descriptorpb.FileDescriptorSet, two *descriptorpb.FileDescriptorSet) {
	diffTwo, err := protodiff.DiffFileDescriptorSetsWire(one, two, "protoparse-protoc")
	assert.NoError(t, err)
	assert.Equal(t, "", diffTwo, "Wire diff:\n%s", diffTwo)
}

func assertFileDescriptorSetsEqualJSON(t *testing.T, one *descriptorpb.FileDescriptorSet, two *descriptorpb.FileDescriptorSet) {
	// TODO: test with resolver?
	// This also has the effect of verifying output order
	diffOne, err := protodiff.DiffFileDescriptorSetsJSON(one, two, "protoparse-protoc")
	assert.NoError(t, err)
	assert.Equal(t, "", diffOne, "JSON diff:\n%s", diffOne)
}

func assertFileDescriptorSetsEqualProto(t *testing.T, one *descriptorpb.FileDescriptorSet, two *descriptorpb.FileDescriptorSet) {
	equal := proto.Equal(one, two)
	assert.True(t, equal, "proto.Equal returned false")
}
