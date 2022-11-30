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

package bufapimodule

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCommitServiceClient struct {
	t       *testing.T
	refResp *registryv1alpha1.GetRepositoryCommitByReferenceResponse
}

func (m *mockCommitServiceClient) ListRepositoryCommitsByBranch(
	_ context.Context,
	_ *connect.Request[registryv1alpha1.ListRepositoryCommitsByBranchRequest],
) (*connect.Response[registryv1alpha1.ListRepositoryCommitsByBranchResponse], error) {
	m.t.Error("unexpected call: ListRepositoryCommitsByBranch")
	return nil, errors.New("unexpected call")
}

func (m *mockCommitServiceClient) ListRepositoryCommitsByReference(
	_ context.Context,
	_ *connect.Request[registryv1alpha1.ListRepositoryCommitsByReferenceRequest],
) (*connect.Response[registryv1alpha1.ListRepositoryCommitsByReferenceResponse], error) {
	m.t.Error("unexpected call: ListRepositoryCommitsByReference")
	return nil, errors.New("unexpected call")
}

func (m *mockCommitServiceClient) GetRepositoryCommitByReference(
	_ context.Context,
	_ *connect.Request[registryv1alpha1.GetRepositoryCommitByReferenceRequest],
) (*connect.Response[registryv1alpha1.GetRepositoryCommitByReferenceResponse], error) {
	return connect.NewResponse(m.refResp), nil
}

func (m *mockCommitServiceClient) GetRepositoryCommitBySequenceId(
	_ context.Context,
	_ *connect.Request[registryv1alpha1.GetRepositoryCommitBySequenceIdRequest],
) (*connect.Response[registryv1alpha1.GetRepositoryCommitBySequenceIdResponse], error) {
	m.t.Error("unexpected call: GetRepositoryCommitBySequenceId")
	return nil, errors.New("unexpected call")
}

func (m *mockCommitServiceClient) ListRepositoryDraftCommits(
	_ context.Context,
	_ *connect.Request[registryv1alpha1.ListRepositoryDraftCommitsRequest],
) (*connect.Response[registryv1alpha1.ListRepositoryDraftCommitsResponse], error) {
	m.t.Error("unexpected call: ListRepositoryDraftCommits")
	return nil, errors.New("unexpected call")
}

func (m *mockCommitServiceClient) DeleteRepositoryDraftCommit(
	_ context.Context,
	_ *connect.Request[registryv1alpha1.DeleteRepositoryDraftCommitRequest],
) (*connect.Response[registryv1alpha1.DeleteRepositoryDraftCommitResponse], error) {
	m.t.Error("unexpected call: DeleteRepositoryDraftCommit")
	return nil, errors.New("unexpected call")
}

func TestGetModulePin(t *testing.T) {
	testGetModulePin(
		t,
		"nominal",
		&registryv1alpha1.GetRepositoryCommitByReferenceResponse{
			RepositoryCommit: &registryv1alpha1.RepositoryCommit{
				Id:     "commitid",
				Digest: "digest",
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
			assert.Equal(t, "commit", pin.Commit())
			assert.Equal(
				t,
				time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC),
				pin.CreateTime(),
			)
		}
	})
}
