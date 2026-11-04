// Copyright 2020-2026 Buf Technologies, Inc.
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

package generate

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"buf.build/go/standard/xlog/xslog"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/buftarget"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/filewatch"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/osext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
)

const (
	// watchDebounceDuration is how long to wait after the last change before regenerating.
	watchDebounceDuration = 100 * time.Millisecond
	// watchMaxLoggedChanges is the maximum number of changed paths logged before a run.
	watchMaxLoggedChanges = 10
)

// runWatch runs generation whenever a relevant file changes, until ctx is done.
func runWatch(ctx context.Context, container appext.Container, runner *runner) error {
	templateFilePath, err := getTemplateFilePath(runner.flags.Template)
	if err != nil {
		return err
	}
	configFilePaths := []string{templateFilePath}
	if isConfigFilePath(runner.flags.Config) {
		configFilePath, err := filepath.Abs(runner.flags.Config)
		if err != nil {
			return err
		}
		configFilePaths = append(configFilePaths, configFilePath)
	}
	watcher, err := filewatch.NewWatcher(container.Logger())
	if err != nil {
		return err
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			container.Logger().WarnContext(ctx, "close file watcher", xslog.ErrorAttr(err))
		}
	}()
	watchRunner := &watchRunner{
		container:       container,
		runner:          runner,
		watcher:         watcher,
		configFilePaths: slices.DeleteFunc(configFilePaths, func(path string) bool { return path == "" }),
	}
	return watchRunner.run(ctx)
}

// watchRunner runs generation and maintains the set of watched paths: the
// module directories and configuration files of the workspaces of the local
// inputs, and the template and config files given as flags.
type watchRunner struct {
	container appext.Container
	runner    *runner
	watcher   filewatch.Watcher
	// configFilePaths are the absolute paths of the template and config files given as flags.
	configFilePaths []string
	// inputFilePaths are the sorted absolute paths of the local files of the
	// images last built, which are valid if hasInputFilePaths is true.
	inputFilePaths    []string
	hasInputFilePaths bool
}

func (w *watchRunner) run(ctx context.Context) error {
	for _, configFilePath := range w.configFilePaths {
		if err := w.watcher.AddFilePath(configFilePath); err != nil {
			return err
		}
	}
	bufGenYAMLFile, inputDirPaths, err := w.readTemplate(ctx)
	if err != nil {
		return err
	}
	if len(inputDirPaths) == 0 {
		return appcmd.NewInvalidArgumentErrorf(
			"--%s requires at least one input that is a local directory or .proto file",
			watchFlagName,
		)
	}
	w.addInputDirPaths(ctx, inputDirPaths)
	w.generate(ctx, bufGenYAMLFile)
	if ctx.Err() == nil {
		w.container.Logger().InfoContext(ctx, "watching for changes")
	}
	filewatch.Run(
		ctx,
		w.container.Logger(),
		w.watcher,
		watchDebounceDuration,
		func(event filewatch.Event) bool {
			return isRelevantEvent(w.configFilePaths, event)
		},
		func(ctx context.Context, changedPaths []string) {
			logChanges(ctx, w.container.Logger(), changedPaths)
			w.runOnce(ctx)
		},
	)
	return nil
}

func (w *watchRunner) runOnce(ctx context.Context) {
	bufGenYAMLFile, inputDirPaths, err := w.readTemplate(ctx)
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		logGenerationError(ctx, w.container.Logger(), err)
		return
	}
	w.addInputDirPaths(ctx, inputDirPaths)
	w.generate(ctx, bufGenYAMLFile)
}

// readTemplate reads the template, and returns it and the directories of the local inputs.
func (w *watchRunner) readTemplate(ctx context.Context) (bufconfig.BufGenYAMLFile, []string, error) {
	bufGenYAMLFile, err := readBufGenYAMLFile(ctx, w.runner.storageosProvider, w.runner.flags.Template)
	if err != nil {
		return nil, nil, err
	}
	inputDirPaths, err := getLocalInputDirPaths(ctx, w.container.Logger(), w.runner.input, bufGenYAMLFile)
	if err != nil {
		return nil, nil, err
	}
	return bufGenYAMLFile, inputDirPaths, nil
}

