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

package bufgen

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufgen/internal"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/bufgenplugin"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/bufgenv1"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/bufgenv2"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestConvertV1ToV2Success(t *testing.T) {
	t.Parallel()
	defaultExpectedInputs := []bufgenv2.ExternalInputConfigV2{
		{
			Directory: toPointer("."),
		},
	}
	placeHolderPluginsV1 := []bufgenplugin.PluginConfig{
		mustCreateNewCuratedPlugin(
			t,
			"buf.build/bufbuild/es:v1.3.0",
			0,
			"gen/es",
			"",
		),
	}
	expectedPlaceHolderPluginsV2 := []bufgenv2.ExternalPluginConfigV2{
		{
			Remote: toPointer("buf.build/bufbuild/es:v1.3.0"),
			Out:    "gen/es",
			Opt:    nil,
		},
	}
	testcases := []struct {
		description    string
		original       *bufgenv1.Config
		expected       *ExternalConfigV2
		findPlugin     func(string) (string, error)
		input          string
		types          []string
		includePaths   []string
		excludePaths   []string
		includeImports bool
		includeWKT     bool
	}{
		{
			description: "local plugin that can be found locally as binary",
			original: &bufgenv1.Config{
				PluginConfigs: []bufgenplugin.PluginConfig{
					mustCreateLocalPluginConfig(
						t,
						"somelocal",
						internal.StrategyDirectory,
						"gen/somelocal",
						"a=b",
					),
				},
			},
			expected: &bufgenv2.ExternalConfigV2{
				Version: "v2",
				Plugins: []bufgenv2.ExternalPluginConfigV2{
					{
						Binary:   "protoc-gen-somelocal",
						Out:      "gen/somelocal",
						Opt:      "a=b",
						Strategy: toPointer("directory"),
					},
				},
				Managed: bufgenv2.ExternalManagedConfigV2{
					Enabled: false,
				},
				Inputs: defaultExpectedInputs,
			},
		},
		{
			description: "local plugin that is builtin to protoc",
			original: &bufgenv1.Config{
				PluginConfigs: []bufgenplugin.PluginConfig{
					mustCreateLocalPluginConfig(
						t,
						"java",
						internal.StrategyAll,
						"gen/java",
						"a=b,c",
					),
				},
			},
			findPlugin: func(s string) (string, error) {
				return "", errors.New("plugin not found")
			},
			expected: &bufgenv2.ExternalConfigV2{
				Version: "v2",
				Plugins: []bufgenv2.ExternalPluginConfigV2{
					{
						ProtocBuiltin: toPointer("java"),
						Out:           "gen/java",
						Opt:           []string{"a=b", "c"},
						Strategy:      toPointer("all"),
					},
				},
				Managed: bufgenv2.ExternalManagedConfigV2{
					Enabled: false,
				},
				Inputs: defaultExpectedInputs,
			},
		},
		{
			description: "binary plugin with args",
			original: &bufgenv1.Config{
				PluginConfigs: []bufgenplugin.PluginConfig{
					mustCreateNewBinaryPlugin(
						t,
						"go",
						[]string{"go", "run", "google.golang.org/protobuf/cmd/protoc-gen-go"},
						internal.StrategyAll,
						"gen/bin1",
						"a,b=c,d,e,f=g",
					),
				},
			},
			expected: &bufgenv2.ExternalConfigV2{
				Version: "v2",
				Plugins: []bufgenv2.ExternalPluginConfigV2{
					{
						Binary:   []string{"go", "run", "google.golang.org/protobuf/cmd/protoc-gen-go"},
						Out:      "gen/bin1",
						Opt:      []string{"a", "b=c", "d", "e", "f=g"},
						Strategy: toPointer("all"),
					},
				},
				Managed: bufgenv2.ExternalManagedConfigV2{
					Enabled: false,
				},
				Inputs: defaultExpectedInputs,
			},
		},
		{
			description: "binary plugin without args",
			original: &bufgenv1.Config{
				PluginConfigs: []bufgenplugin.PluginConfig{
					mustCreateNewBinaryPlugin(
						t,
						"validate",
						[]string{"protoc-gen-validate"},
						internal.StrategyAll,
						"gen/bin1",
						"a,b=c,d,e,f=g",
					),
				},
			},
			expected: &bufgenv2.ExternalConfigV2{
				Version: "v2",
				Plugins: []bufgenv2.ExternalPluginConfigV2{
					{
						Binary:   "protoc-gen-validate",
						Out:      "gen/bin1",
						Opt:      []string{"a", "b=c", "d", "e", "f=g"},
						Strategy: toPointer("all"),
					},
				},
				Managed: bufgenv2.ExternalManagedConfigV2{
					Enabled: false,
				},
				Inputs: defaultExpectedInputs,
			},
		},
		{
			description: "protoc builtin plugin",
			original: &bufgenv1.Config{
				PluginConfigs: []bufgenplugin.PluginConfig{
					mustCreateNewProtocPlugin(
						t,
						"cpp",
						"/bin/protoc",
						"gen/cpp",
						"",
						internal.StrategyAll,
					),
				},
			},
			expected: &bufgenv2.ExternalConfigV2{
				Version: "v2",
				Plugins: []bufgenv2.ExternalPluginConfigV2{
					{
						ProtocBuiltin: toPointer("cpp"),
						ProtocPath:    toPointer("/bin/protoc"),
						Out:           "gen/cpp",
						Opt:           nil,
						Strategy:      toPointer("all"),
					},
				},
				Managed: bufgenv2.ExternalManagedConfigV2{
					Enabled: false,
				},
				Inputs: defaultExpectedInputs,
			},
		},
		{
			description: "curated plugin",
			original: &bufgenv1.Config{
				PluginConfigs: []bufgenplugin.PluginConfig{
					mustCreateNewCuratedPlugin(
						t,
						"buf.build/bufbuild/es:v1.3.0",
						2,
						"gen/es",
						"",
					),
				},
			},
			expected: &bufgenv2.ExternalConfigV2{
				Version: "v2",
				Plugins: []bufgenv2.ExternalPluginConfigV2{
					{
						Remote:   toPointer("buf.build/bufbuild/es:v1.3.0"),
						Revision: toPointer(2),
						Out:      "gen/es",
						Opt:      nil,
					},
				},
				Managed: bufgenv2.ExternalManagedConfigV2{
					Enabled: false,
				},
				Inputs: defaultExpectedInputs,
			},
		},
		{
			description: "include imports",
			original: &bufgenv1.Config{
				PluginConfigs: []bufgenplugin.PluginConfig{
					mustCreateNewCuratedPlugin(
						t,
						"buf.build/bufbuild/es:v1.3.0",
						2,
						"gen/es",
						"",
					),
					mustCreateNewProtocPlugin(
						t,
						"cpp",
						"/bin/protoc",
						"gen/cpp",
						"",
						internal.StrategyAll,
					),
				},
			},
			includeImports: true,
			expected: &bufgenv2.ExternalConfigV2{
				Version: "v2",
				Plugins: []bufgenv2.ExternalPluginConfigV2{
					{
						Remote:         toPointer("buf.build/bufbuild/es:v1.3.0"),
						Revision:       toPointer(2),
						Out:            "gen/es",
						Opt:            nil,
						IncludeImports: true,
					},
					{
						ProtocBuiltin:  toPointer("cpp"),
						ProtocPath:     toPointer("/bin/protoc"),
						Out:            "gen/cpp",
						Opt:            nil,
						Strategy:       toPointer("all"),
						IncludeImports: true,
					},
				},
				Managed: bufgenv2.ExternalManagedConfigV2{
					Enabled: false,
				},
				Inputs: defaultExpectedInputs,
			},
		},
		{
			description: "include imports and wkt",
			original: &bufgenv1.Config{
				PluginConfigs: []bufgenplugin.PluginConfig{
					mustCreateNewCuratedPlugin(
						t,
						"buf.build/bufbuild/es:v1.3.0",
						2,
						"gen/es",
						"",
					),
					mustCreateNewProtocPlugin(
						t,
						"cpp",
						"/bin/protoc",
						"gen/cpp",
						"",
						internal.StrategyAll,
					),
				},
			},
			includeImports: true,
			includeWKT:     true,
			expected: &bufgenv2.ExternalConfigV2{
				Version: "v2",
				Plugins: []bufgenv2.ExternalPluginConfigV2{
					{
						Remote:         toPointer("buf.build/bufbuild/es:v1.3.0"),
						Revision:       toPointer(2),
						Out:            "gen/es",
						Opt:            nil,
						IncludeImports: true,
						IncludeWKT:     true,
					},
					{
						ProtocBuiltin:  toPointer("cpp"),
						ProtocPath:     toPointer("/bin/protoc"),
						Out:            "gen/cpp",
						Opt:            nil,
						Strategy:       toPointer("all"),
						IncludeImports: true,
						IncludeWKT:     true,
					},
				},
				Managed: bufgenv2.ExternalManagedConfigV2{
					Enabled: false,
				},
				Inputs: defaultExpectedInputs,
			},
		},
		{
			description: "empty managed mode enabled",
			original: &bufgenv1.Config{
				PluginConfigs: placeHolderPluginsV1,
				ManagedConfig: &bufgenv1.ManagedConfig{},
			},
			expected: &bufgenv2.ExternalConfigV2{
				Version: "v2",
				Plugins: expectedPlaceHolderPluginsV2,
				Managed: bufgenv2.ExternalManagedConfigV2{
					Enabled: true,
				},
				Inputs: defaultExpectedInputs,
			},
		},
		{
			description: "managed mode config simple options",
			original: &bufgenv1.Config{
				PluginConfigs: placeHolderPluginsV1,
				ManagedConfig: &bufgenv1.ManagedConfig{
					CcEnableArenas:      toPointer(true),
					JavaMultipleFiles:   toPointer(true),
					JavaStringCheckUtf8: toPointer(true),
				},
			},
			expected: &bufgenv2.ExternalConfigV2{
				Version: "v2",
				Plugins: expectedPlaceHolderPluginsV2,
				Managed: bufgenv2.ExternalManagedConfigV2{
					Enabled: true,
					Override: []bufgenv2.ExternalManagedOverrideConfigV2{
						{
							FileOption: "cc_enable_arenas",
							Value:      true,
						},
						{
							FileOption: "java_multiple_files",
							Value:      true,
						},
						{
							FileOption: "java_string_check_utf8",
							Value:      true,
						},
					},
				},
				Inputs: defaultExpectedInputs,
			},
		},
		{
			description: "managed mode with java package, csharp namespace and optimize for",
			original: &bufgenv1.Config{
				PluginConfigs: placeHolderPluginsV1,
				ManagedConfig: &bufgenv1.ManagedConfig{
					JavaPackagePrefixConfig: &bufgenv1.JavaPackagePrefixConfig{
						Default: "net",
						Except: []bufmoduleref.ModuleIdentity{
							mustCreateModuleIdentity(t, "buf.build/acme/weather"),
							mustCreateModuleIdentity(t, "buf.build/googleapis/googleapis"),
						},
						Override: map[bufmoduleref.ModuleIdentity]string{
							mustCreateModuleIdentity(t, "buf.build/acme/petapis"): "dev",
						},
					},
					CsharpNameSpaceConfig: &bufgenv1.CsharpNameSpaceConfig{
						Except: []bufmoduleref.ModuleIdentity{
							mustCreateModuleIdentity(t, "buf.build/googleapis/googleapis"),
						},
						Override: map[bufmoduleref.ModuleIdentity]string{
							mustCreateModuleIdentity(t, "buf.build/acme/petapis"): "X::Y::Z",
						},
					},
					OptimizeForConfig: &bufgenv1.OptimizeForConfig{
						Default: descriptorpb.FileOptions_CODE_SIZE,
						Except: []bufmoduleref.ModuleIdentity{
							mustCreateModuleIdentity(t, "buf.build/acme/petapis"),
						},
						Override: map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode{
							mustCreateModuleIdentity(t, "buf.build/acme/payment"): descriptorpb.FileOptions_LITE_RUNTIME,
						},
					},
				},
			},
			expected: &bufgenv2.ExternalConfigV2{
				Version: "v2",
				Plugins: expectedPlaceHolderPluginsV2,
				Managed: bufgenv2.ExternalManagedConfigV2{
					Enabled: true,
					Disable: []bufgenv2.ExternalManagedDisableConfigV2{
						{
							FileOption: "java_package",
							Module:     "buf.build/acme/weather",
						},
						{
							FileOption: "java_package",
							Module:     "buf.build/googleapis/googleapis",
						},
						{
							FileOption: "csharp_namespace",
							Module:     "buf.build/googleapis/googleapis",
						},
						{
							FileOption: "optimize_for",
							Module:     "buf.build/acme/petapis",
						},
					},
					Override: []bufgenv2.ExternalManagedOverrideConfigV2{
						{
							FileOption: "java_package_prefix",
							Value:      "net",
						},
						{
							FileOption: "java_package_prefix",
							Module:     "buf.build/acme/petapis",
							Value:      "dev",
						},
						{
							FileOption: "csharp_namespace",
							Module:     "buf.build/acme/petapis",
							Value:      "X::Y::Z",
						},
						{
							FileOption: "optimize_for",
							Value:      "CODE_SIZE",
						},
						{
							FileOption: "optimize_for",
							Module:     "buf.build/acme/payment",
							Value:      "LITE_RUNTIME",
						},
					},
				},
				Inputs: defaultExpectedInputs,
			},
		},
		{
			description: "managed mode with go, objc and ruby",
			original: &bufgenv1.Config{
				PluginConfigs: placeHolderPluginsV1,
				ManagedConfig: &bufgenv1.ManagedConfig{
					GoPackagePrefixConfig: &bufgenv1.GoPackagePrefixConfig{
						Default: "github.com/example/proto",
						Except: []bufmoduleref.ModuleIdentity{
							mustCreateModuleIdentity(t, "buf.build/googleapis/googleapis"),
						},
						Override: map[bufmoduleref.ModuleIdentity]string{
							mustCreateModuleIdentity(t, "buf.build/acme/petapis"): "github.com/acme/petapis/proto",
						},
					},
					ObjcClassPrefixConfig: &bufgenv1.ObjcClassPrefixConfig{
						Default: "XYZ",
						Except: []bufmoduleref.ModuleIdentity{
							mustCreateModuleIdentity(t, "buf.build/acme/weather"),
						},
						Override: map[bufmoduleref.ModuleIdentity]string{
							mustCreateModuleIdentity(t, "buf.build/acme/payment"): "ABC",
						},
					},
					RubyPackageConfig: &bufgenv1.RubyPackageConfig{
						Except: []bufmoduleref.ModuleIdentity{
							mustCreateModuleIdentity(t, "buf.build/acme/payment"),
						},
						Override: map[bufmoduleref.ModuleIdentity]string{
							mustCreateModuleIdentity(t, "buf.build/acme/petapis"): "X::Y::Z",
						},
					},
				},
			},
			expected: &bufgenv2.ExternalConfigV2{
				Version: "v2",
				Plugins: expectedPlaceHolderPluginsV2,
				Managed: bufgenv2.ExternalManagedConfigV2{
					Enabled: true,
					Disable: []bufgenv2.ExternalManagedDisableConfigV2{
						{
							FileOption: "go_package",
							Module:     "buf.build/googleapis/googleapis",
						},
						{
							FileOption: "objc_class_prefix",
							Module:     "buf.build/acme/weather",
						},
						{
							FileOption: "ruby_package",
							Module:     "buf.build/acme/payment",
						},
					},
					Override: []bufgenv2.ExternalManagedOverrideConfigV2{
						{
							FileOption: "go_package_prefix",
							Value:      "github.com/example/proto",
						},
						{
							FileOption: "go_package_prefix",
							Module:     "buf.build/acme/petapis",
							Value:      "github.com/acme/petapis/proto",
						},
						{
							FileOption: "objc_class_prefix",
							Value:      "XYZ",
						},
						{
							FileOption: "objc_class_prefix",
							Module:     "buf.build/acme/payment",
							Value:      "ABC",
						},
						{
							FileOption: "ruby_package",
							Module:     "buf.build/acme/petapis",
							Value:      "X::Y::Z",
						},
					},
				},
				Inputs: defaultExpectedInputs,
			},
		},
		{
			description: "managed mode with per-file overrides",
			original: &bufgenv1.Config{
				PluginConfigs: placeHolderPluginsV1,
				ManagedConfig: &bufgenv1.ManagedConfig{
					JavaPackagePrefixConfig: &bufgenv1.JavaPackagePrefixConfig{
						Default: "net",
						Override: map[bufmoduleref.ModuleIdentity]string{
							mustCreateModuleIdentity(t, "buf.build/acme/petapis"): "dev",
						},
					},
					Override: map[string]map[string]string{
						"JAVA_PACKAGE": {
							"dir1/a.proto": "com.example.a",
						},
					},
				},
			},
			expected: &bufgenv2.ExternalConfigV2{
				Version: "v2",
				Plugins: expectedPlaceHolderPluginsV2,
				Managed: bufgenv2.ExternalManagedConfigV2{
					Enabled: true,
					Override: []bufgenv2.ExternalManagedOverrideConfigV2{
						{
							FileOption: "java_package_prefix",
							Value:      "net",
						},
						{
							FileOption: "java_package_prefix",
							Module:     "buf.build/acme/petapis",
							Value:      "dev",
						},
						{
							FileOption: "java_package",
							Path:       "dir1/a.proto",
							Value:      "com.example.a",
						},
					},
				},
				Inputs: defaultExpectedInputs,
			},
		},
		{
			description: "types override",
			original: &bufgenv1.Config{
				PluginConfigs: placeHolderPluginsV1,
			},
			types: []string{
				"a.b.c.Message1",
				"x.y.z.Message2",
			},
			expected: &bufgenv2.ExternalConfigV2{
				Version: "v2",
				Plugins: expectedPlaceHolderPluginsV2,
				Managed: bufgenv2.ExternalManagedConfigV2{
					Enabled: false,
				},
				Inputs: []bufgenv2.ExternalInputConfigV2{
					{
						Directory: toPointer("."),
						Types: []string{
							"a.b.c.Message1",
							"x.y.z.Message2",
						},
					},
				},
			},
		},
		{
			description: "types in configuration",
			original: &bufgenv1.Config{
				PluginConfigs: placeHolderPluginsV1,
				TypesConfig: &bufgenv1.TypesConfig{
					Include: []string{
						"a.b.c.Message1",
						"x.y.z.Message2",
					},
				},
			},
			expected: &bufgenv2.ExternalConfigV2{
				Version: "v2",
				Plugins: expectedPlaceHolderPluginsV2,
				Managed: bufgenv2.ExternalManagedConfigV2{
					Enabled: false,
				},
				Inputs: []bufgenv2.ExternalInputConfigV2{
					{
						Directory: toPointer("."),
						Types: []string{
							"a.b.c.Message1",
							"x.y.z.Message2",
						},
					},
				},
			},
		},
		{
			description: "types in both configuration and override",
			original: &bufgenv1.Config{
				PluginConfigs: placeHolderPluginsV1,
				TypesConfig: &bufgenv1.TypesConfig{
					Include: []string{
						"a.b.c.Message1",
						"x.y.z.Message2",
					},
				},
			},
			types: []string{
				"google.type.DateTime",
			},
			expected: &bufgenv2.ExternalConfigV2{
				Version: "v2",
				Plugins: expectedPlaceHolderPluginsV2,
				Managed: bufgenv2.ExternalManagedConfigV2{
					Enabled: false,
				},
				Inputs: []bufgenv2.ExternalInputConfigV2{
					{
						Directory: toPointer("."),
						Types: []string{
							"google.type.DateTime",
						},
					},
				},
			},
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			logger := zap.NewNop()
			findPlugin := testcase.findPlugin
			if findPlugin == nil {
				findPlugin = func(name string) (string, error) {
					return name, nil
				}
			}
			externalConfigV2, err := convertConfigV1ToExternalConfigV2(
				ctx,
				logger,
				testcase.original,
				findPlugin,
				testcase.input,
				testcase.types,
				testcase.includePaths,
				testcase.excludePaths,
				testcase.includeImports,
				testcase.includeWKT,
			)
			require.NoError(t, err)
			require.Equal(t, testcase.expected, externalConfigV2)
		})
	}
}

