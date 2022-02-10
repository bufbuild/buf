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

package bufimageutil

import (
	"bytes"
	"context"
	"io/ioutil"
	"sort"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoprint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/tools/txtar"
	"google.golang.org/protobuf/proto"
)

func TestImageFilteredByTypes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bucket, err := storagemem.NewReadBucket(map[string][]byte{
		"a.proto": []byte(`syntax = "proto2";
import "e.proto";
import "google/protobuf/descriptor.proto";
package pkg;
option (extendthatthing).foo = "no!";
message Baz { 
	message NestedBaz {
		optional string in_nested_baz = 1 [(extend_that_field) = "foo"];
	}
	optional Baz baz = 1; 
	optional NestedBaz nested_baz = 2; 
	optional EeNum fds = 3;
	extensions 4 to 5; 
}
enum EeNum {
	option (extend_that_enum) = "unset";
	EE_X = 0;
	EE_Y = 1 [(extend_that_enum_value) = "okk"];
	EE_Z = 2;
}
extend Baz { optional string extended_field = 5; }
`),
		"b.proto":  []byte(`syntax = "proto3";import "a.proto"; package pkg; message Bar { Baz baz = 1; }`),
		"c.proto":  []byte(`syntax = "proto3";import "dependency.proto"; import weak "b1.proto"; import public "b.proto"; package pkg; enum FooEnum{X=0;};message XYZ {Qux qux = 2;}; message Foo { Bar baz = 1; dependency.Dep d = 2; FooEnum foo_enum = 3; }`),
		"b1.proto": []byte(`syntax = "proto3";import "b.proto"; package pkg; message Qux { Bar what = 1; }`),
		"s.proto":  []byte(`syntax = "proto3";import "c.proto";import "a.proto"; package pkg; service Quux { rpc Do(Foo) returns (Baz); }`),
		"e.proto": []byte(`syntax = "proto3";
import "google/protobuf/descriptor.proto";

message UnusedFileOption {
	string foo = 1;
}

extend google.protobuf.FieldOptions {
	optional string extend_that_field = 9999998;
}
extend google.protobuf.FileOptions {
	optional string extend_that_file = 9999999;
	optional UnusedFileOption extendthatthing = 999991;
}
extend google.protobuf.EnumOptions {
	optional string extend_that_enum = 9999999;
}
extend google.protobuf.EnumValueOptions {
	optional string extend_that_enum_value = 9999999;
}
// extend google.protobuf.FileOptions {
// }
`),
	})
	require.NoError(t, err)
	moduleIdentity, err := bufmoduleref.NewModuleIdentity("buf.build", "repo", "main")
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForBucket(ctx, bucket, bufmodule.ModuleWithModuleIdentity(moduleIdentity))
	require.NoError(t, err)
	bucketDep, err := storagemem.NewReadBucket(map[string][]byte{
		"dependency.proto": []byte(`syntax = "proto3";package dependency; message Dep{}`),
	})
	require.NoError(t, err)
	moduleIdentityDep, err := bufmoduleref.NewModuleIdentity("buf.build", "repo", "dep")
	require.NoError(t, err)
	moduleDep, err := bufmodule.NewModuleForBucket(ctx, bucketDep, bufmodule.ModuleWithModuleIdentity(moduleIdentityDep))
	require.NoError(t, err)
	image, analysis, err := bufimagebuild.NewBuilder(zaptest.NewLogger(t)).Build(
		ctx,
		bufmodule.NewModuleFileSet(module, []bufmodule.Module{moduleDep}),
		bufimagebuild.WithExcludeSourceCodeInfo(),
	)
	require.NoError(t, err)
	require.Empty(t, analysis)

	// ImageFilteredByTypes changes the backing FileDescriptorSet, so we
	// need to make a copy of the original one to compare to.
	protoImage := bufimage.ImageToProtoImage(image)
	protoImageCopy := proto.Clone(protoImage).(*imagev1.Image)
	originalImage, err := bufimage.NewImageForProto(protoImageCopy)
	require.NoError(t, err)

	_, err = ImageFilteredByTypes(image, []string{"dependency.Dep"})
	require.Error(t, err)
	filteredImage, err := ImageFilteredByTypes(image, []string{"pkg.Baz.NestedBaz"})
	require.NoError(t, err)
	assert.NotNil(t, image)

	// This isnt really supposed to be equal, but prints good enough
	// debugging to see what gets filtered.

	assert.NoError(t, ioutil.WriteFile("testdata/in.txtar", txtarForImage(t, originalImage), 0644))
	assert.NoError(t, ioutil.WriteFile("testdata/out.txtar", txtarForImage(t, filteredImage), 0644))
	t.Error("now diff")
}

func txtarForImage(t *testing.T, image bufimage.Image) []byte {
	fds := bufimage.ImageToFileDescriptorSet(image)
	reflectFDS, err := desc.CreateFileDescriptorsFromSet(fds)
	if err != nil {
		b, err := proto.Marshal(fds)
		require.NoError(t, err)
		err = ioutil.WriteFile("testdata/malformed_image.bin", b, 0644)
		require.NoError(t, err)
	}
	require.NoError(t, err)
	archive := &txtar.Archive{}
	printer := protoprint.Printer{
		SortElements: true,
	}
	for fname, d := range reflectFDS {
		fileBuilder := &bytes.Buffer{}
		require.NoError(t, printer.PrintProtoFile(d, fileBuilder), "expected no error while printing %q", fname)
		archive.Files = append(
			archive.Files,
			txtar.File{
				Name: fname,
				Data: fileBuilder.Bytes(),
			},
		)
	}
	sort.SliceStable(archive.Files, func(i, j int) bool {
		return archive.Files[i].Name < archive.Files[j].Name
	})
	return txtar.Format(archive)
}
