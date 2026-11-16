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

package bufimage

import (
	"reflect"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestCloneImage(t *testing.T) {
	t.Parallel()
	protoImage := imagev1.Image_builder{
		File: []*imagev1.ImageFile{
			imagev1.ImageFile_builder{
				Syntax:     new("proto3"),
				Name:       new("a.proto"),
				Dependency: []string{"b.proto", "c.proto"},
				Package:    new("abc.def"),
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: new("Msg"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:     new("id"),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Number:   proto.Int32(1),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
								JsonName: new("id"),
							},
							{
								Name:     new("en"),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
								Number:   proto.Int32(2),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
								TypeName: new(".abc.def.Enum"),
								JsonName: new("en"),
							},
						},
					},
				},
				BufExtension: imagev1.ImageFileExtension_builder{
					IsImport:            new(false),
					IsSyntaxUnspecified: new(false),
					UnusedDependency:    []int32{1},
					ModuleInfo: imagev1.ModuleInfo_builder{
						Name: imagev1.ModuleName_builder{
							Remote:     new("buf.build"),
							Owner:      new("foo"),
							Repository: new("bar"),
						}.Build(),
						Commit: new("9876543210fedcba9876543210fedcba"),
					}.Build(),
				}.Build(),
			}.Build(),
			imagev1.ImageFile_builder{
				Syntax:  new("proto3"),
				Name:    new("b.proto"),
				Package: new("abc.def"),
				EnumType: []*descriptorpb.EnumDescriptorProto{
					{
						Name: new("Enum"),
						Value: []*descriptorpb.EnumValueDescriptorProto{
							{
								Name:   new("ZERO"),
								Number: proto.Int32(0),
							},
							{
								Name:   new("ONE"),
								Number: proto.Int32(1),
							},
						},
					},
				},
				BufExtension: imagev1.ImageFileExtension_builder{
					IsImport:            new(true),
					IsSyntaxUnspecified: new(false),
					ModuleInfo: imagev1.ModuleInfo_builder{
						Name: imagev1.ModuleName_builder{
							Remote:     new("buf.build"),
							Owner:      new("foo"),
							Repository: new("baz"),
						}.Build(),
						Commit: new("0123456789abcdef0123456789abcdef"),
					}.Build(),
				}.Build(),
			}.Build(),
			imagev1.ImageFile_builder{
				Syntax: new("proto2"),
				Name:   new("c.proto"),
				BufExtension: imagev1.ImageFileExtension_builder{
					IsImport:            new(true),
					IsSyntaxUnspecified: new(true),
					ModuleInfo: imagev1.ModuleInfo_builder{
						Name: imagev1.ModuleName_builder{
							Remote:     new("buf.build"),
							Owner:      new("foo"),
							Repository: new("baz"),
						}.Build(),
						Commit: new("0123456789abcdef0123456789abcdef"),
					}.Build(),
				}.Build(),
			}.Build(),
		},
	}.Build()

	image, err := NewImageForProto(protoImage)
	require.NoError(t, err)

	clone, err := CloneImage(image)
	require.NoError(t, err)

	// Test that they are equal by comparing their proto versions
	protoClone, err := ImageToProtoImage(clone)
	require.NoError(t, err)

	require.Empty(t, cmp.Diff(protoImage, protoClone, protocmp.Transform()))

	// Verify the pointer values are different
	for _, imageFile := range image.Files() {
		cloneFile := clone.GetFile(imageFile.Path())
		require.NotSame(t, imageFile.FileDescriptorProto(), cloneFile.FileDescriptorProto())
		unused := reflect.ValueOf(imageFile.UnusedDependencyIndexes()).Pointer()
		cloneUnused := reflect.ValueOf(cloneFile.UnusedDependencyIndexes()).Pointer()
		if unused != 0 || cloneUnused != 0 {
			// They can both be nil. But otherwise must not be equal since that
			// means the backing arrays are shared.
			require.NotEqual(t, unused, cloneUnused)
		}
	}
}

