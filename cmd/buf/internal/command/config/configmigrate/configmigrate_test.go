// Copyright 2020-2025 Buf Technologies, Inc.
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

package configmigrate

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appcmd/appcmdtesting"
	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/cmd/buf/internal/internaltesting"
	"github.com/bufbuild/buf/private/pkg/osext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigMigrateV1DefaultConfig(t *testing.T) {
	// Cannot be parallel since we chdir.
	testCompareConfigMigrate(t, "testdata/defaultv1", 0, "")
}

func TestConfigMigrateV1BetaV1DefaultConfig(t *testing.T) {
	// Cannot be parallel since we chdir.
	testCompareConfigMigrate(t, "testdata/defaultv1beta1", 0, "")
}

func TestConfigMigrateUnknownVersion(t *testing.T) {
	// Cannot be parallel since we chdir.
	testCompareConfigMigrate(t, "testdata/unknown", 1, "decode buf.yaml: \"version\" is not set. Please add \"version: v2\"")
}

func testCompareConfigMigrate(t *testing.T, dir string, expectCode int, expectStderr string) {
	// Setup temporary bucket with input, then compare it to the output.
	storageosProvider := storageos.NewProvider()
	inputBucket, err := storageosProvider.NewReadWriteBucket(filepath.Join(dir, "input"))
	require.NoError(t, err)
	tempDir := t.TempDir()
	tempBucket, err := storageosProvider.NewReadWriteBucket(tempDir)
	require.NoError(t, err)
	ctx := context.Background()
	_, err = storage.Copy(ctx, inputBucket, tempBucket)
	require.NoError(t, err)
	var outputBucket storage.ReadWriteBucket
	if expectCode == 0 {
		outputBucket, err = storageosProvider.NewReadWriteBucket(filepath.Join(dir, "output"))
		require.NoError(t, err)
	}
	// Run in the temp directory.
	func() {
		pwd, err := osext.Getwd()
		require.NoError(t, err)
		require.NoError(t, osext.Chdir(tempDir))
		defer func() {
			r := recover()
			assert.NoError(t, osext.Chdir(pwd))
			if r != nil {
				panic(r)
			}
		}()
		appcmdtesting.Run(
			t,
			func(use string) *appcmd.Command {
				return NewCommand(use, appext.NewBuilder(use))
			},
			appcmdtesting.WithExpectedExitCode(expectCode),
			appcmdtesting.WithExpectedStderr(expectStderr),
			appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		)
	}()
	if expectCode != 0 {
		return // Nothing to compare.
	}
	var diff bytes.Buffer
	require.NoError(t, storage.Diff(ctx, &diff, outputBucket, tempBucket))
	assert.Empty(t, diff.String())
}
