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
		t, nil, 0, ``,
		"build",
		filepath.Join("testdata", "imports", "success", "people"),
	)
}

func TestValidImportFromCache(t *testing.T) {
	t.Parallel()
	testRunStderrWithCache(
		t, nil, 0, ``,
		"build",
		filepath.Join("testdata", "imports", "success", "students"),
	)
}

func TestValidImportTransitiveFromCache(t *testing.T) {
	t.Parallel()
	testRunStderrWithCache(
		t, nil, 0, ``,
		"build",
		filepath.Join("testdata", "imports", "success", "school"),
	)
}

func TestInvalidNonexistentImport(t *testing.T) {
	t.Parallel()
	testRunStderrWithCache(
		t, nil, 100,
		filepath.FromSlash(`testdata/imports/failure/people/people/v1/people1.proto:5:8:nonexistent.proto: does not exist`),
		"build",
		filepath.Join("testdata", "imports", "failure", "people"),
	)
}

func TestInvalidNonexistentImportFromDirectDep(t *testing.T) {
	t.Parallel()
	testRunStderrWithCache(
		t, nil, 100,
		filepath.FromSlash(`testdata/imports/failure/students/students/v1/students.proto:6:8:`)+`people/v1/people_nonexistent.proto: does not exist`,
		"build",
		filepath.Join("testdata", "imports", "failure", "students"),
	)
}

func TestInvalidImportFromTransitive(t *testing.T) {
	t.Parallel()
	testRunStderrWithCache(
		t, nil, 1,
		`Failure: target proto file "school/v1/school1.proto" imports "people/v1/people1.proto", not found in your local target files or direct dependencies, but found in transitive dependency "bufbuild.test/bufbot/people", please declare that one as explicit dependency in your buf.yaml file; `+ // "people1.proto" failure
			`target proto file "school/v1/school1.proto" imports "people/v1/people2.proto", not found in your local target files or direct dependencies, but found in transitive dependency "bufbuild.test/bufbot/people", please declare that one as explicit dependency in your buf.yaml file`, // "people2.proto" failure
		"build",
		filepath.Join("testdata", "imports", "failure", "school"),
	)
}

func testRunStderrWithCache(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStderr string, args ...string) {
	appcmdtesting.RunCommandExitCodeStderr(
		t,
		func(use string) *appcmd.Command { return NewRootCommand(use) },
		expectedExitCode,
		expectedStderr,
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
