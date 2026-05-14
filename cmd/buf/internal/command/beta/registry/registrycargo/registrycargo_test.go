// Copyright 2020-2026 Buf Technologies, Inc.
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

package registrycargo

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appcmd/appcmdtesting"
	"buf.build/go/app/appext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostFromCargoRegistryURL(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "sparse_https_root",
			input:    "sparse+https://buf.build/gen/cargo/",
			expected: "buf.build",
		},
		{
			name:     "sparse_https_subpath",
			input:    "sparse+https://buf.build/gen/cargo/index/co/nn/connectrpc_eliza_community_neoeinstein-prost",
			expected: "buf.build",
		},
		{
			name:     "sparse_http",
			input:    "sparse+http://buf.build/gen/cargo/",
			expected: "buf.build",
		},
		{
			name:     "https_no_sparse_prefix",
			input:    "https://buf.build/",
			expected: "buf.build",
		},
		{
			name:     "sparse_https_with_port",
			input:    "sparse+https://buf.build:8443/gen/cargo/",
			expected: "buf.build",
		},
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
		{
			name:     "scheme_only",
			input:    "sparse+https://",
			expected: "",
		},
		{
			name:     "path_only_no_scheme",
			input:    "buf.build/gen/cargo/",
			expected: "",
		},
		{
			name:     "sparse_plus_path_only",
			input:    "sparse+buf.build/gen/cargo/",
			expected: "",
		},
		{
			name:     "uppercase_host_lowercased",
			input:    "sparse+https://BUF.BUILD/gen/cargo/",
			expected: "buf.build",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, hostFromCargoRegistryURL(tc.input))
		})
	}
}

func TestNormalizeHost(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "bare", input: "buf.build", expected: "buf.build"},
		{name: "with_port", input: "buf.build:8443", expected: "buf.build"},
		{name: "uppercase", input: "BUF.BUILD", expected: "buf.build"},
		{name: "uppercase_with_port", input: "BUF.BUILD:8443", expected: "buf.build"},
		{name: "ipv6_with_port", input: "[::1]:8443", expected: "::1"},
		{name: "empty", input: "", expected: ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, normalizeHost(tc.input))
		})
	}
}

const cargoIndexBufBuild = "sparse+https://buf.build/gen/cargo/"

// runSilent invokes the command and asserts a silent exit 1: stderr must be
// empty (no "Failure:" line). Use for paths where the URL is missing, no
// host can be extracted, or the host isn't in the allow-list.
func runSilent(
	t *testing.T,
	envOverrides map[string]string,
	args ...string,
) {
	t.Helper()
	runCargo(
		t,
		envOverrides,
		1,
		"",
		nil,
		nil,
		args...,
	)
}

// runSuccess invokes the command and asserts exit 0 with the given stdout.
func runSuccess(
	t *testing.T,
	envOverrides map[string]string,
	expectedStdout string,
	args ...string,
) {
	t.Helper()
	runCargo(
		t,
		envOverrides,
		0,
		expectedStdout,
		nil,
		nil,
		args...,
	)
}

// runLoud invokes the command and asserts exit 1 with stderr containing each
// of the given substrings. Use for paths where the host *is* in the
// allow-list but no token resolves.
func runLoud(
	t *testing.T,
	envOverrides map[string]string,
	stderrSubstrings []string,
	args ...string,
) {
	t.Helper()
	require.NotEmpty(t, stderrSubstrings, "loud failure tests must assert at least one stderr substring")
	runCargo(
		t,
		envOverrides,
		1,
		"",
		stderrSubstrings,
		nil,
		args...,
	)
}

