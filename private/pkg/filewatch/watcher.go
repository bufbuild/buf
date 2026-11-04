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
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"buf.build/go/standard/xlog/xslog"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/fsnotify/fsnotify"
)

// maxQueuedFsnotifyEvents is the maximum number of events from fsnotifyWatcher
// that are queued for handling before further events are dropped.
const maxQueuedFsnotifyEvents = 4096

// watcher keeps the set of directories watched in fsnotifyWatcher in sync with
// the directories and files it was asked to watch.
//
// The directories and files asked for are the only state. The directories to
// watch are derived from them by reconcile, which runs after every change to
// them and after every change to a directory on disk.
type watcher struct {
	logger          *slog.Logger
	fsnotifyWatcher *fsnotify.Watcher

	eventChan chan Event
	errChan   chan error
	// closeChan is closed by Close so that run stops sending on eventChan and errChan.
	closeChan chan struct{}
	closeOnce sync.Once
	closeErr  error
	waitGroup sync.WaitGroup

	// queueLock guards queue and queueOverflowed. Events from fsnotifyWatcher
	// are queued by read, so that it keeps receiving while run handles them, as
	// fsnotifyWatcher.Add and Remove may wait for its events to be received.
	queueLock       sync.Mutex
	queue           []fsnotifyItem
	queueOverflowed bool
	// queueChan is signaled when an item is queued.
	queueChan chan struct{}
	// readDoneChan is closed by read once fsnotifyWatcher is closed.
	readDoneChan chan struct{}

	// lock is held while reconciling, and is never held while sending on
	// eventChan or errChan, as the receiver may be calling a method.
	lock sync.Mutex
	// rootDirPaths are the absolute directories added with AddDirPath.
	rootDirPaths map[string]struct{}
	// filePaths are the absolute files added with AddFilePath.
	filePaths map[string]struct{}
	// excludeDirPaths are the directories given to SetExcludeDirPaths, in
	// normalpath form for use with normalpath.EqualsOrContainsPath.
	excludeDirPaths []string
	// watchedDirPaths are the directories currently watched in fsnotifyWatcher.
	watchedDirPaths map[string]struct{}
}

func newWatcher(logger *slog.Logger) (*watcher, error) {
	fsnotifyWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create file watcher: %w", err)
	}
	watcher := &watcher{
		logger:          logger,
		fsnotifyWatcher: fsnotifyWatcher,
		eventChan:       make(chan Event, 64),
		errChan:         make(chan error),
		closeChan:       make(chan struct{}),
		queueChan:       make(chan struct{}, 1),
		readDoneChan:    make(chan struct{}),
		rootDirPaths:    make(map[string]struct{}),
		filePaths:       make(map[string]struct{}),
		watchedDirPaths: make(map[string]struct{}),
	}
	watcher.waitGroup.Go(watcher.read)
	watcher.waitGroup.Go(watcher.run)
	return watcher, nil
}

func (w *watcher) AddDirPath(dirPath string) error {
	absDirPath, err := filepath.Abs(dirPath)
	if err != nil {
		return err
	}
	// WalkDir does not follow a root directory that is a symbolic link.
	fileInfo, err := os.Lstat(absDirPath)
	if err == nil && fileInfo.Mode()&fs.ModeSymlink != 0 {
		absDirPath, err = filepath.EvalSymlinks(absDirPath)
		if err != nil {
			return err
		}
	}
	w.lock.Lock()
	defer w.lock.Unlock()
	if _, ok := w.rootDirPaths[absDirPath]; ok {
		return nil
	}
	w.rootDirPaths[absDirPath] = struct{}{}
	if _, _, err := w.reconcileLocked(false, ""); err != nil {
		delete(w.rootDirPaths, absDirPath)
		return err
	}
	return nil
}

func (w *watcher) AddFilePath(filePath string) error {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}
	w.lock.Lock()
	defer w.lock.Unlock()
	if _, ok := w.filePaths[absFilePath]; ok {
		return nil
	}
	w.filePaths[absFilePath] = struct{}{}
	if _, _, err := w.reconcileLocked(false, ""); err != nil {
		delete(w.filePaths, absFilePath)
		return err
	}
	return nil
}

