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

package buf

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufctl"
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

func TestValidImportFromCorruptedCacheFile(t *testing.T) {
	t.Parallel()
	appcmdtesting.RunCommandExitCodeStderr(
		t,
		func(use string) *appcmd.Command { return NewRootCommand(use) },
		1,
		`Failure: ***Digest verification failed for module bufbuild.test/bufbot/people:fc7d540124fd42db92511c19a60a1d98***
	Expected digest: "b5:b22338d6faf2a727613841d760c9cbfd21af6950621a589df329e1fe6611125904c39e22a73e0aa8834006a514dbd084e6c33b6bef29c8e4835b4b9dec631465"
	Downloaded data digest: "b5:87403abcc5ec8403180536840a46bef8751df78caa8ad4b46939f4673d8bd58663d0f593668651bb2cd23049fedac4989e8b28c7e0e36b9b524f58ab09bf1053"`,
		func(use string) map[string]string {
			return map[string]string{
				useEnvVar(use, "CACHE_DIR"): filepath.Join("testdata", "imports", "corrupted_cache_file"),
			}
		},
		nil,
		"build",
		filepath.Join("testdata", "imports", "success", "students"),
		"--no-warn",
	)
}

func TestValidImportFromCorruptedCacheDep(t *testing.T) {
	t.Parallel()
	appcmdtesting.RunCommandExitCodeStderr(
		t,
		func(use string) *appcmd.Command { return NewRootCommand(use) },
		1,
		`Failure: ***Digest verification failed for module bufbuild.test/bufbot/students:6c776ed5bee54462b06d31fb7f7c16b8***
	Expected digest: "b5:01764dd31d0e1b8355eb3b262bba4539657af44872df6e4dfec76f57fbd9f1ae645c7c9c607db5c8352fb7041ca97111e3b0f142dafc1028832acbbc14ba1d70"
	Downloaded data digest: "b5:975dad3641303843fb6a06eedf038b0e6ff41da82b8a483920afb36011e0b0a24f720a2407f5e0783389530486ff410b7e132f219add69a5c7324d54f6f89a6c"`,
		func(use string) map[string]string {
			return map[string]string{
				useEnvVar(use, "CACHE_DIR"): filepath.Join("testdata", "imports", "corrupted_cache_dep"),
			}
		},
		nil,
		"build",
		filepath.Join("testdata", "imports", "success", "school"),
		"--no-warn",
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
		t, nil, bufctl.ExitCodeFileAnnotation,
		[]string{filepath.FromSlash(`Failure: testdata/imports/failure/people/people/v1/people1.proto: import "nonexistent.proto": file does not exist`)},
		"build",
		filepath.Join("testdata", "imports", "failure", "people"),
	)
}

func TestInvalidNonexistentImportFromDirectDep(t *testing.T) {
	t.Parallel()
	testRunStderrWithCache(
		t, nil, bufctl.ExitCodeFileAnnotation,
		[]string{filepath.FromSlash(`Failure: testdata/imports/failure/students/students/v1/students.proto: `) + `import "people/v1/people_nonexistent.proto": file does not exist`},
		"build",
		filepath.Join("testdata", "imports", "failure", "students"),
	)
}

func TestInvalidImportFromTransitive(t *testing.T) {
	t.Parallel()
	// We actually want to verify that there are no warnings now. Transitive dependencies not declared
	// in your buf.yaml are acceptable now.
	testRunStderrWithCache(
		t, nil, 0,
		[]string{},
		"build",
		filepath.Join("testdata", "imports", "failure", "school"),
	)
}

func TestInvalidImportFromTransitiveWorkspace(t *testing.T) {
	t.Parallel()
	testRunStderrWithCache(
		t, nil, 0,
		// We actually want to verify that there are no warnings now. deps in your v1 buf.yaml may actually
		// have an effect - they can affect your buf.lock.
		[]string{},
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
		"graph",
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
