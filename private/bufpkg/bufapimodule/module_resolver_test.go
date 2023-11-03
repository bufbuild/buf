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

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
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

type mockResolveServiceClient struct {
	registryv1alpha1connect.UnimplementedResolveServiceHandler

	t       *testing.T
	refResp *registryv1alpha1.GetModulePinsResponse
}

func (m *mockResolveServiceClient) GetModulePins(
	_ context.Context,
	_ *connect.Request[registryv1alpha1.GetModulePinsRequest],
) (*connect.Response[registryv1alpha1.GetModulePinsResponse], error) {
	return connect.NewResponse(m.refResp), nil
}

func TestGetModulePin(t *testing.T) {
	t.Parallel()
	nilDigest, err := bufcas.NewDigestForContent(bytes.NewBuffer(nil))
	require.NoError(t, err)
	testGetModulePin(
		t,
		"nominal",
		&registryv1alpha1.GetRepositoryCommitByReferenceResponse{
			RepositoryCommit: &registryv1alpha1.RepositoryCommit{
				Id:     "commitid",
				Digest: nilDigest.String(),
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

func TestGetModulePins(t *testing.T) {
	t.Parallel()
	testGetModulePins(
		t,
		"nominal",
		&registryv1alpha1.GetModulePinsResponse{
			ModulePins: []*modulev1alpha1.ModulePin{
				{
					Remote:         "remote",
					Owner:          "owner",
					Repository:     "repository",
					Commit:         "commit",
					ManifestDigest: "digest",
				},
			},
		},
		false,
	)
	testGetModulePins(
		t,
		"success, empty pins",
		&registryv1alpha1.GetModulePinsResponse{},
		false,
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
		commitServiceClientFactory := func(_ string) registryv1alpha1connect.RepositoryCommitServiceClient {
			return &mockCommitServiceClient{
				t:       t,
				refResp: resp,
			}
		}
		ctx := context.Background()
		mr := newModuleResolver(
			nil, // logger is unused
			commitServiceClientFactory,
			nil, // resolveServiceClientFactory is unused
		)
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
			assert.Equal(t, resp.RepositoryCommit.Name, pin.Commit())
			assert.Equal(t, resp.RepositoryCommit.ManifestDigest, pin.Digest())
		}
	})
}

func testGetModulePins(
	t *testing.T,
	desc string,
	resp *registryv1alpha1.GetModulePinsResponse,
	isError bool,
) {
	t.Helper()
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		resolveServiceClientFactory := func(_ string) registryv1alpha1connect.ResolveServiceClient {
			return &mockResolveServiceClient{
				t:       t,
				refResp: resp,
			}
		}
		ctx := context.Background()
		mr := newModuleResolver(
			nil, // logger is unused
			nil, // commitServiceClientFactory is unused
			resolveServiceClientFactory,
		)
		moduleReference, err := bufmoduleref.NewModuleReference(
			"remote",
			"owner",
			"repository",
			"reference",
		)
		require.NoError(t, err)
		pins, err := mr.GetModulePins(ctx, []bufmoduleref.ModuleReference{moduleReference}, nil)
		if isError {
			assert.Error(t, err)
		} else {
			assert.Len(t, pins, len(resp.ModulePins))
			for i, gotModulePin := range pins {
				expectedModulePin := resp.ModulePins[i]
				assert.Equal(t, gotModulePin.Remote(), expectedModulePin.Remote)
				assert.Equal(t, gotModulePin.Owner(), expectedModulePin.Owner)
				assert.Equal(t, gotModulePin.Repository(), expectedModulePin.Repository)
				assert.Equal(t, gotModulePin.Commit(), expectedModulePin.Commit)
				assert.Equal(t, gotModulePin.Digest(), expectedModulePin.ManifestDigest)
			}
		}
	})
}
