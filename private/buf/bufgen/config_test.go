// Copyright 2020-2021 Buf Technologies, Inc.
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
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestReadConfigV1Beta1(t *testing.T) {
	truth := true
	successConfig := &Config{
		PluginConfigs: []*PluginConfig{
			{
				Name:     "go",
				Out:      "gen/go",
				Opt:      "plugins=grpc",
				Path:     "/path/to/foo",
				Strategy: StrategyAll,
			},
		},
		ManagedConfig: &ManagedConfig{
			CcEnableArenas:      &truth,
			JavaMultipleFiles:   &truth,
			JavaStringCheckUtf8: nil,
			OptimizeFor:         optimizeModePtr(descriptorpb.FileOptions_CODE_SIZE),
		},
	}
	successConfig2 := &Config{
		ManagedConfig: &ManagedConfig{
			OptimizeFor: optimizeModePtr(descriptorpb.FileOptions_SPEED),
		},
		PluginConfigs: []*PluginConfig{
			{
				Name:     "go",
				Out:      "gen/go",
				Opt:      "plugins=grpc,foo=bar",
				Path:     "/path/to/foo",
				Strategy: StrategyAll,
			},
		},
	}
	successConfig3 := &Config{
		ManagedConfig: &ManagedConfig{
			OptimizeFor: optimizeModePtr(descriptorpb.FileOptions_LITE_RUNTIME),
		},
		PluginConfigs: []*PluginConfig{
			{
				Name:     "go",
				Out:      "gen/go",
				Path:     "/path/to/foo",
				Strategy: StrategyAll,
			},
		},
	}
	ctx := context.Background()
	provider := NewProvider(zap.NewNop())
	readBucket, err := storagemem.NewReadBucket(nil)
	require.NoError(t, err)
	config, err := ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1beta1", "gen_success1.yaml")))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)
	data, err := os.ReadFile(filepath.Join("testdata", "v1beta1", "gen_success1.yaml"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1beta1", "gen_success1.json")))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1beta1", "gen_success1.json"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)

	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1beta1", "gen_success2.yaml")))
	require.NoError(t, err)
	require.Equal(t, successConfig2, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1beta1", "gen_success2.yaml"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig2, config)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1beta1", "gen_success2.json")))
	require.NoError(t, err)
	require.Equal(t, successConfig2, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1beta1", "gen_success2.json"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride((string(data))))
	require.NoError(t, err)
	require.Equal(t, successConfig2, config)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1beta1", "gen_success3.yaml")))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1beta1", "gen_success3.yaml"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride((string(data))))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1beta1", "gen_success3.json")))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1beta1", "gen_success3.json"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1beta1", "gen_success3.yml")))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1beta1", "gen_success3.yml"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)

	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1beta1", "gen_error1.yaml"))
	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1beta1", "gen_error2.yaml"))
	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1beta1", "gen_error3.yaml"))
}

