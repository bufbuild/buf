// Copyright 2020-2025 Buf Technologies, Inc.
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

package buflsp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"

	"buf.build/go/standard/xlog/xslog"
	"github.com/bufbuild/buf/private/buf/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// errUnresolvableWorkspace is an unsupported workspace error.
type errUnresolvableWorkspace protocol.URI

func (e errUnresolvableWorkspace) Error() string {
	return fmt.Sprintf("unresolvable workspace for %q", string(e))
}

// workspaceManager tracks all workspaces the LSP is currently handling, per file.
type workspaceManager struct {
	lsp        *lsp
	workspaces []*workspace
}

// newWorkspaceManager creates a new workspace manager.
func newWorkspaceManager(lsp *lsp) *workspaceManager {
	return &workspaceManager{lsp: lsp}
}

// LeaseWorkspace attempts to find and lease the workspace for the given URI string. If the
// workspace has not been seen before, a new workspace is created. This may fail.
func (w *workspaceManager) LeaseWorkspace(ctx context.Context, uri protocol.URI) (*workspace, error) {
	defer func() {
		// Run a cleanup as a lazy job.
		w.Cleanup(ctx)
		w.lsp.logger.Debug("workspace: lease workspace", slog.Int("active", len(w.workspaces)))
	}()

	workspace, err := w.getOrCreateWorkspace(ctx, uri)
	if err != nil {
		return nil, err
	}
	workspace.Lease()
	return workspace, nil
}

// Cleanup removes any workspaces no longer referenced.
func (w *workspaceManager) Cleanup(ctx context.Context) {
	// Delete in-place.
	index := 0
	for _, workspace := range w.workspaces {
		if workspace.refCount > 0 {
			w.workspaces[index] = workspace
			index++
			continue // workspace leased
		}
		w.lsp.logger.Debug("workspace: cleanup removing workspace", slog.String("parent", workspace.workspaceURI.Filename()))
		for _, file := range workspace.pathToFile {
			file.Close(ctx)
		}
		workspace.pathToFile = nil
	}
	for j := index; j < len(w.workspaces); j++ {
		w.workspaces[j] = nil
	}
	w.workspaces = w.workspaces[:index]
}

// createWorkspace creates a new workspace for the protocol URI.
func (w *workspaceManager) getOrCreateWorkspace(ctx context.Context, uri protocol.URI) (*workspace, error) {
	// This looks for a workspace that already has ownership over the URI.
	// If a new file is added we will create a new workspace.
	// Reusing workspaces is an optimization. Matching is on best-effort.
	fileName := uri.Filename()
	for _, workspace := range w.workspaces {
		if _, ok := workspace.fileNameToFileInfo[fileName]; ok {
			// Workspace already exists for this file. Refresh to update.
			if err := workspace.Refresh(ctx); err != nil {
				return nil, err
			}
			w.lsp.logger.Debug("workspace: reusing workspace", slog.String("file", uri.Filename()), slog.String("parent", workspace.workspaceURI.Filename()))
			return workspace, nil
		}
	}

	// Workspaces are unresolvable for cached files.
	isCache := normalpath.ContainsPath(w.lsp.container.CacheDirPath(), fileName, normalpath.Absolute)
	if isCache {
		w.lsp.logger.Debug("workspace: unresolvable cache file outside workspace", slog.String("path", fileName))
		return nil, errUnresolvableWorkspace(uri)
	}
	// Add the workspace to the manager.
	workspace := &workspace{
		lsp:          w.lsp,
		workspaceURI: uri,
	}
	if err := workspace.Refresh(ctx); err != nil {
		return nil, err
	}
	w.workspaces = append(w.workspaces, workspace)
	return workspace, nil
}

// workspace is a workspace referenced from an open file by the client.
type workspace struct {
	lsp *lsp

	// refCount counts all the files that currently reference this workspace.
	// A refCount of zero will be removed by the workspaceManager on cleanup.
	refCount           int
	workspaceURI       protocol.URI // File that created this workspace.
	workspace          bufworkspace.Workspace
	fileNameToFileInfo map[string]bufmodule.FileInfo
	pathToFile         map[string]*file
	checkClient        bufcheck.Client
}

// Lease increments the reference count.
func (w *workspace) Lease() {
	w.lsp.logger.Debug("workspace: lease", slog.String("path", w.workspaceURI.Filename()))
	w.refCount++
}

// Release decrements the reference count.
func (w *workspace) Release() int {
	w.lsp.logger.Debug("workspace: release", slog.String("path", w.workspaceURI.Filename()))
	w.refCount--
	return w.refCount
}

