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

package generate

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

var testImageInputDirPath = filepath.Join("testdata", "v2", "image")

func TestGenerateImageDefault(t *testing.T) {
	t.Parallel()
	tempDirPath := testGenerateImage(
		t,
		`
version: v2
plugins:
  - image: image.binpb
    out: gen
`,
		testImageInputDirPath,
	)
	image := readTestImage(t, filepath.Join(tempDirPath, "gen", "image.binpb"), nil)
	require.Equal(
		t,
		[]testImageFile{
			{path: "google/protobuf/timestamp.proto", isImport: true},
			{path: "google/protobuf/descriptor.proto", isImport: true},
			{path: "options/v1/options.proto"},
			{path: "a/v1/a.proto"},
			{path: "b/v1/b.proto"},
		},
		getTestImageFiles(image),
	)
	for _, imageFile := range image.GetFile() {
		require.NotEmpty(t, imageFile.GetSourceCodeInfo().GetLocation(), imageFile.GetName())
	}
	// The source-retention option is stripped, the runtime option is kept.
	fooOptions := getTestImageFile(t, image, "a/v1/a.proto").GetMessageType()[0].GetOptions()
	require.Equal(t, []protowire.Number{50001}, getUnknownFieldNumbers(t, fooOptions))
}

func TestGenerateImageAsFileDescriptorSet(t *testing.T) {
	t.Parallel()
	tempDirPath := testGenerateImage(
		t,
		`
version: v2
plugins:
  - image: image.binpb
    out: gen
    as_file_descriptor_set: true
`,
		testImageInputDirPath,
	)
	data := readTestImageData(t, filepath.Join(tempDirPath, "gen", "image.binpb"))
	fileDescriptorSet := &descriptorpb.FileDescriptorSet{}
	require.NoError(t, protoencoding.NewWireUnmarshaler(nil).Unmarshal(data, fileDescriptorSet))
	fileNames := make([]string, 0, len(fileDescriptorSet.GetFile()))
	for _, fileDescriptorProto := range fileDescriptorSet.GetFile() {
		fileNames = append(fileNames, fileDescriptorProto.GetName())
	}
	require.Equal(
		t,
		[]string{
			"google/protobuf/timestamp.proto",
			"google/protobuf/descriptor.proto",
			"options/v1/options.proto",
			"a/v1/a.proto",
			"b/v1/b.proto",
		},
		fileNames,
	)
	// A FileDescriptorSet has no Buf image extension, so this would be an
	// unknown field if the output were an image.
	for _, fileDescriptorProto := range fileDescriptorSet.GetFile() {
		require.Empty(t, getUnknownFieldNumbers(t, fileDescriptorProto), fileDescriptorProto.GetName())
	}
}

func TestGenerateImageEncodings(t *testing.T) {
	t.Parallel()
	// The encodings must all describe the same image as the default.
	expected := readTestImage(t, filepath.Join(testGenerateImage(
		t,
		`
version: v2
plugins:
  - image: image.binpb
    out: gen
`,
		testImageInputDirPath,
	), "gen", "image.binpb"), nil)
	resolver, err := protoencoding.NewResolver(expected.GetFile()...)
	require.NoError(t, err)
	for _, imageFileName := range []string{
		"image.json",
		"image.txtpb",
		"image.yaml",
		"image.binpb.gz",
		"image.json.gz",
		"image.binpb.zst",
		"nested/image.yaml.zst",
	} {
		t.Run(imageFileName, func(t *testing.T) {
			t.Parallel()
			tempDirPath := testGenerateImage(
				t,
				`
version: v2
plugins:
  - image: `+imageFileName+`
    out: gen
`,
				testImageInputDirPath,
			)
			image := readTestImage(t, filepath.Join(tempDirPath, "gen", filepath.FromSlash(imageFileName)), resolver)
			require.True(t, proto.Equal(expected, image))
		})
	}
}