func TestInputStringToInputConfigV2(t *testing.T) {
	t.Parallel()
	logger := zap.NewNop()
	testcases := []struct {
		description      string
		input            string
		types            []string
		includePaths     []string
		excludedPaths    []string
		expectedConfigV2 bufgenv2.ExternalInputConfigV2
	}{
		{
			description: "dot",
			input:       ".",
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				Directory: toPointer("."),
			},
		},
		{
			description: "some directory",
			input:       "path/to/some/dir",
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				Directory: toPointer("path/to/some/dir"),
			},
		},
		{
			description: "module",
			input:       "buf.build/acme/weather",
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				Module: toPointer("buf.build/acme/weather"),
			},
		},
		{
			description: "proto file",
			input:       "path/to/file.proto#include_package_files=false",
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				ProtoFile:           toPointer("path/to/file.proto"),
				IncludePackageFiles: toPointer(false),
			},
		},
		{
			description: "tar",
			input:       "path/to/file.tar",
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				Tarball: toPointer("path/to/file.tar"),
			},
		},
		{
			description: "tar strip components",
			input:       "path/to/file.tar#strip_components=1",
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				Tarball:         toPointer("path/to/file.tar"),
				StripComponents: toPointer(uint32(1)),
			},
		},
		{
			description: "tgz",
			input:       "path/to/file.tgz#strip_components=1",
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				Tarball:         toPointer("path/to/file.tgz"),
				StripComponents: toPointer(uint32(1)),
			},
		},
		{
			description: "zip",
			input:       "path/to/file.zip#strip_components=1",
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				ZipArchive:      toPointer("path/to/file.zip"),
				StripComponents: toPointer(uint32(1)),
			},
		},
		{
			description: "git",
			input:       "ssh://user@hello.com:path/to/dir.git#ref=refs/remotes/origin/HEAD,subdir=protos,branch=main,depth=10,recurse_submodules=true",
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				GitRepo:           toPointer("ssh://user@hello.com:path/to/dir.git"),
				Ref:               toPointer("refs/remotes/origin/HEAD"),
				Subdir:            toPointer("protos"),
				Branch:            toPointer("main"),
				Depth:             toPointer(uint32(10)),
				RecurseSubmodules: toPointer(true),
			},
		},
		{
			description: "git with tag",
			input:       "path/to/dir#format=git,tag=main/foo",
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				GitRepo: toPointer("path/to/dir"),
				Tag:     toPointer("main/foo"),
			},
		},
		{
			description: "bin",
			input:       "path/to/file.bin",
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				BinaryImage: toPointer("path/to/file.bin"),
			},
		},
		{
			description: "bin.gz",
			input:       "path/to/file.bin.gz",
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				BinaryImage: toPointer("path/to/file.bin.gz"),
			},
		},
		{
			description: "json",
			input:       "path/to/file.json",
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				JSONImage: toPointer("path/to/file.json"),
			},
		},
		{
			description: "deprecated bingz",
			input:       "path/to/file#format=bingz",
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				BinaryImage: toPointer("path/to/file"),
				Compression: toPointer("gzip"),
			},
		},
		{
			description: "deprecated jsongz",
			input:       "path/to/file#format=jsongz",
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				JSONImage:   toPointer("path/to/file"),
				Compression: toPointer("gzip"),
			},
		},
		{
			description: "deprecated targz",
			input:       "path/to/file#format=targz",
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				Tarball:     toPointer("path/to/file"),
				Compression: toPointer("gzip"),
			},
		},
		{
			description: "include paths",
			input:       "buf.build/acme/weather",
			includePaths: []string{
				"dir1",
				"dir2/file2",
			},
			excludedPaths: []string{
				"dir1/subdir1",
				"dir1/file1",
			},
			expectedConfigV2: bufgenv2.ExternalInputConfigV2{
				Module: toPointer("buf.build/acme/weather"),
				IncludePaths: []string{
					"dir1",
					"dir2/file2",
				},
				ExcludePaths: []string{
					"dir1/subdir1",
					"dir1/file1",
				},
			},
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			actualInputConfig, err := getExternalInputConfigV2(
				ctx,
				logger,
				testcase.input,
				testcase.types,
				testcase.includePaths,
				testcase.excludedPaths,
			)
			require.NoError(t, err)
			require.Equal(t, testcase.expectedConfigV2, *actualInputConfig)
		})
	}
}

