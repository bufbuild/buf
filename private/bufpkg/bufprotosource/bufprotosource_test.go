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

package bufprotosource

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/stretchr/testify/require"
)

func TestNewFiles(t *testing.T) {
	t.Parallel()
	moduleSet, err := bufmoduletesting.NewModuleSetForDirPath("testdata/nested")
	require.NoError(t, err)
	image, err := bufimage.BuildImage(
		context.Background(),
		slogtestext.NewLogger(t),
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
	)
	require.NoError(t, err)
	files, err := NewFiles(context.Background(), image.Files(), image.Resolver())
	require.NoError(t, err)
	require.Len(t, files, 1)
	file := files[0]
	for _, message := range file.Messages() {
		testMessageName(
			t,
			map[string]string{
				"A":         "A",
				"B":         "A.B",
				"C":         "A.B.C",
				"D":         "A.B.C.D",
				"E":         "A.B.C.D.E",
				"F":         "A.B.C.D.E.F",
				"Message10": "A.B.C.D.E.F.Message10",
				"Message11": "A.B.C.D.E.F.Message10.Message11",
				"Message20": "A.B.C.D.E.F.Message20",
				"Message21": "A.B.C.D.E.F.Message20.Message21",
			},
			message,
		)
	}
}

func testMessageName(t *testing.T, nameToExpectedNestedName map[string]string, message Message) {
	expectedName, ok := nameToExpectedNestedName[message.Name()]
	require.True(t, ok)
	require.Equal(t, expectedName, message.NestedName())
	for _, nestedMessage := range message.Messages() {
		testMessageName(t, nameToExpectedNestedName, nestedMessage)
		_, err := message.AsDescriptor()
		require.NoError(t, err)
	}
}
