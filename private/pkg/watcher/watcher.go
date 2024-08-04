// Package watcher provides a cross-platform interface for file system notifications.
package watcher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// ChangeFunc defines the type for the callback function that is triggered on file system changes.
type ChangeFunc func(context.Context, string, fsnotify.Op) error

// Watcher encapsulates the fsnotify.Watcher and adds functionality to track file checksums.
// Call Add or AddRecursive to add files or directories to the Watcher.
// After Watch is called, the Watcher will trigger the ChangeFunc callback on file changes.
// The Watcher will stop when the context is canceled or an error occurs.
//
// You must dispose of the watcher after first call to Watch.
type Watcher struct {
	logger       *zap.Logger
	watcher      *fsnotify.Watcher
	checksum     map[string][]byte
	checksumLock *sync.Mutex
}

// New initializes a new Watcher with a given logger.
func New(logger *zap.Logger) (*Watcher, error) {
	// Initialize the file watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("initializing filewatcher: %w", err)
	}

	return &Watcher{
		logger:       logger,
		watcher:      watcher,
		checksum:     make(map[string][]byte),
		checksumLock: new(sync.Mutex),
	}, nil
}

// AddRecursive adds a directory and all its subdirectories to the file watcher.
func (w *Watcher) AddRecursive(path string) error {
	// Walk directories recursively and add them to the file watcher.
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			return nil
		}
		if err != nil {
			// Skip paths that we can't access.
			w.logger.Sugar().Warn("Skipping path %s; %v.", path, err)
			return filepath.SkipDir
		}

		err = w.Add(path)
		if err != nil {
			return fmt.Errorf("adding path %s to filewatcher: %w", path, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("walking path %s: %w", path, err)
	}

	return nil
}

// Add adds a single path to the file watcher.
func (w *Watcher) Add(path string) error {
	err := w.watcher.Add(path)
	if err != nil {
		return fmt.Errorf("adding path %s to filewatcher: %w", path, err)
	}
	return nil
}

// Close stops the file watcher and releases associated resources.
// Use in case you don't need to call Watch anymore.
func (w *Watcher) Close() error {
	err := w.watcher.Close()
	if err != nil {
		return fmt.Errorf("closing filewatcher: %w", err)
	}
	return nil
}

// Watch listens for file system events and triggers the ChangeFunc callback on changes.
// The function will return when the context is canceled, an error occurs, or the user sends a SIGTERM or SIGINT signal.
// The callback function should be fast and non-blocking.
// If the callback returns an error, the watcher will stop and return the error.
//
// The Watcher must be disposed of after the first call to Watch.
func (w *Watcher) Watch(ctx context.Context, change ChangeFunc) error {
	// Initialize the signal handler.
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGTERM, syscall.SIGINT)

	defer signal.Stop(termChan)
	defer close(termChan)
	defer w.Close()

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return errors.New("filewatcher unexpectedly closed")
			}

			if isFile, _ := isFile(event.Name); !isFile {
				continue
			}

			changed, err := w.updateChecksum(event.Name)
			if err != nil {
				continue
			}

			if !changed {
				continue
			}

			// Run the callback every time a file is changed.
			// The callback should be fast and non-blocking.
			// If the callback returns an error, the watcher will stop and return the error.
			if err := change(ctx, event.Name, event.Op); err != nil {
				return err
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return errors.New("filewatcher unexpectedly closed")
			}
			return fmt.Errorf("filewatcher error: %w", err)
		case <-ctx.Done():
			return nil
		case <-termChan:
			return nil
		}
	}
}

// updateChecksum computes the checksum for a file and updates the stored checksum if it has changed.
func (w *Watcher) updateChecksum(path string) (changed bool, err error) {
	fileChecksum, err := fileChecksum(path)
	if err != nil {
		return false, fmt.Errorf("computing checksum for file %s: %w", path, err)
	}

	w.checksumLock.Lock()
	defer w.checksumLock.Unlock()

	oldChecksum, ok := w.checksum[path]
	if ok && bytes.Equal(fileChecksum, oldChecksum) {
		return false, nil
	}

	w.checksum[path] = fileChecksum

	return true, nil
}

// fileChecksum computes the SHA256 checksum of the file at the given path.
func fileChecksum(filename string) (checksum []byte, err error) {
	contents, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	h := sha256.New()
	if _, err := h.Write(contents); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

// isFile checks if the given path is a file.
// Returns true if the path is a file, false if it is a directory or does not exist.
func isFile(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("path does not exist: %w", err)
		}
		return false, fmt.Errorf("error stating path: %w", err)
	}
	return !info.IsDir(), nil
}
