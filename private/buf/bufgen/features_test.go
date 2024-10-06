// Copyright 2020-2024 Buf Technologies, Inc.
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
	"bytes"
	"log/slog"
	"math"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestComputeRequiredFeatures(t *testing.T) {
	t.Parallel()
	noRequiredFeatures := makeImageNoRequiredFeatures(t)
	requiresProto3Optional := makeImageRequiresProto3Optional(t)
	requiresEditions := makeImageRequiresEditions(t)
	requiresBoth := makeImageRequiresBoth(t)

	required := computeRequiredFeatures(noRequiredFeatures)
	assert.Empty(t, required.featureToFilenames)
	assert.Empty(t, required.editionToFilenames)

	required = computeRequiredFeatures(requiresProto3Optional)
	assert.Equal(t, map[pluginpb.CodeGeneratorResponse_Feature][]string{
		pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL: {"proto3_optional.proto"},
	}, required.featureToFilenames)
	assert.Empty(t, required.editionToFilenames)

	required = computeRequiredFeatures(requiresEditions)
	assert.Equal(t, map[pluginpb.CodeGeneratorResponse_Feature][]string{
		pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS: {"editions.proto"},
	}, required.featureToFilenames)
	assert.Equal(t, map[descriptorpb.Edition][]string{
		descriptorpb.Edition_EDITION_2023: {"editions.proto"},
	}, required.editionToFilenames)
	// Note that we can't really test a wider range here right now because
	// we don't support building an editions file for anything other than
	// edition 2023 right now.
	assert.Equal(t, descriptorpb.Edition_EDITION_2023, required.minEdition)
	assert.Equal(t, descriptorpb.Edition_EDITION_2023, required.maxEdition)

	required = computeRequiredFeatures(requiresBoth)
	assert.Equal(t, map[pluginpb.CodeGeneratorResponse_Feature][]string{
		pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL:   {"proto3_optional.proto"},
		pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS: {"editions.proto"},
	}, required.featureToFilenames)
	assert.Equal(t, map[descriptorpb.Edition][]string{
		descriptorpb.Edition_EDITION_2023: {"editions.proto"},
	}, required.editionToFilenames)
	assert.Equal(t, descriptorpb.Edition_EDITION_2023, required.minEdition)
	assert.Equal(t, descriptorpb.Edition_EDITION_2023, required.maxEdition)
}