func (w *watchRunner) generate(ctx context.Context, bufGenYAMLFile bufconfig.BufGenYAMLFile) {
	images, imagesErr := w.runner.getImages(ctx, bufGenYAMLFile)
	if ctx.Err() != nil {
		return
	}
	if imagesErr == nil {
		inputFilePaths, err := getLocalFilePaths(images)
		if err != nil {
			logGenerationError(ctx, w.container.Logger(), err)
			return
		}
		w.inputFilePaths = inputFilePaths
		w.hasInputFilePaths = true
	}
	if err := w.updateExcludeDirPaths(bufGenYAMLFile); err != nil {
		logGenerationError(ctx, w.container.Logger(), err)
		return
	}
	if imagesErr != nil {
		logGenerationError(ctx, w.container.Logger(), imagesErr)
		return
	}
	err := w.runner.generate(ctx, w.container, bufGenYAMLFile, images)
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		logGenerationError(ctx, w.container.Logger(), err)
		return
	}
	w.container.Logger().InfoContext(ctx, "generation succeeded")
}

// updateExcludeDirPaths excludes the output directories of the plugins from
// watching, except those that contain input files. Nothing is excluded until
// the images have been built, so that a fix to an input file is not missed.
func (w *watchRunner) updateExcludeDirPaths(bufGenYAMLFile bufconfig.BufGenYAMLFile) error {
	if !w.hasInputFilePaths {
		return w.watcher.SetExcludeDirPaths()
	}
	pluginOutPaths, err := getPluginOutPaths(bufGenYAMLFile, w.runner.flags.BaseOutDirPath)
	if err != nil {
		return err
	}
	return w.watcher.SetExcludeDirPaths(getExcludeDirPaths(pluginOutPaths, w.inputFilePaths)...)
}

// addInputDirPaths watches the modules declared by the workspaces of the
// inputs, or the inputs themselves if not controlled by a workspace.
func (w *watchRunner) addInputDirPaths(ctx context.Context, inputDirPaths []string) {
	for _, inputDirPath := range inputDirPaths {
		if !w.addWorkspacePaths(ctx, inputDirPath) {
			w.addDirPaths(ctx, []string{inputDirPath})
		}
	}
}

func (w *watchRunner) addDirPaths(ctx context.Context, dirPaths []string) {
	for _, dirPath := range dirPaths {
		if err := w.watcher.AddDirPath(dirPath); err != nil {
			w.container.Logger().WarnContext(ctx, "cannot watch directory", xslog.ErrorAttr(err))
		}
	}
}

// addWorkspacePaths watches the configuration files and module directories
// of the workspace that controls dirPath.
//
// It returns false if dirPath itself must be watched instead.
func (w *watchRunner) addWorkspacePaths(ctx context.Context, dirPath string) bool {
	workspaceDirPath, controllingWorkspace, err := getControllingWorkspace(ctx, w.container.Logger(), w.runner.storageosProvider, dirPath)
	if err != nil {
		w.container.Logger().WarnContext(ctx, "cannot find controlling workspace", xslog.ErrorAttr(err))
		return false
	}
	if controllingWorkspace == nil {
		return false
	}
	for _, fileName := range []string{
		bufconfig.DefaultBufYAMLFileName,
		bufconfig.DefaultBufLockFileName,
		bufconfig.DefaultBufWorkYAMLFileName,
	} {
		if err := w.watcher.AddFilePath(filepath.Join(workspaceDirPath, fileName)); err != nil {
			w.container.Logger().WarnContext(ctx, "cannot watch file", xslog.ErrorAttr(err))
		}
	}
	moduleDirPaths := getWorkspaceModuleDirPaths(workspaceDirPath, controllingWorkspace, w.getConfigOverrideBufYAMLFile(ctx))
	if len(moduleDirPaths) == 0 {
		return false
	}
	w.addDirPaths(ctx, moduleDirPaths)
	return true
}

// getConfigOverrideBufYAMLFile returns the buf.yaml given with the config flag,
// or nil if there is none or it cannot be read.
func (w *watchRunner) getConfigOverrideBufYAMLFile(ctx context.Context) bufconfig.BufYAMLFile {
	if w.runner.flags.Config == "" {
		return nil
	}
	bufYAMLFile, err := bufconfig.GetBufYAMLFileForOverride(w.runner.flags.Config)
	if err != nil {
		w.container.Logger().DebugContext(ctx, "cannot read config override", xslog.ErrorAttr(err))
		return nil
	}
	return bufYAMLFile
}