func TestGenerateImageExcludeImports(t *testing.T) {
	t.Parallel()
	expected := []testImageFile{
		{path: "options/v1/options.proto"},
		{path: "a/v1/a.proto"},
		{path: "b/v1/b.proto"},
	}
	template := `
version: v2
plugins:
  - image: image.binpb
    out: gen
    exclude_imports: true
`
	tempDirPath := testGenerateImage(t, template, testImageInputDirPath)
	image := readTestImage(t, filepath.Join(tempDirPath, "gen", "image.binpb"), nil)
	require.Equal(t, expected, getTestImageFiles(image))
	// The --include-imports override does not apply to image plugins.
	tempDirPath = testGenerateImage(t, template, testImageInputDirPath, "--include-imports", "--include-wkt")
	image = readTestImage(t, filepath.Join(tempDirPath, "gen", "image.binpb"), nil)
	require.Equal(t, expected, getTestImageFiles(image))
}

func TestGenerateImageExcludeSourceInfo(t *testing.T) {
	t.Parallel()
	tempDirPath := testGenerateImage(
		t,
		`
version: v2
plugins:
  - image: image.binpb
    out: gen
    exclude_source_info: true
`,
		testImageInputDirPath,
	)
	image := readTestImage(t, filepath.Join(tempDirPath, "gen", "image.binpb"), nil)
	require.Len(t, image.GetFile(), 5)
	for _, imageFile := range image.GetFile() {
		require.False(t, imageFile.HasSourceCodeInfo(), imageFile.GetName())
	}
	fooOptions := getTestImageFile(t, image, "a/v1/a.proto").GetMessageType()[0].GetOptions()
	require.Equal(t, []protowire.Number{50001}, getUnknownFieldNumbers(t, fooOptions))
}

func TestGenerateImageTypes(t *testing.T) {
	t.Parallel()
	tempDirPath := testGenerateImage(
		t,
		`
version: v2
plugins:
  - image: include.binpb
    out: gen
    types:
      - a.v1.Foo
  - image: exclude.binpb
    out: gen
    exclude_types:
      - a.v1.Foo
`,
		testImageInputDirPath,
	)
	image := readTestImage(t, filepath.Join(tempDirPath, "gen", "include.binpb"), nil)
	require.Equal(
		t,
		[]testImageFile{
			{path: "google/protobuf/timestamp.proto", isImport: true},
			{path: "google/protobuf/descriptor.proto", isImport: true},
			{path: "options/v1/options.proto", isImport: true},
			{path: "a/v1/a.proto"},
		},
		getTestImageFiles(image),
	)
	require.Equal(t, []string{"Foo"}, getTestMessageNames(getTestImageFile(t, image, "a/v1/a.proto")))
	image = readTestImage(t, filepath.Join(tempDirPath, "gen", "exclude.binpb"), nil)
	require.Equal(
		t,
		[]testImageFile{
			{path: "google/protobuf/descriptor.proto", isImport: true},
			{path: "options/v1/options.proto"},
			{path: "a/v1/a.proto"},
			{path: "b/v1/b.proto"},
		},
		getTestImageFiles(image),
	)
	require.Equal(t, []string{"Bar"}, getTestMessageNames(getTestImageFile(t, image, "a/v1/a.proto")))
}

func TestGenerateImageManagedMode(t *testing.T) {
	t.Parallel()
	tempDirPath := testGenerateImage(
		t,
		`
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: example.com/gen
plugins:
  - image: image.binpb
    out: gen
`,
		testImageInputDirPath,
	)
	image := readTestImage(t, filepath.Join(tempDirPath, "gen", "image.binpb"), nil)
	require.Equal(
		t,
		"example.com/gen/a/v1;av1",
		getTestImageFile(t, image, "a/v1/a.proto").GetOptions().GetGoPackage(),
	)
}

func TestGenerateImageMergeInputs(t *testing.T) {
	t.Parallel()
	tempDirPath := testGenerateImage(
		t,
		`
version: v2
plugins:
  - image: image.binpb
    out: gen
inputs:
  - directory: testdata/v2/image
    paths:
      - testdata/v2/image/b
  - directory: testdata/v2/image
    paths:
      - testdata/v2/image/a
`,
	)
	image := readTestImage(t, filepath.Join(tempDirPath, "gen", "image.binpb"), nil)
	require.Equal(
		t,
		[]testImageFile{
			{path: "b/v1/b.proto"},
			{path: "google/protobuf/timestamp.proto", isImport: true},
			{path: "google/protobuf/descriptor.proto", isImport: true},
			{path: "options/v1/options.proto", isImport: true},
			{path: "a/v1/a.proto"},
		},
		getTestImageFiles(image),
	)
}