func (w *watcher) SetExcludeDirPaths(dirPaths ...string) error {
	excludeDirPaths := make([]string, 0, len(dirPaths))
	for _, dirPath := range dirPaths {
		absDirPath, err := filepath.Abs(dirPath)
		if err != nil {
			return err
		}
		excludeDirPaths = append(excludeDirPaths, normalpath.Normalize(absDirPath))
	}
	slices.Sort(excludeDirPaths)
	w.lock.Lock()
	defer w.lock.Unlock()
	excludeDirPaths = slices.Compact(excludeDirPaths)
	if slices.Equal(w.excludeDirPaths, excludeDirPaths) {
		return nil
	}
	w.excludeDirPaths = excludeDirPaths
	_, _, err := w.reconcileLocked(false, "")
	return err
}

func (w *watcher) Events() <-chan Event {
	return w.eventChan
}

func (w *watcher) Errors() <-chan error {
	return w.errChan
}

func (w *watcher) Close() error {
	w.closeOnce.Do(func() {
		close(w.closeChan)
		w.closeErr = w.fsnotifyWatcher.Close()
		w.waitGroup.Wait()
	})
	return w.closeErr
}

// read queues the events and errors from fsnotifyWatcher until it is closed.
func (w *watcher) read() {
	defer close(w.readDoneChan)
	fsnotifyEventChan := w.fsnotifyWatcher.Events
	fsnotifyErrChan := w.fsnotifyWatcher.Errors
	for fsnotifyEventChan != nil || fsnotifyErrChan != nil {
		select {
		case fsnotifyEvent, ok := <-fsnotifyEventChan:
			if !ok {
				fsnotifyEventChan = nil
				continue
			}
			w.enqueue(fsnotifyItem{event: fsnotifyEvent})
		case err, ok := <-fsnotifyErrChan:
			if !ok {
				fsnotifyErrChan = nil
				continue
			}
			w.enqueue(fsnotifyItem{err: err})
		case <-w.closeChan:
			return
		}
	}
}

// run handles the queued events and errors until the watcher is closed.
func (w *watcher) run() {
	defer close(w.eventChan)
	defer close(w.errChan)
	for {
		select {
		case <-w.queueChan:
		case <-w.readDoneChan:
			return
		case <-w.closeChan:
			return
		}
		items, overflowed := w.dequeueAll()
		if overflowed {
			w.sendError(ErrEventOverflow)
		}
		for _, item := range items {
			if item.err != nil {
				w.sendError(item.err)
			} else {
				w.handleFsnotifyEvent(item.event)
			}
		}
	}
}

func (w *watcher) enqueue(item fsnotifyItem) {
	w.queueLock.Lock()
	if len(w.queue) < maxQueuedFsnotifyEvents {
		w.queue = append(w.queue, item)
	} else {
		w.queueOverflowed = true
	}
	w.queueLock.Unlock()
	select {
	case w.queueChan <- struct{}{}:
	default:
	}
}

// dequeueAll returns the queued items, and whether any were dropped.
func (w *watcher) dequeueAll() ([]fsnotifyItem, bool) {
	w.queueLock.Lock()
	defer w.queueLock.Unlock()
	items, overflowed := w.queue, w.queueOverflowed
	w.queue, w.queueOverflowed = nil, false
	return items, overflowed
}

func (w *watcher) handleFsnotifyEvent(fsnotifyEvent fsnotify.Event) {
	path := filepath.Clean(fsnotifyEvent.Name)
	// kqueue may report an event without a name for a descriptor it no longer knows.
	if fsnotifyEvent.Op == fsnotify.Chmod || !filepath.IsAbs(path) {
		return
	}
	switch {
	case fsnotifyEvent.Has(fsnotify.Create):
		if fileInfo, err := os.Stat(path); err == nil && fileInfo.IsDir() {
			// The directory is reported through the files within it.
			w.reconcile(path)
			return
		}
	case fsnotifyEvent.Has(fsnotify.Remove), fsnotifyEvent.Has(fsnotify.Rename):
		if w.isWatchedDirPath(path) {
			// Reported by reconcile, which may also have reported it already
			// for a directory that was removed along with it.
			w.reconcile(path)
			return
		}
	}
	if w.isReportedPath(path) {
		w.sendEvent(Event{Path: path})
	}
}

