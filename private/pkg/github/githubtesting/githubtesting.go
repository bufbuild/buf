// Copyright 2020-2023 Buf Technologies, Inc.
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

// Package githubtesting provides testing functionality for GitHub.
package githubtesting

import (
	"context"
	"net/http"

	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/zap"
)

// ArchiveReader reads GitHub archives.
type ArchiveReader interface {
	// GetArchive gets the GitHub archive and untars it to the output directory path.
	//
	// The root directory within the tarball is stripped.
	// If the directory already exists, this is a no-op.
	//
	// Uses file locking to make sure the no-op works properly across multiple process invocations,
	// which is needed for example with go test.
	// This is also thread-safe.
	//
	// Only use for testing.
	GetArchive(
		ctx context.Context,
		outputDirPath string,
		owner string,
		repository string,
		ref string,
	) error
}

// NewArchiveReader returns a new ArchiveReader.
func NewArchiveReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	httpClient *http.Client,
) ArchiveReader {
	return newArchiveReader(
		logger,
		storageosProvider,
		httpClient,
	)
}
