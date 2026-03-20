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
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// testServices defines the two services used across all completion tests:
//
//	acme.foo.v1.FooService  – methods: GetFoo, ListFoos
//	acme.bar.v1.BarService  – methods: CreateBar
//
// The names are chosen to give a multi-level package hierarchy (acme → foo/bar
// → v1 → ServiceName) that exercises the hierarchical completion logic.
var testServices = []protoreflect.FullName{
	"acme.foo.v1.FooService",
	"acme.bar.v1.BarService",
}

// newTestDescriptorResolver builds a protodesc.Resolver containing the two
// test services. Both use google.protobuf.Empty as input/output so they need
// no additional dependencies beyond the well-known types already in the global
// registry.
func newTestDescriptorResolver(t *testing.T) protodesc.Resolver {
	t.Helper()
	// Build one file per service so their proto packages match their names.
	fooFDP := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("acme/foo/v1/foo.proto"),
		Syntax:     proto.String("proto3"),
		Package:    proto.String("acme.foo.v1"),
		Dependency: []string{"google/protobuf/empty.proto"},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("FooService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{Name: proto.String("GetFoo"), InputType: proto.String(".google.protobuf.Empty"), OutputType: proto.String(".google.protobuf.Empty")},
					{Name: proto.String("ListFoos"), InputType: proto.String(".google.protobuf.Empty"), OutputType: proto.String(".google.protobuf.Empty")},
				},
			},
		},
	}
	barFDP := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("acme/bar/v1/bar.proto"),
		Syntax:     proto.String("proto3"),
		Package:    proto.String("acme.bar.v1"),
		Dependency: []string{"google/protobuf/empty.proto"},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("BarService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{Name: proto.String("CreateBar"), InputType: proto.String(".google.protobuf.Empty"), OutputType: proto.String(".google.protobuf.Empty")},
				},
			},
		},
	}
	fooFD, err := protodesc.NewFile(fooFDP, protoregistry.GlobalFiles)
	require.NoError(t, err)
	barFD, err := protodesc.NewFile(barFDP, protoregistry.GlobalFiles)
	require.NoError(t, err)

	files := new(protoregistry.Files)
	require.NoError(t, files.RegisterFile(fooFD))
	require.NoError(t, files.RegisterFile(barFD))
	return files
}

// newTestReflectionServer starts an in-process TLS Connect server that serves
// gRPC reflection (v1 and v1alpha) for the given service names, using resolver
// to look up their descriptors. The server is automatically closed when t
// completes.
func newTestReflectionServer(t *testing.T, resolver protodesc.Resolver, serviceNames ...string) *httptest.Server {
	t.Helper()
	reflector := grpcreflect.NewReflector(
		grpcreflect.NamerFunc(func() []string { return serviceNames }),
		grpcreflect.WithDescriptorResolver(resolver),
	)
	mux := http.NewServeMux()
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}

// newCompletionCmd returns a minimal cobra.Command that has the flags accessed
// by completeURL (schema, insecure, http2-prior-knowledge).
func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{}
	flags := cmd.Flags()
	flags.StringSlice(schemaFlagName, nil, "")
	flags.Bool(insecureFlagName, false, "")
	flags.Bool(http2PriorKnowledgeFlagName, false, "")
	return cmd
}

