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

package filewatch

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/require"
)

const (
	eventTimeout   = 10 * time.Second
	noEventTimeout = 500 * time.Millisecond
)

func TestWatchDirPath(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	// The file exists before the directory is watched. On macOS, a change to
	// a file right after it is created can be missed, as the kqueue watch of
	// the new file is registered after its creation is reported.
	filePath := filepath.Join(dirPath, "a.proto")
	require.NoError(t, os.WriteFile(filePath, []byte("a"), 0600))
	watcher := newTestWatcher(t)
	require.NoError(t, watcher.AddDirPath(dirPath))

	newFilePath := filepath.Join(dirPath, "b.proto")
	require.NoError(t, os.WriteFile(newFilePath, []byte("b"), 0600))
	requireEventForPath(t, watcher, newFilePath)

	require.NoError(t, os.WriteFile(filePath, []byte("ab"), 0600))
	requireEventForPath(t, watcher, filePath)

	require.NoError(t, os.Remove(filePath))
	requireEventForPath(t, watcher, filePath)
}

func TestWatchDirPathExistingSubDir(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	subDirPath := filepath.Join(dirPath, "sub")
	require.NoError(t, os.Mkdir(subDirPath, 0700))
	watcher := newTestWatcher(t)
	require.NoError(t, watcher.AddDirPath(dirPath))

	filePath := filepath.Join(subDirPath, "a.proto")
	require.NoError(t, os.WriteFile(filePath, []byte("a"), 0600))
	requireEventForPath(t, watcher, filePath)
}

func TestWatchDirPathNewSubDir(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	watcher := newTestWatcher(t)
	require.NoError(t, watcher.AddDirPath(dirPath))

	subDirPath := filepath.Join(dirPath, "sub", "nested")
	require.NoError(t, os.MkdirAll(subDirPath, 0700))
	filePath := filepath.Join(subDirPath, "a.proto")
	require.NoError(t, os.WriteFile(filePath, []byte("a"), 0600))
	requireEventForPath(t, watcher, filePath)

	drainEvents(t, watcher)
	otherFilePath := filepath.Join(subDirPath, "b.proto")
	require.NoError(t, os.WriteFile(otherFilePath, []byte("b"), 0600))
	requireEventForPath(t, watcher, otherFilePath)
}

func TestWatchDirPathRemovedAndRecreated(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	subDirPath := filepath.Join(dirPath, "sub")
	require.NoError(t, os.Mkdir(subDirPath, 0700))
	watcher := newTestWatcher(t)
	require.NoError(t, watcher.AddDirPath(dirPath))

	require.NoError(t, os.RemoveAll(subDirPath))
	event := requireEventForPath(t, watcher, subDirPath)
	require.True(t, event.IsRemovedDir)
	require.NoError(t, os.Mkdir(subDirPath, 0700))
	drainEvents(t, watcher)

	filePath := filepath.Join(subDirPath, "a.proto")
	require.NoError(t, os.WriteFile(filePath, []byte("a"), 0600))
	requireEventForPath(t, watcher, filePath)
	drainEvents(t, watcher)

	// Removing and recreating the root watched directory itself watches it again.
	require.NoError(t, os.RemoveAll(dirPath))
	event = requireEventForPath(t, watcher, dirPath)
	require.True(t, event.IsRemovedDir)
	require.NoError(t, os.MkdirAll(subDirPath, 0700))
	require.NoError(t, os.WriteFile(filePath, []byte("ab"), 0600))
	requireEventForPath(t, watcher, filePath)
}

func TestWatchDirPathNotExistYet(t *testing.T) {
	t.Parallel()
	rootDirPath := t.TempDir()
	dirPath := filepath.Join(rootDirPath, "internal", "proto")
	watcher := newTestWatcher(t)
	require.NoError(t, watcher.AddDirPath(dirPath))

	// Files outside the watched directory are not reported while waiting for its creation.
	require.NoError(t, os.WriteFile(filepath.Join(rootDirPath, "other.proto"), []byte("a"), 0600))
	requireNoEvent(t, watcher)

	subDirPath := filepath.Join(dirPath, "a", "v1")
	require.NoError(t, os.MkdirAll(subDirPath, 0700))
	filePath := filepath.Join(subDirPath, "a.proto")
	require.NoError(t, os.WriteFile(filePath, []byte("a"), 0600))
	requireEventForPath(t, watcher, filePath)
	drainEvents(t, watcher)

	otherFilePath := filepath.Join(subDirPath, "b.proto")
	require.NoError(t, os.WriteFile(otherFilePath, []byte("b"), 0600))
	requireEventForPath(t, watcher, otherFilePath)
}

