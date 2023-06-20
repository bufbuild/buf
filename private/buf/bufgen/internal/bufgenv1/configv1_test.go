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

package bufgenv1

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufgen/internal"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/plugingen"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestReadConfigV1Beta1(t *testing.T) {
	truth := true
	successConfig := &Config{
		PluginConfigs: []plugingen.PluginConfig{
			mustCreateBinaryPlugin(
				t,
				"go",
				[]string{"/path/to/foo"},
				internal.StrategyAll,
				"gen/go",
				"plugins=connect",
			),
		},
		ManagedConfig: &ManagedConfig{
			CcEnableArenas:      &truth,
			JavaMultipleFiles:   &truth,
			JavaStringCheckUtf8: nil,
			OptimizeForConfig: &OptimizeForConfig{
				Default:  descriptorpb.FileOptions_CODE_SIZE,
				Except:   make([]bufmoduleref.ModuleIdentity, 0),
				Override: make(map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode),
			},
		},
	}
	successConfig2 := &Config{
		ManagedConfig: &ManagedConfig{
			OptimizeForConfig: &OptimizeForConfig{
				Default:  descriptorpb.FileOptions_SPEED,
				Except:   make([]bufmoduleref.ModuleIdentity, 0),
				Override: make(map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode),
			},
		},
		PluginConfigs: []plugingen.PluginConfig{
			mustCreateBinaryPlugin(
				t,
				"go",
				[]string{"/path/to/foo"},
				internal.StrategyAll,
				"gen/go",
				"plugins=connect,foo=bar",
			),
		},
	}
	successConfig3 := &Config{
		ManagedConfig: &ManagedConfig{
			OptimizeForConfig: &OptimizeForConfig{
				Default:  descriptorpb.FileOptions_LITE_RUNTIME,
				Except:   make([]bufmoduleref.ModuleIdentity, 0),
				Override: make(map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode),
			},
		},
		PluginConfigs: []plugingen.PluginConfig{
			mustCreateBinaryPlugin(
				t,
				"go",
				[]string{"/path/to/foo"},
				internal.StrategyAll,
				"gen/go",
				"",
			),
		},
	}
	successConfig4 := &Config{
		ManagedConfig: &ManagedConfig{
			OptimizeForConfig: &OptimizeForConfig{
				Default:  descriptorpb.FileOptions_LITE_RUNTIME,
				Except:   make([]bufmoduleref.ModuleIdentity, 0),
				Override: make(map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode),
			},
		},
		PluginConfigs: []plugingen.PluginConfig{
			mustCreateLocalPlugin(
				t,
				"go",
				internal.StrategyAll,
				"gen/go",
				"",
			),
		},
	}
	ctx := context.Background()
	nopLogger := zap.NewNop()
	readBucket, err := storagemem.NewReadBucket(nil)
	require.NoError(t, err)

	testReadConfigSuccess := func(t *testing.T, configPath string, expected *Config) {
		t.Helper()
		config, err := readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(configPath))
		require.NoError(t, err)
		assert.Equal(t, expected, config)
		data, err := os.ReadFile(configPath)
		require.NoError(t, err)
		config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
		require.NoError(t, err)
		assert.Equal(t, expected, config)
	}

	testReadConfigSuccess(t, filepath.Join("testdata", "v1beta1", "gen_success1.yaml"), successConfig)
	testReadConfigSuccess(t, filepath.Join("testdata", "v1beta1", "gen_success1.json"), successConfig)
	testReadConfigSuccess(t, filepath.Join("testdata", "v1beta1", "gen_success2.yaml"), successConfig2)
	testReadConfigSuccess(t, filepath.Join("testdata", "v1beta1", "gen_success2.json"), successConfig2)
	testReadConfigSuccess(t, filepath.Join("testdata", "v1beta1", "gen_success3.yaml"), successConfig3)
	testReadConfigSuccess(t, filepath.Join("testdata", "v1beta1", "gen_success3.json"), successConfig3)
	testReadConfigSuccess(t, filepath.Join("testdata", "v1beta1", "gen_success3.yml"), successConfig3)
	testReadConfigSuccess(t, filepath.Join("testdata", "v1beta1", "gen_success4_nopath.yaml"), successConfig4)

	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1beta1", "gen_error1.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1beta1", "gen_error2.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1beta1", "gen_error3.yaml"))
}

