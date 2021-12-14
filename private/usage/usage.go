// Copyright 2020-2021 Buf Technologies, Inc.
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

package usage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
)

func init() {
	if err := check(); err != nil {
		panic(err.Error())
	}
}

func check() error {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		if shouldSkip() {
			return nil
		}
		return errors.New("github.com/bufbuild/buf/private code must only be imported by github.com/bufbuild projects")
	}
	if buildInfo.Main.Path == "" {
		if shouldSkip() {
			return nil
		}
		return errors.New("github.com/bufbuild/buf/private code must only be imported by github.com/bufbuild projects")
	}
	if !strings.HasPrefix(buildInfo.Main.Path, "github.com/bufbuild") {
		return fmt.Errorf("github.com/bufbuild/buf/private code must only be imported by github.com/bufbuild projects but was used in %s", buildInfo.Main.Path)
	}
	return nil
}

func shouldSkip() bool {
	return strings.HasSuffix(os.Args[0], testSuffix) || filepath.Base(os.Args[0]) == debugBin
}
