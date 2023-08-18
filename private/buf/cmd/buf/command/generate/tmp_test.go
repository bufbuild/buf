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
  java_package_prefix: com
plugins:
  - plugin: java
    out: gen/java
types:
  include:
    - a.v1.Foo
`

const v1ContentMinimal = `version: v1
managed:
  enabled: true
  cc_enable_arenas: false
  java_multiple_files: true
  java_package_prefix: net
  java_string_check_utf8: false
  optimize_for: CODE_SIZE
plugins:
  - plugin: java
    out: gen/java
`

const v1ContentWithPerFileOverride = `version: v1
managed:
  enabled: true
  cc_enable_arenas: true
  java_multiple_files: false
  java_package_prefix:
    default: xyz
    except:
      - bufbuild.test/generate/bar
    override:
      bufbuild.test/generate/baz: dev
      bufbuild.test/generate/qux: net
  java_string_check_utf8: true
  optimize_for:
    default: LITE_RUNTIME
    except:
      - bufbuild.test/generate/qux
    override:
      bufbuild.test/generate/bar: SPEED
      bufbuild.test/generate/baz: CODE_SIZE
  go_package_prefix:
    default:  example.com/generate
    override:
      bufbuild.test/generate/bar: example.com/baroverride
      bufbuild.test/generate/baz: example.com/bazoverride
  objc_class_prefix:
    default:  XYZ
    except:
      - bufbuild.test/generate/baz
    override:
      bufbuild.test/generate/bar: BAR
      bufbuild.test/generate/qux: QUX
  csharp_namespace:
    except:
      - bufbuild.test/generate/baz
    override:
      bufbuild.test/generate/bar: B::A::R
      bufbuild.test/generate/qux: Q::A::X
  ruby_package:
    except:
      - bufbuild.test/generate/baz
    override:
      bufbuild.test/generate/bar: B::A::R
      bufbuild.test/generate/qux: Q::A::X
  override:
    JAVA_PACKAGE:
      a.proto: ajavapkg
      x.proto: xjavapkg # note that x.proto's module is excluded
      v1/n.proto: njavapkg
    GO_PACKAGE:
      b.proto: b/gopkg
    RUBY_PACKAGE:
      v1/m.proto: mrubypkg
    CC_ENABLE_ARENAS:
      v1/n.proto: false
    OPTIMIZE_FOR:
      t.proto: CODE_SIZE
    CSHARP_NAMESPACE:
      y.proto: YPROTO::YPROTO
      b.proto: BPROTO:BPROTO
    JAVA_MULTIPLE_FILES:
      v1/n.proto: true
      t.proto: true
    OBJC_CLASS_PREFIX:
      a.proto: APRO
      x.proto: XPRO
    PHP_METADATA_NAMESPACE:
      y.proto: YProto\Metadata
      v1/m.proto: MProto\Metadata
    PHP_NAMESPACE:
      x.proto: XProtoNamespace
      b.proto: BProtoNamespace
plugins:
  - plugin: java
    out: gen/java
  - plugin: go
    out: gen/go
  - plugin: objc
    out: gen/objc
