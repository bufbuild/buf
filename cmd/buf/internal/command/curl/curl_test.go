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

package curl

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appcmd/appcmdtesting"
	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/cmd/buf/internal/internaltesting"
	"github.com/bufbuild/buf/private/pkg/slogapp"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateHTTP2PriorKnowledge verifies that validate implies
// --http2-prior-knowledge for plain-text URLs when server reflection or the
// gRPC protocol is used, since both require HTTP/2, and leaves it alone
// otherwise.
func TestValidateHTTP2PriorKnowledge(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		args     []string
		isSecure bool
		expected bool
	}{
		{
			name:     "http with default reflection",
			expected: true,
		},
		{
			name:     "http with explicit reflection and schema",
			args:     []string{"--schema", ".", "--reflect"},
			expected: true,
		},
		{
			name:     "http with grpc protocol and schema",
			args:     []string{"--schema", ".", "--protocol", "grpc"},
			expected: true,
		},
		{
			name:     "http with connect protocol and schema",
			args:     []string{"--schema", "."},
			expected: false,
		},
		{
			name:     "http with grpcweb protocol and schema",
			args:     []string{"--schema", ".", "--protocol", "grpcweb"},
			expected: false,
		},
		{
			name:     "http with flag set and schema",
			args:     []string{"--schema", ".", "--http2-prior-knowledge"},
			expected: true,
		},
		{
			name:     "https with default reflection",
			isSecure: true,
			expected: false,
		},
		{
			name:     "https with grpc protocol and schema",
			args:     []string{"--schema", ".", "--protocol", "grpc"},
			isSecure: true,
			expected: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			flags := newFlags()
			flagSet := pflag.NewFlagSet("curl", pflag.ContinueOnError)
			flags.Bind(flagSet)
			require.NoError(t, flagSet.Parse(testCase.args))
			require.NoError(t, flags.validate(true, testCase.isSecure))
			assert.Equal(t, testCase.expected, flags.HTTP2PriorKnowledge)
		})
	}
}

// TestWrapPlainTextHTTP2Error verifies that only the HTTP/2 framer error for
// an HTTP 1.1 response over a plain-text prior knowledge connection is
// rewritten.
func TestWrapPlainTextHTTP2Error(t *testing.T) {
	t.Parallel()
	framerErr := errors.New("unavailable: http2: failed reading the frame payload: http2: frame too large, note that the frame header looked like an HTTP/1.1 header")
	otherErr := errors.New("unavailable: dial tcp 127.0.0.1:1: connect: connection refused")

	t.Run("nil error", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, wrapPlainTextHTTP2Error(nil, &flags{HTTP2PriorKnowledge: true, Reflect: true}, "localhost:80", false))
	})
	t.Run("secure URL is untouched", func(t *testing.T) {
		t.Parallel()
		assert.Same(t, framerErr, wrapPlainTextHTTP2Error(framerErr, &flags{HTTP2PriorKnowledge: true, Reflect: true}, "localhost:443", true))
	})
	t.Run("without prior knowledge is untouched", func(t *testing.T) {
		t.Parallel()
		assert.Same(t, framerErr, wrapPlainTextHTTP2Error(framerErr, &flags{}, "localhost:80", false))
	})
	t.Run("other error is untouched", func(t *testing.T) {
		t.Parallel()
		assert.Same(t, otherErr, wrapPlainTextHTTP2Error(otherErr, &flags{HTTP2PriorKnowledge: true, Reflect: true}, "localhost:80", false))
	})
	t.Run("http1 response with prior knowledge is rewritten", func(t *testing.T) {
		t.Parallel()
		err := wrapPlainTextHTTP2Error(framerErr, &flags{HTTP2PriorKnowledge: true}, "localhost:80", false)
		require.Error(t, err)
		assert.Equal(t,
			"the RPC protocol or method requires HTTP/2, but the server at localhost:80 responded with HTTP/1.1 and does not appear to support HTTP/2 over plain-text (h2c): "+framerErr.Error(),
			err.Error(),
		)
		// Not wrapped, so the CLI error interceptor does not rewrite it.
		assert.NotErrorIs(t, err, framerErr)
	})
}

