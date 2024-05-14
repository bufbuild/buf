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

package git

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetSourceControlURL(t *testing.T) {
	t.Parallel()
	t.Run("ssh, bitbucket", func(t *testing.T) {
		t.Parallel()
		hostname, repositoryPath := parseRawRemoteURL("ssh://user@bitbucket.org:1234/user/repo.git")
		require.Equal(t, "bitbucket.org", hostname)
		require.Equal(t, "/user/repo", repositoryPath)
	})
	t.Run("ssh, github", func(t *testing.T) {
		t.Parallel()
		hostname, repositoryPath := parseRawRemoteURL("ssh://git@github.com/user/repo.git")
		require.Equal(t, "github.com", hostname)
		require.Equal(t, "/user/repo", repositoryPath)
	})
	t.Run("scp-like ssh, github", func(t *testing.T) {
		t.Parallel()
		hostname, repositoryPath := parseRawRemoteURL("git@github.com:user/repo.git")
		require.Equal(t, "github.com", hostname)
		require.Equal(t, "/user/repo", repositoryPath)
	})
	t.Run("ssh, gitlab", func(t *testing.T) {
		t.Parallel()
		hostname, repositoryPath := parseRawRemoteURL("ssh://user@gitlab.mycompany.com:1234/user/repo.git")
		require.Equal(t, "gitlab.mycompany.com", hostname)
		require.Equal(t, "/user/repo", repositoryPath)
	})
	t.Run("scp-like ssh, gitlab", func(t *testing.T) {
		t.Parallel()
		hostname, repositoryPath := parseRawRemoteURL("git@gitlab.com:user/repo.git")
		require.Equal(t, "gitlab.com", hostname)
		require.Equal(t, "/user/repo", repositoryPath)
	})
	t.Run("https, bitbucket", func(t *testing.T) {
		t.Parallel()
		hostname, repositoryPath := parseRawRemoteURL("https://bitbucket.mycompany.com/user/repo.git")
		require.Equal(t, "bitbucket.mycompany.com", hostname)
		require.Equal(t, "/user/repo", repositoryPath)
	})
	t.Run("https, github", func(t *testing.T) {
		t.Parallel()
		hostname, repositoryPath := parseRawRemoteURL("https://github.mycompany.com:4321/user/repo.git")
		require.Equal(t, "github.mycompany.com", hostname)
		require.Equal(t, "/user/repo", repositoryPath)
	})
	t.Run("https, gitlab", func(t *testing.T) {
		t.Parallel()
		hostname, repositoryPath := parseRawRemoteURL("https://gitlab.com/user/repo.git")
		require.Equal(t, "gitlab.com", hostname)
		require.Equal(t, "/user/repo", repositoryPath)
	})
}