// reconcile updates the watched directories after a change to the directory
// at path. It reports the watched directories that were removed, and the
// files within the directories that are newly watched, as they may have been
// created before the directory was watched.
func (w *watcher) reconcile(path string) {
	w.lock.Lock()
	removedDirPaths, newDirPaths, err := w.reconcileLocked(true, path)
	w.lock.Unlock()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		w.sendError(err)
	}
	for _, dirPath := range removedDirPaths {
		w.sendEvent(Event{Path: dirPath, IsRemovedDir: true})
	}
	for _, dirPath := range newDirPaths {
		dirEntries, err := os.ReadDir(dirPath)
		if err != nil {
			// Removed again already, which is reported by its own event.
			continue
		}
		for _, dirEntry := range dirEntries {
			if dirEntry.IsDir() {
				continue
			}
			if path := filepath.Join(dirPath, dirEntry.Name()); w.isReportedPath(path) {
				w.sendEvent(Event{Path: path})
			}
		}
	}
}

// reconcileLocked watches the directories that should be watched, and stops
// watching the others.
//
// Watches are dropped by fsnotifyWatcher when their directories are removed
// or renamed. If sync is true, as when called for an event, the watched
// directories are first synced with fsnotifyWatcher, so that a directory
// created again is watched again, and the reported directories that were
// removed are returned. Otherwise, a watched directory that no longer exists
// is left for its event to report.
//
// If scopePath is not empty, only the roots and files related to it, by one
// containing the other, are walked, and the directories watched for the
// others are kept as they are.
//
// It returns the removed directories and the newly watched directories.
func (w *watcher) reconcileLocked(sync bool, scopePath string) ([]string, []string, error) {
	var previouslyWatchedDirPaths map[string]struct{}
	if sync {
		previouslyWatchedDirPaths = w.watchedDirPaths
		w.watchedDirPaths = make(map[string]struct{})
		for _, dirPath := range w.fsnotifyWatcher.WatchList() {
			w.watchedDirPaths[filepath.Clean(dirPath)] = struct{}{}
		}
	}
	var errs []error
	var newDirPaths []string
	desiredDirPaths := make(map[string]struct{})
	activeExcludeDirPaths := w.activeExcludeDirPathsLocked()
	var normalizedUnrelatedDirPaths []string
	normalizedScopePath := normalpath.Normalize(scopePath)
	isInScope := func(dirPath string) bool {
		normalizedDirPath := normalpath.Normalize(dirPath)
		if scopePath == "" || isRelatedNormalizedPath(normalizedDirPath, normalizedScopePath) {
			return true
		}
		normalizedUnrelatedDirPaths = append(normalizedUnrelatedDirPaths, normalizedDirPath)
		return false
	}
	for _, rootDirPath := range slices.Sorted(maps.Keys(w.rootDirPaths)) {
		if !isInScope(rootDirPath) {
			continue
		}
		if err := w.watchDirPathLocked(rootDirPath, true, activeExcludeDirPaths, desiredDirPaths, &newDirPaths); err != nil {
			errs = append(errs, err)
		}
	}
	for _, filePath := range slices.Sorted(maps.Keys(w.filePaths)) {
		fileDirPath := filepath.Dir(filePath)
		if !isInScope(fileDirPath) {
			continue
		}
		if err := w.watchDirPathLocked(fileDirPath, false, nil, desiredDirPaths, &newDirPaths); err != nil {
			errs = append(errs, err)
		}
	}
	// Nothing changed for the roots and files out of scope, so the
	// directories watched for them, including ancestors watched for their
	// creation, stay as they are.
	for dirPath := range w.watchedDirPaths {
		normalizedDirPath := normalpath.Normalize(dirPath)
		if slices.ContainsFunc(normalizedUnrelatedDirPaths, func(normalizedUnrelatedDirPath string) bool {
			return isRelatedNormalizedPath(normalizedDirPath, normalizedUnrelatedDirPath)
		}) {
			desiredDirPaths[dirPath] = struct{}{}
		}
	}
	for dirPath := range w.watchedDirPaths {
		if _, ok := desiredDirPaths[dirPath]; ok {
			continue
		}
		if !sync {
			if _, err := os.Stat(dirPath); err != nil {
				continue
			}
		}
		// The watch is already gone if the directory was removed.
		if err := w.fsnotifyWatcher.Remove(dirPath); err != nil && !errors.Is(err, fsnotify.ErrNonExistentWatch) {
			w.logger.Debug("cannot unwatch directory", slog.String("path", dirPath), xslog.ErrorAttr(err))
		}
		delete(w.watchedDirPaths, dirPath)
	}
	var removedDirPaths []string
	for dirPath := range previouslyWatchedDirPaths {
		if _, ok := desiredDirPaths[dirPath]; ok {
			continue
		}
		if _, err := os.Stat(dirPath); err == nil || !w.isReportedPathLocked(dirPath) {
			continue
		}
		removedDirPaths = append(removedDirPaths, dirPath)
	}
	slices.Sort(removedDirPaths)
	return removedDirPaths, newDirPaths, errors.Join(errs...)
}