func TestWatchDirPathSymlink(t *testing.T) {
	t.Parallel()
	targetDirPath, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	symlinkDirPath := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(targetDirPath, symlinkDirPath))
	watcher := newTestWatcher(t)
	require.NoError(t, watcher.AddDirPath(symlinkDirPath))

	// Changes are reported with the resolved path.
	filePath := filepath.Join(targetDirPath, "a.proto")
	require.NoError(t, os.WriteFile(filePath, []byte("a"), 0600))
	requireEventForPath(t, watcher, filePath)
}

func TestWatchDirPathNotExistYetWithOtherRoot(t *testing.T) {
	t.Parallel()
	rootDirPath := t.TempDir()
	dirPath := filepath.Join(rootDirPath, "proto")
	otherDirPath := filepath.Join(t.TempDir(), "other")
	require.NoError(t, os.Mkdir(otherDirPath, 0700))
	watcher := newTestWatcher(t)
	require.NoError(t, watcher.AddDirPath(dirPath))
	require.NoError(t, watcher.AddDirPath(otherDirPath))

	// A change within another root does not disturb the watch for the
	// creation of the missing one.
	require.NoError(t, os.Mkdir(filepath.Join(otherDirPath, "sub"), 0700))
	drainEvents(t, watcher)
	filePath := filepath.Join(dirPath, "a.proto")
	require.NoError(t, os.Mkdir(dirPath, 0700))
	require.NoError(t, os.WriteFile(filePath, []byte("a"), 0600))
	requireEventForPath(t, watcher, filePath)
}

func TestWatchDirPathIgnoresHiddenDirs(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	hiddenDirPath := filepath.Join(dirPath, ".hidden")
	require.NoError(t, os.Mkdir(hiddenDirPath, 0700))
	watcher := newTestWatcher(t)
	require.NoError(t, watcher.AddDirPath(dirPath))

	require.NoError(t, os.WriteFile(filepath.Join(hiddenDirPath, "a.proto"), []byte("a"), 0600))
	requireNoEvent(t, watcher)

	newHiddenDirPath := filepath.Join(dirPath, ".new")
	require.NoError(t, os.Mkdir(newHiddenDirPath, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(newHiddenDirPath, "a.proto"), []byte("a"), 0600))
	requireNoEvent(t, watcher)
}

func TestSetExcludeDirPaths(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	excludedDirPath := filepath.Join(dirPath, "gen")
	require.NoError(t, os.Mkdir(excludedDirPath, 0700))
	watcher := newTestWatcher(t)
	require.NoError(t, watcher.SetExcludeDirPaths(excludedDirPath))
	require.NoError(t, watcher.AddDirPath(dirPath))

	require.NoError(t, os.WriteFile(filepath.Join(excludedDirPath, "a.go"), []byte("a"), 0600))
	requireNoEvent(t, watcher)

	filePath := filepath.Join(dirPath, "a.proto")
	require.NoError(t, os.WriteFile(filePath, []byte("a"), 0600))
	requireEventForPath(t, watcher, filePath)

	// Directories that are no longer excluded are watched again, including
	// subdirectories created while excluded.
	excludedSubDirPath := filepath.Join(excludedDirPath, "sub")
	require.NoError(t, os.Mkdir(excludedSubDirPath, 0700))
	require.NoError(t, watcher.SetExcludeDirPaths())
	excludedSubFilePath := filepath.Join(excludedSubDirPath, "c.proto")
	require.NoError(t, os.WriteFile(excludedSubFilePath, []byte("c"), 0600))
	requireEventForPath(t, watcher, excludedSubFilePath)
	// The creation may be reported again as a write.
	drainEvents(t, watcher)

	// Exclusions apply to directories that are already watched.
	require.NoError(t, watcher.SetExcludeDirPaths(excludedDirPath))
	require.NoError(t, os.WriteFile(excludedSubFilePath, []byte("cd"), 0600))
	requireNoEvent(t, watcher)

	// Watched files are reported even if excluded.
	watchedFilePath := filepath.Join(excludedDirPath, "buf.gen.yaml")
	require.NoError(t, watcher.AddFilePath(watchedFilePath))
	require.NoError(t, os.WriteFile(watchedFilePath, []byte("a"), 0600))
	requireEventForPath(t, watcher, watchedFilePath)
}

