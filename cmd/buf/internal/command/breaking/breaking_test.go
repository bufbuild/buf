// Copyright 2020-2025 Buf Technologies, Inc.
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

package breaking

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appcmd/appcmdtesting"
	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/cmd/buf/internal/internaltesting"
)

// TestBreakingWorkspaceNewModuleWithImport tests that adding a new module to a
// workspace does not produce false breaking change errors when the new module
// imports files from an existing module.
//
// This is a regression test for the case where the new module's imported files
// cause filterImageWithConfigsNotInAgainstImages to incorrectly match the new
// module to an existing against image, stealing the match from the module that
// actually owns those files.
func TestBreakingWorkspaceNewModuleWithImport(t *testing.T) {
	t.Parallel()
	testRunStdoutStderr(
		t,
		nil,
		0,
		"",
		"",
		filepath.Join("testdata", "workspace_new_module", "head"),
		"--against",
		filepath.Join("testdata", "workspace_new_module", "against"),
	)
}

func testRunStdoutStderr(
	t *testing.T,
	stdin io.Reader,
	expectedExitCode int,
	expectedStdout string,
	expectedStderr string,
	args ...string,
) {
	t.Helper()
	appcmdtesting.Run(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appext.NewBuilder(
					name,
					appext.BuilderWithInterceptor(
						func(next func(context.Context, appext.Container) error) func(context.Context, appext.Container) error {
							return func(ctx context.Context, container appext.Container) error {
								err := next(ctx, container)
								if err == nil {
									return nil
								}
								return fmt.Errorf("Failure: %w", err)
							}
						},
					),
				),
			)
		},
		appcmdtesting.WithExpectedExitCode(expectedExitCode),
		appcmdtesting.WithExpectedStdout(expectedStdout),
		appcmdtesting.WithExpectedStderr(expectedStderr),
		appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithArgs(args...),
	)
}