func TestMigrateV1ToV2Error(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		description    string
		original       *bufgenv1.Config
		expectedError  string
		findPlugin     func(string) (string, error)
		input          string
		types          []string
		includePaths   []string
		excludePaths   []string
		includeImports bool
		includeWKT     bool
	}{
		{
			description: "local plugin not found locally and not protoc builtin",
			original: &bufgenv1.Config{
				PluginConfigs: []bufgenplugin.PluginConfig{
					mustCreateLocalPluginConfig(
						t,
						"somelocal",
						internal.StrategyDirectory,
						"gen/somelocal",
						"a=b",
					),
				},
			},
			findPlugin: func(s string) (string, error) {
				return "", errors.New("not found")
			},
			expectedError: `unable to migrate plugin "somelocal": plugin protoc-gen-somelocal is not found locally and somelocal is not built-in to protoc`,
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			logger := zap.NewNop()
			findPlugin := testcase.findPlugin
			if findPlugin == nil {
				findPlugin = func(name string) (string, error) {
					return name, nil
				}
			}
			externalConfigV2, err := convertConfigV1ToExternalConfigV2(
				ctx,
				logger,
				testcase.original,
				findPlugin,
				testcase.input,
				testcase.types,
				testcase.includePaths,
				testcase.excludePaths,
				testcase.includeImports,
				testcase.includeWKT,
			)
			require.ErrorContains(t, err, testcase.expectedError)
			require.Nil(t, externalConfigV2)
		})
	}
}

