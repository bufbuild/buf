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

package bufimagetesting

import (
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImageWithOnlyPathsPreservesIsImportForCrossModuleImports is a regression
// test for a bug in ImageWithOnlyPathsAllowNotExist (and ImageWithOnlyPaths)
// where a file whose BufImageExtension.is_import is already true in the input
// Image gets reclassified as is_import=false purely because its path happens
// to match one of the path filters.
//
// This is observable when a pre-built Image is passed through a paths filter
// (for example, the binary_image: -> controller.filterImage path, which calls
// ImageWithOnlyPathsAllowNotExist when imageCameFromAWorkspace is false): a
// cross-module dependency that shares a path prefix with the filter is
// correctly flagged is_import=true in the input, but that flag is clobbered
// by addFileWithImports in private/bufpkg/bufimage/util.go, which rebuilds
// is_import from nonImportPaths instead of preserving the input flag.
//
// The workspace input path avoids this because filterImage is called with
// imageCameFromAWorkspace=true and skips the post-hoc path filter entirely.
func TestImageWithOnlyPathsPreservesIsImportForCrossModuleImports(t *testing.T) {
	t.Parallel()

	// A declared target under the shared prefix, importing a file that also
	// lives under the shared prefix but belongs to a different source.
	protoTargetFile := NewProtoImageFile(
		t,
		"shared/prefix/target.proto",
		"shared/prefix/external_dep.proto",
	)

	// The imported file: path happens to share the "shared/prefix/" prefix
	// with the target, but it was flagged as an import upstream and must
	// stay one after filtering.
	protoImportedFile := NewProtoImageFileIsImport(
		t,
		"shared/prefix/external_dep.proto",
	)

	image, err := bufimage.NewImage([]bufimage.ImageFile{
		NewImageFile(t, protoTargetFile, nil, uuid.Nil,
			"local/shared/prefix/target.proto",
			"local/shared/prefix/target.proto",
			false, // is_import: false — declared target
			false, nil,
		),
		NewImageFile(t, protoImportedFile, nil, uuid.Nil,
			"external/shared/prefix/external_dep.proto",
			"external/shared/prefix/external_dep.proto",
			true, // is_import: true — originated from outside the local source
			false, nil,
		),
	})
	require.NoError(t, err)

	// Preconditions: is_import flags match the scenario above.
	require.False(t, image.GetFile("shared/prefix/target.proto").IsImport())
	require.True(t, image.GetFile("shared/prefix/external_dep.proto").IsImport(),
		"precondition: imported file must start flagged is_import=true",
	)

	// Apply the same filter that `paths: [shared/prefix]` in a buf.gen.yaml
	// produces for a non-workspace input (controller.filterImage ->
	// ImageWithOnlyPathsAllowNotExist).
	filtered, err := bufimage.ImageWithOnlyPathsAllowNotExist(
		image,
		[]string{"shared/prefix"},
		nil,
	)
	require.NoError(t, err)

	// Both files remain present — the filter matches the shared prefix.
	require.NotNil(t, filtered.GetFile("shared/prefix/target.proto"))
	require.NotNil(t, filtered.GetFile("shared/prefix/external_dep.proto"))

	// The declared target stays a target.
	assert.False(t,
		filtered.GetFile("shared/prefix/target.proto").IsImport(),
		"declared target must remain is_import=false after path filter",
	)

	// The bug: the imported file is promoted to is_import=false because its
	// path matches the filter. This assertion currently fails on main and
	// on released versions. It should pass: a file that was already an
	// import must remain an import regardless of whether its path happens
	// to match one of the filter entries.
	assert.True(t,
		filtered.GetFile("shared/prefix/external_dep.proto").IsImport(),
		"file originally flagged is_import=true must NOT be promoted to "+
			"is_import=false when its path matches a filter entry; see "+
			"addFileWithImports in private/bufpkg/bufimage/util.go, which "+
			"rebuilds is_import from nonImportPaths instead of preserving "+
			"the existing flag",
	)
}

// TestImageWithOnlyPathsExactFileMatchPreservesIsImport exercises the
// exact-file-match branch of imageWithOnlyPaths (util.go: the branch where
// image.GetFile(fileOrDirPath) resolves to a specific ImageFile). A filter
// entry that exactly names a file already flagged is_import=true must not
// promote it to a target. The file may still appear in the output as an
// import if an actual target transitively depends on it.
func TestImageWithOnlyPathsExactFileMatchPreservesIsImport(t *testing.T) {
	t.Parallel()

	protoTarget := NewProtoImageFile(t, "a/target.proto", "shared/dep.proto")
	protoDep := NewProtoImageFileIsImport(t, "shared/dep.proto")

	image, err := bufimage.NewImage([]bufimage.ImageFile{
		NewImageFile(t, protoTarget, nil, uuid.Nil,
			"a/target.proto", "a/target.proto",
			false, false, nil,
		),
		NewImageFile(t, protoDep, nil, uuid.Nil,
			"shared/dep.proto", "shared/dep.proto",
			true, false, nil,
		),
	})
	require.NoError(t, err)

	// Both paths are exact .proto matches — no prefix expansion. This drives
	// the Loop 2 branch of imageWithOnlyPaths on both entries.
	filtered, err := bufimage.ImageWithOnlyPaths(
		image,
		[]string{"a/target.proto", "shared/dep.proto"},
		nil,
	)
	require.NoError(t, err)

	require.NotNil(t, filtered.GetFile("a/target.proto"))
	require.NotNil(t, filtered.GetFile("shared/dep.proto"),
		"import should still be present — pulled in transitively from a/target.proto",
	)
	assert.False(t,
		filtered.GetFile("a/target.proto").IsImport(),
		"declared target must remain is_import=false",
	)
	assert.True(t,
		filtered.GetFile("shared/dep.proto").IsImport(),
		"exact-path filter entry must not promote a pre-existing import to a target",
	)
}

// TestImageWithOnlyPathsDropsUnreferencedImportMatchingFilter pins down the
// semantics of the fix: a pre-existing import whose path matches a filter
// entry but which no non-import target transitively depends on is dropped
// from the filtered image, not carried over as an orphan. This mirrors the
// long-standing exclude-only branch (util.go:58) which also skips imports
// when collecting target candidates — a path filter selects targets, not
// arbitrary files to retain.
func TestImageWithOnlyPathsDropsUnreferencedImportMatchingFilter(t *testing.T) {
	t.Parallel()

	// a/target.proto imports nothing; shared/orphan.proto is an import that
	// nothing in the image depends on.
	protoTarget := NewProtoImageFile(t, "a/target.proto")
	protoOrphan := NewProtoImageFileIsImport(t, "shared/orphan.proto")

	image, err := bufimage.NewImage([]bufimage.ImageFile{
		NewImageFile(t, protoTarget, nil, uuid.Nil,
			"a/target.proto", "a/target.proto",
			false, false, nil,
		),
		NewImageFile(t, protoOrphan, nil, uuid.Nil,
			"shared/orphan.proto", "shared/orphan.proto",
			true, false, nil,
		),
	})
	require.NoError(t, err)

	filtered, err := bufimage.ImageWithOnlyPathsAllowNotExist(
		image,
		[]string{"a", "shared"},
		nil,
	)
	require.NoError(t, err)

	require.NotNil(t, filtered.GetFile("a/target.proto"))
	assert.Nil(t,
		filtered.GetFile("shared/orphan.proto"),
		"a pre-existing import whose path matches the filter but which no "+
			"target transitively depends on is not a target and should not "+
			"appear in the output",
	)
}

// TestImageWithOnlyPathsExcludeOnlyPreservesIsImport locks in the behavior
// of the exclude-only branch (Loop 1 in imageWithOnlyPaths, util.go:56-73)
// which has always guarded against treating pre-existing imports as
// targets. It is included here to keep the three target-collection
// branches — exclude-only, exact file match, directory/prefix — symmetric
// under the same is_import invariant.
func TestImageWithOnlyPathsExcludeOnlyPreservesIsImport(t *testing.T) {
	t.Parallel()

	protoTarget := NewProtoImageFile(t, "a/target.proto", "shared/dep.proto")
	protoOther := NewProtoImageFile(t, "a/other.proto")
	protoDep := NewProtoImageFileIsImport(t, "shared/dep.proto")

	image, err := bufimage.NewImage([]bufimage.ImageFile{
		NewImageFile(t, protoTarget, nil, uuid.Nil,
			"a/target.proto", "a/target.proto",
			false, false, nil,
		),
		NewImageFile(t, protoOther, nil, uuid.Nil,
			"a/other.proto", "a/other.proto",
			false, false, nil,
		),
		NewImageFile(t, protoDep, nil, uuid.Nil,
			"shared/dep.proto", "shared/dep.proto",
			true, false, nil,
		),
	})
	require.NoError(t, err)

	// Empty paths + non-empty excludes routes to Loop 1.
	filtered, err := bufimage.ImageWithOnlyPathsAllowNotExist(
		image,
		nil,
		[]string{"a/other.proto"},
	)
	require.NoError(t, err)

	require.NotNil(t, filtered.GetFile("a/target.proto"))
	require.NotNil(t, filtered.GetFile("shared/dep.proto"),
		"import pulled in transitively from a/target.proto must remain in the image",
	)
	assert.Nil(t,
		filtered.GetFile("a/other.proto"),
		"excluded file must not be in the filtered image",
	)
	assert.False(t,
		filtered.GetFile("a/target.proto").IsImport(),
		"declared target must remain is_import=false",
	)
	assert.True(t,
		filtered.GetFile("shared/dep.proto").IsImport(),
		"pre-existing import must remain is_import=true in exclude-only mode",
	)
}
