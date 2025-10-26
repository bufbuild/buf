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
	"iter"
	"log/slog"
	"time"

	"buf.build/go/standard/xlog/xslog"
	"github.com/bufbuild/buf/private/buf/buftarget"
	"github.com/bufbuild/buf/private/buf/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
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
		} else {
			w.lsp.logger.Debug("workspace: cleanup removing workspace", slog.String("parent", workspace.workspaceURI.Filename()))
		}
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

	// verison of the workspace.
	version int32

	// refCount counts all the files that currently reference this workspace.
	// A refCount of zero will be removed by the workspaceManager on cleanup.
	refCount           int
	workspaceURI       protocol.URI // File that created this workspace.
	workspaceDir       string       // Directory containing the buf.yaml.
	workspace          bufworkspace.Workspace
	fileNameToFileInfo map[string]bufmodule.FileInfo

	// cancelChecks if not nil, cancels any running check context.
	cancelChecks func()
}

// Lease increments the reference count.
func (w *workspace) Lease() {
	w.refCount++
}

// Release decrements the reference count.
func (w *workspace) Release() int {
	w.refCount--
	return w.refCount
}

// Version returns the workspace iteration count.
func (w *workspace) Version() int32 {
	if w == nil {
		return 0
	}
	return w.version
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

	// Determine the workspace directory using buftarget.
	workspaceDir, err := w.getWorkspaceDir(ctx, fileName)
	if err != nil {
		w.lsp.logger.Warn("workspace: failed to determine workspace directory", xslog.ErrorAttr(err))
		// Fall back to using the directory of the file.
		workspaceDir = normalpath.Dir(fileName)
	}
	w.lsp.logger.Debug(
		"workspace: determined workspace directory",
		slog.String("uri", fileName),
		slog.String("workspaceDir", workspaceDir),
	)

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
	// Update the workspace.
	w.version++
	w.workspace = bufWorkspace
	w.workspaceDir = workspaceDir
	w.fileNameToFileInfo = fileNameToFileInfo
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

// CancelChecks cancels any currently running checks for this workspace.
func (w *workspace) CancelChecks(ctx context.Context) {
	if w == nil {
		return
	}
	if w.cancelChecks != nil {
		w.cancelChecks()
		w.cancelChecks = nil
	}
}

// RunChecks triggers the run of checks within the workspace. Diagnostics are published asynchronously.
func (w *workspace) RunChecks(ctx context.Context) {
	if w == nil {
		return
	}
	w.CancelChecks(ctx)

	const checkTimeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), checkTimeout)
	w.cancelChecks = cancel

	go func() {
		annotations, err := w.runChecks(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				w.lsp.logger.DebugContext(ctx, "workspace: checks cancelled", slog.String("uri", w.workspaceURI.Filename()), xslog.ErrorAttr(err))
			} else if errors.Is(err, context.DeadlineExceeded) {
				w.lsp.logger.WarnContext(ctx, "workspace: checks deadline exceeded", slog.String("uri", w.workspaceURI.Filename()), xslog.ErrorAttr(err))
			} else {
				w.lsp.logger.ErrorContext(ctx, "workspace: checks failed", slog.String("uri", w.workspaceURI.Filename()), xslog.ErrorAttr(err))
			}
			return
		}

		w.lsp.lock.Lock()
		defer w.lsp.lock.Unlock()

		select {
		case <-ctx.Done():
			return // Context cancelled before publishing.
		default:
		}

		// Group annotations by file.
		pathToLocalPath := make(map[string]string)
		for fileInfo := range w.FileInfo() {
			pathToLocalPath[fileInfo.Path()] = fileInfo.LocalPath()
		}
		annotationsByPath := make(map[string][]bufanalysis.FileAnnotation)
		for _, annotation := range annotations {
			fileInfo := annotation.FileInfo()
			if fileInfo == nil {
				continue
			}
			path := fileInfo.Path()
			annotationsByPath[path] = append(annotationsByPath[path], annotation)
		}

		// Append diagnostics to each file and publish.
		for path, fileAnnotations := range annotationsByPath {
			localPath, ok := pathToLocalPath[path]
			if !ok {
				// File path not found in workspace, skip it.
				continue
			}
			fileURI := uri.File(localPath)
			file := w.lsp.fileManager.Get(fileURI)
			if file == nil {
				// File is not tracked, skip it.
				continue
			}
			if !file.IsOpenInEditor() {
				// Only publish diagnostics for files open in the editor.
				continue
			}
			for _, annotation := range fileAnnotations {
				file.appendAnnotation("buf lint", annotation)
			}
			file.PublishDiagnostics(ctx)
		}
	}()
}