func TestGenerateImageMergeInputsConflict(t *testing.T) {
	t.Parallel()
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: file "b/v1/b.proto" differs between images`,
		"--template",
		`
version: v2
plugins:
  - image: image.binpb
    out: gen
inputs:
  - directory: testdata/v2/image
  - directory: testdata/v2/image_conflict
`,
		"-o",
		t.TempDir(),
	)
}

func TestGenerateImageWithLocalPlugin(t *testing.T) {
	t.Parallel()
	tempDirPath := testGenerateImage(
		t,
		`
version: v2
plugins:
  - local: protoc-gen-top-level-type-names-yaml
    out: gen
  - image: image.binpb
    out: gen
`,
		testImageInputDirPath,
		"--path",
		filepath.Join(testImageInputDirPath, "a"),
	)
	data, err := os.ReadFile(filepath.Join(tempDirPath, "gen", "a", "v1", "a.top-level-type-names.yaml"))
	require.NoError(t, err)
	require.Equal(t, "messages:\n    - a.v1.Bar\n    - a.v1.Foo\n", string(data))
	image := readTestImage(t, filepath.Join(tempDirPath, "gen", "image.binpb"), nil)
	require.Equal(
		t,
		[]testImageFile{
			{path: "google/protobuf/timestamp.proto", isImport: true},
			{path: "google/protobuf/descriptor.proto", isImport: true},
			{path: "options/v1/options.proto", isImport: true},
			{path: "a/v1/a.proto"},
		},
		getTestImageFiles(image),
	)
}

func TestGenerateImageZip(t *testing.T) {
	t.Parallel()
	tempDirPath := testGenerateImage(
		t,
		`
version: v2
plugins:
  - image: image.binpb
    out: gen.zip
`,
		testImageInputDirPath,
	)
	zipReader, err := zip.OpenReader(filepath.Join(tempDirPath, "gen.zip"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, zipReader.Close()) })
	file, err := zipReader.Open("image.binpb")
	require.NoError(t, err)
	data, err := io.ReadAll(file)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	image := &imagev1.Image{}
	require.NoError(t, protoencoding.NewWireUnmarshaler(nil).Unmarshal(data, image))
	require.Len(t, image.GetFile(), 5)
}

func TestGenerateImageClean(t *testing.T) {
	t.Parallel()
	tempDirPath := t.TempDir()
	staleFilePath := filepath.Join(tempDirPath, "gen", "stale.binpb")
	require.NoError(t, os.MkdirAll(filepath.Dir(staleFilePath), 0755))
	require.NoError(t, os.WriteFile(staleFilePath, []byte("stale"), 0600))
	testRunSuccess(
		t,
		"--template",
		`
version: v2
clean: true
plugins:
  - image: image.binpb
    out: gen
`,
		"-o",
		tempDirPath,
		testImageInputDirPath,
	)
	_, err := os.Stat(staleFilePath)
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(filepath.Join(tempDirPath, "gen", "image.binpb"))
	require.NoError(t, err)
}

func TestGenerateImageDuplicateFileFail(t *testing.T) {
	t.Parallel()
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: file "gen/a/v1/a.top-level-type-names.yaml" was generated multiple times: once by plugin "protoc-gen-top-level-type-names-yaml" and again by plugin "a/v1/a.top-level-type-names.yaml"`,
		"--template",
		`
version: v2
plugins:
  - local: protoc-gen-top-level-type-names-yaml
    out: gen
  - image: a/v1/a.top-level-type-names.yaml
    out: gen
`,
		"-o",
		t.TempDir(),
		testImageInputDirPath,
		"--path",
		filepath.Join(testImageInputDirPath, "a"),
	)
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: file "gen/image.binpb" was generated multiple times: once by plugin "image.binpb" and again by plugin "image.binpb"`,
		"--template",
		`