func mustCreateLocalPluginConfig(
	t *testing.T,
	name string,
	strategy internal.Strategy,
	out string,
	opt string,
) bufgenplugin.LocalPluginConfig {
	plugin, err := bufgenplugin.NewLocalPluginConfig(
		name,
		strategy,
		out,
		opt,
		false,
		false,
	)
	require.NoError(t, err)
	return plugin
}

func mustCreateNewBinaryPlugin(
	t *testing.T,
	name string,
	path []string,
	strategy internal.Strategy,
	out string,
	opt string,
) bufgenplugin.BinaryPluginConfig {
	plugin, err := bufgenplugin.NewBinaryPluginConfig(
		name,
		path,
		strategy,
		out,
		opt,
		false,
		false,
	)
	require.NoError(t, err)
	return plugin
}

func mustCreateNewProtocPlugin(
	t *testing.T,
	name string,
	protocPath string,
	out string,
	opt string,
	strategy internal.Strategy,
) bufgenplugin.ProtocBuiltinPluginConfig {
	plugin, err := bufgenplugin.NewProtocBuiltinPluginConfig(
		name,
		protocPath,
		out,
		opt,
		false,
		false,
		strategy,
	)
	require.NoError(t, err)
	return plugin
}

func mustCreateNewCuratedPlugin(
	t *testing.T,
	fullName string,
	revision int,
	out string,
	opt string,
) bufgenplugin.CuratedPluginConfig {
	plugin, err := bufgenplugin.NewCuratedPluginConfig(
		fullName,
		revision,
		out,
		opt,
		false,
		false,
	)
	require.NoError(t, err)
	return plugin
}

func toPointer[T any](value T) *T {
	return &value
}

func mustCreateModuleIdentity(
	t *testing.T,
	identityString string,
) bufmoduleref.ModuleIdentity {
	parts := strings.Split(identityString, "/")
	require.Len(t, parts, 3)
	moduleIdentity, err := bufmoduleref.NewModuleIdentity(parts[0], parts[1], parts[2])
	require.NoError(t, err)
	return moduleIdentity
}