// runCargo is the underlying harness. The interceptor mirrors the subset of
// cmd/buf/buf.go's wrapError that this command actually exercises: empty-
// message non-Connect errors pass through unchanged (silent failure), and
// any other error gets the "Failure: " prefix. wrapError has additional
// handling for *connect.Error, syserror, ImportNotExistError, etc. that the
// real binary applies in production but this command never produces, so the
// interceptor does not replicate them.
//
// expectedStderrPartials: nil means assert stderr is empty; a non-nil slice
// means assert each substring is present.
//
// forbiddenStderrSubstrings: any substring listed here must NOT appear in
// stderr. Used for leak-prevention assertions (e.g. that a wrapped error
// does not echo a sensitive value from the underlying cause).
func runCargo(
	t *testing.T,
	envOverrides map[string]string,
	expectedExitCode int,
	expectedStdout string,
	expectedStderrPartials []string,
	forbiddenStderrSubstrings []string,
	args ...string,
) {
	t.Helper()
	var stderrBuf *bytes.Buffer
	options := []appcmdtesting.RunOption{
		appcmdtesting.WithEnv(func(string) map[string]string {
			env := map[string]string{
				"PATH": os.Getenv("PATH"),
			}
			maps.Copy(env, envOverrides)
			return env
		}),
		appcmdtesting.WithExpectedExitCode(expectedExitCode),
		appcmdtesting.WithExpectedStdout(expectedStdout),
		appcmdtesting.WithArgs(args...),
	}
	if len(forbiddenStderrSubstrings) > 0 {
		stderrBuf = &bytes.Buffer{}
		options = append(options, appcmdtesting.WithStderr(stderrBuf))
	}
	if expectedStderrPartials == nil {
		options = append(options, appcmdtesting.WithExpectedStderrPartials())
	} else {
		options = append(options, appcmdtesting.WithExpectedStderrPartials(expectedStderrPartials...))
	}
	appcmdtesting.Run(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appext.NewBuilder(
					name,
					appext.BuilderWithInterceptor(
						func(next func(context.Context, appext.Container) error) func(context.Context, appext.Container) error {
							return func(ctx context.Context, container appext.Container) error {
								err := next(ctx, container)
								if err == nil {
									return nil
								}
								if err.Error() == "" {
									return err
								}
								return fmt.Errorf("Failure: %w", err)
							}
						},
					),
				),
			)
		},
		options...,
	)
	if stderrBuf != nil {
		stderrText := stderrBuf.String()
		for _, forbidden := range forbiddenStderrSubstrings {
			assert.NotContains(t, stderrText, forbidden, "stderr leaked forbidden substring")
		}
	}
}

// writeNetrc writes a netrc file with a single machine entry and returns its
// path.
func writeNetrc(t *testing.T, machine, login, password string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".netrc")
	contents := fmt.Sprintf("machine %s\n  login %s\n  password %s\n", machine, login, password)
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

func TestCargo_UnsetURL_SilentExit1(t *testing.T) {
	t.Parallel()
	runSilent(t, map[string]string{})
}

func TestCargo_NoArgs_DefaultAllowList_BufBuild_UnscopedToken(t *testing.T) {
	t.Parallel()
	runSuccess(t,
		map[string]string{
			"CARGO_REGISTRY_INDEX_URL": cargoIndexBufBuild,
			"BUF_TOKEN":                "plain-token",
		},
		"Bearer plain-token",
	)
}

func TestCargo_NoArgs_DefaultAllowList_OtherHost_SilentExit1(t *testing.T) {
	t.Parallel()
	runSilent(t,
		map[string]string{
			"CARGO_REGISTRY_INDEX_URL": "sparse+https://other.example.com/gen/cargo/",
			"BUF_TOKEN":                "plain-token",
		},
	)
}

func TestCargo_ExplicitHost_Matches(t *testing.T) {
	t.Parallel()
	runSuccess(t,
		map[string]string{
			"CARGO_REGISTRY_INDEX_URL": "sparse+https://bsr.example.com/gen/cargo/",
			"BUF_TOKEN":                "plain-token",
		},
		"Bearer plain-token",
		"bsr.example.com",
	)
}

func TestCargo_PositionalHostWithPort_MatchesURL(t *testing.T) {
	t.Parallel()
	runSuccess(t,
		map[string]string{
			"CARGO_REGISTRY_INDEX_URL": "sparse+https://bsr.example.com:8443/gen/cargo/",
			"BUF_TOKEN":                "plain-token",
		},
		"Bearer plain-token",
		"bsr.example.com:8443",
	)
}

func TestCargo_PositionalHostUppercase_MatchesURL(t *testing.T) {
	t.Parallel()
	runSuccess(t,
		map[string]string{
			"CARGO_REGISTRY_INDEX_URL": "sparse+https://bsr.example.com/gen/cargo/",
			"BUF_TOKEN":                "plain-token",
		},
		"Bearer plain-token",
		"BSR.example.com",
	)
}

func TestCargo_ExplicitHost_DoesNotIncludeDefault_BufBuild_Silent(t *testing.T) {
	t.Parallel()
	runSilent(t,
		map[string]string{
			"CARGO_REGISTRY_INDEX_URL": cargoIndexBufBuild,
			"BUF_TOKEN":                "plain-token",
		},
		"bsr.example.com",
	)
}