// watchDirPathLocked watches dirPath, and its subdirectories if recursive, or
// watches its nearest existing ancestor for its creation if it does not exist.
//
// Every directory that should be watched is added to desiredDirPaths, and
// every directory that was not watched before is appended to newDirPaths.
func (w *watcher) watchDirPathLocked(
	dirPath string,
	recursive bool,
	activeExcludeDirPaths []string,
	desiredDirPaths map[string]struct{},
	newDirPaths *[]string,
) error {
	for {
		err := w.walkDirPathLocked(dirPath, recursive, activeExcludeDirPaths, desiredDirPaths, newDirPaths)
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		ancestorDirPath := nearestExistingAncestorDirPath(dirPath)
		if ancestorDirPath == "" {
			return nil
		}
		_, wasWatched := w.watchedDirPaths[ancestorDirPath]
		if err := w.addWatchLocked(ancestorDirPath, desiredDirPaths, newDirPaths); err != nil {
			return err
		}
		if wasWatched {
			// Its creation will be reported.
			return nil
		}
		// It may have been created before the ancestor was watched.
	}
}

// walkDirPathLocked watches dirPath and, if recursive, its subdirectories
// that are not hidden or excluded. Each directory is watched before it is
// read, so that no subdirectory created in the meantime is missed.
func (w *watcher) walkDirPathLocked(
	dirPath string,
	recursive bool,
	activeExcludeDirPaths []string,
	desiredDirPaths map[string]struct{},
	newDirPaths *[]string,
) error {
	return filepath.WalkDir(
		dirPath,
		func(path string, dirEntry fs.DirEntry, err error) error {
			if err != nil {
				if path == dirPath {
					return err
				}
				if !errors.Is(err, fs.ErrNotExist) {
					w.logger.Warn("skipping unreadable directory", slog.String("path", path), xslog.ErrorAttr(err))
				}
				return filepath.SkipDir
			}
			if !dirEntry.IsDir() {
				if path == dirPath {
					return fmt.Errorf("watch %s: not a directory", path)
				}
				return nil
			}
			if path != dirPath {
				if !recursive {
					return filepath.SkipDir
				}
				if strings.HasPrefix(dirEntry.Name(), ".") || isExcludedPath(path, activeExcludeDirPaths) {
					return filepath.SkipDir
				}
			}
			if err := w.addWatchLocked(path, desiredDirPaths, newDirPaths); err != nil {
				if path == dirPath {
					return err
				}
				if !errors.Is(err, fs.ErrNotExist) {
					w.logger.Warn("cannot watch directory", slog.String("path", path), xslog.ErrorAttr(err))
				}
				return filepath.SkipDir
			}
			return nil
		},
	)
}

