// Copyright 2020-2024 Buf Technologies, Inc.
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

// This file defines a manager for tracking individual files.

package buflsp

import (
	"context"

	"github.com/bufbuild/buf/private/pkg/refcount"
	"go.lsp.dev/protocol"
)

// fileManager tracks all files the LSP is currently handling, whether read from disk or opened
// by the editor.
type fileManager struct {
	lsp       *lsp
	uriToFile refcount.Map[protocol.URI, file]
	mutexPool mutexPool
}

// newFiles creates a new file manager.
func newFileManager(lsp *lsp) *fileManager {
	return &fileManager{lsp: lsp}
}

// Open finds a file with the given URI, or creates one.
//
// This will increment the file's refcount.
func (fm *fileManager) Open(ctx context.Context, uri protocol.URI) *file {
	file, found := fm.uriToFile.Insert(uri)
	if !found {
		file.lsp = fm.lsp
		file.uri = uri
		file.lock = fm.mutexPool.NewMutex()
	}

	return file
}

// Get finds a file with the given URI, or returns nil.
func (fm *fileManager) Get(uri protocol.URI) *file {
	return fm.uriToFile.Get(uri)
}

// Close marks a file as closed.
//
// This will not necessarily evict the file, since there may be more than one user
// for this file.
func (fm *fileManager) Close(ctx context.Context, uri protocol.URI) {
	if deleted := fm.uriToFile.Delete(uri); deleted != nil {
		deleted.Reset(ctx)
	}
}
