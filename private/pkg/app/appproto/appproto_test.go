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

package appproto

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestWriteInsertionPoint(t *testing.T) {
	t.Parallel()
	// \u205F is "Medium Mathematical Space"
	whitespacePrefix := "\u205F\t\t\t"
	targetFileContent := `This is a test file
// @@protoc_insertion_point(ip1)

that has more than one insertion point
    // @@protoc_insertion_point(ip2)

at varied indentation levels
` + whitespacePrefix + "// @@protoc_insertion_point(ip3)\n"
	targetFileName := "test.proto"
	insertionPointContent := "!!! this content was inserted; こんにちは"

	t.Run("basic", func(t *testing.T) {
		t.Parallel()
		insertionPointName := "ip1"
		insertionPointConsumer := &pluginpb.CodeGeneratorResponse_File{
			Name:           &targetFileName,
			InsertionPoint: &insertionPointName,
			Content:        &insertionPointContent,
		}

		postInsertionContent, err := writeInsertionPoint(
			context.Background(),
			insertionPointConsumer,
			strings.NewReader(targetFileContent),
		)
		require.NoError(t, err)
		expectContent := []byte(`This is a test file
!!! this content was inserted; こんにちは
// @@protoc_insertion_point(ip1)

that has more than one insertion point
    // @@protoc_insertion_point(ip2)

at varied indentation levels
` + whitespacePrefix + "// @@protoc_insertion_point(ip3)")

		assert.Equal(t, expectContent, postInsertionContent)
	})
	t.Run("basic_indent", func(t *testing.T) {
		t.Parallel()
		insertionPointName := "ip2"
		insertionPointConsumer := &pluginpb.CodeGeneratorResponse_File{
			Name:           &targetFileName,
			InsertionPoint: &insertionPointName,
			Content:        &insertionPointContent,
		}

		postInsertionContent, err := writeInsertionPoint(
			context.Background(),
			insertionPointConsumer,
			strings.NewReader(targetFileContent),
		)
		require.NoError(t, err)
		expectContent := []byte(`This is a test file
// @@protoc_insertion_point(ip1)

that has more than one insertion point
    !!! this content was inserted; こんにちは
    // @@protoc_insertion_point(ip2)

at varied indentation levels
` + whitespacePrefix + "// @@protoc_insertion_point(ip3)")

		assert.Equal(t, expectContent, postInsertionContent)
	})
	t.Run("basic_unicode_indent", func(t *testing.T) {
		t.Parallel()
		insertionPointName := "ip3"
		insertionPointConsumer := &pluginpb.CodeGeneratorResponse_File{
			Name:           &targetFileName,
			InsertionPoint: &insertionPointName,
			Content:        &insertionPointContent,
		}

		postInsertionContent, err := writeInsertionPoint(
			context.Background(),
			insertionPointConsumer,
			strings.NewReader(targetFileContent),
		)
		require.NoError(t, err)
		expectContent := []byte(`This is a test file
// @@protoc_insertion_point(ip1)

that has more than one insertion point
    // @@protoc_insertion_point(ip2)

at varied indentation levels
` +
			whitespacePrefix +
			"!!! this content was inserted; こんにちは" + "\n" +
			whitespacePrefix +
			"// @@protoc_insertion_point(ip3)")

		assert.Equal(t, expectContent, postInsertionContent)
	})
}

func BenchmarkWriteInsertionPoint(b *testing.B) {
	// \u205F is "Medium Mathematical Space"
	whitespacePrefix := "\u205F\t\t\t"
	targetFileContent := `This is a test file
// @@protoc_insertion_point(ip1)

that has more than one insertion point
    // @@protoc_insertion_point(ip2)

at varied indentation levels` + whitespacePrefix + "// @@protoc_insertion_point(ip3)\n"
	targetFileName := "test.proto"
	insertionPointContent := "!!! this content was inserted; こんにちは"
	// Our generated files in private/gen/proto are on average 1100 lines.
	inflatedLines := 1100

	inflatedTargetFileContent := targetFileContent
	for i := 0; i < inflatedLines-1; i++ {
		inflatedTargetFileContent += "// this is just extra garbage\n"
	}
	// no trailing newline
	inflatedTargetFileContent += "// this is just extra garbage"

	b.Run("basic", func(b *testing.B) {
		var postInsertionContent []byte
		insertionPointName := "ip1"
		insertionPointConsumer := &pluginpb.CodeGeneratorResponse_File{
			Name:           &targetFileName,
			InsertionPoint: &insertionPointName,
			Content:        &insertionPointContent,
		}
		expectContent := []byte(`This is a test file
!!! this content was inserted; こんにちは
// @@protoc_insertion_point(ip1)

that has more than one insertion point
    // @@protoc_insertion_point(ip2)

at varied indentation levels` + whitespacePrefix + "// @@protoc_insertion_point(ip3)")

		for i := 0; i < b.N; i++ {
			b.ReportAllocs()

			postInsertionContent, _ = writeInsertionPoint(
				context.Background(),
				insertionPointConsumer,
				strings.NewReader(targetFileContent),
			)
		}
		assert.Equal(b, expectContent, postInsertionContent)
	})

	b.Run("inflated", func(b *testing.B) {
		insertionPointName := "ip1"
		insertionPointConsumer := &pluginpb.CodeGeneratorResponse_File{
			Name:           &targetFileName,
			InsertionPoint: &insertionPointName,
			Content:        &insertionPointContent,
		}
		inflatedExpectContent := []byte(`This is a test file
!!! this content was inserted; こんにちは
// @@protoc_insertion_point(ip1)

that has more than one insertion point
    // @@protoc_insertion_point(ip2)

at varied indentation levels` + whitespacePrefix + "// @@protoc_insertion_point(ip3)\n")
		for i := 0; i < inflatedLines-1; i++ {
			inflatedExpectContent = append(inflatedExpectContent, []byte("// this is just extra garbage\n")...)
		}
		// no trailing newline
		inflatedExpectContent = append(inflatedExpectContent, []byte("// this is just extra garbage")...)

		var postInsertionContent []byte
		for i := 0; i < b.N; i++ {
			b.ReportAllocs()

			postInsertionContent, _ = writeInsertionPoint(
				context.Background(),
				insertionPointConsumer,
				strings.NewReader(inflatedTargetFileContent),
			)
		}
		assert.Equal(b, inflatedExpectContent, postInsertionContent)
	})
}
