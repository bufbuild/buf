// Copyright 2020-2026 Buf Technologies, Inc.
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

package bufprotopluginos

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestResponseWriterSkipsUnchangedFile(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	content := "package foo\n"
	filePath := filepath.Join(outDir, "foo.go")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0600))
	past := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(filePath, past, past))

	runResponseWriter(t, outDir, newResponseFile("foo.go", content))

	info, err := os.Stat(filePath)
	require.NoError(t, err)
	require.Equal(t, past.Truncate(time.Second), info.ModTime().Truncate(time.Second))
}

func TestResponseWriterWritesChangedFile(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	filePath := filepath.Join(outDir, "foo.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package old\n"), 0600))
	past := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(filePath, past, past))

	newContent := "package new\n"
	runResponseWriter(t, outDir, newResponseFile("foo.go", newContent))

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, newContent, string(data))
	info, err := os.Stat(filePath)
	require.NoError(t, err)
	require.Greater(t, info.ModTime(), past)
}

func TestResponseWriterWritesNewFile(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	content := "package foo\n"

	runResponseWriter(t, outDir, newResponseFile("foo.go", content))

	data, err := os.ReadFile(filepath.Join(outDir, "foo.go"))
	require.NoError(t, err)
	require.Equal(t, content, string(data))
}

func TestResponseWriterMixedFiles(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	unchangedContent := "package unchanged\n"
	unchangedPath := filepath.Join(outDir, "unchanged.go")
	changedPath := filepath.Join(outDir, "changed.go")
	newPath := filepath.Join(outDir, "new.go")
	require.NoError(t, os.WriteFile(unchangedPath, []byte(unchangedContent), 0600))
	require.NoError(t, os.WriteFile(changedPath, []byte("package old\n"), 0600))
	past := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(unchangedPath, past, past))
	require.NoError(t, os.Chtimes(changedPath, past, past))

	runResponseWriter(t, outDir,
		newResponseFile("unchanged.go", unchangedContent),
		newResponseFile("changed.go", "package changed\n"),
		newResponseFile("new.go", "package new\n"),
	)

	unchangedInfo, err := os.Stat(unchangedPath)
	require.NoError(t, err)
	require.Equal(t, past.Truncate(time.Second), unchangedInfo.ModTime().Truncate(time.Second))

	changedData, err := os.ReadFile(changedPath)
	require.NoError(t, err)
	require.Equal(t, "package changed\n", string(changedData))
	changedInfo, err := os.Stat(changedPath)
	require.NoError(t, err)
	require.Greater(t, changedInfo.ModTime(), past)

	newData, err := os.ReadFile(newPath)
	require.NoError(t, err)
	require.Equal(t, "package new\n", string(newData))
}

func runResponseWriter(t *testing.T, outDir string, files ...*pluginpb.CodeGeneratorResponse_File) {
	t.Helper()
	writer := NewResponseWriter(
		slogtestext.NewLogger(t),
		storageos.NewProvider(),
		ResponseWriterWithCreateOutDirIfNotExists(),
	)
	require.NoError(t, writer.AddResponse(
		t.Context(),
		&pluginpb.CodeGeneratorResponse{File: files},
		outDir,
	))
	require.NoError(t, writer.Close())
}

func newResponseFile(name, content string) *pluginpb.CodeGeneratorResponse_File {
	return &pluginpb.CodeGeneratorResponse_File{
		Name:    &name,
		Content: &content,
	}
}
