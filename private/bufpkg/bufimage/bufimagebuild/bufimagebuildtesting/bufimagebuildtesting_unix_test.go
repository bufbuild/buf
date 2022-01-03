// Copyright 2020-2022 Buf Technologies, Inc.
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

//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd js,wasm linux netbsd openbsd solaris

package bufimagebuildtesting

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/stretchr/testify/require"
)

func TestCorpus(t *testing.T) {
	// To focus on just one test in the corpus, put its file name here. Don't forget to revert before committing.
	focus := ""
	ctx := context.Background()
	runner := command.NewRunner()
	require.NoError(t, filepath.Walk("corpus", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if focus != "" && info.Name() != focus {
			return nil
		}
		t.Run(info.Name(), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("corpus", info.Name()))
			require.NoError(t, err)
			result, err := fuzz(ctx, runner, data)
			require.NoError(t, err)
			require.NoError(t, result.error(ctx))
		})
		return nil
	}))
}
