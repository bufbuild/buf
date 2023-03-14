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

package bufapimodule

import (
	"bytes"
	"context"
	"testing"
	"time"

	"buf.build/gen/go/bufbuild/buf/bufbuild/connect-go/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "buf.build/gen/go/bufbuild/buf/protocolbuffers/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCommitServiceClient struct {
	registryv1alpha1connect.UnimplementedRepositoryCommitServiceHandler

	t       *testing.T
	refResp *registryv1alpha1.GetRepositoryCommitByReferenceResponse
}

func (m *mockCommitServiceClient) GetRepositoryCommitByReference(
	_ context.Context,
	_ *connect.Request[registryv1alpha1.GetRepositoryCommitByReferenceRequest],
) (*connect.Response[registryv1alpha1.GetRepositoryCommitByReferenceResponse], error) {
	return connect.NewResponse(m.refResp), nil
}

func TestGetModulePin(t *testing.T) {
	digester, err := manifest.NewDigester(manifest.DigestTypeShake256)
	require.NoError(t, err)
	nullDigest, err := digester.Digest(&bytes.Buffer{})
	require.NoError(t, err)
	testGetModulePin(
		t,
		"nominal",
		&registryv1alpha1.GetRepositoryCommitByReferenceResponse{
			RepositoryCommit: &registryv1alpha1.RepositoryCommit{
				Id:     "commitid",
				Digest: nullDigest.String(),
				Name:   "commit",
				Branch: "unsupported-feature",
				Author: "John Doe",
			},
		},
		false,
	)
	testGetModulePin(
		t,
		"success, nil repository commit",
		&registryv1alpha1.GetRepositoryCommitByReferenceResponse{},
		true,
	)
}

func testGetModulePin(
	t *testing.T,
	desc string,
	resp *registryv1alpha1.GetRepositoryCommitByReferenceResponse,
	isError bool,
) {
	t.Helper()
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		clientFactory := func(_ string) registryv1alpha1connect.RepositoryCommitServiceClient {
			return &mockCommitServiceClient{
				t:       t,
				refResp: resp,
			}
		}
		ctx := context.Background()
		mr := newModuleResolver(nil, clientFactory) // logger is unused
		moduleReference, err := bufmoduleref.NewModuleReference(
			"remote",
			"owner",
			"repository",
			"reference",
		)
		require.NoError(t, err)
		pin, err := mr.GetModulePin(ctx, moduleReference)
		if isError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, "remote", pin.Remote())
			assert.Equal(t, "owner", pin.Owner())
			assert.Equal(t, "repository", pin.Repository())
			assert.Equal(t, "", pin.Branch())
			assert.Equal(t, resp.RepositoryCommit.Name, pin.Commit())
			assert.Equal(t, resp.RepositoryCommit.Digest, pin.Digest())
			assert.Equal(
				t,
				time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC),
				pin.CreateTime(),
			)
		}
	})
}
