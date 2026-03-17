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

	runResponseWriter(t, outDir, false, newResponseFile("foo.go", content))

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
	runResponseWriter(t, outDir, false, newResponseFile("foo.go", newContent))

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

	runResponseWriter(t, outDir, false, newResponseFile("foo.go", "package foo\n"))

	data, err := os.ReadFile(filepath.Join(outDir, "foo.go"))
	require.NoError(t, err)
	require.Equal(t, "package foo\n", string(data))
}

func TestResponseWriterMixedFiles(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	unchangedContent := "package unchanged\n"
	unchangedPath := filepath.Join(outDir, "unchanged.go")
	changedPath := filepath.Join(outDir, "changed.go")
	require.NoError(t, os.WriteFile(unchangedPath, []byte(unchangedContent), 0600))
	require.NoError(t, os.WriteFile(changedPath, []byte("package old\n"), 0600))
	past := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(unchangedPath, past, past))
	require.NoError(t, os.Chtimes(changedPath, past, past))

	runResponseWriter(t, outDir, false,
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

	newData, err := os.ReadFile(filepath.Join(outDir, "new.go"))
	require.NoError(t, err)
	require.Equal(t, "package new\n", string(newData))
}

func TestResponseWriterSmartCleanDeletesStaleFile(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	stalePath := filepath.Join(outDir, "stale.go")
	require.NoError(t, os.WriteFile(stalePath, []byte("package stale\n"), 0600))

	// Generate only foo.go; stale.go should be deleted.
	runResponseWriter(t, outDir, true, newResponseFile("foo.go", "package foo\n"))

	_, err := os.Stat(stalePath)
	require.ErrorIs(t, err, os.ErrNotExist)
	data, err := os.ReadFile(filepath.Join(outDir, "foo.go"))
	require.NoError(t, err)
	require.Equal(t, "package foo\n", string(data))
}

func TestResponseWriterSmartCleanPreservesMtimeForUnchanged(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	content := "package foo\n"
	filePath := filepath.Join(outDir, "foo.go")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0600))
	past := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(filePath, past, past))

	runResponseWriter(t, outDir, true, newResponseFile("foo.go", content))

	info, err := os.Stat(filePath)
	require.NoError(t, err)
	require.Equal(t, past.Truncate(time.Second), info.ModTime().Truncate(time.Second))
}

func TestResponseWriterSmartCleanMixedFiles(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	unchangedContent := "package unchanged\n"
	unchangedPath := filepath.Join(outDir, "unchanged.go")
	changedPath := filepath.Join(outDir, "changed.go")
	stalePath := filepath.Join(outDir, "stale.go")
	require.NoError(t, os.WriteFile(unchangedPath, []byte(unchangedContent), 0600))
	require.NoError(t, os.WriteFile(changedPath, []byte("package old\n"), 0600))
	require.NoError(t, os.WriteFile(stalePath, []byte("package stale\n"), 0600))
	past := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(unchangedPath, past, past))
	require.NoError(t, os.Chtimes(changedPath, past, past))

	runResponseWriter(t, outDir, true,
		newResponseFile("unchanged.go", unchangedContent),
		newResponseFile("changed.go", "package changed\n"),
		newResponseFile("new.go", "package new\n"),
	)

	// Unchanged: mtime preserved.
	unchangedInfo, err := os.Stat(unchangedPath)
	require.NoError(t, err)
	require.Equal(t, past.Truncate(time.Second), unchangedInfo.ModTime().Truncate(time.Second))

	// Changed: new content, updated mtime.
	changedData, err := os.ReadFile(changedPath)
	require.NoError(t, err)
	require.Equal(t, "package changed\n", string(changedData))
	changedInfo, err := os.Stat(changedPath)
	require.NoError(t, err)
	require.Greater(t, changedInfo.ModTime(), past)

	// New: created.
	newData, err := os.ReadFile(filepath.Join(outDir, "new.go"))
	require.NoError(t, err)
	require.Equal(t, "package new\n", string(newData))

	// Stale: deleted.
	_, err = os.Stat(stalePath)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestResponseWriterZipPreservesMtimeWhenUnchanged(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "output.zip")

	// First run creates the zip.
	runResponseWriter(t, outFile, false, newResponseFile("foo.go", "package foo\n"))
	require.FileExists(t, outFile)

	past := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(outFile, past, past))

	// Second run with identical content should not rewrite the zip.
	runResponseWriter(t, outFile, false, newResponseFile("foo.go", "package foo\n"))

	info, err := os.Stat(outFile)
	require.NoError(t, err)
	require.Equal(t, past.Truncate(time.Second), info.ModTime().Truncate(time.Second))
}