// isRelevantEvent returns true if the change should result in regeneration.
func isRelevantEvent(configFilePaths []string, event filewatch.Event) bool {
	if event.IsRemovedDir {
		// Changes to the files within it may not be reported.
		return true
	}
	path := event.Path
	if slices.Contains(configFilePaths, path) {
		return true
	}
	switch filepath.Base(path) {
	case bufconfig.DefaultBufYAMLFileName,
		bufconfig.DefaultBufLockFileName,
		bufconfig.DefaultBufWorkYAMLFileName:
		return true
	}
	return filepath.Ext(path) == ".proto"
}

// getTemplateFilePath returns the absolute path of the template file, or empty
// if the template is inline data.
func getTemplateFilePath(template string) (string, error) {
	switch {
	case template == "":
		return filepath.Abs(bufconfig.DefaultBufGenYAMLFileName)
	case isConfigFilePath(template):
		return filepath.Abs(template)
	default:
		return "", nil
	}
}

// getLocalInputDirPaths returns the absolute directories of the inputs that
// are local directories or .proto files.
func getLocalInputDirPaths(
	ctx context.Context,
	logger *slog.Logger,
	inputSpecified string,
	bufGenYAMLFile bufconfig.BufGenYAMLFile,
) ([]string, error) {
	refParser := buffetch.NewRefParser(logger)
	var refs []buffetch.Ref
	if inputSpecified != "" || len(bufGenYAMLFile.InputConfigs()) == 0 {
		input := "."
		if inputSpecified != "" {
			input = inputSpecified
		}
		ref, err := refParser.GetRef(ctx, input)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	} else {
		for _, inputConfig := range bufGenYAMLFile.InputConfigs() {
			ref, err := refParser.GetRefForInputConfig(ctx, inputConfig)
			if err != nil {
				return nil, err
			}
			refs = append(refs, ref)
		}
	}
	var dirPaths []string
	for _, ref := range refs {
		var dirPath string
		switch ref := ref.(type) {
		case buffetch.ProtoFileRef:
			if ref.IsDevPath() {
				continue
			}
			dirPath = filepath.Dir(ref.ProtoFilePath())
		case buffetch.DirRef:
			dirPath = ref.DirPath()
		default:
			continue
		}
		absDirPath, err := filepath.Abs(dirPath)
		if err != nil {
			return nil, err
		}
		dirPaths = append(dirPaths, absDirPath)
	}
	return dirPaths, nil
}

// getWorkspaceModuleDirPaths returns the absolute directories of the modules
// that the workspace declares, or that overrideBufYAMLFile declares if not nil.
// A v1 override makes the input a single module, so no directories are returned for it.
func getWorkspaceModuleDirPaths(
	workspaceDirPath string,
	controllingWorkspace buftarget.ControllingWorkspace,
	overrideBufYAMLFile bufconfig.BufYAMLFile,
) []string {
	var relDirPaths []string
	if overrideBufYAMLFile != nil {
		if overrideBufYAMLFile.FileVersion() == bufconfig.FileVersionV2 {
			for _, moduleConfig := range overrideBufYAMLFile.ModuleConfigs() {
				relDirPaths = append(relDirPaths, moduleConfig.DirPath())
			}
		}
	} else if bufWorkYAMLFile := controllingWorkspace.BufWorkYAMLFile(); bufWorkYAMLFile != nil {
		relDirPaths = bufWorkYAMLFile.DirPaths()
	} else if bufYAMLFile := controllingWorkspace.BufYAMLFile(); bufYAMLFile != nil {
		for _, moduleConfig := range bufYAMLFile.ModuleConfigs() {
			relDirPaths = append(relDirPaths, moduleConfig.DirPath())
		}
	}
	dirPaths := make([]string, 0, len(relDirPaths))
	for _, relDirPath := range relDirPaths {
		dirPaths = append(dirPaths, filepath.Join(workspaceDirPath, normalpath.Unnormalize(relDirPath)))
	}
	return dirPaths
}