// Refresh rebuilds the workspace and required context.
func (w *workspace) Refresh(ctx context.Context) error {
	if w == nil {
		return nil
	}
	fileName := w.workspaceURI.Filename()
	bufWorkspace, err := w.lsp.controller.GetWorkspace(ctx, fileName)
	if err != nil {
		w.lsp.logger.Error("workspace: get workspace", slog.String("file", fileName), xslog.ErrorAttr(err))
		return err
	}
	fileNameToFileInfo := make(map[string]bufmodule.FileInfo)
	for _, module := range bufWorkspace.Modules() {
		if err := module.WalkFileInfos(ctx, func(fileInfo bufmodule.FileInfo) error {
			if fileInfo.FileType() != bufmodule.FileTypeProto {
				return nil
			}
			fileNameToFileInfo[fileInfo.LocalPath()] = fileInfo
			return nil
		}); err != nil {
			return err
		}
	}
	// Get the check client for the workspace.
	checkClient, err := w.lsp.controller.GetCheckClientForWorkspace(ctx, bufWorkspace, w.lsp.wasmRuntime)
	if err != nil {
		w.lsp.logger.Warn("workspace: get check client", slog.String("file", fileName), xslog.ErrorAttr(err))
	}

	// Update the workspace.
	w.workspace = bufWorkspace
	w.fileNameToFileInfo = fileNameToFileInfo
	w.checkClient = checkClient
	w.indexFiles(ctx)
	return nil
}

// FileInfo returns an iterator over the files in the workspace.
func (w *workspace) FileInfo() iter.Seq[bufmodule.FileInfo] {
	return func(yield func(bufmodule.FileInfo) bool) {
		if w == nil {
			return
		}
		for _, fileInfo := range w.fileNameToFileInfo {
			if !yield(fileInfo) {
				return
			}
		}
	}
}

// Workspace returns the buf Workspace.
func (w *workspace) Workspace() bufworkspace.Workspace {
	if w == nil {
		return nil
	}
	return w.workspace
}

// GetModule resolves the Module for the protocol URI.
func (w *workspace) GetModule(uri protocol.URI) bufmodule.Module {
	if w == nil {
		return nil
	}
	fileName := uri.Filename()
	if fileInfo, ok := w.fileNameToFileInfo[fileName]; ok {
		return fileInfo.Module()
	}
	w.lsp.logger.Warn("workspace: module not found", slog.String("file", fileName), slog.String("parent", w.workspaceURI.Filename()))
	return nil
}

// CheckClient returns the buf check Client configured for the workspace.
func (w *workspace) CheckClient() bufcheck.Client {
	if w == nil {
		return nil
	}
	return w.checkClient
}

// PathToFile is an index of all files within the workspace.
func (w *workspace) PathToFile() map[string]*file {
	if w == nil {
		return nil
	}
	return w.pathToFile
}

// indexFiles builds the pathToFile mapping.
func (w *workspace) indexFiles(ctx context.Context) {
	w.lsp.logger.Debug("workspace: index files", slog.String("path", w.workspaceURI.Filename()))
	previous := w.pathToFile
	w.pathToFile = make(map[string]*file, len(previous))

	for fileInfo := range w.fileInfos(ctx) {
		file, ok := previous[fileInfo.Path()]
		if !ok {
			fileURI := uri.File(fileInfo.LocalPath())
			file = w.lsp.fileManager.Track(fileURI)
			w.lsp.logger.Debug("workspace: index track file", slog.String("path", file.uri.Filename()))
		}

		// Currently we only associate a file with one workspace. This assumption isn't accurate
		// for shared dependencies. Here we update to the lastest, most recently used, workspace.
		// This will make goto definition and find references only work in that workspace.
		if oldWorkspace := file.workspace; oldWorkspace != nil && oldWorkspace != w {
			oldWorkspace.Release()
			w.Lease()
			file.workspace = w
		}

		file.objectInfo = fileInfo
		if err := file.ReadFromWorkspace(ctx); err != nil {
			w.lsp.logger.Error(
				"failed to read contents for file",
				xslog.ErrorAttr(err),
				slog.String("file", fileInfo.Path()),
			)
		}

		// Update index.
		w.pathToFile[fileInfo.Path()] = file
		delete(previous, fileInfo.Path())
	}
	// Drop all unused files. It was deleted from the workspace.
	for _, file := range previous {
		w.lsp.logger.Debug("workspace: index drop file", slog.String("path", file.uri.Filename()))
		file.Close(ctx)
	}
}

// fileInfos returns all files within the workspace.
//
// This consists of files within the workspace plus WKTs.
func (w *workspace) fileInfos(ctx context.Context) iter.Seq[storage.ObjectInfo] {
	return func(yield func(storage.ObjectInfo) bool) {
		seen := make(map[string]struct{})
		for fileInfo := range w.FileInfo() {
			if !yield(fileInfo) {
				return
			}
			seen[fileInfo.Path()] = struct{}{}
		}
		// Add all wellknown types if not provided within the workspace.
		if err := w.lsp.wktBucket.Walk(ctx, "", func(objectInfo storage.ObjectInfo) error {
			if _, ok := seen[objectInfo.Path()]; !ok {
				if !yield(wktObjectInfo{objectInfo}) {
					return io.EOF
				}
			}
			return nil
		}); err != nil && !errors.Is(err, io.EOF) {
			w.lsp.logger.Error("wkt bucket failed", xslog.ErrorAttr(err))
		}
	}
}