func TestResponseWriterZipUpdatesWhenChanged(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "output.zip")

	runResponseWriter(t, outFile, false, newResponseFile("foo.go", "package foo\n"))

	past := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(outFile, past, past))

	// Second run with different content should rewrite.
	runResponseWriter(t, outFile, false, newResponseFile("foo.go", "package bar\n"))

	info, err := os.Stat(outFile)
	require.NoError(t, err)
	require.Greater(t, info.ModTime(), past)
}

func TestResponseWriterSmartCleanRemovesEmptyDirs(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	subDir := filepath.Join(outDir, "subpkg")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	// Pre-existing file in a subdirectory that will become stale.
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "stale.go"), []byte("package stale\n"), 0600))

	// Generate only to the root dir; nothing goes into subpkg.
	runResponseWriter(t, outDir, true, newResponseFile("foo.go", "package foo\n"))

	// stale.go deleted, subpkg now empty and also removed.
	_, err := os.Stat(subDir)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestResponseWriterSmartCleanRemovesNestedEmptyDirs(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	// Create a/b/c/stale.go - all three directories should be removed once
	// stale.go is deleted, because each parent becomes empty after its child
	// is removed.
	require.NoError(t, os.MkdirAll(filepath.Join(outDir, "a", "b", "c"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(outDir, "a", "b", "c", "stale.go"), []byte("package stale\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(outDir, "a", "b", "kept.go"), []byte("package kept\n"), 0600))
	// a/d is a pre-existing empty directory with no files.
	require.NoError(t, os.MkdirAll(filepath.Join(outDir, "a", "d"), 0755))

	runResponseWriter(t, outDir, true,
		newResponseFile("foo.go", "package foo\n"),
		newResponseFile("a/b/kept.go", "package kept\n"),
	)

	// a/b/c removed (stale file deleted, dir now empty).
	_, err := os.Stat(filepath.Join(outDir, "a", "b", "c"))
	require.ErrorIs(t, err, os.ErrNotExist)
	// a/d removed (pre-existing empty directory).
	_, err = os.Stat(filepath.Join(outDir, "a", "d"))
	require.ErrorIs(t, err, os.ErrNotExist)
	// a/b still present because a/b/kept.go is generated output.
	require.FileExists(t, filepath.Join(outDir, "a", "b", "kept.go"))
	require.DirExists(t, filepath.Join(outDir, "a", "b"))
	require.DirExists(t, filepath.Join(outDir, "a"))
}

func runResponseWriter(t *testing.T, outPath string, deleteOuts bool, files ...*pluginpb.CodeGeneratorResponse_File) {
	t.Helper()
	opts := []ResponseWriterOption{
		ResponseWriterWithCreateOutDirIfNotExists(),
	}
	if deleteOuts {
		opts = append(opts, ResponseWriterWithDeleteOuts())
	}
	writer := NewResponseWriter(
		slogtestext.NewLogger(t),
		storageos.NewProvider(),
		opts...,
	)
	require.NoError(t, writer.AddResponse(
		t.Context(),
		&pluginpb.CodeGeneratorResponse{File: files},
		outPath,
	))
	require.NoError(t, writer.Close())
}

func newResponseFile(name, content string) *pluginpb.CodeGeneratorResponse_File {
	return &pluginpb.CodeGeneratorResponse_File{
		Name:    &name,
		Content: &content,
	}
}
