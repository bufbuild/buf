// Copyright 2020-2023 Buf Technologies, Inc.
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

//go:build windows
// +build windows

package filepathext

import (
	"os"
)

var (
	// The environment variable that shows the drive that holds the
	// Windows folder. This is a drive name and not a folder name (`C:` not `C:\`).
	// https://learn.microsoft.com/en-us/windows/deployment/usmt/usmt-recognized-environment-variables#variables-that-are-processed-for-the-operating-system-and-in-the-context-of-each-user
	FSRoot = os.Getenv("SYSTEMDRIVE") + string(os.PathSeparator)
)
