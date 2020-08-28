// Copyright 2020 Buf Technologies, Inc.
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

package bufmodulebuild_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmoduletesting"
	modulev1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/module/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestModuleFileSetBuilderDeduplicates(t *testing.T) {
	moduleFileSetBuilder := bufmodulebuild.NewModuleFileSetBuilder(
		zap.NewNop(),
		&testModuleReader{
			alreadyRequested: make(map[string]struct{}),
		},
	)
	module, err := bufmodule.NewModuleForProto(context.Background(), &modulev1.Module{
		Files: []*modulev1.ModuleFile{
			{
				Path:    "file.proto",
				Content: []byte(`syntax="proto3";`),
			},
		},
		Dependencies: []*modulev1.ModuleName{
			{
				Server:     "buf.build",
				Owner:      "foo",
				Repository: "bar",
				Version:    "v1",
				Digest:     bufmoduletesting.TestDigest,
			},
			{
				Server:     "buf.build",
				Owner:      "google",
				Repository: "googleapis",
				Version:    "v1",
				Digest:     bufmoduletesting.TestDigest,
			},
		},
	})
	require.NoError(t, err)
	_, err = moduleFileSetBuilder.Build(context.Background(), module)
	require.NoError(t, err)
}

type testModuleReader struct {
	alreadyRequested map[string]struct{}
}

func (m *testModuleReader) GetModule(ctx context.Context, moduleName bufmodule.ModuleName) (bufmodule.Module, error) {
	if _, ok := m.alreadyRequested[moduleName.String()]; ok {
		// This is the test failure condition - we should only request each module once
		return nil, errors.New("module already requested")
	}
	m.alreadyRequested[moduleName.String()] = struct{}{}
	switch moduleName.String() {
	case fmt.Sprintf("buf.build/foo/bar/v1:%s", bufmoduletesting.TestDigest):
		return bufmodule.NewModuleForProto(context.Background(), &modulev1.Module{
			Files: []*modulev1.ModuleFile{
				{
					Path:    "file.proto",
					Content: []byte(`syntax="proto3";`),
				},
			},
			Dependencies: []*modulev1.ModuleName{
				// Will force second lookup of googleapis
				{
					Server:     "buf.build",
					Owner:      "google",
					Repository: "googleapis",
					Version:    "v1",
					Digest:     bufmoduletesting.TestDigest,
				},
			},
		})
	case fmt.Sprintf("buf.build/google/googleapis/v1:%s", bufmoduletesting.TestDigest):
		return bufmodule.NewModuleForProto(context.Background(), &modulev1.Module{
			Files: []*modulev1.ModuleFile{
				{
					Path:    "file.proto",
					Content: []byte(`syntax="proto3";`),
				},
			},
		})
	default:
		return nil, fmt.Errorf("unknown module: %s", moduleName.String())
	}

}
