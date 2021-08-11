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

package bufimagebuildtesting

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCorpus(t *testing.T) {
	require.NoError(t, filepath.Walk("corpus", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		t.Run(info.Name(), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("corpus", info.Name()))
			require.NoError(t, err)
			testFuzz(t, data)
		})
		return nil
	}))
}

func TestCrashers(t *testing.T) {
	t.Skip("skipping known crashers")
	require.NoError(t, filepath.Walk("corpus", func(path string, entry fs.FileInfo, err error) error {
		// We only want files that don't end in .quoted or .output. *.quoted and *.output files are created by go-fuzz
		// for human consumption. They are not valid test data.
		if entry.IsDir() ||
			strings.HasSuffix(entry.Name(), ".quoted") ||
			strings.HasSuffix(entry.Name(), ".output") {
			return nil
		}
		t.Run(entry.Name(), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("crashers", entry.Name()))
			require.NoError(t, err)
			testFuzz(t, data)
		})
		return nil
	}))
}

// testFuzz runs a fuzz test and fails if data is invalid or if the Fuzz would have panicked
func testFuzz(t *testing.T, data []byte) {
	ctx := context.Background()
	result, err := fuzz(ctx, data)
	require.NoError(t, err)
	require.NoError(t, result.error(ctx))
}
