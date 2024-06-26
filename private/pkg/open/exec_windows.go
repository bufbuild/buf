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

//go:build windows

package open

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
)

func open(
	ctx context.Context,
	runner command.Runner,
	envContainer app.EnvContainer,
	input string,
) error {
	// Replace characters that are not allowed by cmd/bash.
	input = strings.ReplaceAll(input, "&", `^&`)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if err := runner.Run(
		ctx,
		"cmd",
		command.RunWithArgs("/c", "start"),
		command.RunWithStdout(stdout),
		command.RunWithStderr(stderr),
		command.RunWithEnv(app.EnvironMap(envContainer)),
	); err != nil {
		return fmt.Errorf("could not open %q: %w, %s", input, err, stderr.String())
	}
	return nil
}
