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

package generate

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufgen"
	"github.com/bufbuild/buf/private/buf/cmd/buf/internal/internaltesting"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appcmd/appcmdtesting"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/testingextended"
	"github.com/stretchr/testify/require"
)

const v1Content = `version: v1
managed:
  enabled: true
  cc_enable_arenas: false
  java_multiple_files: true
  java_package_prefix: com
  java_string_check_utf8: false
  optimize_for: CODE_SIZE
  go_package_prefix:
    default: github.com/acme/weather/private/gen/proto/go
    except:
      - buf.build/googleapis/googleapis
    override:
      buf.build/acme/weather: github.com/acme/weather/gen/proto/go
  override:
    JAVA_PACKAGE:
      acme/weather/v1/weather.proto: "org"
plugins:
  - plugin: java
    out: gen/java
`

const v1ContentWithTypes = `version: v1
managed:
  enabled: true
  cc_enable_arenas: false
  java_multiple_files: true
  java_package_prefix: com
  java_string_check_utf8: false
  optimize_for: CODE_SIZE
  go_package_prefix:
    default: github.com/acme/weather/private/gen/proto/go
    except:
      - buf.build/googleapis/googleapis
    override:
      buf.build/acme/weather: github.com/acme/weather/gen/proto/go
  override:
    JAVA_PACKAGE:
      acme/weather/v1/weather.proto: "org"
plugins:
  - plugin: java
    out: gen/java
types:
  include:
    - a.v1.Foo
`

func TestMigrateWithIncludeAndExcludePaths(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	tmpDir := t.TempDir()

	testcases := []struct {
		description             string
		input                   string
		templateContent         string
		templatePath            string // must be unique across testcases
		outDir                  string // must be unique across testcases
		additionalFlags         []string
		filesThatShouldExist    []string
		filesThatShouldNotExist []string
	}{
		{
			description:     "include and exclude paths",
			input:           filepath.Join("testdata", "paths"),
			templateContent: v1Content,
			templatePath:    filepath.Join(tmpDir, "buf.include.exclude.yaml"),
			outDir:          filepath.Join(tmpDir, "IncludeExclude"),
			additionalFlags: []string{
				"--path",
				filepath.Join("testdata", "paths", "b", "v1", "b.proto"),
				"--path",
				filepath.Join("testdata", "paths", "a"),
				"--exclude-path",
				filepath.Join("testdata", "paths", "a", "v2"),
			},
			filesThatShouldExist: []string{
				"gen/java/com/a/v1/AProto.java",
				"gen/java/com/a/v3/AProto.java",
				"gen/java/com/a/v3/foo/BarProto.java",
				"gen/java/com/b/v1/BProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/com/a/v2/AProto.java",
			},
		},
		{
			description:     "types only new flags",
			input:           filepath.Join("testdata", "paths"),
			templateContent: v1Content,
			templatePath:    filepath.Join(tmpDir, "buf.types.new.yaml"),
			outDir:          filepath.Join(tmpDir, "NewTypes"),
			additionalFlags: []string{
				"--type",
				"b.v1.Bar",
				"--type",
				"a.v3.foo.Bar",
				"--type",
				"a.v2.Foo",
			},
			filesThatShouldExist: []string{
				"gen/java/com/b/v1/BProto.java",
				"gen/java/com/a/v3/foo/BarProto.java",
				"gen/java/com/a/v2/AProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/com/a/v1/AProto.java",
				"gen/java/com/a/v3/AProto.java",
			},
		},
		{
			description:     "types only old flags",
			input:           filepath.Join("testdata", "paths"),
			templateContent: v1Content,
			templatePath:    filepath.Join(tmpDir, "buf.types.old.yaml"),
			outDir:          filepath.Join(tmpDir, "OldTypes"),
			additionalFlags: []string{
				"--include-types",
				"a.v1.Foo",
				"--include-types",
				"a.v3.foo.Foo",
				"--include-types",
				"a.v3.Foo",
			},
			filesThatShouldExist: []string{
				"gen/java/com/a/v1/AProto.java",
				"gen/java/com/a/v3/AProto.java",
				"gen/java/com/a/v3/foo/FooProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/com/a/v3/foo/BarProto.java",
				"gen/java/com/b/v1/BProto.java",
				"gen/java/com/a/v2/AProto.java",
			},
		},
		{
			description:     "types both new and old flags",
			input:           filepath.Join("testdata", "paths"),
			templateContent: v1Content,
			templatePath:    filepath.Join(tmpDir, "buf.types.mixed.yaml"),
			outDir:          filepath.Join(tmpDir, "MixedTypes"),
			additionalFlags: []string{
				"--type",
				"b.v1.Bar",
				"--include-types",
				"a.v3.foo.Bar",
				"--type",
				"a.v2.Foo",
			},
			filesThatShouldExist: []string{
				"gen/java/com/b/v1/BProto.java",
				"gen/java/com/a/v3/foo/BarProto.java",
				"gen/java/com/a/v2/AProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/com/a/v1/AProto.java",
				"gen/java/com/a/v3/AProto.java",
			},
		},
		{
			description:     "types in both flags and v1 config",
			input:           filepath.Join("testdata", "paths"),
			templateContent: v1ContentWithTypes,
			templatePath:    filepath.Join(tmpDir, "buf.types.flags.config.yaml"),
			outDir:          filepath.Join(tmpDir, "FlagsAndConfig"),
			additionalFlags: []string{
				"--type",
				"b.v1.Bar",
				"--include-types",
				"a.v3.foo.Bar",
				"--type",
				"a.v2.Foo",
			},
			filesThatShouldExist: []string{
				"gen/java/com/b/v1/BProto.java",
				"gen/java/com/a/v3/foo/BarProto.java",
				"gen/java/com/a/v2/AProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/com/a/v1/AProto.java",
				"gen/java/com/a/v3/AProto.java",
			},
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			err := os.WriteFile(testcase.templatePath, []byte(v1Content), 0600)
			require.NoError(t, err)
			outDir := testcase.outDir + "v1"
			argAndFlags := append(
				[]string{
					testcase.input,
					"--template",
					testcase.templatePath,
					"--output",
					outDir,
				},
				testcase.additionalFlags...,
			)
			// generate with v1 template
			testRunSuccess(
				t,
				argAndFlags...,
			)
			bucketV1, err := storageosProvider.NewReadWriteBucket(
				outDir,
				storageos.ReadWriteBucketWithSymlinksIfSupported(),
			)
			require.NoError(t, err)

			// migrate from v1 to v2
			outDir = testcase.outDir + "migrate"
			argAndFlags = append(
				[]string{
					testcase.input,
					"--template",
					testcase.templatePath,
					"--output",
					outDir,
					"--migrate",
				},
				testcase.additionalFlags...,
			)
			printedLastLine := runAndGetStderrSecondLastLine(
				t,
				argAndFlags...,
			)
			bucketMigrate, err := storageosProvider.NewReadWriteBucket(
				outDir,
				storageos.ReadWriteBucketWithSymlinksIfSupported(),
			)
			require.NoError(t, err)
			expectedNewArgs := []string{
				"--template",
				testcase.templatePath,
				"--output",
				outDir,
			}
			expectedNewCommand := strings.Join(
				append(
					[]string{
						"buf",
						"generate",
					},
					expectedNewArgs...,
				),
				" ",
			)
			require.Equal(t, expectedNewCommand, printedLastLine)
			requireVersionV2(t, testcase.templatePath)

			// generate with v2 template
			outDir = testcase.outDir + "v2"
			testRunSuccess(
				t,
				"--template",
				testcase.templatePath,
				"-o",
				outDir,
			)
			bucketV2, err := storageosProvider.NewReadWriteBucket(
				outDir,
				storageos.ReadWriteBucketWithSymlinksIfSupported(),
			)
			require.NoError(t, err)

			diff, err := storage.DiffBytes(
				context.Background(),
				command.NewRunner(),
				bucketV1,
				bucketMigrate,
				transformGolangProtocVersionToUnknown(t),
			)
			require.NoError(t, err)
			require.Empty(t, string(diff))

			diff, err = storage.DiffBytes(
				context.Background(),
				command.NewRunner(),
				bucketMigrate,
				bucketV2,
				transformGolangProtocVersionToUnknown(t),
			)
			require.NoError(t, err)
			require.Empty(t, string(diff))

			for _, fileThatShouldExist := range testcase.filesThatShouldExist {
				requireFileExists(t, bucketV2, fileThatShouldExist)
			}
			for _, fileThatShouldNotExist := range testcase.filesThatShouldNotExist {
				requireFileDoesNotExist(t, bucketV2, fileThatShouldNotExist)
			}
		})
	}
}

