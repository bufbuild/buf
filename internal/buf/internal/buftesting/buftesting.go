// Copyright 2020-2021 Buf Technologies, Inc.
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

package buftesting

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/pkg/github/githubtesting"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/prototesting"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	// NumGoogleapisFiles is the number of googleapis files on the current test commit.
	NumGoogleapisFiles = 1574
	// NumGoogleapisFilesWithImports is the number of googleapis files on the current test commit with imports.
	NumGoogleapisFilesWithImports = 1585

	testGoogleapisCommit = "37c923effe8b002884466074f84bc4e78e6ade62"
)

var (
	testHTTPClient = &http.Client{
		Timeout: 10 * time.Second,
	}
	testStorageosProvider = storageos.NewProvider(storageos.ProviderWithSymlinks())
	testArchiveReader     = githubtesting.NewArchiveReader(
		zap.NewNop(),
		testStorageosProvider,
		testHTTPClient,
	)
	testGoogleapisDirPath = filepath.Join("cache", "googleapis")
)

// GetActualProtocFileDescriptorSet gets the FileDescriptorSet for actual protoc.
func GetActualProtocFileDescriptorSet(
	t *testing.T,
	includeImports bool,
	includeSourceInfo bool,
	dirPath string,
	filePaths []string,
) *descriptorpb.FileDescriptorSet {
	fileDescriptorSet, err := prototesting.GetProtocFileDescriptorSet(
		context.Background(),
		[]string{dirPath},
		filePaths,
		includeImports,
		includeSourceInfo,
	)
	require.NoError(t, err)
	return fileDescriptorSet
}

// RunActualProtoc runs actual protoc.
func RunActualProtoc(
	t *testing.T,
	includeImports bool,
	includeSourceInfo bool,
	dirPath string,
	filePaths []string,
	env map[string]string,
	stdout io.Writer,
	extraFlags ...string,
) {
	t.Helper()
	err := prototesting.RunProtoc(
		context.Background(),
		[]string{dirPath},
		filePaths,
		includeImports,
		includeSourceInfo,
		env,
		stdout,
		extraFlags...,
	)
	require.NoError(t, err)
}

// GetGoogleapisDirPath gets the path to a clone of googleapis.
func GetGoogleapisDirPath(t *testing.T, buftestingDirPath string) string {
	googleapisDirPath := filepath.Join(buftestingDirPath, testGoogleapisDirPath)
	require.NoError(
		t,
		testArchiveReader.GetArchive(
			context.Background(),
			googleapisDirPath,
			"googleapis",
			"googleapis",
			testGoogleapisCommit,
		),
	)
	return googleapisDirPath
}

// GetProtocFilePaths gets the file paths for protoc.
//
// Limit limits the number of files returned if > 0.
// protoc has a fixed size for number of characters to argument list.
func GetProtocFilePaths(t *testing.T, dirPath string, limit int) []string {
	t.Helper()
	module, err := bufmodulebuild.NewModuleIncludeBuilder(zap.NewNop(), testStorageosProvider).BuildForIncludes(
		context.Background(),
		[]string{dirPath},
	)
	require.NoError(t, err)
	targetFileInfos, err := module.TargetFileInfos(context.Background())
	require.NoError(t, err)
	realFilePaths := make([]string, len(targetFileInfos))
	for i, fileInfo := range targetFileInfos {
		realFilePaths[i] = normalpath.Unnormalize(normalpath.Join(dirPath, fileInfo.Path()))
	}
	if limit > 0 && len(realFilePaths) > limit {
		return realFilePaths[:limit]
	}
	return realFilePaths
}
