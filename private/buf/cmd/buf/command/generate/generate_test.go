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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufgen"
	"github.com/bufbuild/buf/private/buf/cmd/buf/internal/internaltesting"
	"github.com/bufbuild/buf/private/bufpkg/buftesting"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appcmd/appcmdtesting"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/testingextended"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: this has to change if we split up this repository
var buftestingDirPath = filepath.Join(
	"..",
	"..",
	"..",
	"..",
	"..",
	"..",
	"private",
	"bufpkg",
	"buftesting",
)

const (
	v1 = "v1"
	v2 = "v2"
)

func TestCompareGeneratedStubsGoogleapisGo(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubs(
		t,
		command.NewRunner(),
		googleapisDirPath,
		[]*testPluginInfo{
			{name: "go", opt: "Mgoogle/api/auth.proto=foo", isBinary: true},
		},
	)
}

func TestCompareGeneratedStubsGoogleapisGoZip(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubsArchive(
		t,
		command.NewRunner(),
		googleapisDirPath,
		[]*testPluginInfo{
			{name: "go", opt: "Mgoogle/api/auth.proto=foo", isBinary: true},
		},
		false,
	)
}

func TestCompareGeneratedStubsGoogleapisGoJar(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubsArchive(
		t,
		command.NewRunner(),
		googleapisDirPath,
		[]*testPluginInfo{
			{name: "go", opt: "Mgoogle/api/auth.proto=foo", isBinary: true},
		},
		true,
	)
}

func TestCompareGeneratedStubsGoogleapisObjc(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubs(
		t,
		command.NewRunner(),
		googleapisDirPath,
		[]*testPluginInfo{{name: "objc"}},
	)
}

func TestCompareGeneratedStubsGoogleapisPyi(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubs(
		t,
		command.NewRunner(),
		googleapisDirPath,
		[]*testPluginInfo{{name: "pyi"}},
	)
}

func TestCompareInsertionPointOutput(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	insertionTestdataDirPath := filepath.Join("testdata", "insertion")
	testCompareGeneratedStubs(
		t,
		command.NewRunner(),
		insertionTestdataDirPath,
		[]*testPluginInfo{
			{name: "insertion-point-receiver", isBinary: true},
			{name: "insertion-point-writer", isBinary: true},
		},
	)
}

func TestOutputFlag(t *testing.T) {
	t.Parallel()
	tempDirPath := t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "simple", "buf.gen.yaml"),
		filepath.Join("testdata", "simple"),
	)
	_, err := os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "A.java"))
	require.NoError(t, err)

	tempDirPath = t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "simple", "buf.genv2.yaml"),
		filepath.Join("testdata", "simple"),
	)
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "A.java"))
	require.NoError(t, err)
}

func TestProtoFileRefIncludePackageFiles(t *testing.T) {
	t.Parallel()
	tempDirPath := t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "protofileref", "buf.gen.yaml"),
		fmt.Sprintf("%s#include_package_files=true", filepath.Join("testdata", "protofileref", "a", "v1", "a.proto")),
	)
	_, err := os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "A.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "B.java"))
	require.NoError(t, err)

	tempDirPath = t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "protofileref", "buf.gen2.yaml"),
		fmt.Sprintf("%s#include_package_files=true", filepath.Join("testdata", "protofileref", "a", "v1", "a.proto")),
	)
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "A.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "B.java"))
	require.NoError(t, err)
}

func TestGenerateDuplicatePlugins(t *testing.T) {
	t.Parallel()
	tempDirPath := t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "duplicate_plugins", "buf.gen.yaml"),
		filepath.Join("testdata", "duplicate_plugins"),
	)
	_, err := os.Stat(filepath.Join(tempDirPath, "foo", "a", "v1", "A.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "bar", "a", "v1", "A.java"))
	require.NoError(t, err)

	tempDirPath = t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "duplicate_plugins", "buf.genv2.yaml"),
		filepath.Join("testdata", "duplicate_plugins"),
	)
	_, err = os.Stat(filepath.Join(tempDirPath, "foo", "a", "v1", "A.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "bar", "a", "v1", "A.java"))
	require.NoError(t, err)
}

func TestOutputWithPathEqualToExclude(t *testing.T) {
	t.Parallel()
	tempDirPath := t.TempDir()
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: cannot set the same path for both --path and --exclude-path flags: a/v1/a.proto`),
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "paths", "buf.gen.yaml"),
		"--exclude-path",
		filepath.Join("testdata", "paths", "a", "v1", "a.proto"),
		"--path",
		filepath.Join("testdata", "paths", "a", "v1", "a.proto"),
		filepath.Join("testdata", "paths"),
	)

	tempDirPath = t.TempDir()
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: cannot set the same path for both --path and --exclude-path flags: a/v1/a.proto`),
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "paths", "buf.genv2.yaml"),
		"--exclude-path",
		filepath.Join("testdata", "paths", "a", "v1", "a.proto"),
		"--path",
		filepath.Join("testdata", "paths", "a", "v1", "a.proto"),
		filepath.Join("testdata", "paths"),
	)
}