// TestRunPlainTextReflection verifies that server reflection works against a
// plain-text server that supports HTTP/2 via prior knowledge (h2c) without
// the --http2-prior-knowledge flag, for both the Connect and gRPC protocols.
func TestRunPlainTextReflection(t *testing.T) {
	t.Parallel()
	resolver := newTestDescriptorResolver(t)
	server := newTestPlainTextReflectionServer(t, resolver, true, "acme.foo.v1.FooService", "acme.bar.v1.BarService")

	t.Run("list services", func(t *testing.T) {
		t.Parallel()
		appcmdtesting.Run(
			t,
			newTestCurlCommand,
			appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
			appcmdtesting.WithExpectedStdout("acme.bar.v1.BarService\nacme.foo.v1.FooService\n"),
			appcmdtesting.WithArgs("curl", "--list-services", server.URL),
		)
	})
	t.Run("list methods with grpc protocol", func(t *testing.T) {
		t.Parallel()
		appcmdtesting.Run(
			t,
			newTestCurlCommand,
			appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
			appcmdtesting.WithExpectedStdout(
				"acme.bar.v1.BarService/CreateBar\nacme.foo.v1.FooService/GetFoo\nacme.foo.v1.FooService/ListFoos\n",
			),
			appcmdtesting.WithArgs("curl", "--list-methods", "--protocol", "grpc", server.URL),
		)
	})
}

// TestRunPlainTextBidiStream verifies that invoking a bidirectional streaming
// method with --schema against a plain-text h2c server works without the
// --http2-prior-knowledge flag. The reflection service itself is the bidi
// method under test, described by a minimal schema written to a temp dir.
func TestRunPlainTextBidiStream(t *testing.T) {
	t.Parallel()
	resolver := newTestDescriptorResolver(t)
	server := newTestPlainTextReflectionServer(t, resolver, true, "acme.foo.v1.FooService", "acme.bar.v1.BarService")

	schemaDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(schemaDir, "reflection.proto"), []byte(`
syntax = "proto3";
package grpc.reflection.v1;
service ServerReflection {
  rpc ServerReflectionInfo(stream ServerReflectionRequest) returns (stream ServerReflectionResponse);
}
message ServerReflectionRequest {
  string host = 1;
  oneof message_request {
    string list_services = 7;
  }
}
message ServerReflectionResponse {
  string valid_host = 1;
  ServerReflectionRequest original_request = 2;
  oneof message_response {
    ListServiceResponse list_services_response = 6;
  }
}
message ListServiceResponse {
  repeated ServiceResponse service = 1;
}
message ServiceResponse {
  string name = 1;
}
`), 0o600))

	stdout := bytes.NewBuffer(nil)
	appcmdtesting.Run(
		t,
		newTestCurlCommand,
		appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		appcmdtesting.WithStdout(stdout),
		appcmdtesting.WithArgs(
			"curl",
			"--schema", schemaDir,
			"--data", `{"list_services": ""}`,
			server.URL+"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
		),
	)
	assert.Contains(t, stdout.String(), "acme.foo.v1.FooService")
	assert.Contains(t, stdout.String(), "acme.bar.v1.BarService")
}

// TestRunPlainTextReflectionWithoutHTTP2 verifies the failure when the
// plain-text server only speaks HTTP 1.1, which cannot carry the
// bidirectional stream that server reflection uses.
func TestRunPlainTextReflectionWithoutHTTP2(t *testing.T) {
	t.Parallel()
	resolver := newTestDescriptorResolver(t)
	server := newTestPlainTextReflectionServer(t, resolver, false, "acme.foo.v1.FooService")
	stderr := bytes.NewBuffer(nil)
	appcmdtesting.Run(
		t,
		newTestCurlCommand,
		appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		appcmdtesting.WithExpectedExitCode(1),
		appcmdtesting.WithStderr(stderr),
		// The client speaks HTTP/2 via prior knowledge, but the server answers
		// with HTTP 1.1, which the HTTP/2 framer rejects.
		appcmdtesting.WithExpectedStderrPartials(
			"the RPC protocol or method requires HTTP/2, but the server at "+strings.TrimPrefix(server.URL, "http://")+
				" responded with HTTP/1.1 and does not appear to support HTTP/2 over plain-text (h2c)",
			"note that the frame header looked like an HTTP/1.1 header",
		),
		appcmdtesting.WithArgs("curl", "--list-services", server.URL),
	)
	// The flag is implied, so the error must no longer tell the user to set it.
	assert.NotContains(t, stderr.String(), http2PriorKnowledgeFlagName)
}

// newTestCurlCommand returns a root command with the curl command as its only
// sub-command, wired up the same way as in the buf CLI.
func newTestCurlCommand(use string) *appcmd.Command {
	builder := appext.NewBuilder(
		use,
		appext.BuilderWithTimeout(0),
		appext.BuilderWithLoggerProvider(slogapp.LoggerProvider),
	)
	return &appcmd.Command{
		Use:                 use,
		BindPersistentFlags: builder.BindRoot,
		SubCommands: []*appcmd.Command{
			NewCommand("curl", builder),
		},
	}
}
