// Copyright 2020 Buf Technologies, Inc.
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
	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/buf/bufmod"
	"github.com/bufbuild/buf/internal/buf/bufsrc"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/github"
	"github.com/bufbuild/buf/internal/pkg/httpauth"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/proto/protoc"
	"github.com/bufbuild/buf/internal/pkg/proto/protodiff"
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
				fileDescriptorSet := bufcore.ImageToFileDescriptorSet(image)
				protocFileDescriptorSet := testBuildProtocGoogleapis(t, includeSourceInfo)
				assertFileDescriptorSetsEqualJSON(t, fileDescriptorSet, protocFileDescriptorSet)
				// Cannot compare due to unknown field issue
				//assertFileDescriptorSetsEqualProto(t, fileDescriptorSet, protocFileDescriptorSet)
			},
		)
	}
}

func TestCompareCustomOptions1(t *testing.T) {
	testCompare(t, "customoptions1")
}

func TestCompareProto3Optional1(t *testing.T) {
	testCompare(t, "proto3optional1")
}

func TestCustomOptionsError1(t *testing.T) {
	t.Parallel()
	_, fileAnnotations := testBuild(t, false, filepath.Join("testdata", "customoptionserror1"))
	require.Equal(t, 1, len(fileAnnotations), fileAnnotations)
	require.Equal(
		t,
		"testdata/customoptionserror1/b.proto:9:27:field a.Baz.bat: option (a.foo).bat: field bat of a.Foo does not exist",
		fileAnnotations[0].String(),
	)
}

func testCompare(t *testing.T, relDirPath string) {
	t.Parallel()
	dirPath := filepath.Join("testdata", relDirPath)
	image, fileAnnotations := testBuild(t, false, dirPath)
	require.Equal(t, 0, len(fileAnnotations), fileAnnotations)
	image = bufcore.ImageWithoutImports(image)
	fileDescriptorSet := bufcore.ImageToFileDescriptorSet(image)
	protocFileDescriptorSet := testBuildProtoc(t, false, false, dirPath)
	assertFileDescriptorSetsEqualWire(t, fileDescriptorSet, protocFileDescriptorSet)
	assertFileDescriptorSetsEqualJSON(t, fileDescriptorSet, protocFileDescriptorSet)
	assertFileDescriptorSetsEqualProto(t, fileDescriptorSet, protocFileDescriptorSet)
}

func testBuildGoogleapis(t *testing.T, includeSourceInfo bool) bufcore.Image {
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
		testGetImageImportPaths(image),
	)

	imageWithoutImports := bufcore.ImageWithoutImports(image)
	assert.Equal(t, 1574, len(imageWithoutImports.Files()))
	imageWithoutImports = bufcore.ImageWithoutImports(imageWithoutImports)
	assert.Equal(t, 1574, len(imageWithoutImports.Files()))

	imageWithSpecificNames, err := bufcore.ImageWithOnlyPathsAllowNotExist(
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
		testGetImageFilePaths(imageWithSpecificNames),
	)
	imageWithoutImports = bufcore.ImageWithoutImports(imageWithSpecificNames)
	assert.Equal(
		t,
		[]string{
			"google/protobuf/api.proto",
			"google/protobuf/descriptor.proto",
			"google/type/date.proto",
		},
		testGetImageFilePaths(imageWithoutImports),
	)
	_, err = bufcore.ImageWithOnlyPaths(
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

func testBuild(t *testing.T, includeSourceInfo bool, dirPath string) (bufcore.Image, []bufanalysis.FileAnnotation) {
	module := testGetModule(t, dirPath)
	var options []BuildOption
	if !includeSourceInfo {
		options = append(options, WithExcludeSourceCodeInfo())
	}
	image, fileAnnotations, err := NewBuilder(zap.NewNop()).Build(
		context.Background(),
		module,
		options...,
	)
	require.NoError(t, err)
	return image, fileAnnotations
}

func testBuildProtoc(
	t *testing.T,
	includeImports bool,
	includeSourceInfo bool,
	dirPath string,
) *descriptorpb.FileDescriptorSet {
	module := testGetModule(t, dirPath)
	targetFileInfos, err := module.TargetFileInfos(context.Background())
	require.NoError(t, err)
	realFilePaths := make([]string, len(targetFileInfos))
	for i, fileInfo := range targetFileInfos {
		realFilePaths[i] = normalpath.Unnormalize(normalpath.Join(dirPath, fileInfo.Path()))
	}
	fileDescriptorSet, err := protoc.GetFileDescriptorSet(
		context.Background(),
		[]string{dirPath},
		realFilePaths,
		includeImports,
		includeSourceInfo,
		true,
	)
	require.NoError(t, err)
	return fileDescriptorSet
}

func testGetModule(t *testing.T, dirPath string) bufcore.Module {
	readWriteBucket, err := storageos.NewReadWriteBucket(dirPath)
	require.NoError(t, err)
	config, err := bufmod.NewConfig(bufmod.ExternalConfig{})
	require.NoError(t, err)
	module, err := bufmod.NewBuilder(zap.NewNop()).BuildForBucket(
		context.Background(),
		readWriteBucket,
		config,
	)
	require.NoError(t, err)
	return module
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

func testGetImageFilePaths(image bufcore.Image) []string {
	var fileNames []string
	for _, file := range image.Files() {
		fileNames = append(fileNames, file.Path())
	}
	sort.Strings(fileNames)
	return fileNames
}

func testGetImageImportPaths(image bufcore.Image) []string {
	var importNames []string
	for _, file := range image.Files() {
		if file.IsImport() {
			importNames = append(importNames, file.Path())
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