// getControllingWorkspace returns the absolute directory of the workspace
// that controls dirPath, and the workspace, which is nil if there is none.
func getControllingWorkspace(
	ctx context.Context,
	logger *slog.Logger,
	storageosProvider storageos.Provider,
	dirPath string,
) (string, buftarget.ControllingWorkspace, error) {
	absDirPath, err := normalpath.NormalizeAndAbsolute(dirPath)
	if err != nil {
		return "", nil, err
	}
	fsRoot := normalpath.Components(absDirPath)[0]
	fsRelDirPath, err := normalpath.Rel(fsRoot, absDirPath)
	if err != nil {
		return "", nil, err
	}
	fsRootBucket, err := storageosProvider.NewReadWriteBucket(fsRoot, storageos.ReadWriteBucketWithSymlinksIfSupported())
	if err != nil {
		return "", nil, err
	}
	bucketTargeting, err := buftarget.NewBucketTargeting(
		ctx,
		logger,
		fsRootBucket,
		fsRelDirPath,
		nil,
		nil,
		buftarget.TerminateAtControllingWorkspace,
	)
	if err != nil {
		return "", nil, err
	}
	controllingWorkspace := bucketTargeting.ControllingWorkspace()
	if controllingWorkspace == nil {
		return "", nil, nil
	}
	return normalpath.Unnormalize(normalpath.Join(fsRoot, controllingWorkspace.Path())), controllingWorkspace, nil
}

// getPluginOutPaths returns the absolute paths that each plugin writes to.
func getPluginOutPaths(bufGenYAMLFile bufconfig.BufGenYAMLFile, baseOutDirPath string) ([]string, error) {
	pluginConfigs := bufGenYAMLFile.GenerateConfig().GeneratePluginConfigs()
	outPaths := make([]string, 0, len(pluginConfigs))
	for _, pluginConfig := range pluginConfigs {
		out := pluginConfig.Out()
		if baseOutDirPath != "" && baseOutDirPath != "." {
			out = filepath.Join(baseOutDirPath, out)
		}
		absOut, err := filepath.Abs(out)
		if err != nil {
			return nil, err
		}
		outPaths = append(outPaths, absOut)
	}
	return outPaths, nil
}

// getLocalFilePaths returns the sorted absolute paths of the files of the
// images that are on disk.
func getLocalFilePaths(images []bufimage.Image) ([]string, error) {
	var filePaths []string
	for _, image := range images {
		for _, imageFile := range image.Files() {
			localPath := imageFile.LocalPath()
			if localPath == "" {
				continue
			}
			filePath, err := filepath.Abs(localPath)
			if err != nil {
				return nil, err
			}
			filePaths = append(filePaths, filePath)
		}
	}
	slices.Sort(filePaths)
	return slices.Compact(filePaths), nil
}

// getExcludeDirPaths returns the output directories that contain none of the
// sorted input files.
func getExcludeDirPaths(pluginOutPaths []string, inputFilePaths []string) []string {
	var excludeDirPaths []string
	for _, pluginOutPath := range pluginOutPaths {
		prefix := pluginOutPath + string(filepath.Separator)
		if strings.HasSuffix(pluginOutPath, string(filepath.Separator)) {
			prefix = pluginOutPath
		}
		// The files with the prefix, if any, start at its insertion point.
		index, _ := slices.BinarySearch(inputFilePaths, prefix)
		if index < len(inputFilePaths) && strings.HasPrefix(inputFilePaths[index], prefix) {
			continue
		}
		excludeDirPaths = append(excludeDirPaths, pluginOutPath)
	}
	return excludeDirPaths
}

func logChanges(ctx context.Context, logger *slog.Logger, paths []string) {
	for i, path := range paths {
		if i == watchMaxLoggedChanges {
			logger.InfoContext(ctx, "more changes detected", slog.Int("count", len(paths)-i))
			return
		}
		logger.InfoContext(ctx, "change detected", slog.String("path", displayPath(path)))
	}
}

func logGenerationError(ctx context.Context, logger *slog.Logger, err error) {
	if errors.Is(err, bufctl.ErrFileAnnotation) {
		// The annotations were already printed.
		logger.ErrorContext(ctx, "generation failed")
		return
	}
	logger.ErrorContext(ctx, "generation failed", xslog.ErrorAttr(err))
}

// displayPath returns the path relative to the working directory if it is within it.
func displayPath(absPath string) string {
	pwd, err := osext.Getwd()
	if err != nil {
		return absPath
	}
	relPath, err := filepath.Rel(pwd, absPath)
	if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return absPath
	}
	return relPath
}