func TestGenerateInsertionPoint(t *testing.T) {
	t.Parallel()
	runner := command.NewRunner()
	testGenerateInsertionPoint(t, runner, ".", ".", filepath.Join("testdata", "insertion_point"))
	testGenerateInsertionPoint(t, runner, "gen/proto/insertion", "gen/proto/insertion", filepath.Join("testdata", "nested_insertion_point"))
	testGenerateInsertionPoint(t, runner, "gen/proto/insertion/", "./gen/proto/insertion", filepath.Join("testdata", "nested_insertion_point"))
}

func TestGenerateInsertionPointFail(t *testing.T) {
	t.Parallel()
	successTemplate := `
version: v1
plugins:
  - name: insertion-point-receiver
    out: gen/proto/insertion
  - name: insertion-point-writer
    out: .
`
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: plugin insertion-point-writer: test.txt: does not exist`,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		successTemplate,
		"-o",
		t.TempDir(),
	)

	successTemplate = `
version: v2
plugins:
  - binary: protoc-gen-insertion-point-receiver
    out: gen/proto/insertion
  - binary: protoc-gen-insertion-point-writer
    out: .
`
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: plugin protoc-gen-insertion-point-writer: test.txt: does not exist`,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		successTemplate,
		"-o",
		t.TempDir(),
	)
}

func TestGenerateDuplicateFileFail(t *testing.T) {
	t.Parallel()
	successTemplate := `
version: v1
plugins:
  - name: insertion-point-receiver
    out: .
  - name: insertion-point-receiver
    out: .
`
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: file "test.txt" was generated multiple times: once by plugin "insertion-point-receiver" and again by plugin "insertion-point-receiver"`,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		successTemplate,
		"-o",
		t.TempDir(),
	)

	successTemplate = `
version: v2
plugins:
  - binary: protoc-gen-insertion-point-receiver
    out: .
  - binary: protoc-gen-insertion-point-receiver
    out: .
`
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: file "test.txt" was generated multiple times: once by plugin "protoc-gen-insertion-point-receiver" and again by plugin "protoc-gen-insertion-point-receiver"`,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		successTemplate,
		"-o",
		t.TempDir(),
	)
}

func TestGenerateInsertionPointMixedPathsFail(t *testing.T) {
	t.Parallel()
	wd, err := os.Getwd()
	require.NoError(t, err)
	testGenerateInsertionPointMixedPathsFail(t, ".", wd)
	testGenerateInsertionPointMixedPathsFail(t, wd, ".")
}