`

func TestMigrateWithIncludeAndExcludePaths(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()

	testcases := []struct {
		description             string
		input                   string
		templateContent         string
		additionalFlags         []string
		filesThatShouldExist    []string
		filesThatShouldNotExist []string
		// skipMigrateComp this skips comparing running <command> vs running <command> --migrate
		// this should only be set to true when the v1 file has `types`
		skipMigrateComp bool
	}{
		{
			description:     "include and exclude paths",
			input:           filepath.Join("testdata", "paths"),
			templateContent: v1Content,
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
			description:     "types in old flags and v1 config",
			input:           filepath.Join("testdata", "paths"),
			templateContent: v1ContentWithTypes,
			additionalFlags: []string{
				"--type",
				"b.v1.Bar",
			},
			filesThatShouldExist: []string{
				"gen/java/com/b/v1/BProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/com/a/v2/AProto.java",
				"gen/java/com/a/v3/foo/BarProto.java",
				"gen/java/com/a/v1/AProto.java",
				"gen/java/com/a/v3/AProto.java",
			},
		},
		{
			description:     "types only in config",
			input:           filepath.Join("testdata", "paths"),
			templateContent: v1ContentWithTypes,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/com/a/v1/AProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/com/b/v1/BProto.java",
				"gen/java/com/a/v2/AProto.java",
				"gen/java/com/a/v3/AProto.java",
				"gen/java/com/a/v3/foo/BarProto.java",
			},
			skipMigrateComp: true,
		},
		{
			description:     "module dir include imports and wkt",
			input:           filepath.Join("testdata", "formats", "workspace_dir", "foo"),
			templateContent: v1ContentMinimal,
			additionalFlags: []string{
				"--include-imports",
				"--include-wkt",
			},
			filesThatShouldExist: []string{
				"gen/java/net/foo/AProto.java",
				"gen/java/net/foo/BProto.java",
				"gen/java/net/bar/XProto.java",
				"gen/java/net/bar/YProto.java",
				"gen/java/net/baz/v1/MProto.java",
				"gen/java/net/baz/v1/NProto.java",
				"gen/java/net/qux/TProto.java",
				"gen/java/com/google/protobuf/TimestampProto.java",
			},
		},
		{
			description:     "module dir include imports but not wkt",
			input:           filepath.Join("testdata", "formats", "workspace_dir", "foo"),
			templateContent: v1ContentMinimal,
			additionalFlags: []string{
				"--include-imports",
			},
			filesThatShouldExist: []string{
				"gen/java/net/foo/AProto.java",
				"gen/java/net/foo/BProto.java",
				"gen/java/net/bar/XProto.java",
				"gen/java/net/bar/YProto.java",
				"gen/java/net/baz/v1/NProto.java",
				"gen/java/net/qux/TProto.java",
				"gen/java/net/baz/v1/MProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/com/google/protobuf/TimestampProto.java",
			},
		},
		{
			description:     "module dir without including imports",
			input:           filepath.Join("testdata", "formats", "workspace_dir", "foo"),
			templateContent: v1ContentMinimal,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/foo/AProto.java",
				"gen/java/net/foo/BProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/net/bar/XProto.java",
				"gen/java/net/baz/v1/MProto.java",
				"gen/java/com/google/protobuf/TimestampProto.java",
			},
		},
		{
			description:     "json image",
			input:           filepath.Join("testdata", "formats", "workspace_image.json"),
			templateContent: v1ContentMinimal,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/foo/AProto.java",
				"gen/java/net/foo/Foo.java",
				"gen/java/net/foo/BProto.java",
				"gen/java/net/foo/Baz.java",
			},
		},
		{
			description:     "binary image with .binpb",
			input:           filepath.Join("testdata", "formats", "workspace_image.binpb"),
			templateContent: v1ContentMinimal,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/foo/AProto.java",
				"gen/java/net/foo/Foo.java",
				"gen/java/net/foo/BProto.java",
				"gen/java/net/foo/Baz.java",
			},
		},
		{
			description:     "binary image with .binpb.gz",
			input:           filepath.Join("testdata", "formats", "workspace_image.binpb.gz"),
			templateContent: v1ContentMinimal,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/foo/AProto.java",
				"gen/java/net/foo/Foo.java",
				"gen/java/net/foo/BProto.java",
				"gen/java/net/foo/Baz.java",
			},
		},
		{
			description:     "text image",
			input:           filepath.Join("testdata", "formats", "workspace_image.txtpb"),
			templateContent: v1ContentMinimal,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/foo/AProto.java",
				"gen/java/net/foo/Foo.java",
				"gen/java/net/foo/BProto.java",
				"gen/java/net/foo/Baz.java",
			},
		},
		{
			description:     "directory not module",
			input:           filepath.Join("testdata", "formats", "not_module"),
			templateContent: v1ContentMinimal,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/A/AProto.java",
				"gen/java/net/B/BProto.java",
			},
		},
		{
			description:     "zip archive",
			input:           filepath.Join("testdata", "formats", "not_module.zip"),
			templateContent: v1ContentMinimal,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/A/AProto.java",
				"gen/java/net/B/BProto.java",
			},
		},
		{
			description:     "zip archive",
			input:           filepath.Join("testdata", "formats", "not_module.zip"),
			templateContent: v1ContentMinimal,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/A/AProto.java",
				"gen/java/net/B/BProto.java",
			},
		},
		{
			description:     "zip archive with sub dir",
			input:           filepath.Join("testdata", "formats", "not_module.zip#subdir=a"),
			templateContent: v1ContentMinimal,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/A/AProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/net/B/BProto.java",
			},
		},
		{
			description:     "proto file ref",
			input:           filepath.Join("testdata", "protofileref", "a", "v1", "a.proto"),
			templateContent: v1ContentMinimal,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/A/v1/AProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/net/A/v1/BProto.java",
			},
		},
		{
			description:     "proto file ref include package file",
			input:           filepath.Join("testdata", "protofileref", "a", "v1", "a.proto#include_package_files=true"),
			templateContent: v1ContentMinimal,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/A/v1/AProto.java",
				"gen/java/net/A/v1/BProto.java",
			},
		},
		{
			description:     "module dir with managed mode and per-file overrides",
			input:           filepath.Join("testdata", "formats", "workspace_dir", "foo"),
			templateContent: v1ContentWithPerFileOverride,
			additionalFlags: []string{
				"--include-imports",
				"--include-wkt",
			},
			filesThatShouldExist: []string{
				"gen/java/ajavapkg/AProto.java",
				"gen/java/xyz/foo/BProto.java",
				"gen/java/bar/XProto.java",
				"gen/java/bar/YProto.java",
				"gen/java/dev/baz/v1/MProto.java",
				"gen/java/njavapkg/NProto.java",
				"gen/java/net/qux/TProto.java",
				"gen/java/com/google/protobuf/TimestampProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/xyz/foo/AProto.java",
				"gen/java/dev/baz/v1/NProto.java",
			},
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
			tempDir := t.TempDir()
			templatePath := filepath.Join(tempDir, "buf.gen.test.yaml")
			outDirBase := filepath.Join(tempDir, "out")
			err := os.WriteFile(templatePath, []byte(testcase.templateContent), 0600)
			require.NoError(t, err)
			outDir := outDirBase + "v1"
			argAndFlags := append(
				[]string{
					testcase.input,
					"--template",
					templatePath,
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
			outDir = outDirBase + "migrate"
			argAndFlags = append(
				[]string{
					testcase.input,
					"--template",
					templatePath,
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
				templatePath,
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
			requireVersionV2(t, templatePath)

			// generate with v2 template
			outDir = outDirBase + "v2"
			testRunSuccess(
				t,
				"--template",
				templatePath,
				"-o",
				outDir,
			)
			bucketV2, err := storageosProvider.NewReadWriteBucket(
				outDir,
				storageos.ReadWriteBucketWithSymlinksIfSupported(),
			)
			require.NoError(t, err)

			// generate with v2 template with --migrate flag
			outDir = outDirBase + "v2migrate"
			printedLastLine = runAndGetStderrSecondLastLine(
				t,
				"--template",
				templatePath,
				"-o",
				outDir,
				"--migrate",
			)
			require.True(t, strings.HasSuffix(printedLastLine, "is already in v2"))
			bucketV2Migrate, err := storageosProvider.NewReadWriteBucket(
				outDir,
				storageos.ReadWriteBucketWithSymlinksIfSupported(),
			)
			require.NoError(t, err)

			if !testcase.skipMigrateComp {
				diff, err := storage.DiffBytes(
					context.Background(),
					command.NewRunner(),
					bucketV1,
					bucketMigrate,
					transformGolangProtocVersionToUnknown(t),
				)
				require.NoError(t, err)
				require.Empty(t, string(diff))
			}

			diff, err := storage.DiffBytes(
				context.Background(),
				command.NewRunner(),
				bucketV1,
				bucketV2,
				transformGolangProtocVersionToUnknown(t),
			)
			require.NoError(t, err)
			require.Empty(t, string(diff))

			diff, err = storage.DiffBytes(
				context.Background(),
				command.NewRunner(),
				bucketV2,
				bucketV2Migrate,
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
	var filesInBucket []string
	if err != nil {
		bucket.Walk(
			context.Background(),
			"",
			func(oi storage.ObjectInfo) error {
				filesInBucket = append(filesInBucket, oi.Path())
				return nil
			},
		)
	}
	require.NoErrorf(t, err, "%s should exist but is not found among %v", fileName, filesInBucket)
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
	require.Errorf(t, err, "%s should not exist but is found", fileName)
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
