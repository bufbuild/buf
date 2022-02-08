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
	"context"
	"os"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoprint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
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
		"c.proto":  []byte(`syntax = "proto3";import "b.proto";import "b1.proto"; package pkg; enum FooEnum{X=0;};message Foo { Bar baz = 1; Qux qux = 2; FooEnum foo_enum = 3; }`),
		"b1.proto": []byte(`syntax = "proto3";import "b.proto"; package pkg; message Qux { string what = 1; }`),
		"s.proto":  []byte(`syntax = "proto3";import "c.proto";import "a.proto"; package pkg; service Quux { rpc Do(Foo) returns (Baz); }`),
	})
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForBucket(ctx, bucket)
	require.NoError(t, err)
	image, analysis, err := bufimagebuild.NewBuilder(zaptest.NewLogger(t)).Build(
		ctx,
		bufmodule.NewModuleFileSet(module, nil),
	)
	require.NoError(t, err)
	require.Empty(t, analysis)

	filteredImage, err := ImageFilteredByTypes(image, []string{"pkg.Quux"})
	require.NoError(t, err)
	assert.NotNil(t, image)
	printer := protoprint.Printer{}
	fds := bufimage.ImageToFileDescriptorSet(filteredImage)
	reflectFDS, err := desc.CreateFileDescriptorsFromSet(fds)
	require.NoError(t, err)

	// todo: protoprint the in and the out image then diff
	for fname, d := range reflectFDS {
		os.Stderr.WriteString("----" + fname + "----\n")
		assert.NoError(t, printer.PrintProtoFile(d, os.Stderr))
	}
	t.Error("na")
}