func TestReadConfigV1(t *testing.T) {
	truth := true
	successConfig := &Config{
		PluginConfigs: []*PluginConfig{
			{
				Name:     "go",
				Out:      "gen/go",
				Opt:      "plugins=grpc",
				Path:     "/path/to/foo",
				Strategy: StrategyAll,
			},
		},
		ManagedConfig: &ManagedConfig{
			CcEnableArenas:      &truth,
			JavaMultipleFiles:   &truth,
			JavaStringCheckUtf8: &truth,
			JavaPackagePrefix:   "org",
			OptimizeFor:         optimizeModePtr(descriptorpb.FileOptions_CODE_SIZE),
		},
	}
	successConfig2 := &Config{
		ManagedConfig: &ManagedConfig{
			OptimizeFor: optimizeModePtr(descriptorpb.FileOptions_SPEED),
		},
		PluginConfigs: []*PluginConfig{
			{
				Name:     "go",
				Out:      "gen/go",
				Opt:      "plugins=grpc,foo=bar",
				Path:     "/path/to/foo",
				Strategy: StrategyAll,
			},
		},
	}
	successConfig3 := &Config{
		ManagedConfig: &ManagedConfig{
			OptimizeFor: optimizeModePtr(descriptorpb.FileOptions_LITE_RUNTIME),
		},
		PluginConfigs: []*PluginConfig{
			{
				Name:     "go",
				Out:      "gen/go",
				Path:     "/path/to/foo",
				Strategy: StrategyAll,
			},
		},
	}
	successConfig4 := &Config{
		PluginConfigs: []*PluginConfig{
			{
				Remote:   "someremote.com/owner/plugins/myplugin",
				Out:      "gen/go",
				Version:  "v1.1.0-1",
				Strategy: StrategyDirectory,
			},
		},
	}
	ctx := context.Background()
	provider := NewProvider(zap.NewNop())
	readBucket, err := storagemem.NewReadBucket(nil)
	require.NoError(t, err)
	config, err := ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success1.yaml")))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)
	data, err := os.ReadFile(filepath.Join("testdata", "v1", "gen_success1.yaml"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success1.json")))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success1.json"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)

	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success2.yaml")))
	require.NoError(t, err)
	require.Equal(t, successConfig2, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success2.yaml"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig2, config)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success2.json")))
	require.NoError(t, err)
	require.Equal(t, successConfig2, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success2.json"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride((string(data))))
	require.NoError(t, err)
	require.Equal(t, successConfig2, config)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success3.yaml")))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success3.yaml"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride((string(data))))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success3.json")))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success3.json"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success3.yml")))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success3.yml"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig3, config)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success4.yaml")))
	require.NoError(t, err)
	require.Equal(t, successConfig4, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success4.yaml"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride((string(data))))
	require.NoError(t, err)
	require.Equal(t, successConfig4, config)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success4.json")))
	require.NoError(t, err)
	require.Equal(t, successConfig4, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success4.json"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig4, config)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1", "gen_success4.yml")))
	require.NoError(t, err)
	require.Equal(t, successConfig4, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "gen_success4.yml"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig4, config)

	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1", "gen_error1.yaml"))
	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1", "gen_error2.yaml"))
	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1", "gen_error3.yaml"))
	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1", "gen_error4.yaml"))
	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1", "gen_error5.yaml"))
	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1", "gen_error6.yaml"))
	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1", "gen_error7.yaml"))
	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1", "gen_error8.yaml"))

	successConfig = &Config{
		PluginConfigs: []*PluginConfig{
			{
				Name:     "go",
				Out:      "gen/go",
				Opt:      "plugins=grpc",
				Path:     "/path/to/foo",
				Strategy: StrategyAll,
			},
		},
		ManagedConfig: &ManagedConfig{
			GoPackagePrefixConfig: &GoPackagePrefixConfig{
				Default:  "github.com/foo/bar/gen/go",
				Except:   make([]bufmodule.ModuleIdentity, 0),
				Override: make(map[bufmodule.ModuleIdentity]string),
			},
		},
	}
	readBucket, err = storagemem.NewReadBucket(nil)
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1", "go_gen_success1.yaml")))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "go_gen_success1.yaml"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(filepath.Join("testdata", "v1", "go_gen_success1.json")))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)
	data, err = os.ReadFile(filepath.Join("testdata", "v1", "go_gen_success1.json"))
	require.NoError(t, err)
	config, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(string(data)))
	require.NoError(t, err)
	require.Equal(t, successConfig, config)

	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1", "go_gen_error1.yaml"))
	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1", "go_gen_error2.yaml"))
	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1", "go_gen_error3.yaml"))
	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1", "go_gen_error4.yaml"))
	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1", "go_gen_error5.yaml"))
	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1", "go_gen_error6.yaml"))
	testReadConfigError(t, provider, readBucket, filepath.Join("testdata", "v1", "go_gen_error7.yaml"))
}

func testReadConfigError(t *testing.T, provider Provider, readBucket storage.ReadBucket, testFilePath string) {
	ctx := context.Background()
	_, err := ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(testFilePath))
	require.Error(t, err)
	data, err := os.ReadFile(testFilePath)
	require.NoError(t, err)
	_, err = ReadConfig(ctx, provider, readBucket, ReadConfigWithOverride(string(data)))
	require.Error(t, err)
}