func TestReadConfigV1(t *testing.T) {
	truth := true
	successConfig := &Config{
		PluginConfigs: []plugingen.PluginConfig{
			mustCreateBinaryPlugin(
				t,
				"go",
				[]string{"/path/to/foo"},
				internal.StrategyAll,
				"gen/go",
				"plugins=connect",
			),
		},
		ManagedConfig: &ManagedConfig{
			CcEnableArenas:      &truth,
			JavaMultipleFiles:   &truth,
			JavaStringCheckUtf8: &truth,
			JavaPackagePrefixConfig: &JavaPackagePrefixConfig{
				Default:  "org",
				Except:   make([]bufmoduleref.ModuleIdentity, 0),
				Override: make(map[bufmoduleref.ModuleIdentity]string),
			},
			OptimizeForConfig: &OptimizeForConfig{
				Default:  descriptorpb.FileOptions_CODE_SIZE,
				Except:   make([]bufmoduleref.ModuleIdentity, 0),
				Override: make(map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode),
			},
			Override: map[string]map[string]string{
				bufimagemodify.JavaPackageID: {"a.proto": "override"},
			},
		},
		TypesConfig: &TypesConfig{
			Include: []string{
				"buf.alpha.lint.v1.IDPaths",
			},
		},
	}
	successConfig2 := &Config{
		ManagedConfig: &ManagedConfig{
			OptimizeForConfig: &OptimizeForConfig{
				Default:  descriptorpb.FileOptions_SPEED,
				Except:   make([]bufmoduleref.ModuleIdentity, 0),
				Override: make(map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode),
			},
		},
		PluginConfigs: []plugingen.PluginConfig{
			mustCreateBinaryPlugin(
				t,
				"go",
				[]string{"/path/to/foo"},
				internal.StrategyAll,
				"gen/go",
				"plugins=connect,foo=bar",
			),
		},
	}
	successConfig3 := &Config{
		ManagedConfig: &ManagedConfig{
			OptimizeForConfig: &OptimizeForConfig{
				Default:  descriptorpb.FileOptions_LITE_RUNTIME,
				Except:   make([]bufmoduleref.ModuleIdentity, 0),
				Override: make(map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode),
			},
		},
		PluginConfigs: []plugingen.PluginConfig{
			mustCreateBinaryPlugin(
				t,
				"go",
				[]string{"/path/to/foo"},
				internal.StrategyAll,
				"gen/go",
				"",
			),
		},
	}
	successConfig4 := &Config{
		PluginConfigs: []plugingen.PluginConfig{
			mustCreateLegacyRemotePlugin(
				t,
				"someremote.com/owner/plugins/myplugin:v1.1.0-1",
				"gen/go",
				"",
			),
		},
	}
	successConfig5 := &Config{
		PluginConfigs: []plugingen.PluginConfig{
			mustCreateLegacyRemotePlugin(
				t,
				"someremote.com/owner/plugins/myplugin",
				"gen/go",
				"",
			),
		},
	}
	moduleIdentity, err := bufmoduleref.NewModuleIdentity(
		"someremote.com",
		"owner",
		"repo",
	)
	require.NoError(t, err)
	successConfig6 := &Config{
		ManagedConfig: &ManagedConfig{
			JavaPackagePrefixConfig: &JavaPackagePrefixConfig{
				Default:  "org",
				Except:   []bufmoduleref.ModuleIdentity{moduleIdentity},
				Override: make(map[bufmoduleref.ModuleIdentity]string),
			},
		},
		PluginConfigs: []plugingen.PluginConfig{
			mustCreateLegacyRemotePlugin(
				t,
				"someremote.com/owner/plugins/myplugin",
				"gen/go",
				"",
			),
		},
	}
	successConfig7 := &Config{
		PluginConfigs: []plugingen.PluginConfig{
			mustCreateBinaryPlugin(
				t,
				"go",
				[]string{"/path/to/foo"},
				internal.StrategyAll,
				"gen/go",
				"plugins=connect",
			),
		},
		ManagedConfig: nil,
	}
	moduleIdentity1 := mustCreateModuleIdentity(
		t,
		"someremote.com",
		"owner",
		"repo",
	)
	moduleIdentity2 := mustCreateModuleIdentity(
		t,
		"someremote.com",
		"owner",
		"foo",
	)
	moduleIdentity3 := mustCreateModuleIdentity(
		t,
		"someremote.com",
		"owner",
		"bar",
	)
	moduleIdentity4 := mustCreateModuleIdentity(
		t,
		"someremote.com",
		"owner",
		"baz",
	)
	successConfig8 := &Config{
		ManagedConfig: &ManagedConfig{
			CsharpNameSpaceConfig: &CsharpNameSpaceConfig{
				Except: []bufmoduleref.ModuleIdentity{
					moduleIdentity1,
				},
				Override: map[bufmoduleref.ModuleIdentity]string{
					moduleIdentity2: "a",
					moduleIdentity3: "b",
					moduleIdentity4: "c",
				},
			},
			ObjcClassPrefixConfig: &ObjcClassPrefixConfig{
				Default: "default",
				Except: []bufmoduleref.ModuleIdentity{
					moduleIdentity1,
				},
				Override: map[bufmoduleref.ModuleIdentity]string{
					moduleIdentity2: "a",
					moduleIdentity3: "b",
					moduleIdentity4: "c",
				},
			},
			RubyPackageConfig: &RubyPackageConfig{
				Except: []bufmoduleref.ModuleIdentity{
					moduleIdentity1,
				},
				Override: map[bufmoduleref.ModuleIdentity]string{
					moduleIdentity2: "x",
					moduleIdentity3: "y",
					moduleIdentity4: "z",
				},
			},
		},
		PluginConfigs: []plugingen.PluginConfig{
			mustCreateLegacyRemotePlugin(
				t,
				"someremote.com/owner/plugins/myplugin",
				"gen/go",
				"",
			),
		},
	}
	successConfig9 := &Config{
		ManagedConfig: &ManagedConfig{
			OptimizeForConfig: &OptimizeForConfig{
				Default: descriptorpb.FileOptions_CODE_SIZE,
				Except: []bufmoduleref.ModuleIdentity{
					mustCreateModuleIdentity(
						t,
						"someremote.com",
						"owner",
						"repo",
					),
					mustCreateModuleIdentity(
						t,
						"someremote.com",
						"owner",
						"foo",
					),
				},
				Override: map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode{
					mustCreateModuleIdentity(
						t,
						"someremote.com",
						"owner",
						"bar",
					): descriptorpb.FileOptions_SPEED,
					mustCreateModuleIdentity(
						t,
						"someremote.com",
						"owner",
						"baz",
					): descriptorpb.FileOptions_LITE_RUNTIME,
				},
			},
		},
		PluginConfigs: []plugingen.PluginConfig{
			mustCreateLegacyRemotePlugin(
				t,
				"someremote.com/owner/plugins/myplugin",
				"gen/go",
				"",
			),
		},
	}

	ctx := context.Background()
	nopLogger := zap.NewNop()
	readBucket, err := storagemem.NewReadBucket(nil)
	require.NoError(t, err)
	config, err := readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success1.yaml")))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)
	data, err := os.ReadFile(filepath.Join("testdata", "v1", "gen_success1.yaml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success1.json")))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success1.json"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)

	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success2.yaml")))
	require.NoError(t, err)
	require.Equal(t, successConfig2, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success2.yaml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig2, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success2.json")))
	require.NoError(t, err)
	require.Equal(t, successConfig2, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success2.json"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig2, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success3.yaml")))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success3.yaml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success3.json")))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success3.json"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success3.yml")))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success3.yml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success4.yaml")))
	require.NoError(t, err)
	require.Equal(t, successConfig4, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success4.yaml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig4, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success4.json")))
	require.NoError(t, err)
	require.Equal(t, successConfig4, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success4.json"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig4, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success4.yml")))
	require.NoError(t, err)
	require.Equal(t, successConfig4, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success4.yml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig4, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success5.yaml")))
	require.NoError(t, err)
	require.Equal(t, successConfig5, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success5.yaml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig5, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success5.json")))
	require.NoError(t, err)
	require.Equal(t, successConfig5, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success5.json"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig5, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success5.yml")))
	require.NoError(t, err)
	require.Equal(t, successConfig5, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success5.yml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig5, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success6.yaml")))
	require.NoError(t, err)
	require.Equal(t, successConfig6, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success6.yaml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig6, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success6.json")))
	require.NoError(t, err)
	require.Equal(t, successConfig6, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success6.json"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig6, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success6.yml")))
	require.NoError(t, err)
	require.Equal(t, successConfig6, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success6.yml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig6, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success7.yaml")))
	require.NoError(t, err)
	require.Equal(t, successConfig7, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success7.yaml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig7, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success7.json")))
	require.NoError(t, err)
	require.Equal(t, successConfig7, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success7.json"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig7, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success7.yml")))
	require.NoError(t, err)
	require.Equal(t, successConfig7, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success7.yml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig7, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success8.yaml")))
	require.NoError(t, err)
	assertConfigsWithEqualCsharpnamespace(t, successConfig8, config)
	assertConfigsWithEqualObjcPrefix(t, successConfig8, config)
	assertConfigsWithEqualRubyPackage(t, successConfig8, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success8.yaml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	assertConfigsWithEqualCsharpnamespace(t, successConfig8, config)
	assertConfigsWithEqualObjcPrefix(t, successConfig8, config)
	assertConfigsWithEqualRubyPackage(t, successConfig8, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success8.json")))
	require.NoError(t, err)
	assertConfigsWithEqualCsharpnamespace(t, successConfig8, config)
	assertConfigsWithEqualObjcPrefix(t, successConfig8, config)
	assertConfigsWithEqualRubyPackage(t, successConfig8, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success8.json"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	assertConfigsWithEqualCsharpnamespace(t, successConfig8, config)
	assertConfigsWithEqualObjcPrefix(t, successConfig8, config)
	assertConfigsWithEqualRubyPackage(t, successConfig8, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success8.yml")))
	require.NoError(t, err)
	assertConfigsWithEqualCsharpnamespace(t, successConfig8, config)
	assertConfigsWithEqualObjcPrefix(t, successConfig8, config)
	assertConfigsWithEqualRubyPackage(t, successConfig8, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success8.yml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	assertConfigsWithEqualCsharpnamespace(t, successConfig8, config)
	assertConfigsWithEqualObjcPrefix(t, successConfig8, config)
	assertConfigsWithEqualRubyPackage(t, successConfig8, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success9.yaml")))
	require.NoError(t, err)
	assertConfigsWithEqualOptimizeFor(t, successConfig9, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success9.yaml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	assertConfigsWithEqualOptimizeFor(t, successConfig9, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success9.json")))
	require.NoError(t, err)
	assertConfigsWithEqualOptimizeFor(t, successConfig9, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success9.json"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	assertConfigsWithEqualOptimizeFor(t, successConfig9, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success9.yml")))
	require.NoError(t, err)
	assertConfigsWithEqualOptimizeFor(t, successConfig9, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success9.yml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	assertConfigsWithEqualOptimizeFor(t, successConfig9, config)

	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "gen_error1.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "gen_error2.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "gen_error3.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "gen_error4.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "gen_error5.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "gen_error6.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "gen_error7.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "gen_error8.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "gen_error9.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "gen_error10.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "gen_error11.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "gen_error12.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "gen_error13.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "gen_error14.yaml"))

	successConfig = &Config{
		PluginConfigs: []plugingen.PluginConfig{
			mustCreateBinaryPlugin(
				t,
				"go",
				[]string{"/path/to/foo"},
				internal.StrategyAll,
				"gen/go",
				"plugins=connect",
			),
		},
		ManagedConfig: &ManagedConfig{
			GoPackagePrefixConfig: &GoPackagePrefixConfig{
				Default:  "github.com/foo/bar/gen/go",
				Except:   make([]bufmoduleref.ModuleIdentity, 0),
				Override: make(map[bufmoduleref.ModuleIdentity]string),
			},
		},
	}
	readBucket, err = storagemem.NewReadBucket(nil)
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "go_gen_success1.yaml")))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "go_gen_success1.yaml"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(filepath.Join("testdata", "v1", "go_gen_success1.json")))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "go_gen_success1.json"))
	require.NoError(t, err)
	config, err = readConfigV1(ctx, nopLogger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)

	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "go_gen_error2.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "go_gen_error3.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "go_gen_error4.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "go_gen_error5.yaml"))
	testReadConfigError(t, nopLogger, readBucket, filepath.Join("testdata", "v1", "go_gen_error6.yaml"))
}

func testReadConfigError(t *testing.T, logger *zap.Logger, readBucket storage.ReadBucket, testFilePath string) {
	ctx := context.Background()
	_, err := readConfigV1(ctx, logger, readBucket, internal.ReadConfigWithOverride(testFilePath))
	require.Error(t, err)
	data, err := os.ReadFile(testFilePath)
	require.NoError(t, err)
	_, err = readConfigV1(ctx, logger, readBucket, internal.ReadConfigWithOverride(string(data)))
	require.Error(t, err)
}

func mustCreateModuleIdentity(
	t *testing.T,
	remote string,
	owner string,
	repository string,
) bufmoduleref.ModuleIdentity {
	moduleIdentity, err := bufmoduleref.NewModuleIdentity(remote, owner, repository)
	require.NoError(t, err)
	return moduleIdentity
}

func assertConfigsWithEqualObjcPrefix(t *testing.T, successConfig *Config, config *Config) {
	require.Equal(t, successConfig.PluginConfigs, config.PluginConfigs)
	require.NotNil(t, successConfig.ManagedConfig)
	require.NotNil(t, config.ManagedConfig)
	require.NotNil(t, successConfig.ManagedConfig.ObjcClassPrefixConfig)
	require.NotNil(t, config.ManagedConfig.ObjcClassPrefixConfig)
	successObjcPrefixConfig := successConfig.ManagedConfig.ObjcClassPrefixConfig
	objcPrefixConfig := config.ManagedConfig.ObjcClassPrefixConfig
	require.Equal(t, successObjcPrefixConfig.Default, objcPrefixConfig.Default)
	require.Equal(t, successObjcPrefixConfig.Except, objcPrefixConfig.Except)
	assertEqualModuleIdentityKeyedMaps(t, successObjcPrefixConfig.Override, objcPrefixConfig.Override)
}

func assertConfigsWithEqualCsharpnamespace(t *testing.T, successConfig *Config, config *Config) {
	require.Equal(t, successConfig.PluginConfigs, config.PluginConfigs)
	require.NotNil(t, successConfig.ManagedConfig)
	require.NotNil(t, config.ManagedConfig)
	require.NotNil(t, successConfig.ManagedConfig.CsharpNameSpaceConfig)
	require.NotNil(t, config.ManagedConfig.CsharpNameSpaceConfig)
	successCsharpConfig := successConfig.ManagedConfig.CsharpNameSpaceConfig
	csharpConfig := config.ManagedConfig.CsharpNameSpaceConfig
	require.Equal(t, successCsharpConfig.Except, csharpConfig.Except)
	assertEqualModuleIdentityKeyedMaps(t, successCsharpConfig.Override, csharpConfig.Override)
}

func assertConfigsWithEqualRubyPackage(t *testing.T, successConfig *Config, config *Config) {
	require.Equal(t, successConfig.PluginConfigs, config.PluginConfigs)
	require.NotNil(t, successConfig.ManagedConfig)
	require.NotNil(t, config.ManagedConfig)
	require.NotNil(t, successConfig.ManagedConfig.RubyPackageConfig)
	require.NotNil(t, config.ManagedConfig.RubyPackageConfig)
	successRubyConfig := successConfig.ManagedConfig.RubyPackageConfig
	rubyConfig := config.ManagedConfig.RubyPackageConfig
	require.Equal(t, successRubyConfig.Except, rubyConfig.Except)
	assertEqualModuleIdentityKeyedMaps(t, successRubyConfig.Override, rubyConfig.Override)
}

func assertConfigsWithEqualOptimizeFor(t *testing.T, successConfig *Config, config *Config) {
	require.Equal(t, successConfig.PluginConfigs, config.PluginConfigs)
	require.NotNil(t, successConfig.ManagedConfig)
	require.NotNil(t, config.ManagedConfig)
	successOptimizeForConfig := successConfig.ManagedConfig.OptimizeForConfig
	require.NotNil(t, successOptimizeForConfig)
	optimizeForConfig := config.ManagedConfig.OptimizeForConfig
	require.NotNil(t, optimizeForConfig)
	require.Equal(t, successOptimizeForConfig.Default, optimizeForConfig.Default)
	require.Equal(t, successOptimizeForConfig.Except, optimizeForConfig.Except)
	assertEqualModuleIdentityKeyedMaps(t, optimizeForConfig.Override, optimizeForConfig.Override)
}

func assertEqualModuleIdentityKeyedMaps[V any](t *testing.T, m1 map[bufmoduleref.ModuleIdentity]V, m2 map[bufmoduleref.ModuleIdentity]V) {
	require.Equal(t, len(m1), len(m2))
	keyedM1 := make(map[string]V, len(m1))
	keyedM2 := make(map[string]V, len(m2))
	for k, v := range m1 {
		keyedM1[k.IdentityString()] = v
	}
	for k, v := range m2 {
		keyedM2[k.IdentityString()] = v
	}
	require.Equal(t, keyedM1, keyedM2)
}

func mustCreateBinaryPlugin(
	t *testing.T,
	name string,
	path []string,
	strategy internal.Strategy,
	out string,
	opt string,
) plugingen.BinaryPluginConfig {
	plugin, err := plugingen.NewBinaryPluginConfig(
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

func mustCreateLocalPlugin(
	t *testing.T,
	name string,
	strategy internal.Strategy,
	out string,
	opt string,
) plugingen.LocalPluginConfig {
	plugin, err := plugingen.NewLocalPluginConfig(
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

func mustCreateLegacyRemotePlugin(
	t *testing.T,
	remote string,
	out string,
	opt string,
) plugingen.LegacyRemotePluginConfig {
	plugin, err := plugingen.NewLegacyRemotePluginConfig(
		remote,
		out,
		opt,
		false,
		false,
	)
	require.NoError(t, err)
	return plugin
}