func TestMergeImages(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name   string
		images [][]mergeTestFile
		// expected is nil when an error is expected.
		expected    []mergeTestFile
		errContains string
	}
	testCases := []testCase{
		{
			name: "disjoint",
			images: [][]mergeTestFile{
				{{path: "a.proto"}, {path: "b.proto"}},
				{{path: "c.proto"}, {path: "d.proto"}},
			},
			expected: []mergeTestFile{{path: "a.proto"}, {path: "b.proto"}, {path: "c.proto"}, {path: "d.proto"}},
		},
		{
			name: "shared_import",
			images: [][]mergeTestFile{
				{{path: "dep.proto", isImport: true}, {path: "a.proto", dependencies: []string{"dep.proto"}}},
				{{path: "dep.proto", isImport: true}, {path: "b.proto", dependencies: []string{"dep.proto"}}},
			},
			expected: []mergeTestFile{
				{path: "dep.proto", isImport: true},
				{path: "a.proto", dependencies: []string{"dep.proto"}},
				{path: "b.proto", dependencies: []string{"dep.proto"}},
			},
		},
		{
			name: "import_then_target",
			images: [][]mergeTestFile{
				{{path: "dep.proto", isImport: true}, {path: "a.proto", dependencies: []string{"dep.proto"}}},
				{{path: "dep.proto"}},
			},
			expected: []mergeTestFile{
				{path: "dep.proto"},
				{path: "a.proto", dependencies: []string{"dep.proto"}},
			},
		},
		{
			name: "target_then_import",
			images: [][]mergeTestFile{
				{{path: "dep.proto"}},
				{{path: "dep.proto", isImport: true}, {path: "a.proto", dependencies: []string{"dep.proto"}}},
			},
			expected: []mergeTestFile{
				{path: "dep.proto"},
				{path: "a.proto", dependencies: []string{"dep.proto"}},
			},
		},
		{
			name: "dependency_first_seen_in_earlier_image",
			images: [][]mergeTestFile{
				{{path: "dep.proto"}},
				{
					{path: "other.proto", isImport: true},
					{path: "dep.proto", isImport: true},
					{path: "b.proto", dependencies: []string{"dep.proto", "other.proto"}},
				},
			},
			expected: []mergeTestFile{
				{path: "dep.proto"},
				{path: "other.proto", isImport: true},
				{path: "b.proto", dependencies: []string{"dep.proto", "other.proto"}},
			},
		},
		{
			name: "conflicting_descriptors",
			images: [][]mergeTestFile{
				{{path: "a.proto", pkg: "foo"}},
				{{path: "a.proto", pkg: "bar"}},
			},
			errContains: `file "a.proto" differs between images`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			images := make([]Image, len(testCase.images))
			for i, testFiles := range testCase.images {
				images[i] = newMergeTestImage(t, nil, uuid.Nil, testFiles...)
			}
			merged, err := MergeImages(images...)
			if testCase.errContains != "" {
				require.ErrorContains(t, err, testCase.errContains)
				return
			}
			require.NoError(t, err)
			expected := newMergeTestImage(t, nil, uuid.Nil, testCase.expected...)
			expectedProto, err := ImageToProtoImage(expected)
			require.NoError(t, err)
			mergedProto, err := ImageToProtoImage(merged)
			require.NoError(t, err)
			require.Empty(t, cmp.Diff(expectedProto, mergedProto, protocmp.Transform()))
		})
	}
	t.Run("single", func(t *testing.T) {
		t.Parallel()
		image := newMergeTestImage(t, nil, uuid.Nil, mergeTestFile{path: "a.proto", isImport: true})
		merged, err := MergeImages(image)
		require.NoError(t, err)
		require.Same(t, image, merged)
	})
	t.Run("none", func(t *testing.T) {
		t.Parallel()
		_, err := MergeImages()
		require.Error(t, err)
	})
	t.Run("same_file_different_modules", func(t *testing.T) {
		t.Parallel()
		fooFullName, err := bufparse.ParseFullName("buf.build/acme/foo")
		require.NoError(t, err)
		barFullName, err := bufparse.ParseFullName("buf.build/acme/bar")
		require.NoError(t, err)
		image, err := MergeImages(
			newMergeTestImage(t, fooFullName, uuid.Nil, mergeTestFile{path: "a.proto"}),
			newMergeTestImage(t, barFullName, mergeTestCommitID, mergeTestFile{path: "a.proto"}),
		)
		require.NoError(t, err)
		require.Len(t, image.Files(), 1)
		require.Equal(t, fooFullName.String(), image.Files()[0].FullName().String())
		require.Equal(t, uuid.Nil, image.Files()[0].CommitID())
	})
	t.Run("same_module_different_commits", func(t *testing.T) {
		t.Parallel()
		fooFullName, err := bufparse.ParseFullName("buf.build/acme/foo")
		require.NoError(t, err)
		barFullName, err := bufparse.ParseFullName("buf.build/acme/bar")
		require.NoError(t, err)
		// A local module without a commit and the same module as a
		// dependency at a commit, with shared and disjoint files.
		localImage := newMergeTestImage(
			t,
			fooFullName,
			uuid.Nil,
			mergeTestFile{path: "a.proto"},
			mergeTestFile{path: "b.proto"},
		)
		remoteFooImage := newMergeTestImage(
			t,
			fooFullName,
			mergeTestCommitID,
			mergeTestFile{path: "a.proto", isImport: true},
			mergeTestFile{path: "c.proto", isImport: true},
		)
		remoteBarImage := newMergeTestImage(
			t,
			barFullName,
			mergeTestCommitID,
			mergeTestFile{path: "d.proto", dependencies: []string{"a.proto", "c.proto"}},
		)
		remoteImage, err := NewImage(append(remoteFooImage.Files(), remoteBarImage.Files()...))
		require.NoError(t, err)
		image, err := MergeImages(localImage, remoteImage)
		require.NoError(t, err)
		var paths []string
		for _, imageFile := range image.Files() {
			paths = append(paths, imageFile.Path())
			if imageFile.FullName().String() == fooFullName.String() {
				require.Equal(t, uuid.Nil, imageFile.CommitID(), imageFile.Path())
			} else {
				require.Equal(t, mergeTestCommitID, imageFile.CommitID(), imageFile.Path())
			}
		}
		require.Equal(t, []string{"a.proto", "b.proto", "c.proto", "d.proto"}, paths)
	})
}

type mergeTestFile struct {
	path         string
	isImport     bool
	dependencies []string
	pkg          string
}

var mergeTestCommitID = uuid.MustParse("0123456789abcdef0123456789abcdef")

func newMergeTestImage(
	t *testing.T,
	moduleFullName bufparse.FullName,
	commitID uuid.UUID,
	testFiles ...mergeTestFile,
) Image {
	t.Helper()
	imageFiles := make([]ImageFile, len(testFiles))
	for i, testFile := range testFiles {
		fileDescriptorProto := &descriptorpb.FileDescriptorProto{
			Name:       new(testFile.path),
			Syntax:     new("proto3"),
			Dependency: testFile.dependencies,
		}
		if testFile.pkg != "" {
			fileDescriptorProto.Package = new(testFile.pkg)
		}
		imageFile, err := NewImageFile(
			fileDescriptorProto,
			moduleFullName,
			commitID,
			"",
			"",
			testFile.isImport,
			false,
			nil,
		)
		require.NoError(t, err)
		imageFiles[i] = imageFile
	}
	image, err := NewImage(imageFiles)
	require.NoError(t, err)
	return image
}
