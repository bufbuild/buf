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
	gitCommitSha := "007bdc4ddc7e4854b4bf3ff6c1f61eda"
	t.Run("ssh, bitbucket", func(t *testing.T) {
		t.Parallel()
		sourceControlURL, err := parseSourceControlURL("ssh://user@bitbucket.org:1234/user/repo.git", gitCommitSha)
		require.NoError(t, err)
		require.Equal(t, "https://bitbucket.org/user/repo/commits/007bdc4ddc7e4854b4bf3ff6c1f61eda", sourceControlURL)
	})
	t.Run("ssh, github", func(t *testing.T) {
		t.Parallel()
		sourceControlURL, err := parseSourceControlURL("ssh://git@github.com/user/repo.git", gitCommitSha)
		require.NoError(t, err)
		require.Equal(t, "https://github.com/user/repo/commit/007bdc4ddc7e4854b4bf3ff6c1f61eda", sourceControlURL)
	})
	t.Run("ssh, gitlab", func(t *testing.T) {
		t.Parallel()
		sourceControlURL, err := parseSourceControlURL("ssh://user@gitlab.mycompany.com:1234/user/repo.git", gitCommitSha)
		require.NoError(t, err)
		require.Equal(t, "https://gitlab.mycompany.com/user/repo/commit/007bdc4ddc7e4854b4bf3ff6c1f61eda", sourceControlURL)
	})
	t.Run("https, bitbucket", func(t *testing.T) {
		t.Parallel()
		sourceControlURL, err := parseSourceControlURL("https://bitbucket.mycompany.com/user/repo.git", gitCommitSha)
		require.NoError(t, err)
		require.Equal(t, "https://bitbucket.mycompany.com/user/repo/commits/007bdc4ddc7e4854b4bf3ff6c1f61eda", sourceControlURL)
	})
	t.Run("https, github", func(t *testing.T) {
		t.Parallel()
		sourceControlURL, err := parseSourceControlURL("https://github.mycompany.com:4321/user/repo.git", gitCommitSha)
		require.NoError(t, err)
		require.Equal(t, "https://github.mycompany.com/user/repo/commit/007bdc4ddc7e4854b4bf3ff6c1f61eda", sourceControlURL)
	})
	t.Run("https, gitlab", func(t *testing.T) {
		t.Parallel()
		sourceControlURL, err := parseSourceControlURL("https://gitlab.com/user/repo.git", gitCommitSha)
		require.NoError(t, err)
		require.Equal(t, "https://gitlab.com/user/repo/commit/007bdc4ddc7e4854b4bf3ff6c1f61eda", sourceControlURL)
	})
}
