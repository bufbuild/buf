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

//go:build linux

package open

import (
	"bytes"
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
)

// open opens the input with xdg-open.
// http://sources.debian.net/src/xdg-utils/1.1.0~rc1%2Bgit20111210-7.1/scripts/xdg-open/
// http://sources.debian.net/src/xdg-utils/1.1.0~rc1%2Bgit20111210-7.1/scripts/xdg-mime/
func open(
	ctx context.Context,
	runner command.Runner,
	envContainer app.EnvContainer,
	input string,
) error {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if err := runner.Run(
		ctx,
		"xdg-open",
		command.RunWithArgs(input),
		command.RunWithStdout(stdout),
		command.RunWithStderr(stderr),
		command.RunWithEnv(app.EnvironMap(envContainer)),
	); err != nil {
		return fmt.Errorf("could not open %q: %w, %s", input, err, stderr.String())
	}
	return nil
}
