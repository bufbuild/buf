// Copyright 2020-2022 Buf Technologies, Inc.
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

//go:build go1.19

package appprotoexec

import (
	"errors"
	"os/exec"
)

// unsafeLookPath is a wrapper around exec.LookPath that restores the original
// pre-Go 1.19 behavior of resolving queries that would use relative PATH
// entries. We consider it acceptable for the use case of locating plugins.
//
// On Go 1.18 and below, this function is just a direct call to exec.LookPath.
//
// https://pkg.go.dev/os/exec#hdr-Executables_in_the_current_directory
func unsafeLookPath(file string) (string, error) {
	path, err := exec.LookPath(file)
	if errors.Is(err, exec.ErrDot) {
		err = nil
	}
	return path, err
}