version: v2
plugins:
  - image: image.binpb
    out: gen
  - image: image.binpb
    out: gen
    exclude_imports: true
`,
		"-o",
		t.TempDir(),
		testImageInputDirPath,
	)
}

// testGenerateImage runs generate with the template into a new temporary
// directory, and returns the directory.
func testGenerateImage(t *testing.T, template string, args ...string) string {
	tempDirPath := t.TempDir()
	testRunSuccess(t, append([]string{"--template", template, "-o", tempDirPath}, args...)...)
	return tempDirPath
}

type testImageFile struct {
	path     string
	isImport bool
}

func getTestImageFiles(image *imagev1.Image) []testImageFile {
	testImageFiles := make([]testImageFile, 0, len(image.GetFile()))
	for _, imageFile := range image.GetFile() {
		testImageFiles = append(testImageFiles, testImageFile{
			path:     imageFile.GetName(),
			isImport: imageFile.GetBufExtension().GetIsImport(),
		})
	}
	return testImageFiles
}

func getTestImageFile(t *testing.T, image *imagev1.Image, path string) *imagev1.ImageFile {
	for _, imageFile := range image.GetFile() {
		if imageFile.GetName() == path {
			return imageFile
		}
	}
	require.Failf(t, "file not found in image", "%s", path)
	return nil
}

func getTestMessageNames(imageFile *imagev1.ImageFile) []string {
	messageNames := make([]string, 0, len(imageFile.GetMessageType()))
	for _, messageType := range imageFile.GetMessageType() {
		messageNames = append(messageNames, messageType.GetName())
	}
	return messageNames
}

func getUnknownFieldNumbers(t *testing.T, message proto.Message) []protowire.Number {
	var numbers []protowire.Number
	unknown := message.ProtoReflect().GetUnknown()
	for len(unknown) > 0 {
		number, _, n := protowire.ConsumeField(unknown)
		require.GreaterOrEqual(t, n, 0)
		numbers = append(numbers, number)
		unknown = unknown[n:]
	}
	return numbers
}

// readTestImage reads the image file, decompressing and decoding it as given
// by its extensions. The resolver is used to decode JSON, text and YAML, and
// may be nil for binary. Custom options are returned as unknown fields.
func readTestImage(t *testing.T, filePath string, resolver protoencoding.Resolver) *imagev1.Image {
	data := readTestImageData(t, filePath)
	var unmarshaler protoencoding.Unmarshaler
	switch extension := filepath.Ext(strings.TrimSuffix(strings.TrimSuffix(filePath, ".gz"), ".zst")); extension {
	case ".binpb":
		unmarshaler = protoencoding.NewWireUnmarshaler(nil)
	case ".json":
		unmarshaler = protoencoding.NewJSONUnmarshaler(resolver)
	case ".txtpb":
		unmarshaler = protoencoding.NewTxtpbUnmarshaler(resolver)
	case ".yaml":
		unmarshaler = protoencoding.NewYAMLUnmarshaler(resolver)
	default:
		require.Failf(t, "unknown extension", "%s", extension)
	}
	image := &imagev1.Image{}
	require.NoError(t, unmarshaler.Unmarshal(data, image))
	// Round trip through the wire format so that custom options are unknown
	// fields for every encoding.
	wireData, err := protoencoding.NewWireMarshaler().Marshal(image)
	require.NoError(t, err)
	wireImage := &imagev1.Image{}
	require.NoError(t, protoencoding.NewWireUnmarshaler(nil).Unmarshal(wireData, wireImage))
	return wireImage
}

func readTestImageData(t *testing.T, filePath string) []byte {
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	switch filepath.Ext(filePath) {
	case ".gz":
		gzipReader, err := gzip.NewReader(bytes.NewReader(data))
		require.NoError(t, err)
		data, err = io.ReadAll(gzipReader)
		require.NoError(t, err)
		require.NoError(t, gzipReader.Close())
	case ".zst":
		zstdDecoder, err := zstd.NewReader(nil)
		require.NoError(t, err)
		defer zstdDecoder.Close()
		data, err = zstdDecoder.DecodeAll(data, nil)
		require.NoError(t, err)
	}
	return data
}