func TestCheckRequiredFeatures(t *testing.T) {
	t.Parallel()
	noRequiredFeatures := makeImageNoRequiredFeatures(t)
	requiresProto3Optional := makeImageRequiresProto3Optional(t)
	requiresEditions := makeImageRequiresEditions(t)
	requiresBoth := makeImageRequiresBoth(t)

	supportsNoFeatures := &pluginpb.CodeGeneratorResponse{}
	supportsBoth := &pluginpb.CodeGeneratorResponse{
		SupportedFeatures: proto.Uint64(uint64(
			pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL |
				pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS,
		)),
		MinimumEdition: (*int32)(descriptorpb.Edition_EDITION_2023.Enum()),
		MaximumEdition: (*int32)(descriptorpb.Edition_EDITION_2024.Enum()),
	}
	supportsEditionsButOutOfRange := &pluginpb.CodeGeneratorResponse{
		SupportedFeatures: proto.Uint64(uint64(pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS)),
		MinimumEdition:    (*int32)(descriptorpb.Edition_EDITION_2024.Enum()),
		MaximumEdition:    (*int32)(descriptorpb.Edition_EDITION_MAX.Enum()),
	}
	supportsEditionsButNoRange := &pluginpb.CodeGeneratorResponse{
		SupportedFeatures: proto.Uint64(uint64(pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS)),
	}
	supportsEditionsButMalformedRange := &pluginpb.CodeGeneratorResponse{
		SupportedFeatures: proto.Uint64(uint64(pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS)),
		MinimumEdition:    (*int32)(descriptorpb.Edition_EDITION_2024.Enum()),
		MaximumEdition:    (*int32)(descriptorpb.Edition_EDITION_2023.Enum()),
	}

	// Successful cases
	testCheckRequiredFeatures(t, noRequiredFeatures, supportsNoFeatures, "", "")
	testCheckRequiredFeatures(t, requiresProto3Optional, supportsBoth, "", "")
	testCheckRequiredFeatures(t, requiresEditions, supportsBoth, "", "")
	testCheckRequiredFeatures(t, requiresBoth, supportsBoth, "", "")
	// These are successful because we intentionally don't validate the editions range unless the
	// file uses editions. This is to work around known issue in older versions of plugins in protoc
	// from when editions were still experimental. These plugins advertise supported for plugins but
	// don't correctly set the min and max edition fields.
	testCheckRequiredFeatures(t, noRequiredFeatures, supportsEditionsButNoRange, "", "")
	testCheckRequiredFeatures(t, noRequiredFeatures, supportsEditionsButMalformedRange, "", "")

	// Error cases
	testCheckRequiredFeatures(
		t,
		requiresProto3Optional,
		supportsNoFeatures,
		`plugin "test" does not support required features.
  Feature "proto3 optional" is required by 1 file(s):
    proto3_optional.proto`,
		"", // No error expected
	)
	testCheckRequiredFeatures(
		t,
		requiresEditions,
		supportsNoFeatures,
		"", // No stderr expected
		`plugin "test" does not support feature "supports editions" which is required by "editions.proto"`,
	)
	testCheckRequiredFeatures(
		t,
		requiresBoth,
		supportsNoFeatures,
		`plugin "test" does not support required features.
  Feature "proto3 optional" is required by 1 file(s):
    proto3_optional.proto`,
		`plugin "test" does not support feature "supports editions" which is required by "editions.proto"`,
	)
	testCheckRequiredFeatures(
		t,
		requiresEditions,
		supportsEditionsButOutOfRange,
		"", // No stderr expected
		`plugin "test" does not support edition "2023" which is required by "editions.proto"`,
	)
	testCheckRequiredFeatures(
		t,
		requiresEditions,
		supportsEditionsButNoRange,
		"", // No stderr expected
		`plugin "test" advertises that it supports editions but did not indicate a minimum supported edition`,
	)
	testCheckRequiredFeatures(
		t,
		requiresEditions,
		supportsEditionsButMalformedRange,
		"", // No stderr expected
		`plugin "test" indicates a maximum supported edition (EDITION_2023) that is less than its minimum supported edition (EDITION_2024)`,
	)
}

func testCheckRequiredFeatures(
	t *testing.T,
	image bufimage.Image,
	codeGenResponse *pluginpb.CodeGeneratorResponse,
	expectedStdErr string,
	expectedErr string,
) {
	t.Helper()
	required := computeRequiredFeatures(image)
	buffer := bytes.NewBuffer(nil)
	logger := slog.New(slog.NewTextHandler(buffer, &slog.HandlerOptions{}))
	err := checkRequiredFeatures(
		logger,
		required,
		[]*pluginpb.CodeGeneratorResponse{
			codeGenResponse,
			// this makes sure we handle multiple responses; this one never fails
			{
				SupportedFeatures: proto.Uint64(math.MaxUint), // all features enabled
				MinimumEdition:    proto.Int32(0),
				MaximumEdition:    proto.Int32(int32(descriptorpb.Edition_EDITION_MAX)),
			},
		},
		[]bufconfig.GeneratePluginConfig{
			newMockPluginConfig("test"),
			newMockPluginConfig("never_fails"),
		},
	)
	if expectedStdErr != "" {
		require.NotEmpty(t, buffer.String())
	} else {
		require.Empty(t, buffer.String())
	}
	if expectedErr != "" {
		require.ErrorContains(t, err, expectedErr)
	} else {
		require.NoError(t, err)
	}
}

