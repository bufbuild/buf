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
package pkg;
message Baz { 
	message NestedBaz {
		optional Baz in_nested_baz = 1;
	}
	optional Baz baz = 1; 
	optional NestedBaz nested_baz = 2; 
	extensions 3 to 4; 
} 
extend Baz { optional string extended_field = 3; }
`),
		"b.proto":  []byte(`syntax = "proto3";import "a.proto"; package pkg; message Bar { Baz baz = 1; }`),
		"c.proto":  []byte(`syntax = "proto3";import weak "dependency.proto"; import weak "b1.proto";import "b.proto"; package pkg; enum FooEnum{X=0;};message XYZ {Qux qux = 2;}; message Foo { Bar baz = 1; dependency.Dep d = 2; FooEnum foo_enum = 3; }`),
		"b1.proto": []byte(`syntax = "proto3";import "b.proto"; package pkg; message Qux { Bar what = 1; }`),
		"s.proto":  []byte(`syntax = "proto3";import "c.proto";import "a.proto"; package pkg; service Quux { rpc Do(Foo) returns (Baz); }`),
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
	filteredImage, err := ImageFilteredByTypes(image, []string{"pkg.Quux"})
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
	require.NoError(t, err)
	archive := &txtar.Archive{}
	printer := protoprint.Printer{}
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