func TestGenerateWithV1AndV2(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	v1Simple := `version: v1
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
	v1WithTypes := `version: v1
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
	v1WithPerFileOverride := `version: v1
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
	v1ContentWithStrategyAndOpt := `version: v1
plugins:
  - plugin: plugin-config
    out: gen/sall
    strategy: all
    opt: "a=b"
  - plugin: plugin-config
    out: gen/sdir
    strategy: directory
    opt:
      - c=d
      - xyz
      - a=b=c
  - plugin: plugin-config
    out: gen/default
`
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
			templateContent: v1Simple,
			additionalFlags: []string{
				"--path",
				filepath.Join("testdata", "paths", "b", "v1", "b.proto"),
				"--path",
				filepath.Join("testdata", "paths", "a"),
				"--exclude-path",
				filepath.Join("testdata", "paths", "a", "v2"),
			},
			filesThatShouldExist: []string{
				"gen/java/net/a/v1/AProto.java",
				"gen/java/net/a/v3/AProto.java",
				"gen/java/net/a/v3/foo/BarProto.java",
				"gen/java/net/b/v1/BProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/net/a/v2/AProto.java",
			},
		},
		{
			description:     "types only new flags",
			input:           filepath.Join("testdata", "paths"),
			templateContent: v1Simple,
			additionalFlags: []string{
				"--type",
				"b.v1.Bar",
				"--type",
				"a.v3.foo.Bar",
				"--type",
				"a.v2.Foo",
			},
			filesThatShouldExist: []string{
				"gen/java/net/b/v1/BProto.java",
				"gen/java/net/a/v3/foo/BarProto.java",
				"gen/java/net/a/v2/AProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/net/a/v1/AProto.java",
				"gen/java/net/a/v3/AProto.java",
			},
		},
		{
			description:     "types only old flags",
			input:           filepath.Join("testdata", "paths"),
			templateContent: v1Simple,
			additionalFlags: []string{
				"--include-types",
				"a.v1.Foo",
				"--include-types",
				"a.v3.foo.Foo",
				"--include-types",
				"a.v3.Foo",
			},
			filesThatShouldExist: []string{
				"gen/java/net/a/v1/AProto.java",
				"gen/java/net/a/v3/AProto.java",
				"gen/java/net/a/v3/foo/FooProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/net/a/v3/foo/BarProto.java",
				"gen/java/net/b/v1/BProto.java",
				"gen/java/net/a/v2/AProto.java",
			},
		},
		{
			description:     "types both new and old flags",
			input:           filepath.Join("testdata", "paths"),
			templateContent: v1Simple,
			additionalFlags: []string{
				"--type",
				"b.v1.Bar",
				"--include-types",
				"a.v3.foo.Bar",
				"--type",
				"a.v2.Foo",
			},
			filesThatShouldExist: []string{
				"gen/java/net/b/v1/BProto.java",
				"gen/java/net/a/v3/foo/BarProto.java",
				"gen/java/net/a/v2/AProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/net/a/v1/AProto.java",
				"gen/java/net/a/v3/AProto.java",
			},
		},
		{
			description:     "types in both flags and v1 config",
			input:           filepath.Join("testdata", "paths"),
			templateContent: v1WithTypes,
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
			templateContent: v1WithTypes,
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
			templateContent: v1WithTypes,
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
			templateContent: v1Simple,
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
			templateContent: v1Simple,
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
			templateContent: v1Simple,
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
			templateContent: v1Simple,
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
			templateContent: v1Simple,
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
			templateContent: v1Simple,
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
			templateContent: v1Simple,
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
			input:           filepath.Join("testdata", "formats", "not_module", "src", "protos"),
			templateContent: v1Simple,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/a/AProto.java",
				"gen/java/net/b/BProto.java",
			},
		},
		{
			description:     "zip archive",
			input:           filepath.Join("testdata", "formats", "not_module.zip#strip_components=3"),
			templateContent: v1Simple,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/a/AProto.java",
				"gen/java/net/b/BProto.java",
			},
		},
		{
			description:     "zip archive with sub dir",
			input:           filepath.Join("testdata", "formats", "not_module.zip#subdir=not_module/src/protos"),
			templateContent: v1Simple,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/a/AProto.java",
				"gen/java/net/b/BProto.java",
			},
		},
		{
			description:     "zip archive with sub dir and strip components",
			input:           filepath.Join("testdata", "formats", "not_module.zip#strip_components=1,subdir=src/protos"),
			templateContent: v1Simple,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/a/AProto.java",
				"gen/java/net/b/BProto.java",
			},
		},
		{
			description:     "tape archive",
			input:           filepath.Join("testdata", "formats", "not_module.tar#strip_components=3"),
			templateContent: v1Simple,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/a/AProto.java",
				"gen/java/net/b/BProto.java",
			},
		},
		{
			description:     "tape archive with compression specified",
			input:           filepath.Join("testdata", "formats", "not_module_gzip#format=tar,strip_components=3,compression=gzip"),
			templateContent: v1Simple,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/a/AProto.java",
				"gen/java/net/b/BProto.java",
			},
		},
		{
			description:     "tape archive with compression inferred",
			input:           filepath.Join("testdata", "formats", "not_module.tar.gz#strip_components=3"),
			templateContent: v1Simple,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/a/AProto.java",
				"gen/java/net/b/BProto.java",
			},
		},
		{
			description:     "proto file ref",
			input:           filepath.Join("testdata", "protofileref", "a", "v1", "a.proto"),
			templateContent: v1Simple,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/a/v1/AProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/net/a/v1/BProto.java",
			},
		},
		{
			description:     "proto file ref include package file",
			input:           filepath.Join("testdata", "protofileref", "a", "v1", "a.proto#include_package_files=true"),
			templateContent: v1Simple,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/java/net/a/v1/AProto.java",
				"gen/java/net/a/v1/BProto.java",
			},
		},
		{
			description:     "module dir with managed mode and per-file overrides",
			input:           filepath.Join("testdata", "formats", "workspace_dir", "foo"),
			templateContent: v1WithPerFileOverride,
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
		{
			description:     "module dir with file options with managed mode and per-file overrides",
			input:           filepath.Join("testdata", "formats", "workspace_dir_with_file_options", "foo"),
			templateContent: v1WithPerFileOverride,
			additionalFlags: []string{
				"--include-imports",
				"--include-wkt",
			},
			filesThatShouldExist: []string{
				"gen/java/ajavapkg/AProto.java",
				"gen/java/xyz/foo/BProto.java",
				"gen/java/foo/XProto.java",
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
		{
			description:     "module dir without packages with managed mode and per-file overrides",
			input:           filepath.Join("testdata", "formats", "workspace_dir_without_package", "foo"),
			templateContent: v1WithPerFileOverride,
			additionalFlags: []string{
				"--include-imports",
				"--include-wkt",
			},
			filesThatShouldExist: []string{
				"gen/java/ajavapkg/AProto.java",
				"gen/java/original/javapkg/from/a/BProto.java",
				"gen/java/XProto.java",
				"gen/java/YProto.java",
				"gen/java/MProto.java",
				"gen/java/njavapkg/NProto.java",
				"gen/java/foo/TProto.java",
				"gen/java/com/google/protobuf/Timestamp.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/xyz/BProto.java",
				"gen/java/xyz/AProto.java",
				"gen/java/bar/XProto.java",
				"gen/java/bar/YProto.java",
				"gen/java/dev/NProto.java",
				"gen/java/dev/MProto.java",
				"gen/java/net/TProto.java",
			},
		},
		{
			description:     "module dir without package with file options with managed mode and per-file overrides",
			input:           filepath.Join("testdata", "formats", "workspace_dir_without_package_with_options", "foo"),
			templateContent: v1WithPerFileOverride,
			additionalFlags: []string{
				"--include-imports",
				"--include-wkt",
			},
			filesThatShouldExist: []string{
				"gen/java/ajavapkg/AProto.java",
				"gen/java/foobarbaz/BProto.java",
				"gen/java/barfoobar/XProto.java",
				"gen/java/barfoobar/YProto.java",
				"gen/java/bazfoobar/MProto.java",
				"gen/java/njavapkg/NProto.java",
				"gen/java/foo/T.java",
				"gen/java/com/google/protobuf/Timestamp.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/xyzAProto.java",
				"gen/java/xyz/BProto.java",
				"gen/java/XProto.java",
				"gen/java/YProto.java",
				"gen/java/MProto.java",
				"gen/java/bazfoobar/NProto.java",
				"gen/java/dev/baz/v1/NProto.java",
			},
		},
		{
			description:     "strategy and opt per plugin",
			input:           filepath.Join("testdata", "formats", "not_module", "src", "protos"),
			templateContent: v1ContentWithStrategyAndOpt,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/sall/a/a.response.txt",
				"gen/sall/b/b.response.txt",
				"gen/sdir/a/a.response.txt",
				"gen/sdir/b/b.response.txt",
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

func TestPerPluginIncludeImports(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	v2BothPluginsIncludeImports := `version: v2
managed:
  enabled: true
  override:
    - file_option: go_package
      value: example.com/protos
    - file_option: java_multiple_files
      value: false
plugins:
  - protoc_builtin: java
    out: gen/java
    include_imports: true
    include_wkt: true
  - binary: protoc-gen-go
    out: gen/go
    include_imports: true
    include_wkt: true
`
	v2OnlyOnePluginIncludesWKT := `version: v2
managed:
  enabled: true
  override:
    - file_option: go_package
      value: example.com/protos
    - file_option: java_multiple_files
      value: false
plugins:
  - protoc_builtin: java
    out: gen/java
    include_imports: true
    include_wkt: true
  - binary: protoc-gen-go
    out: gen/go
`
	v2OnlyOnePluginIncludesImports := `version: v2
managed:
  enabled: true
  override:
    - file_option: go_package
      value: example.com/protos
    - file_option: java_multiple_files
      value: false
plugins:
  - protoc_builtin: java
    out: gen/java
    include_imports: true
  - binary: protoc-gen-go
    out: gen/go
`

	testcases := []struct {
		description             string
		input                   string
		templateContent         string
		additionalFlags         []string
		filesThatShouldExist    []string
		filesThatShouldNotExist []string
	}{
		{
			description:     "both include imports and wkt",
			input:           filepath.Join("testdata", "formats", "workspace_dir", "foo"),
			templateContent: v2BothPluginsIncludeImports,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/go/example.com/protos/a.pb.go",
				"gen/go/example.com/protos/b.pb.go",
				"gen/go/example.com/protos/m.pb.go",
				"gen/go/example.com/protos/n.pb.go",
				"gen/go/example.com/protos/t.pb.go",
				"gen/go/example.com/protos/x.pb.go",
				"gen/go/example.com/protos/y.pb.go",
				"gen/go/google.golang.org/protobuf/types/known/timestamppb/timestamp.pb.go",
				"gen/java/com/bar/XProto.java",
				"gen/java/com/bar/YProto.java",
				"gen/java/com/baz/v1/MProto.java",
				"gen/java/com/baz/v1/NProto.java",
				"gen/java/com/foo/AProto.java",
				"gen/java/com/foo/BProto.java",
				"gen/java/com/google/protobuf/Timestamp.java",
				"gen/java/com/qux/TProto.java",
			},
		},
		{
			description:     "only one plugin includes WKT",
			input:           filepath.Join("testdata", "formats", "workspace_dir", "foo"),
			templateContent: v2OnlyOnePluginIncludesWKT,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/go/example.com/protos/a.pb.go",
				"gen/go/example.com/protos/b.pb.go",
				"gen/java/com/bar/XProto.java",
				"gen/java/com/bar/YProto.java",
				"gen/java/com/baz/v1/MProto.java",
				"gen/java/com/baz/v1/NProto.java",
				"gen/java/com/foo/AProto.java",
				"gen/java/com/foo/BProto.java",
				"gen/java/com/google/protobuf/Timestamp.java",
				"gen/java/com/qux/TProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/go/example.com/protos/m.pb.go",
				"gen/go/example.com/protos/n.pb.go",
				"gen/go/example.com/protos/t.pb.go",
				"gen/go/example.com/protos/x.pb.go",
				"gen/go/example.com/protos/y.pb.go",
				"gen/go/google.golang.org/protobuf/types/known/timestamppb/timestamp.pb.go",
			},
		},
		{
			description:     "only one plugin includes imports",
			input:           filepath.Join("testdata", "formats", "workspace_dir", "foo"),
			templateContent: v2OnlyOnePluginIncludesImports,
			additionalFlags: nil,
			filesThatShouldExist: []string{
				"gen/go/example.com/protos/a.pb.go",
				"gen/go/example.com/protos/b.pb.go",
				"gen/java/com/bar/XProto.java",
				"gen/java/com/bar/YProto.java",
				"gen/java/com/baz/v1/MProto.java",
				"gen/java/com/baz/v1/NProto.java",
				"gen/java/com/foo/AProto.java",
				"gen/java/com/foo/BProto.java",
				"gen/java/com/qux/TProto.java",
			},
			filesThatShouldNotExist: []string{
				"gen/java/com/google/protobuf/Timestamp.java",
				"gen/go/example.com/protos/m.pb.go",
				"gen/go/example.com/protos/n.pb.go",
				"gen/go/example.com/protos/t.pb.go",
				"gen/go/example.com/protos/x.pb.go",
				"gen/go/example.com/protos/y.pb.go",
				"gen/go/google.golang.org/protobuf/types/known/timestamppb/timestamp.pb.go",
			},
		},
		{
			description:     "only one plugin includes WKT with include imports flag",
			input:           filepath.Join("testdata", "formats", "workspace_dir", "foo"),
			templateContent: v2OnlyOnePluginIncludesWKT,
			additionalFlags: []string{
				"--include-imports",
			},
			filesThatShouldExist: []string{
				"gen/go/example.com/protos/a.pb.go",
				"gen/go/example.com/protos/b.pb.go",
				"gen/java/com/bar/XProto.java",
				"gen/java/com/bar/YProto.java",
				"gen/java/com/baz/v1/MProto.java",
				"gen/java/com/baz/v1/NProto.java",
				"gen/java/com/foo/AProto.java",
				"gen/java/com/foo/BProto.java",
				"gen/java/com/google/protobuf/Timestamp.java",
				"gen/java/com/qux/TProto.java",
				"gen/go/example.com/protos/m.pb.go",
				"gen/go/example.com/protos/n.pb.go",
				"gen/go/example.com/protos/t.pb.go",
				"gen/go/example.com/protos/x.pb.go",
				"gen/go/example.com/protos/y.pb.go",
			},
			filesThatShouldNotExist: []string{
				"gen/go/google.golang.org/protobuf/types/known/timestamppb/timestamp.pb.go",
			},
		},
		{
			description:     "only one plugin includes imports with include imports flag",
			input:           filepath.Join("testdata", "formats", "workspace_dir", "foo"),
			templateContent: v2OnlyOnePluginIncludesImports,
			additionalFlags: []string{
				"--include-imports",
			},
			filesThatShouldExist: []string{
				"gen/go/example.com/protos/a.pb.go",
				"gen/go/example.com/protos/b.pb.go",
				"gen/java/com/bar/XProto.java",
				"gen/java/com/bar/YProto.java",
				"gen/java/com/baz/v1/MProto.java",
				"gen/java/com/baz/v1/NProto.java",
				"gen/java/com/foo/AProto.java",
				"gen/java/com/foo/BProto.java",
				"gen/java/com/qux/TProto.java",
				"gen/go/example.com/protos/m.pb.go",
				"gen/go/example.com/protos/n.pb.go",
				"gen/go/example.com/protos/t.pb.go",
				"gen/go/example.com/protos/x.pb.go",
				"gen/go/example.com/protos/y.pb.go",
			},
			filesThatShouldNotExist: []string{
				"gen/java/com/google/protobuf/Timestamp.java",
				"gen/go/google.golang.org/protobuf/types/known/timestamppb/timestamp.pb.go",
			},
		},
		{
			description:     "only one plugin includes imports with include WKT flag",
			input:           filepath.Join("testdata", "formats", "workspace_dir", "foo"),
			templateContent: v2OnlyOnePluginIncludesImports,
			additionalFlags: []string{
				"--include-imports",
				"--include-wkt",
			},
			filesThatShouldExist: []string{
				"gen/go/example.com/protos/a.pb.go",
				"gen/go/example.com/protos/b.pb.go",
				"gen/java/com/bar/XProto.java",
				"gen/java/com/bar/YProto.java",
				"gen/java/com/baz/v1/MProto.java",
				"gen/java/com/baz/v1/NProto.java",
				"gen/java/com/foo/AProto.java",
				"gen/java/com/foo/BProto.java",
				"gen/java/com/qux/TProto.java",
				"gen/java/com/google/protobuf/Timestamp.java",
				"gen/go/example.com/protos/m.pb.go",
				"gen/go/example.com/protos/n.pb.go",
				"gen/go/example.com/protos/t.pb.go",
				"gen/go/example.com/protos/x.pb.go",
				"gen/go/example.com/protos/y.pb.go",
				"gen/go/google.golang.org/protobuf/types/known/timestamppb/timestamp.pb.go",
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
			outDir := filepath.Join(tempDir, "out")
			err := os.WriteFile(templatePath, []byte(testcase.templateContent), 0600)
			require.NoError(t, err)
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
			bucket, err := storageosProvider.NewReadWriteBucket(
				outDir,
				storageos.ReadWriteBucketWithSymlinksIfSupported(),
			)
			require.NoError(t, err)

			for _, fileThatShouldExist := range testcase.filesThatShouldExist {
				requireFileExists(t, bucket, fileThatShouldExist)
			}
			for _, fileThatShouldNotExist := range testcase.filesThatShouldNotExist {
				requireFileDoesNotExist(t, bucket, fileThatShouldNotExist)
			}
		})
	}
}

func testGenerateInsertionPoint(
	t *testing.T,
	runner command.Runner,
	receiverOut string,
	writerOut string,
	expectedOutputPath string,
) {
	testGenerateInsertionPointV1(
		t,
		runner,
		receiverOut,
		writerOut,
		expectedOutputPath,
	)
	testGenerateInsertionPointV2(
		t,
		runner,
		receiverOut,
		writerOut,
		expectedOutputPath,
	)
}

func testGenerateInsertionPointV1(
	t *testing.T,
	runner command.Runner,
	receiverOut string,
	writerOut string,
	expectedOutputPath string,
) {
	successTemplate := `
version: v1
plugins:
  - name: insertion-point-receiver
    out: %s
  - name: insertion-point-writer
    out: %s
`
	storageosProvider := storageos.NewProvider()
	tempDir, readWriteBucket := internaltesting.CopyReadBucketToTempDir(
		context.Background(),
		t,
		storageosProvider,
		storagemem.NewReadWriteBucket(),
	)
	testRunSuccess(
		t,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		fmt.Sprintf(successTemplate, receiverOut, writerOut),
		"-o",
		tempDir,
	)
	expectedOutput, err := storageosProvider.NewReadWriteBucket(expectedOutputPath)
	require.NoError(t, err)
	diff, err := storage.DiffBytes(context.Background(), runner, expectedOutput, readWriteBucket)
	require.NoError(t, err)
	require.Empty(t, string(diff))
}

func testGenerateInsertionPointV2(
	t *testing.T,
	runner command.Runner,
	receiverOut string,
	writerOut string,
	expectedOutputPath string,
) {
	successTemplate := `
version: v2
plugins:
  - binary: protoc-gen-insertion-point-receiver
    out: %s
  - binary: protoc-gen-insertion-point-writer
    out: %s
`
	storageosProvider := storageos.NewProvider()
	tempDir, readWriteBucket := internaltesting.CopyReadBucketToTempDir(
		context.Background(),
		t,
		storageosProvider,
		storagemem.NewReadWriteBucket(),
	)
	testRunSuccess(
		t,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		fmt.Sprintf(successTemplate, receiverOut, writerOut),
		"-o",
		tempDir,
	)
	expectedOutput, err := storageosProvider.NewReadWriteBucket(expectedOutputPath)
	require.NoError(t, err)
	diff, err := storage.DiffBytes(context.Background(), runner, expectedOutput, readWriteBucket)
	require.NoError(t, err)
	require.Empty(t, string(diff))
}

// testGenerateInsertionPointMixedPathsFail demonstrates that insertion points are only
// able to generate to the same output directory, even if the absolute path points to the
// same place. This is equivalent to protoc's behavior.
func testGenerateInsertionPointMixedPathsFail(t *testing.T, receiverOut string, writerOut string) {
	successTemplate := `
version: v1
plugins:
  - name: insertion-point-receiver
    out: %s
  - name: insertion-point-writer
    out: %s
`
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: plugin insertion-point-writer: test.txt: does not exist`,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		fmt.Sprintf(successTemplate, receiverOut, writerOut),
		"-o",
		t.TempDir(),
	)

	successTemplate = `
version: v2
plugins:
  - binary: protoc-gen-insertion-point-receiver
    out: %s
  - binary: protoc-gen-insertion-point-writer
    out: %s
`
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: plugin protoc-gen-insertion-point-writer: test.txt: does not exist`,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		fmt.Sprintf(successTemplate, receiverOut, writerOut),
		"-o",
		t.TempDir(),
	)
}

func testCompareGeneratedStubs(
	t *testing.T,
	runner command.Runner,
	dirPath string,
	testPluginInfos []*testPluginInfo,
) {
	filePaths := buftesting.GetProtocFilePaths(t, dirPath, 100)
	actualProtocDir := t.TempDir()
	var actualProtocPluginFlags []string
	for _, testPluginInfo := range testPluginInfos {
		actualProtocPluginFlags = append(actualProtocPluginFlags, fmt.Sprintf("--%s_out=%s", testPluginInfo.name, actualProtocDir))
		if testPluginInfo.opt != "" {
			actualProtocPluginFlags = append(actualProtocPluginFlags, fmt.Sprintf("--%s_opt=%s", testPluginInfo.name, testPluginInfo.opt))
		}
	}
	buftesting.RunActualProtoc(
		t,
		runner,
		false,
		false,
		dirPath,
		filePaths,
		map[string]string{
			"PATH": os.Getenv("PATH"),
		},
		nil,
		actualProtocPluginFlags...,
	)
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	actualReadWriteBucket, err := storageosProvider.NewReadWriteBucket(
		actualProtocDir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)

	bufReadWriteBucket := generateIntoBucket(
		t,
		v1,
		storageosProvider,
		dirPath,
		testPluginInfos,
		filePaths,
	)
	diff, err := storage.DiffBytes(
		context.Background(),
		runner,
		actualReadWriteBucket,
		bufReadWriteBucket,
		transformGolangProtocVersionToUnknown(t),
	)
	require.NoError(t, err)
	assert.Empty(t, string(diff))

	bufReadWriteBucket = generateIntoBucket(
		t,
		v2,
		storageosProvider,
		dirPath,
		testPluginInfos,
		filePaths,
	)
	diff, err = storage.DiffBytes(
		context.Background(),
		runner,
		actualReadWriteBucket,
		bufReadWriteBucket,
		transformGolangProtocVersionToUnknown(t),
	)
	require.NoError(t, err)
	assert.Empty(t, string(diff))
}

func testCompareGeneratedStubsArchive(
	t *testing.T,
	runner command.Runner,
	dirPath string,
	testPluginInfos []*testPluginInfo,
	useJar bool,
) {
	fileExt := ".zip"
	if useJar {
		fileExt = ".jar"
	}
	filePaths := buftesting.GetProtocFilePaths(t, dirPath, 100)
	tempDir := t.TempDir()
	actualProtocFile := filepath.Join(tempDir, "actual-protoc"+fileExt)
	var actualProtocPluginFlags []string
	for _, testPluginInfo := range testPluginInfos {
		actualProtocPluginFlags = append(actualProtocPluginFlags, fmt.Sprintf("--%s_out=%s", testPluginInfo.name, actualProtocFile))
		if testPluginInfo.opt != "" {
			actualProtocPluginFlags = append(actualProtocPluginFlags, fmt.Sprintf("--%s_opt=%s", testPluginInfo.name, testPluginInfo.opt))
		}
	}
	buftesting.RunActualProtoc(
		t,
		runner,
		false,
		false,
		dirPath,
		filePaths,
		map[string]string{
			"PATH": os.Getenv("PATH"),
		},
		nil,
		actualProtocPluginFlags...,
	)
	actualData, err := os.ReadFile(actualProtocFile)
	require.NoError(t, err)
	actualReadWriteBucket := storagemem.NewReadWriteBucket()
	err = storagearchive.Unzip(
		context.Background(),
		bytes.NewReader(actualData),
		int64(len(actualData)),
		actualReadWriteBucket,
		nil,
		0,
	)
	require.NoError(t, err)

	bufReadWriteBucket := generateZipIntoBucket(
		t,
		v1,
		fileExt,
		dirPath,
		testPluginInfos,
		filePaths,
	)
	diff, err := storage.DiffBytes(
		context.Background(),
		runner,
		actualReadWriteBucket,
		bufReadWriteBucket,
		transformGolangProtocVersionToUnknown(t),
	)
	require.NoError(t, err)
	assert.Empty(t, string(diff))

	bufReadWriteBucket = generateZipIntoBucket(
		t,
		v2,
		fileExt,
		dirPath,
		testPluginInfos,
		filePaths,
	)
	diff, err = storage.DiffBytes(
		context.Background(),
		runner,
		actualReadWriteBucket,
		bufReadWriteBucket,
		transformGolangProtocVersionToUnknown(t),
	)
	require.NoError(t, err)
	assert.Empty(t, string(diff))
}

func testRunSuccess(t *testing.T, args ...string) {
	appcmdtesting.RunCommandSuccess(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appflag.NewBuilder(name),
			)
		},
		internaltesting.NewEnvFunc(t),
		nil,
		nil,
		args...,
	)
}

func testRunStdoutStderr(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStdout string, expectedStderr string, args ...string) {
	appcmdtesting.RunCommandExitCodeStdoutStderr(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appflag.NewBuilder(name),
			)
		},
		expectedExitCode,
		expectedStdout,
		expectedStderr,
		internaltesting.NewEnvFunc(t),
		stdin,
		args...,
	)
}

func generateIntoBucket(
	t *testing.T,
	version string,
	storageosProvider storageos.Provider,
	dirPath string,
	testPluginInfos []*testPluginInfo,
	filePaths []string,
) storage.ReadWriteBucket {
	out := t.TempDir()
	var genFlags []string
	switch version {
	case v1:
		genFlags = []string{
			dirPath,
			"--template",
			newExternalConfigV1String(t, testPluginInfos, out),
		}
	case v2:
		genFlags = []string{
			dirPath,
			"--template",
			newExternalConfigV2String(t, testPluginInfos, out),
		}
	default:
		require.Fail(t, "invalid test case")
	}
	for _, filePath := range filePaths {
		genFlags = append(
			genFlags,
			"--path",
			filePath,
		)
	}
	testRunSuccess(
		t,
		genFlags...,
	)
	bufReadWriteBucket, err := storageosProvider.NewReadWriteBucket(
		out,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	return bufReadWriteBucket
}

func generateZipIntoBucket(
	t *testing.T,
	version string,
	fileExt string,
	dirPath string,
	testPluginInfos []*testPluginInfo,
	filePaths []string,
) storage.ReadWriteBucket {
	out := filepath.Join(t.TempDir(), "buf-generate"+fileExt)
	var genFlags []string
	switch version {
	case v1:
		genFlags = []string{
			dirPath,
			"--template",
			newExternalConfigV1String(t, testPluginInfos, out),
		}
	case v2:
		genFlags = []string{
			dirPath,
			"--template",
			newExternalConfigV2String(t, testPluginInfos, out),
		}
	default:
		require.Fail(t, "invalid test case")
	}
	for _, filePath := range filePaths {
		genFlags = append(
			genFlags,
			"--path",
			filePath,
		)
	}
	testRunSuccess(
		t,
		genFlags...,
	)
	bufData, err := os.ReadFile(out)
	require.NoError(t, err)
	bufReadWriteBucket := storagemem.NewReadWriteBucket()
	err = storagearchive.Unzip(
		context.Background(),
		bytes.NewReader(bufData),
		int64(len(bufData)),
		bufReadWriteBucket,
		nil,
		0,
	)
	require.NoError(t, err)
	return bufReadWriteBucket
}

func newExternalConfigV1String(t *testing.T, plugins []*testPluginInfo, out string) string {
	externalConfig := bufgen.ExternalConfigV1{
		Version: "v1",
	}
	for _, plugin := range plugins {
		externalConfig.Plugins = append(
			externalConfig.Plugins,
			bufgen.ExternalPluginConfigV1{
				Name: plugin.name,
				Out:  out,
				Opt:  plugin.opt,
			},
		)
	}
	data, err := json.Marshal(externalConfig)
	require.NoError(t, err)
	return string(data)
}

func newExternalConfigV2String(t *testing.T, plugins []*testPluginInfo, out string) string {
	externalConfig := bufgen.ExternalConfigV2{
		Version: "v2",
	}
	for _, plugin := range plugins {
		var pluginConfig bufgen.ExternalPluginConfigV2
		if plugin.isBinary {
			pluginConfig = bufgen.ExternalPluginConfigV2{
				Binary: fmt.Sprintf("protoc-gen-%s", plugin.name),
				Opt:    plugin.opt,
				Out:    out,
			}
		} else {
			pluginConfig = bufgen.ExternalPluginConfigV2{
				ProtocBuiltin: &plugin.name,
				Opt:           plugin.opt,
				Out:           out,
			}
		}
		externalConfig.Plugins = append(
			externalConfig.Plugins,
			pluginConfig,
		)
	}
	data, err := json.Marshal(externalConfig)
	require.NoError(t, err)
	return string(data)
}

type testPluginInfo struct {
	name     string
	opt      string
	isBinary bool
}

func transformGolangProtocVersionToUnknown(t *testing.T) storage.DiffOption {
	return storage.DiffWithTransform(func(_, _ string, content []byte) []byte {
		lines := bytes.Split(content, []byte("\n"))
		filteredLines := make([][]byte, 0, len(lines))
		commentPrefix := []byte("//")
		protocVersionIndicator := []byte("protoc")
		for _, line := range lines {
			if !(bytes.HasPrefix(line, commentPrefix) && bytes.Contains(line, protocVersionIndicator)) {
				filteredLines = append(filteredLines, line)
			}
		}
		return bytes.Join(filteredLines, []byte("\n"))
	})
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
		walkErr := bucket.Walk(
			context.Background(),
			"",
			func(oi storage.ObjectInfo) error {
				filesInBucket = append(filesInBucket, oi.Path())
				return nil
			},
		)
		require.NoError(t, walkErr)
	}
	require.NoErrorf(t, err, "%s should exist but is not found among: \n%s\n", fileName, strings.Join(filesInBucket, "\n"))
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