func TestSetExcludeDirPathsContainingRoot(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	moduleDirPath := filepath.Join(dirPath, "proto")
	require.NoError(t, os.Mkdir(moduleDirPath, 0700))
	watcher := newTestWatcher(t)
	require.NoError(t, watcher.AddDirPath(moduleDirPath))

	// An excluded directory that contains a watched directory, as with
	// out: . and a module at the root, does not apply.
	require.NoError(t, watcher.SetExcludeDirPaths(dirPath))
	filePath := filepath.Join(moduleDirPath, "a.proto")
	require.NoError(t, os.WriteFile(filePath, []byte("a"), 0600))
	requireEventForPath(t, watcher, filePath)

	// Nor does an excluded directory once it is watched itself.
	genDirPath := filepath.Join(moduleDirPath, "gen")
	require.NoError(t, os.Mkdir(genDirPath, 0700))
	require.NoError(t, watcher.SetExcludeDirPaths(genDirPath))
	drainEvents(t, watcher)
	genFilePath := filepath.Join(genDirPath, "b.proto")
	require.NoError(t, os.WriteFile(genFilePath, []byte("b"), 0600))
	requireNoEvent(t, watcher)
	require.NoError(t, watcher.AddDirPath(genDirPath))
	require.NoError(t, os.WriteFile(genFilePath, []byte("bc"), 0600))
	requireEventForPath(t, watcher, genFilePath)
}

func TestSetExcludeDirPathsKeepsWatchedFiles(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	excludedDirPath := filepath.Join(dirPath, "gen")
	require.NoError(t, os.Mkdir(excludedDirPath, 0700))
	watchedFilePath := filepath.Join(excludedDirPath, "buf.gen.yaml")
	require.NoError(t, os.WriteFile(watchedFilePath, []byte("a"), 0600))
	watcher := newTestWatcher(t)
	require.NoError(t, watcher.AddFilePath(watchedFilePath))
	require.NoError(t, watcher.AddDirPath(dirPath))

	// Excluding the directory of a watched file after it was added keeps
	// reporting the file.
	require.NoError(t, watcher.SetExcludeDirPaths(excludedDirPath))
	require.NoError(t, os.WriteFile(filepath.Join(excludedDirPath, "a.proto"), []byte("a"), 0600))
	requireNoEvent(t, watcher)
	require.NoError(t, os.WriteFile(watchedFilePath, []byte("ab"), 0600))
	requireEventForPath(t, watcher, watchedFilePath)
}

func TestWatchFilePath(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	filePath := filepath.Join(dirPath, "buf.gen.yaml")
	watcher := newTestWatcher(t)
	require.NoError(t, watcher.AddFilePath(filePath))

	require.NoError(t, os.WriteFile(filepath.Join(dirPath, "other.yaml"), []byte("a"), 0600))
	requireNoEvent(t, watcher)

	require.NoError(t, os.WriteFile(filePath, []byte("a"), 0600))
	requireEventForPath(t, watcher, filePath)
	// The creation may be reported again as a write.
	drainEvents(t, watcher)

	subDirPath := filepath.Join(dirPath, "sub")
	require.NoError(t, os.Mkdir(subDirPath, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(subDirPath, "a.proto"), []byte("a"), 0600))
	requireNoEvent(t, watcher)
}

func TestWatchFilePathParentRecreated(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	subDirPath := filepath.Join(dirPath, "sub")
	require.NoError(t, os.Mkdir(subDirPath, 0700))
	filePath := filepath.Join(subDirPath, "buf.yaml")
	watcher := newTestWatcher(t)
	require.NoError(t, watcher.AddFilePath(filePath))

	// The underlying watch is lost with the directory, and is established
	// again once the directory is recreated.
	require.NoError(t, os.RemoveAll(subDirPath))
	drainEvents(t, watcher)
	require.NoError(t, os.Mkdir(subDirPath, 0700))
	require.NoError(t, os.WriteFile(filePath, []byte("a"), 0600))
	requireEventForPath(t, watcher, filePath)
}

func TestRun(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	watcher := newTestWatcher(t)
	require.NoError(t, watcher.AddDirPath(dirPath))
	ctx, cancel := context.WithCancel(t.Context())
	changesChan := make(chan []string)
	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan)
		Run(
			ctx,
			slog.New(slog.DiscardHandler),
			watcher,
			noEventTimeout,
			func(event Event) bool { return filepath.Ext(event.Path) == ".proto" },
			func(_ context.Context, changedPaths []string) { changesChan <- changedPaths },
		)
	}()

	// Changes within the debounce duration are reported together, sorted
	// and without duplicates, and irrelevant changes are ignored.
	bFilePath := filepath.Join(dirPath, "b.proto")
	aFilePath := filepath.Join(dirPath, "a.proto")
	require.NoError(t, os.WriteFile(bFilePath, []byte("b"), 0600))
	require.NoError(t, os.WriteFile(aFilePath, []byte("a"), 0600))
	require.NoError(t, os.WriteFile(aFilePath, []byte("ab"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dirPath, "README.md"), []byte("a"), 0600))
	select {
	case changedPaths := <-changesChan:
		require.Equal(t, []string{aFilePath, bFilePath}, changedPaths)
	case <-time.After(eventTimeout):
		require.Fail(t, "timed out waiting for changes")
	}

	cancel()
	select {
	case <-doneChan:
	case <-time.After(eventTimeout):
		require.Fail(t, "timed out waiting for Run to return")
	}
}

