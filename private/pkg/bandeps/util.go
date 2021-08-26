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

package bandeps

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app"
)

func sortViolations(violations []Violation) {
	sort.Slice(
		violations,
		func(i int, j int) bool {
			one := violations[i]
			two := violations[j]
			if one.Package() < two.Package() {
				return true
			}
			if one.Package() > two.Package() {
				return false
			}
			if one.Dep() < two.Dep() {
				return true
			}
			if one.Dep() > two.Dep() {
				return false
			}
			return one.Note() < two.Note()
		},
	)
}

func runStdout(
	ctx context.Context,
	envContainer app.EnvContainer,
	stdin io.Reader,
	name string,
	args ...string,
) (io.Reader, error) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = app.Environ(envContainer)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s\n%s", err.Error(), stderr.String())
	}
	return stdout, nil
}

func getNonEmptyLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if line := strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func addMaps(base map[string]struct{}, maps ...map[string]struct{}) {
	for _, m := range maps {
		for k, v := range m {
			base[k] = v
		}
	}
}

func subtractMaps(base map[string]struct{}, maps ...map[string]struct{}) {
	for _, m := range maps {
		for k := range m {
			delete(base, k)
		}
	}
}
