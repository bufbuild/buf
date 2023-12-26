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

package buf

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appcmd/appcmdtesting"
)

func TestValidNoImports(t *testing.T) {
	t.Parallel()
	testRunStderrWithCache(
		t, nil, 0, nil,
		"build",
		filepath.Join("testdata", "imports", "success", "people"),
	)
}

func TestValidImportFromCache(t *testing.T) {
	t.Parallel()
	testRunStderrWithCache(
		t, nil, 0, nil,
		"build",
		filepath.Join("testdata", "imports", "success", "students"),
	)
}

func TestValidImportFromCorruptedCache(t *testing.T) {
	t.Parallel()
	testRunStderrWithCache(
		t, nil, 0, nil,
		"build",
		filepath.Join("testdata", "imports", "success", "students"),
	)
	appcmdtesting.RunCommandExitCodeStderr(
		t,
		func(use string) *appcmd.Command { return NewRootCommand(use) },
		1,
		`Failure: verification failed for module bufbuild.test/bufbot/people: expected module digest "b5:b22338d6faf2a727613841d760c9cbfd21af6950621a589df329e1fe6611125904c39e22a73e0aa8834006a514dbd084e6c33b6bef29c8e4835b4b9dec631465" but downloaded data had digest "b5:fbb4d43ed11ddcd1ff1c3ee0b97b573360208f602c4ac3a3b9a4be9bbdf00b99fdfd4180d1d326a547b298417b094e2b9c00b431c66b13732d1227795c3b26d2"`,
		func(use string) map[string]string {
			return map[string]string{
				useEnvVar(use, "CACHE_DIR"): filepath.Join("testdata", "imports", "corrupted_cache"),
			}
		},
		nil,
		"build",
		filepath.Join("testdata", "imports", "success", "students"),
	)
}

func TestValidImportTransitiveFromCache(t *testing.T) {
	t.Parallel()
	testRunStderrWithCache(
		t, nil, 0, nil,
		"build",
		filepath.Join("testdata", "imports", "success", "school"),
	)
}

func TestValidImportWKT(t *testing.T) {
	t.Parallel()
	testRunStdoutStderr(
		t, nil, 0,
		"", // no warnings
		"build",
		filepath.Join("testdata", "imports", "success", "wkt"),
	)
}

func TestInvalidNonexistentImport(t *testing.T) {
	t.Parallel()
	testRunStderrWithCache(
		t, nil, 100,
		[]string{filepath.FromSlash(`testdata/imports/failure/people/people/v1/people1.proto:5:8:stat nonexistent.proto: file does not exist`)},
		"build",
		filepath.Join("testdata", "imports", "failure", "people"),
	)
}

func TestInvalidNonexistentImportFromDirectDep(t *testing.T) {
	t.Parallel()
	testRunStderrWithCache(
		t, nil, 100,
		[]string{filepath.FromSlash(`testdata/imports/failure/students/students/v1/students.proto:6:8:`) + `stat people/v1/people_nonexistent.proto: file does not exist`},
		"build",
		filepath.Join("testdata", "imports", "failure", "students"),
	)
}

func TestInvalidImportFromTransitive(t *testing.T) {
	t.Parallel()
	testRunStderrWithCache(
		t, nil, 0,
		[]string{
			"WARN",
			`Module bufbuild.test/bufbot/people is a transitive remote dependency not declared in your buf.yaml deps. Add bufbuild.test/bufbot/people to your deps.`,
		},
		"build",
		filepath.Join("testdata", "imports", "failure", "school"),
	)
}

func TestInvalidImportFromTransitiveWorkspace(t *testing.T) {
	t.Parallel()
	testRunStderrWithCache(
		t, nil, 0,
		[]string{
			"WARN",
			// a -> c
			`Module bufbuild.test/workspace/second is declared in your buf.yaml deps but is a module in your workspace. Declaring a dep within your workspace has no effect.`,
			`Module bufbuild.test/workspace/third is declared in your buf.yaml deps but is a module in your workspace. Declaring a dep within your workspace has no effect.`,
		},
		"build",
		filepath.Join("testdata", "imports", "failure", "workspace", "transitive_imports"),
	)
}

func TestValidImportFromLocalOnlyWorkspaceUnnamedModules(t *testing.T) {
	t.Parallel()
	testRunStdoutStderr(
		t, nil, 0,
		"", // no warnings
		"build",
		filepath.Join("testdata", "imports", "success", "workspace", "unnamed_local_only_modules"),
	)
}

func TestGraphNoWarningsValidImportFromWorkspaceNamedModules(t *testing.T) {
	t.Parallel()
	testRunStdoutStderr(
		t, nil, 0,
		"", // no warnings
		"beta", "graph",
		filepath.Join("testdata", "imports", "success", "workspace", "valid_explicit_deps"),
	)
}

func testRunStderrWithCache(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStderrPartials []string, args ...string) {
	appcmdtesting.RunCommandExitCodeStderrContains(
		t,
		func(use string) *appcmd.Command { return NewRootCommand(use) },
		expectedExitCode,
		expectedStderrPartials,
		func(use string) map[string]string {
			return map[string]string{
				useEnvVar(use, "CACHE_DIR"): filepath.Join("testdata", "imports", "cache"),
			}
		},
		stdin,
		args...,
	)
}

func useEnvVar(use string, suffix string) string {
	return strings.ToUpper(use) + "_" + suffix
}
