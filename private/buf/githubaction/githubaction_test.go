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

package githubaction

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const (
	testGithubToken      = "githubToken"
	testGithubRepository = "owner/repo"
	testGithubSHA        = "1111111111111111111111111111111111111111"
	testGithubParentSHA  = "2222222222222222222222222222222222222222"
	testGithubRefName    = "test-branch"
	testBufCommitName    = "33333333333333333333333333333333"
	testBufCommitName2   = "44444444444444444444444444444444"

	testUserAgent  = "userAgent"
	testTrack      = "test-track"
	testTrackID    = "test-track-id"
	testBufVersion = "1.0.0-rc1"

	testBufToken = "bufToken"
	testOwner    = "buf-owner"
	testRepo     = "buf-repo"
	testRemote   = "buf.example.com"
)

func TestPush(t *testing.T) {
	ctx := context.Background()
	moduleIdentity, err := bufmoduleref.NewModuleIdentity(testRemote, testOwner, testRepo)
	require.NoError(t, err)
	protoModule := bufmoduletesting.TestDataProto
	module, err := bufmodule.NewModuleForProto(ctx, protoModule)
	require.NoError(t, err)

	t.Run("behind github branch", func(t *testing.T) {
		ghServer, apiProvider := setupTest(t)
		env := defaultTestEnvironment(ghServer)

		ghServer.handlers["/repos/owner/repo/compare/test-branch...1111111111111111111111111111111111111111"] =
			func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, fmt.Sprintf("Bearer %s", testGithubToken), r.Header.Get("Authorization"))
				assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
					"status": compareCommitsStatusBehind,
				}))
			}

		err = Push(
			context.WithValue(ctx, oauth2.HTTPClient, ghServer.server.Client()),
			app.NewEnvContainer(env),
			apiProvider,
			moduleIdentity,
			module,
			protoModule,
			testBufVersion,
		)
		require.EqualError(t, err, "github ref test-branch is behind 1111111111111111111111111111111111111111")
	})

	t.Run("track doesn't exist", func(t *testing.T) {
		ghServer, apiProvider := setupTest(t)
		env := defaultTestEnvironment(ghServer)

		ghServer.handlers["/repos/owner/repo/compare/test-branch...1111111111111111111111111111111111111111"] =
			func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, fmt.Sprintf("Bearer %s", testGithubToken), r.Header.Get("Authorization"))
				assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
					"status": compareCommitsStatusIdentical,
				}))
			}

		apiProvider.referenceService.getReferenceByName =
			func(ctx context.Context, name string, owner string, repositoryName string) (*registryv1alpha1.Reference, error) {
				assert.Equal(t, testTrack, name)
				assert.Equal(t, testOwner, owner)
				assert.Equal(t, testRepo, repositoryName)
				return nil, rpc.NewNotFoundError("not found")
			}

		apiProvider.pushService.push =
			func(ctx context.Context, owner string, repository string, branch string, mod *modulev1alpha1.Module, tags []string, tracks []string) (*registryv1alpha1.LocalModulePin, error) {
				assert.Equal(t, testOwner, owner)
				assert.Equal(t, testRepo, repository)
				assert.Equal(t, "", branch)
				assert.Equal(t, protoModule, mod)
				assert.Empty(t, tags)
				assert.Equal(t, []string{testTrack}, tracks)
				return &registryv1alpha1.LocalModulePin{Commit: testBufCommitName2}, nil
			}

		ghServer.handlers["/repos/owner/repo/statuses/1111111111111111111111111111111111111111"] =
			func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, fmt.Sprintf("Bearer %s", testGithubToken), r.Header.Get("Authorization"))
				assert.Equal(t, r.Method, http.MethodPost)
			}

		err = Push(
			context.WithValue(ctx, oauth2.HTTPClient, ghServer.server.Client()),
			app.NewEnvContainer(env),
			apiProvider,
			moduleIdentity,
			module,
			protoModule,
			testBufVersion,
		)
		require.NoError(t, err)
	})

	t.Run("empty track", func(t *testing.T) {
		ghServer, apiProvider := setupTest(t)
		env := defaultTestEnvironment(ghServer)

		ghServer.handlers["/repos/owner/repo/compare/test-branch...1111111111111111111111111111111111111111"] =
			func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, fmt.Sprintf("Bearer %s", testGithubToken), r.Header.Get("Authorization"))
				assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
					"status": compareCommitsStatusIdentical,
				}))
			}

		apiProvider.referenceService.getReferenceByName =
			func(ctx context.Context, name string, owner string, repositoryName string) (*registryv1alpha1.Reference, error) {
				assert.Equal(t, testTrack, name)
				assert.Equal(t, testOwner, owner)
				assert.Equal(t, testRepo, repositoryName)
				return &registryv1alpha1.Reference{
					Reference: &registryv1alpha1.Reference_Track{Track: &registryv1alpha1.RepositoryTrack{Id: testTrackID}},
				}, nil
			}

		apiProvider.repositoryCommitService.getRepositoryCommitByReference =
			func(ctx context.Context, repositoryOwner string, repositoryName string, reference string) (*registryv1alpha1.RepositoryCommit, error) {
				assert.Equal(t, testOwner, repositoryOwner)
				assert.Equal(t, testRepo, repositoryName)
				switch reference {
				case testGithubRefName:
					return nil, rpc.NewNotFoundError("not found")
				case testGithubSHA:
					return nil, rpc.NewNotFoundError("not found")
				}
				t.Errorf("unexpected reference: %s", reference)
				return nil, fmt.Errorf("unexpected reference: %s", reference)
			}

		apiProvider.pushService.push =
			func(ctx context.Context, owner string, repository string, branch string, mod *modulev1alpha1.Module, tags []string, tracks []string) (*registryv1alpha1.LocalModulePin, error) {
				assert.Equal(t, testOwner, owner)
				assert.Equal(t, testRepo, repository)
				assert.Equal(t, "", branch)
				assert.Equal(t, protoModule, mod)
				assert.Empty(t, tags)
				assert.Equal(t, []string{testTrack}, tracks)
				return &registryv1alpha1.LocalModulePin{Commit: testBufCommitName2}, nil
			}

		ghServer.handlers["/repos/owner/repo/statuses/1111111111111111111111111111111111111111"] =
			func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, fmt.Sprintf("Bearer %s", testGithubToken), r.Header.Get("Authorization"))
				assert.Equal(t, r.Method, http.MethodPost)
			}

		err = Push(
			context.WithValue(ctx, oauth2.HTTPClient, ghServer.server.Client()),
			app.NewEnvContainer(env),
			apiProvider,
			moduleIdentity,
			module,
			protoModule,
			testBufVersion,
		)
		require.NoError(t, err)
	})

	t.Run("ahead of track", func(t *testing.T) {
		ghServer, apiProvider := setupTest(t)
		env := defaultTestEnvironment(ghServer)

		ghServer.handlers["/repos/owner/repo/compare/test-branch...1111111111111111111111111111111111111111"] =
			func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, fmt.Sprintf("Bearer %s", testGithubToken), r.Header.Get("Authorization"))
				assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
					"status": compareCommitsStatusIdentical,
				}))
			}

		apiProvider.referenceService.getReferenceByName =
			func(ctx context.Context, name string, owner string, repositoryName string) (*registryv1alpha1.Reference, error) {
				assert.Equal(t, testTrack, name)
				assert.Equal(t, testOwner, owner)
				assert.Equal(t, testRepo, repositoryName)
				return &registryv1alpha1.Reference{
					Reference: &registryv1alpha1.Reference_Track{Track: &registryv1alpha1.RepositoryTrack{Id: testTrackID}},
				}, nil
			}

		apiProvider.repositoryCommitService.getRepositoryCommitByReference =
			func(ctx context.Context, repositoryOwner string, repositoryName string, reference string) (*registryv1alpha1.RepositoryCommit, error) {
				assert.Equal(t, testOwner, repositoryOwner)
				assert.Equal(t, testRepo, repositoryName)
				switch reference {
				case testGithubRefName:
					return &registryv1alpha1.RepositoryCommit{
						Name:   testBufCommitName,
						Digest: "b1-fake-digest",
						Tags:   []*registryv1alpha1.RepositoryTag{{Name: testGithubParentSHA}},
					}, nil
				case testGithubSHA:
					return nil, rpc.NewNotFoundError("not found")
				}
				t.Errorf("unexpected reference: %s", reference)
				return nil, fmt.Errorf("unexpected reference: %s", reference)
			}

		apiProvider.pushService.push =
			func(ctx context.Context, owner string, repository string, branch string, mod *modulev1alpha1.Module, tags []string, tracks []string) (*registryv1alpha1.LocalModulePin, error) {
				assert.Equal(t, testOwner, owner)
				assert.Equal(t, testRepo, repository)
				assert.Equal(t, "", branch)
				assert.Equal(t, protoModule, mod)
				assert.Empty(t, tags)
				assert.Equal(t, []string{testTrack}, tracks)
				return &registryv1alpha1.LocalModulePin{Commit: testBufCommitName2}, nil
			}

		ghServer.handlers["/repos/owner/repo/compare/2222222222222222222222222222222222222222...1111111111111111111111111111111111111111"] =
			func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, fmt.Sprintf("Bearer %s", testGithubToken), r.Header.Get("Authorization"))
				assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
					"status": compareCommitsStatusAhead,
				}))
			}

		ghServer.handlers["/repos/owner/repo/statuses/1111111111111111111111111111111111111111"] =
			func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, fmt.Sprintf("Bearer %s", testGithubToken), r.Header.Get("Authorization"))
				assert.Equal(t, r.Method, http.MethodPost)
			}

		err = Push(
			context.WithValue(ctx, oauth2.HTTPClient, ghServer.server.Client()),
			app.NewEnvContainer(env),
			apiProvider,
			moduleIdentity,
			module,
			protoModule,
			testBufVersion,
		)
		require.NoError(t, err)
	})

	t.Run("missing environment vars", func(t *testing.T) {
		ghServer, _ := setupTest(t)
		testMissingEnvironmentVariable := func(varName string, expectMessage string) {
			env := defaultTestEnvironment(ghServer)
			delete(env, varName)
			err := Push(ctx, app.NewEnvContainer(env), nil, nil, nil, nil, "")
			require.EqualError(t, err, expectMessage)
		}

		// environment variables from action inputs
		testMissingEnvironmentVariable(bufTrackEnvKey, "inputs.track is required")
		testMissingEnvironmentVariable(tokenEnvKey, "inputs.buf_token is required")
		testMissingEnvironmentVariable(githubTokenEnvKey, "inputs.github_token is required")

		// environment variables that provided by github actions itself
		testMissingEnvironmentVariable(githubShaEnvKey, "environment variable GITHUB_SHA is required")
		testMissingEnvironmentVariable(githubRepositoryEnvKey, "environment variable GITHUB_REPOSITORY is required")
		testMissingEnvironmentVariable(githubRefNameEnvKey, "environment variable GITHUB_REF_NAME is required")
		testMissingEnvironmentVariable(githubRefTypeEnvKey, "environment variable GITHUB_REF_TYPE is required")
		testMissingEnvironmentVariable(githubAPIURLEnvKey, "environment variable GITHUB_API_URL is required")
	})
}

func setupTest(t *testing.T) (*testServer, fakeRegistryProvider) {
	return newTestServer(t), fakeRegistryProvider{t: t}
}

func defaultTestEnvironment(ghServer *testServer) map[string]string {
	return map[string]string{
		githubShaEnvKey:        testGithubSHA,
		githubRepositoryEnvKey: testGithubRepository,
		githubRefNameEnvKey:    testGithubRefName,
		githubRefTypeEnvKey:    "branch",
		githubTokenEnvKey:      testGithubToken,
		githubAPIURLEnvKey:     ghServer.server.URL,
		tokenEnvKey:            testBufToken,
		bufTrackEnvKey:         testTrack,
	}
}