// getWorkspaceDir determines the workspace directory by finding the controlling workspace.
func (w *workspace) getWorkspaceDir(ctx context.Context, fileName string) (string, error) {
	absPath, err := normalpath.NormalizeAndAbsolute(fileName)
	if err != nil {
		return "", err
	}
	// Split the absolute path into components to get the FS root.
	absPathComponents := normalpath.Components(absPath)
	fsRoot := absPathComponents[0]
	fsRelPath, err := normalpath.Rel(fsRoot, absPath)
	if err != nil {
		return "", err
	}
	storageosProvider := storageos.NewProvider()
	osRootBucket, err := storageosProvider.NewReadWriteBucket(
		fsRoot,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return "", err
	}
	bucketTargeting, err := buftarget.NewBucketTargeting(
		ctx,
		w.lsp.logger,
		osRootBucket,
		fsRelPath,
		nil, // no target paths
		nil, // no target exclude paths
		buftarget.TerminateAtControllingWorkspace,
	)
	if err != nil {
		return "", err
	}
	controllingWorkspace := bucketTargeting.ControllingWorkspace()
	if controllingWorkspace == nil {
		return "", fmt.Errorf("no controlling workspace found")
	}
	// The controlling workspace path is relative to the bucket (fileDir).
	// Combine it to get the absolute workspace directory.
	return normalpath.Join(fsRoot, controllingWorkspace.Path()), nil
}

// runChecks invokes checks on the workspace.
func (w *workspace) runChecks(ctx context.Context) (annotations []bufanalysis.FileAnnotation, err error) {
	defer xslog.DebugProfile(
		w.lsp.logger,
		slog.String("uri", w.workspaceURI.Filename()),
		slog.String("workspaceDir", w.workspaceDir),
	)()

	imageWithConfigs, checkClient, err := w.lsp.controller.GetTargetImageWithConfigsAndCheckClient(
		ctx,
		w.workspaceDir,
		w.lsp.wasmRuntime,
	)
	if err != nil {
		return nil, err
	}
	var allFileAnnotations []bufanalysis.FileAnnotation
	allCheckConfigs := make([]bufconfig.CheckConfig, 0, len(imageWithConfigs)*2)
	for _, imageWithConfig := range imageWithConfigs {
		allCheckConfigs = append(allCheckConfigs, imageWithConfig.LintConfig())
		allCheckConfigs = append(allCheckConfigs, imageWithConfig.BreakingConfig())
	}
	for _, imageWithConfig := range imageWithConfigs {
		w.lsp.logger.DebugContext(
			ctx, "workspace: running lint",
			slog.String("uri", w.workspaceURI.Filename()),
			slog.String("workspaceDir", w.workspaceDir),
			slog.String("module", imageWithConfig.ModuleOpaqueID()),
		)
		lintOptions := []bufcheck.LintOption{
			bufcheck.WithPluginConfigs(imageWithConfig.PluginConfigs()...),
			bufcheck.WithPolicyConfigs(imageWithConfig.PolicyConfigs()...),
			bufcheck.WithRelatedCheckConfigs(allCheckConfigs...),
		}
		if err := checkClient.Lint(
			ctx,
			imageWithConfig.LintConfig(),
			imageWithConfig,
			lintOptions...,
		); err != nil {
			var fileAnnotationSet bufanalysis.FileAnnotationSet
			if errors.As(err, &fileAnnotationSet) {
				allFileAnnotations = append(allFileAnnotations, fileAnnotationSet.FileAnnotations()...)
			} else {
				return nil, err
			}
		}
	}
	return allFileAnnotations, nil
}