func makeImageNoRequiredFeatures(t *testing.T) bufimage.Image {
	t.Helper()
	testFile, err := bufimage.NewImageFile(
		&descriptorpb.FileDescriptorProto{
			Name:   proto.String("test.proto"),
			Syntax: proto.String("proto3"),
			Dependency: []string{
				"imported_editions.proto",
				"imported_proto3_optional.proto",
			},
		},
		nil,
		uuid.UUID{},
		"test.proto",
		"test.proto",
		false,
		false,
		[]int32{0, 1},
	)
	require.NoError(t, err)
	// Imported files can use features since we're not doing code gen for them.
	importedFileEditions := makeImageFileRequiresEditions(t, "imported_editions.proto", true)
	importedFileProto3Optional := makeImageFileRequiresProto3Optional(t, "imported_proto3_optional.proto", true)
	image, err := bufimage.NewImage([]bufimage.ImageFile{importedFileEditions, importedFileProto3Optional, testFile})
	require.NoError(t, err)
	return image
}

func makeImageRequiresProto3Optional(t *testing.T) bufimage.Image {
	t.Helper()
	proto3OptionalFile := makeImageFileRequiresProto3Optional(t, "proto3_optional.proto", false)
	image, err := bufimage.NewImage([]bufimage.ImageFile{proto3OptionalFile})
	require.NoError(t, err)
	return image
}

func makeImageRequiresEditions(t *testing.T) bufimage.Image {
	t.Helper()
	editionsFile := makeImageFileRequiresEditions(t, "editions.proto", false)
	image, err := bufimage.NewImage([]bufimage.ImageFile{editionsFile})
	require.NoError(t, err)
	return image
}

func makeImageRequiresBoth(t *testing.T) bufimage.Image {
	t.Helper()
	editionsFile := makeImageFileRequiresEditions(t, "editions.proto", false)
	proto3OptionalFile := makeImageFileRequiresProto3Optional(t, "proto3_optional.proto", false)
	image, err := bufimage.NewImage([]bufimage.ImageFile{editionsFile, proto3OptionalFile})
	require.NoError(t, err)
	return image
}

func makeImageFileRequiresProto3Optional(t *testing.T, name string, isImport bool) bufimage.ImageFile {
	t.Helper()
	imageFile, err := bufimage.NewImageFile(
		&descriptorpb.FileDescriptorProto{
			Syntax: proto.String("proto3"),
			Name:   proto.String(name),
			MessageType: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("Foo"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:           proto.String("bar"),
							Label:          descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
							Type:           descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							JsonName:       proto.String("bar"),
							OneofIndex:     proto.Int32(0),
							Proto3Optional: proto.Bool(true),
						},
					},
					OneofDecl: []*descriptorpb.OneofDescriptorProto{
						{
							Name: proto.String("_bar"),
						},
					},
				},
			},
		},
		nil,
		uuid.UUID{},
		name,
		name,
		isImport,
		false,
		nil,
	)
	require.NoError(t, err)
	return imageFile
}

func makeImageFileRequiresEditions(t *testing.T, name string, isImport bool) bufimage.ImageFile {
	t.Helper()
	imageFile, err := bufimage.NewImageFile(
		&descriptorpb.FileDescriptorProto{
			Syntax:  proto.String("editions"),
			Edition: descriptorpb.Edition_EDITION_2023.Enum(),
			Name:    proto.String(name),
			MessageType: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("Bar"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:     proto.String("baz"),
							Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							JsonName: proto.String("baz"),
						},
					},
				},
			},
		},
		nil,
		uuid.UUID{},
		name,
		name,
		isImport,
		false,
		nil,
	)
	require.NoError(t, err)
	return imageFile
}

type mockPluginConfig struct {
	bufconfig.GeneratePluginConfig

	name string
}

func newMockPluginConfig(name string) bufconfig.GeneratePluginConfig {
	return mockPluginConfig{name: name}
}

func (p mockPluginConfig) Name() string {
	return p.name
}