// TestFlagCompletions verifies that --protocol and --reflect-protocol return
// the expected static lists with no file completions.
func TestFlagCompletions(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{}
	// completeCurlCommand requires the flags to already be registered.
	newFlags().Bind(cmd.Flags())
	require.NoError(t, completeCurlCommand(cmd))

	protocolFn, _ := cmd.GetFlagCompletionFunc(protocolFlagName)
	require.NotNil(t, protocolFn)
	protocols, directive := protocolFn(cmd, nil, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.ElementsMatch(t, []string{connect.ProtocolConnect, connect.ProtocolGRPC, connect.ProtocolGRPCWeb}, protocols)

	reflectFn, _ := cmd.GetFlagCompletionFunc(reflectProtocolFlagName)
	require.NotNil(t, reflectFn)
	reflectProtocols, directive := reflectFn(cmd, nil, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.ElementsMatch(t, []string{"grpc-v1", "grpc-v1alpha"}, reflectProtocols)
}

// TestCompletePathFromServices_ServiceHierarchy exercises the hierarchical
// package-segment completion for the service-name portion of the URL.
func TestCompletePathFromServices_ServiceHierarchy(t *testing.T) {
	t.Parallel()
	const base = "https://api.example.com"

	// getDesc is unused for service-level tests but must not panic.
	noDesc := func(string) (protoreflect.ServiceDescriptor, error) {
		t.Fatal("getServiceDescriptor should not be called during service-level completion")
		return nil, nil
	}

	t.Run("empty prefix skips to first fork", func(t *testing.T) {
		t.Parallel()
		// Both services share the unambiguous prefix "acme.", so the completer
		// advances past it automatically and returns the two diverging branches.
		completions, directive := completePathFromServices(base, testServices, "", noDesc, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoSpace|cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Equal(t, []string{base + "/acme.bar.", base + "/acme.foo."}, completions)
	})

	t.Run("acme. prefix shows same fork", func(t *testing.T) {
		t.Parallel()
		// Explicitly typing "acme." produces the same result as the empty prefix.
		completions, directive := completePathFromServices(base, testServices, "acme.", noDesc, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoSpace|cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Equal(t, []string{base + "/acme.bar.", base + "/acme.foo."}, completions)
	})

	t.Run("unambiguous branch jumps to service name", func(t *testing.T) {
		t.Parallel()
		// "acme.foo." has only one service beneath it, so the completer skips
		// the intermediate "v1." segment and lands on the full service name.
		completions, directive := completePathFromServices(base, testServices, "acme.foo.", noDesc, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoSpace|cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Equal(t, []string{base + "/acme.foo.v1.FooService/"}, completions)
	})

	t.Run("full service name adds trailing slash", func(t *testing.T) {
		t.Parallel()
		completions, directive := completePathFromServices(base, testServices, "acme.foo.v1.", noDesc, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoSpace|cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Equal(t, []string{base + "/acme.foo.v1.FooService/"}, completions)
	})

	t.Run("exact service name without trailing slash adds slash", func(t *testing.T) {
		t.Parallel()
		completions, directive := completePathFromServices(base, testServices, "acme.foo.v1.FooService", noDesc, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoSpace|cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Equal(t, []string{base + "/acme.foo.v1.FooService/"}, completions)
	})

	t.Run("no match returns empty list", func(t *testing.T) {
		t.Parallel()
		completions, directive := completePathFromServices(base, testServices, "notexist.", noDesc, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoSpace|cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Empty(t, completions)
	})

	t.Run("source appended to terminal service names only", func(t *testing.T) {
		t.Parallel()
		// Intermediate package segments get no description; only the terminal
		// service name (ending with "/") gets the source description.
		forkCompletions, _ := completePathFromServices(base, testServices, "acme.", noDesc, "test-source")
		assert.Equal(t, []string{base + "/acme.bar.", base + "/acme.foo."}, forkCompletions,
			"intermediate package segments should have no description")

		terminalCompletions, _ := completePathFromServices(base, testServices, "acme.foo.", noDesc, "test-source")
		assert.Equal(t, []string{base + "/acme.foo.v1.FooService/\ttest-source"}, terminalCompletions,
			"terminal service name should have description")
	})
}

// TestCompletePathFromServices_Methods exercises method completion once a
// slash is present in the raw path.
func TestCompletePathFromServices_Methods(t *testing.T) {
	t.Parallel()
	const base = "https://api.example.com"

	resolver := newTestDescriptorResolver(t)
	getDesc := func(svcName string) (protoreflect.ServiceDescriptor, error) {
		desc, err := resolver.FindDescriptorByName(protoreflect.FullName(svcName))
		if err != nil {
			return nil, err
		}
		svcDesc, ok := desc.(protoreflect.ServiceDescriptor)
		if !ok {
			return nil, fmt.Errorf("%s is not a service", svcName)
		}
		return svcDesc, nil
	}

	t.Run("all methods listed after trailing slash", func(t *testing.T) {
		t.Parallel()
		completions, directive := completePathFromServices(base, testServices, "acme.foo.v1.FooService/", getDesc, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Equal(t, []string{
			base + "/acme.foo.v1.FooService/GetFoo",
			base + "/acme.foo.v1.FooService/ListFoos",
		}, completions)
	})

	t.Run("method prefix filters results", func(t *testing.T) {
		t.Parallel()
		completions, directive := completePathFromServices(base, testServices, "acme.foo.v1.FooService/List", getDesc, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Equal(t, []string{base + "/acme.foo.v1.FooService/ListFoos"}, completions)
	})

	t.Run("non-matching method prefix returns empty", func(t *testing.T) {
		t.Parallel()
		completions, directive := completePathFromServices(base, testServices, "acme.foo.v1.FooService/Delete", getDesc, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Empty(t, completions)
	})

	t.Run("descriptor lookup error returns no completions", func(t *testing.T) {
		t.Parallel()
		errDesc := func(string) (protoreflect.ServiceDescriptor, error) {
			return nil, fmt.Errorf("not found")
		}
		completions, directive := completePathFromServices(base, testServices, "acme.foo.v1.FooService/", errDesc, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Nil(t, completions)
	})

	t.Run("source appended to method completions", func(t *testing.T) {
		t.Parallel()
		completions, directive := completePathFromServices(base, testServices, "acme.foo.v1.FooService/", getDesc, "test-source")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Equal(t, []string{
			base + "/acme.foo.v1.FooService/GetFoo\ttest-source",
			base + "/acme.foo.v1.FooService/ListFoos\ttest-source",
		}, completions)
	})

}

// TestCompleteURL_ReflectionServer verifies end-to-end completion via a real
// in-process Connect server with gRPC reflection enabled.
func TestCompleteURL_ReflectionServer(t *testing.T) {
	t.Parallel()
	resolver := newTestDescriptorResolver(t)
	server := newTestReflectionServer(t, resolver, "acme.foo.v1.FooService", "acme.bar.v1.BarService")

	cmd := newCompletionCmd()
	require.NoError(t, cmd.Flags().Set(insecureFlagName, "true"))

	t.Run("root skips to first fork", func(t *testing.T) {
		t.Parallel()
		// Unambiguous "acme." prefix is skipped automatically.
		completions, directive := completeURL(cmd, nil, server.URL+"/")
		assert.Equal(t, cobra.ShellCompDirectiveNoSpace|cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Equal(t, []string{server.URL + "/acme.bar.", server.URL + "/acme.foo."}, completions)
	})

	t.Run("unambiguous branch jumps to service name", func(t *testing.T) {
		t.Parallel()
		// "acme.foo." has only one service; intermediate "v1." is skipped.
		completions, directive := completeURL(cmd, nil, server.URL+"/acme.foo.")
		assert.Equal(t, cobra.ShellCompDirectiveNoSpace|cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Equal(t, []string{server.URL + "/acme.foo.v1.FooService/\treflection"}, completions)
	})

	t.Run("lists methods for a service", func(t *testing.T) {
		t.Parallel()
		completions, directive := completeURL(cmd, nil, server.URL+"/acme.foo.v1.FooService/")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Equal(t, []string{
			server.URL + "/acme.foo.v1.FooService/GetFoo\treflection",
			server.URL + "/acme.foo.v1.FooService/ListFoos\treflection",
		}, completions)
	})

	t.Run("empty toComplete returns no completions", func(t *testing.T) {
		t.Parallel()
		completions, directive := completeURL(cmd, nil, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Nil(t, completions)
	})

	t.Run("non-URL toComplete returns no completions", func(t *testing.T) {
		t.Parallel()
		completions, directive := completeURL(cmd, nil, "not-a-url")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Nil(t, completions)
	})
}

// TestCompleteURLFromReflection_Unavailable verifies that when a server does not
// support reflection, completeURLFromReflection returns ok=false so the caller
// can try an alternative source.
func TestCompleteURLFromReflection_Unavailable(t *testing.T) {
	t.Parallel()
	// A plain HTTPS server with no reflection handlers; every request returns 404.
	server := httptest.NewUnstartedServer(http.NotFoundHandler())
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)

	transport, ok := server.Client().Transport.(*http.Transport)
	require.True(t, ok)
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:   transport.TLSClientConfig,
			ForceAttemptHTTP2: true,
			Protocols:         protocols,
		},
	}

	ctx := t.Context()
	completions, directive, ok := completeURLFromReflection(ctx, httpClient, server.URL, "")
	assert.False(t, ok, "expected ok=false when server does not support reflection")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.Nil(t, completions)
}

// TestMakeCompletionHTTPClient verifies the two code paths in makeCompletionHTTPClient.
func TestMakeCompletionHTTPClient(t *testing.T) {
	t.Parallel()

	t.Run("https returns client", func(t *testing.T) {
		t.Parallel()
		cmd := newCompletionCmd()
		client, ok := makeCompletionHTTPClient(cmd, true)
		assert.True(t, ok)
		assert.NotNil(t, client)
	})

	t.Run("http without prior knowledge returns nothing", func(t *testing.T) {
		t.Parallel()
		cmd := newCompletionCmd()
		client, ok := makeCompletionHTTPClient(cmd, false)
		assert.False(t, ok)
		assert.Nil(t, client)
	})

	t.Run("http with prior knowledge returns client", func(t *testing.T) {
		t.Parallel()
		cmd := newCompletionCmd()
		require.NoError(t, cmd.Flags().Set(http2PriorKnowledgeFlagName, "true"))
		client, ok := makeCompletionHTTPClient(cmd, false)
		assert.True(t, ok)
		assert.NotNil(t, client)
	})
}

// TestCompletePathFromServices_ErrorReporting verifies that descriptor lookup
// errors are written to BASH_COMP_DEBUG_FILE when a source is set, and are
// silent when source is empty. These tests modify an environment variable so
// they cannot be run in parallel.
func TestCompletePathFromServices_ErrorReporting(t *testing.T) {
	const base = "https://api.example.com"
	errDesc := func(string) (protoreflect.ServiceDescriptor, error) {
		return nil, fmt.Errorf("lookup failed")
	}

	t.Run("error logged when source is set", func(t *testing.T) {
		debugFile := filepath.Join(t.TempDir(), "comp_debug.log")
		t.Setenv("BASH_COMP_DEBUG_FILE", debugFile)

		completions, directive := completePathFromServices(base, testServices, "acme.foo.v1.FooService/", errDesc, "reflection")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Nil(t, completions)

		contents, err := os.ReadFile(debugFile)
		require.NoError(t, err, "expected error to be written to debug file")
		assert.Contains(t, string(contents), "acme.foo.v1.FooService")
		assert.Contains(t, string(contents), "lookup failed")
	})

	t.Run("error silent when source is empty", func(t *testing.T) {
		debugFile := filepath.Join(t.TempDir(), "comp_debug.log")
		t.Setenv("BASH_COMP_DEBUG_FILE", debugFile)

		completions, directive := completePathFromServices(base, testServices, "acme.foo.v1.FooService/", errDesc, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Nil(t, completions)

		_, err := os.ReadFile(debugFile)
		assert.True(t, os.IsNotExist(err), "debug file should not be created when source is empty")
	})
}