func TestCargo_MultipleExplicitHosts_RequestMatchesOne(t *testing.T) {
	t.Parallel()
	runSuccess(t,
		map[string]string{
			"CARGO_REGISTRY_INDEX_URL": cargoIndexBufBuild,
			"BUF_TOKEN":                "plain-token",
		},
		"Bearer plain-token",
		"a.example.com", "buf.build",
	)
}

func TestCargo_ScopedToken_Match(t *testing.T) {
	t.Parallel()
	runSuccess(t,
		map[string]string{ //nolint:gosec // G101: BUF_TOKEN literals are synthetic env fixtures for contract tests.
			"CARGO_REGISTRY_INDEX_URL": cargoIndexBufBuild,
			"BUF_TOKEN":                "scoped-tok@buf.build",
		},
		"Bearer scoped-tok",
	)
}

func TestCargo_MalformedBufToken_LoudFailureMentionsBufToken(t *testing.T) {
	t.Parallel()
	const secret = "supersecret-do-not-leak"
	runCargo(
		t,
		map[string]string{
			"CARGO_REGISTRY_INDEX_URL": cargoIndexBufBuild,
			// Duplicate remote triggers newMultipleTokenProvider's
			// "repeated remote address" error, which embeds the raw token.
			"BUF_TOKEN": secret + "@buf.build," + secret + "@buf.build",
		},
		1,
		"",
		[]string{
			"Failure:",
			"BUF_TOKEN environment variable could not be parsed",
			"buf registry login buf.build",
			"unset BUF_TOKEN",
		},
		// The wrapped error must NOT echo the raw token; the parser-level
		// detail is debug-only.
		[]string{secret},
	)
}

func TestCargo_ScopedToken_NoMatchForHost_LoudFailure(t *testing.T) {
	t.Parallel()
	runLoud(t,
		map[string]string{ //nolint:gosec // G101: BUF_TOKEN literals are synthetic env fixtures for contract tests.
			"CARGO_REGISTRY_INDEX_URL": cargoIndexBufBuild,
			"BUF_TOKEN":                "scoped-tok@elsewhere.example.com",
		},
		[]string{
			"Failure:",
			"no token found for buf.build",
			"\"buf registry login buf.build\"",
		},
	)
}

func TestCargo_NetrcMatch(t *testing.T) {
	t.Parallel()
	netrcPath := writeNetrc(t, "buf.build", "user", "netrc-secret")
	runSuccess(t,
		map[string]string{
			"CARGO_REGISTRY_INDEX_URL": cargoIndexBufBuild,
			"NETRC":                    netrcPath,
		},
		"Bearer netrc-secret",
	)
}

func TestCargo_ScopedTokenNoMatch_FallsThroughToNetrc(t *testing.T) {
	t.Parallel()
	netrcPath := writeNetrc(t, "buf.build", "user", "netrc-secret")
	runSuccess(t,
		map[string]string{ //nolint:gosec // G101: BUF_TOKEN literals are synthetic env fixtures for contract tests.
			"CARGO_REGISTRY_INDEX_URL": cargoIndexBufBuild,
			"BUF_TOKEN":                "scoped-tok@elsewhere.example.com",
			"NETRC":                    netrcPath,
		},
		"Bearer netrc-secret",
	)
}

func TestCargo_BufTokenWinsOverNetrc(t *testing.T) {
	t.Parallel()
	netrcPath := writeNetrc(t, "buf.build", "user", "netrc-loses")
	runSuccess(t,
		map[string]string{
			"CARGO_REGISTRY_INDEX_URL": cargoIndexBufBuild,
			"BUF_TOKEN":                "env-wins",
			"NETRC":                    netrcPath,
		},
		"Bearer env-wins",
	)
}

func TestCargo_NetrcMiss_NoEnvVar_LoudFailure(t *testing.T) {
	t.Parallel()
	emptyNetrc := writeNetrc(t, "other.example.com", "user", "irrelevant")
	runLoud(t,
		map[string]string{
			"CARGO_REGISTRY_INDEX_URL": cargoIndexBufBuild,
			"NETRC":                    emptyNetrc,
		},
		[]string{
			"Failure:",
			"no token found for buf.build",
		},
	)
}
