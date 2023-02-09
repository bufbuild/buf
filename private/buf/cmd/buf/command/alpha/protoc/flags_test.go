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

package protoc

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFlags(t *testing.T) {
	absFilePath, err := filepath.Abs("out")
	require.NoError(t, err)
	testCases := []struct {
		Args          []string
		Expected      *env
		ExpectedError error
	}{
		{
			ExpectedError: errNoInputFiles,
		},
		{
			Args: []string{
				"foo.proto",
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: defaultIncludeDirPaths,
					ErrorFormat:     defaultErrorFormat,
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"-I",
				"proto",
				"--error_format",
				"text",
				"foo.proto",
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: []string{
						"proto",
					},
					ErrorFormat: "text",
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"-I",
				"proto",
				"--error_format",
				"text",
				"--go_out",
				"plugins=connect:go_out",
				"foo.proto",
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: []string{
						"proto",
					},
					ErrorFormat: "text",
				},
				PluginNamesSortedByOutIndex: []string{
					"go",
				},
				PluginNameToPluginInfo: map[string]*pluginInfo{
					"go": {
						Out: "go_out",
						Opt: []string{"plugins=connect"},
					},
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"-I",
				"proto",
				"--error_format",
				"text",
				"--go_out",
				"go_out",
				"--go_opt",
				"plugins=connect",
				"foo.proto",
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: []string{
						"proto",
					},
					ErrorFormat: "text",
				},
				PluginNamesSortedByOutIndex: []string{
					"go",
				},
				PluginNameToPluginInfo: map[string]*pluginInfo{
					"go": {
						Out: "go_out",
						Opt: []string{"plugins=connect"},
					},
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"-I",
				"proto",
				"--error_format",
				"text",
				"--go_out",
				"go_out",
				"--go_opt",
				"plugins=connect",
				"--plugin",
				"/bin/protoc-gen-go",
				"foo.proto",
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: []string{
						"proto",
					},
					ErrorFormat: "text",
				},
				PluginNamesSortedByOutIndex: []string{
					"go",
				},
				PluginNameToPluginInfo: map[string]*pluginInfo{
					"go": {
						Out:  "go_out",
						Opt:  []string{"plugins=connect"},
						Path: "/bin/protoc-gen-go",
					},
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"-I",
				"proto",
				"--error_format",
				"text",
				"--go_out",
				"go_out",
				"--go_opt",
				"plugins=connect",
				"--plugin",
				"protoc-gen-go=/bin/foo",
				"foo.proto",
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: []string{
						"proto",
					},
					ErrorFormat: "text",
				},
				PluginNamesSortedByOutIndex: []string{
					"go",
				},
				PluginNameToPluginInfo: map[string]*pluginInfo{
					"go": {
						Out:  "go_out",
						Opt:  []string{"plugins=connect"},
						Path: "/bin/foo",
					},
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"@" + filepath.Join("testdata", "1", "flags.txt"),
				"foo.proto",
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: []string{
						"proto",
					},
					ErrorFormat: "text",
				},
				PluginNamesSortedByOutIndex: []string{
					"go",
				},
				PluginNameToPluginInfo: map[string]*pluginInfo{
					"go": {
						Out:  "go_out",
						Opt:  []string{"plugins=connect"},
						Path: "/bin/protoc-gen-go",
					},
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"@" + filepath.Join("testdata", "2", "flags1.txt"),
				"foo.proto",
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: []string{
						"proto",
					},
					ErrorFormat: "text",
				},
				PluginNamesSortedByOutIndex: []string{
					"go",
				},
				PluginNameToPluginInfo: map[string]*pluginInfo{
					"go": {
						Out:  "go_out",
						Opt:  []string{"plugins=connect"},
						Path: "/bin/protoc-gen-go",
					},
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"@" + filepath.Join("testdata", "3", "flags1.txt"),
				"foo.proto",
			},
			ExpectedError: newRecursiveReferenceError(filepath.Join("testdata", "3", "flags1.txt")),
		},
		{
			Args: []string{
				"-I",
				"proto",
				"--error_format",
				"text",
				"--go_out",
				"plugins=connect:go_out",
				"--go_opt",
				"foo=bar",
				"foo.proto",
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: []string{
						"proto",
					},
					ErrorFormat: "text",
				},
				PluginNamesSortedByOutIndex: []string{
					"go",
				},
				PluginNameToPluginInfo: map[string]*pluginInfo{
					"go": {
						Out: "go_out",
						Opt: []string{
							"plugins=connect",
							"foo=bar",
						},
					},
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"-I",
				"proto",
				"--error_format",
				"text",
				"--go_out",
				"plugins=connect:go_out",
				"--go_opt",
				"foo=bar",
				"--go_opt",
				"baz=bat",
				"foo.proto",
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: []string{
						"proto",
					},
					ErrorFormat: "text",
				},
				PluginNamesSortedByOutIndex: []string{
					"go",
				},
				PluginNameToPluginInfo: map[string]*pluginInfo{
					"go": {
						Out: "go_out",
						Opt: []string{
							"plugins=connect",
							"foo=bar",
							"baz=bat",
						},
					},
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"-I",
				"proto",
				"--error_format",
				"text",
				"--go_out",
				"go_out",
				"--go_opt",
				"foo=bar",
				"--go_opt",
				"baz=bat",
				"foo.proto",
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: []string{
						"proto",
					},
					ErrorFormat: "text",
				},
				PluginNamesSortedByOutIndex: []string{
					"go",
				},
				PluginNameToPluginInfo: map[string]*pluginInfo{
					"go": {
						Out: "go_out",
						Opt: []string{
							"foo=bar",
							"baz=bat",
						},
					},
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"-I",
				"proto",
				"--error_format",
				"text",
				"--go_out",
				"foo=bar,baz=bat:go_out",
				"--go_opt",
				"one=two,three=four",
				"--go_opt",
				"five=six",
				"foo.proto",
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: []string{
						"proto",
					},
					ErrorFormat: "text",
				},
				PluginNamesSortedByOutIndex: []string{
					"go",
				},
				PluginNameToPluginInfo: map[string]*pluginInfo{
					"go": {
						Out: "go_out",
						Opt: []string{
							"foo=bar",
							"baz=bat",
							"one=two",
							"three=four",
							"five=six",
						},
					},
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"--go_out",
				"go_out",
				"--go_out",
				"go_out",
				"foo.proto",
			},
			ExpectedError: newDuplicateOutError("go"),
		},
		{
			Args: []string{
				"--foo_out=foo",
				"--bar_out=bar",
				"--baz_out=baz",
				"foo.proto",
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: defaultIncludeDirPaths,
					ErrorFormat:     defaultErrorFormat,
				},
				PluginNamesSortedByOutIndex: []string{
					"foo",
					"bar",
					"baz",
				},
				PluginNameToPluginInfo: map[string]*pluginInfo{
					"foo": {
						Out: "foo",
					},
					"bar": {
						Out: "bar",
					},
					"baz": {
						Out: "baz",
					},
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"-I",
				"foo" + includeDirPathSeparator + "bar" + includeDirPathSeparator + "baz",
				"-I",
				"bat",
				"foo.proto",
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: []string{
						"foo",
						"bar",
						"baz",
						"bat",
					},
					ErrorFormat: defaultErrorFormat,
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"-I",
				"foo" + includeDirPathSeparator + includeDirPathSeparator + "baz",
				"-I",
				"bat",
				"foo.proto",
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: []string{
						"foo",
						"baz",
						"bat",
					},
					ErrorFormat: defaultErrorFormat,
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"-I",
				"foo" + includeDirPathSeparator + "bar" + includeDirPathSeparator,
				"-I",
				"bat",
				"foo.proto",
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: []string{
						"foo",
						"bar",
						"bat",
					},
					ErrorFormat: defaultErrorFormat,
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"-I",
				"proto",
				"foo.proto",
				fmt.Sprintf("--go_out=%s", absFilePath),
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: []string{
						"proto",
					},
					ErrorFormat: "gcc",
				},
				PluginNamesSortedByOutIndex: []string{
					"go",
				},
				PluginNameToPluginInfo: map[string]*pluginInfo{
					"go": {
						Out: absFilePath,
					},
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
		{
			Args: []string{
				"-I",
				"proto",
				"foo.proto",
				fmt.Sprintf("--go_out=opt:%s", absFilePath),
			},
			Expected: &env{
				flags: flags{
					IncludeDirPaths: []string{
						"proto",
					},
					ErrorFormat: "gcc",
				},
				PluginNamesSortedByOutIndex: []string{
					"go",
				},
				PluginNameToPluginInfo: map[string]*pluginInfo{
					"go": {
						Out: absFilePath,
						Opt: []string{"opt"},
					},
				},
				FilePaths: []string{
					"foo.proto",
				},
			},
		},
	}
	for i, testCase := range testCases {
		name := fmt.Sprintf("%d", i)
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			env, err := testParseFlags(name, testCase.Args)
			if testCase.ExpectedError != nil {
				assert.Equal(t, testCase.ExpectedError, err)
			} else {
				require.NoError(t, err)
				if env != nil {
					// testify counts nil and empty as different
					// we do not want to have to set empty values in our expected env
					// so we set them to nil here for comparison
					if len(env.IncludeDirPaths) == 0 {
						env.IncludeDirPaths = nil
					}
					if len(env.PluginNamesSortedByOutIndex) == 0 {
						env.PluginNamesSortedByOutIndex = nil
					}
					if len(env.PluginNameToPluginInfo) == 0 {
						env.PluginNameToPluginInfo = nil
					}
					if len(env.FilePaths) == 0 {
						env.FilePaths = nil
					}
				}
				assert.Equal(t, testCase.Expected, env)
			}
		})
	}
}

func testParseFlags(name string, args []string) (*env, error) {
	flagsBuilder := newFlagsBuilder()
	flagSet := pflag.NewFlagSet(name, pflag.ContinueOnError)
	flagsBuilder.Bind(flagSet)
	flagSet.SetNormalizeFunc(normalizeFunc(flagsBuilder.Normalize))
	if err := flagSet.Parse(args); err != nil {
		return nil, err
	}
	return flagsBuilder.Build(flagSet.Args())
}