func (w *watcher) addWatchLocked(dirPath string, desiredDirPaths map[string]struct{}, newDirPaths *[]string) error {
	desiredDirPaths[dirPath] = struct{}{}
	if _, ok := w.watchedDirPaths[dirPath]; ok {
		return nil
	}
	if err := w.fsnotifyWatcher.Add(dirPath); err != nil {
		return fmt.Errorf("watch %s: %w", dirPath, err)
	}
	w.watchedDirPaths[dirPath] = struct{}{}
	*newDirPaths = append(*newDirPaths, dirPath)
	return nil
}

// activeExcludeDirPathsLocked returns the excluded directories that apply,
// which are those that do not contain a root directory.
func (w *watcher) activeExcludeDirPathsLocked() []string {
	var activeExcludeDirPaths []string
	for _, excludeDirPath := range w.excludeDirPaths {
		if !w.containsRootDirPathLocked(excludeDirPath) {
			activeExcludeDirPaths = append(activeExcludeDirPaths, excludeDirPath)
		}
	}
	return activeExcludeDirPaths
}

func (w *watcher) containsRootDirPathLocked(normalizedDirPath string) bool {
	for rootDirPath := range w.rootDirPaths {
		if normalpath.EqualsOrContainsPath(normalizedDirPath, normalpath.Normalize(rootDirPath), normalpath.Absolute) {
			return true
		}
	}
	return false
}

// isReportedPath returns true if a change to path is reported: it is a
// watched file, or it is within a root directory and not excluded.
func (w *watcher) isReportedPath(path string) bool {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.isReportedPathLocked(path)
}

func (w *watcher) isReportedPathLocked(path string) bool {
	if _, ok := w.filePaths[path]; ok {
		return true
	}
	if isExcludedPath(path, w.activeExcludeDirPathsLocked()) {
		return false
	}
	normalizedPath := normalpath.Normalize(path)
	for rootDirPath := range w.rootDirPaths {
		if normalpath.EqualsOrContainsPath(normalpath.Normalize(rootDirPath), normalizedPath, normalpath.Absolute) {
			return true
		}
	}
	return false
}

func (w *watcher) isWatchedDirPath(dirPath string) bool {
	w.lock.Lock()
	defer w.lock.Unlock()
	_, ok := w.watchedDirPaths[dirPath]
	return ok
}

func (w *watcher) sendEvent(event Event) {
	select {
	case w.eventChan <- event:
	case <-w.closeChan:
	}
}

func (w *watcher) sendError(err error) {
	select {
	case w.errChan <- err:
	case <-w.closeChan:
	}
}

// fsnotifyItem is an event or error from fsnotifyWatcher.
type fsnotifyItem struct {
	event fsnotify.Event
	err   error
}

func isExcludedPath(path string, activeExcludeDirPaths []string) bool {
	if len(activeExcludeDirPaths) == 0 {
		return false
	}
	normalizedPath := normalpath.Normalize(path)
	for _, activeExcludeDirPath := range activeExcludeDirPaths {
		if normalpath.EqualsOrContainsPath(activeExcludeDirPath, normalizedPath, normalpath.Absolute) {
			return true
		}
	}
	return false
}

// isRelatedNormalizedPath returns true if either absolute normalized path
// equals or contains the other.
func isRelatedNormalizedPath(normalizedPath string, normalizedOtherPath string) bool {
	if normalizedPath == normalizedOtherPath || normalizedPath == "/" || normalizedOtherPath == "/" {
		return true
	}
	return strings.HasPrefix(normalizedPath, normalizedOtherPath+"/") ||
		strings.HasPrefix(normalizedOtherPath, normalizedPath+"/")
}

// nearestExistingAncestorDirPath returns the nearest ancestor of path that
// exists, or empty if there is none.
func nearestExistingAncestorDirPath(path string) string {
	for {
		parentDirPath := filepath.Dir(path)
		if parentDirPath == path {
			return ""
		}
		if fileInfo, err := os.Stat(parentDirPath); err == nil && fileInfo.IsDir() {
			return parentDirPath
		}
		path = parentDirPath
	}
}