// runAndGetStderrSecondLastLine runs the command and requires that it succeeds, and
// returns the second last line printed to stderr. The command itself may have side effects.
func runAndGetStderrSecondLastLine(t *testing.T, args ...string) string {
	stderr := bytes.NewBuffer(nil)
	appcmdtesting.RunCommandExitCode(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appflag.NewBuilder(name),
			)
		},
		0,
		internaltesting.NewEnvFunc(t),
		nil,
		nil,
		stderr,
		args...,
	)
	printedLines := strings.Split(stderr.String(), "\n")
	require.GreaterOrEqual(t, len(printedLines), 2)
	return printedLines[len(printedLines)-2]
}

func requireFileExists(
	t *testing.T,
	bucket storage.ReadBucket,
	fileName string,
) {
	_, err := bucket.Stat(
		context.Background(),
		filepath.FromSlash(fileName),
	)
	require.NoError(t, err)
}

func requireFileDoesNotExist(
	t *testing.T,
	bucket storage.ReadBucket,
	fileName string,
) {
	_, err := bucket.Stat(
		context.Background(),
		filepath.FromSlash(fileName),
	)
	require.Error(t, err)
}

func requireVersionV2(
	t *testing.T,
	templatePath string,
) {
	data, err := os.ReadFile(templatePath)
	require.NoError(t, err)
	versionConfig := bufgen.ExternalConfigVersion{}
	err = encoding.UnmarshalYAMLNonStrict(data, &versionConfig)
	require.NoError(t, err)
	require.Equal(t, "v2", versionConfig.Version)
}