func TestIgnoresEventWithoutName(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	watcher, err := newWatcher(slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, watcher.Close())
	})
	require.NoError(t, watcher.AddDirPath(dirPath))
	require.NoError(t, watcher.SetExcludeDirPaths(filepath.Join(dirPath, "gen")))

	// kqueue may report such an event, whose relative path must not be
	// matched against the excluded directories, as that never terminates.
	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan)
		watcher.handleFsnotifyEvent(fsnotify.Event{Name: "", Op: fsnotify.Create})
	}()
	select {
	case <-doneChan:
	case <-time.After(eventTimeout):
		require.Fail(t, "timed out handling event without name")
	}
	requireNoEvent(t, watcher)
}

func TestReadsWhileEventsNotReceived(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	watcher, err := newWatcher(slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, watcher.Close())
	})
	require.NoError(t, watcher.AddDirPath(dirPath))

	// Events from fsnotify are still received while Events is not read, as
	// fsnotify may wait for them to be received before adding a watch.
	for i := range 2 * cap(watcher.eventChan) {
		require.NoError(t, os.WriteFile(filepath.Join(dirPath, fmt.Sprintf("%d.proto", i)), nil, 0600))
	}
	require.Eventually(
		t,
		func() bool {
			watcher.queueLock.Lock()
			defer watcher.queueLock.Unlock()
			return len(watcher.queue) > 0
		},
		eventTimeout,
		10*time.Millisecond,
	)
	require.NoError(t, watcher.AddDirPath(t.TempDir()))
}

func TestEnqueueOverflow(t *testing.T) {
	t.Parallel()
	watcher := &watcher{queueChan: make(chan struct{}, 1)}
	for range maxQueuedFsnotifyEvents + 1 {
		watcher.enqueue(fsnotifyItem{event: fsnotify.Event{Name: "/a.proto", Op: fsnotify.Write}})
	}
	items, overflowed := watcher.dequeueAll()
	require.Len(t, items, maxQueuedFsnotifyEvents)
	require.True(t, overflowed)
	items, overflowed = watcher.dequeueAll()
	require.Empty(t, items)
	require.False(t, overflowed)
}

func TestClose(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	watcher, err := NewWatcher(slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	require.NoError(t, watcher.AddDirPath(dirPath))
	require.NoError(t, watcher.Close())
	require.NoError(t, watcher.Close())
	_, ok := <-watcher.Events()
	require.False(t, ok)
	_, ok = <-watcher.Errors()
	require.False(t, ok)
}

func newTestWatcher(t *testing.T) Watcher {
	watcher, err := NewWatcher(slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, watcher.Close())
	})
	return watcher
}

// requireEventForPath waits until an Event for the given path is received,
// discarding Events for other paths.
func requireEventForPath(t *testing.T, watcher Watcher, path string) Event {
	timer := time.NewTimer(eventTimeout)
	defer timer.Stop()
	for {
		select {
		case event, ok := <-watcher.Events():
			require.True(t, ok, "watcher closed while waiting for event for %s", path)
			if event.Path == path {
				return event
			}
		case err := <-watcher.Errors():
			require.NoError(t, err)
		case <-timer.C:
			require.Fail(t, "timed out waiting for event", "path: %s", path)
		}
	}
}

// requireNoEvent requires that no Event is received for a short time.
func requireNoEvent(t *testing.T, watcher Watcher) {
	timer := time.NewTimer(noEventTimeout)
	defer timer.Stop()
	select {
	case event := <-watcher.Events():
		require.Fail(t, "unexpected event", "path: %s", event.Path)
	case err := <-watcher.Errors():
		require.NoError(t, err)
	case <-timer.C:
	}
}

// drainEvents discards any Events received for a short time.
func drainEvents(t *testing.T, watcher Watcher) {
	timer := time.NewTimer(noEventTimeout)
	defer timer.Stop()
	for {
		select {
		case <-watcher.Events():
		case err := <-watcher.Errors():
			require.NoError(t, err)
		case <-timer.C:
			return
		}
	}
}
