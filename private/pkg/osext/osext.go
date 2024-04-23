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

// Package osext provides os utilities.
package osext

import (
	"errors"
	"os"
	"sync"
)

var (
	globalWorkDirPath    string
	globalWorkDirPathErr error

	globalLock sync.RWMutex

	errOSGetwdEmpty = errors.New("os.Getwd returned empty and no error")
)

// Getwd replaces os.Getwd and caches the result.
func Getwd() (string, error) {
	globalLock.RLock()
	workDirPath, workDirPathErr := globalWorkDirPath, globalWorkDirPathErr
	globalLock.RUnlock()
	if workDirPath != "" || workDirPathErr != nil {
		return workDirPath, workDirPathErr
	}
	globalLock.Lock()
	defer globalLock.Unlock()
	globalWorkDirPath, globalWorkDirPathErr = getwdUncached()
	return globalWorkDirPath, globalWorkDirPathErr
}

// Chdir calls os.Chdir and clears any cached result of Getwd.
func Chdir(dir string) error {
	globalLock.Lock()
	defer globalLock.Unlock()
	globalWorkDirPath = ""
	globalWorkDirPathErr = nil
	return os.Chdir(dir)
}

func getwdUncached() (string, error) {
	workDirPath, workDirPathErr := os.Getwd()
	if workDirPath == "" && workDirPathErr == nil {
		return "", errOSGetwdEmpty
	}
	return workDirPath, workDirPathErr
}
