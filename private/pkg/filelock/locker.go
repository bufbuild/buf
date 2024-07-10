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

package filelock

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
)

type locker struct {
	rootDirPath    string
	lockTimeout    time.Duration
	lockRetryDelay time.Duration
}

func newLocker(rootDirPath string, options ...LockerOption) (*locker, error) {
	// allow symlinks
	fileInfo, err := os.Stat(normalpath.Unnormalize(rootDirPath))
	if err != nil {
		return nil, err
	}
	if !fileInfo.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", rootDirPath)
	}
	lockerOptions := newLockerOptions()
	for _, option := range options {
		option(lockerOptions)
	}
	return &locker{
		// do not validate - allow anything including absolute paths and jumping context
		rootDirPath:    normalpath.Normalize(rootDirPath),
		lockTimeout:    lockerOptions.lockTimeout,
		lockRetryDelay: lockerOptions.lockRetryDelay,
	}, nil
}

func (l *locker) Lock(ctx context.Context, path string, options ...LockOption) (Unlocker, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}
	options = slicesext.Concat(
		[]LockOption{
			LockWithTimeout(l.lockTimeout),
			LockWithRetryDelay(l.lockRetryDelay),
		},
		options, // Any additional options set will be applied last
	)
	return lock(
		ctx,
		normalpath.Unnormalize(normalpath.Join(l.rootDirPath, path)),
		options...,
	)
}

func (l *locker) RLock(ctx context.Context, path string, options ...LockOption) (Unlocker, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}
	options = slicesext.Concat(
		[]LockOption{
			LockWithTimeout(l.lockTimeout),
			LockWithRetryDelay(l.lockRetryDelay),
		},
		options, // Any additional options set will be applied last
	)
	return rlock(
		ctx,
		normalpath.Unnormalize(normalpath.Join(l.rootDirPath, path)),
		options...,
	)
}

func validatePath(path string) error {
	normalPath, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return err
	}
	if path != normalPath {
		// just extra safety
		return fmt.Errorf("expected file lock path %q to be equal to normalized path %q", path, normalPath)
	}
	return nil
}

type lockerOptions struct {
	lockTimeout    time.Duration
	lockRetryDelay time.Duration
}

func newLockerOptions() *lockerOptions {
	return &lockerOptions{
		lockTimeout:    DefaultLockTimeout,
		lockRetryDelay: DefaultLockRetryDelay,
	}
}
