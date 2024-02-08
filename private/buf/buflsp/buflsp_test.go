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

package buflsp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/httpauth"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"github.com/bufbuild/buf/private/pkg/zaputil"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestBufLsp(t *testing.T) {
	t.Parallel()
	server, doc, err := newTestServerWithFilePath(t, "testdata/buftest/buf/lsp/test/v1alpha1/test_cases.proto")
	require.NoError(t, err)
	entry, ok := server.getCachedFileEntryForURI(doc)
	require.True(t, ok, "%s not in cache", doc.Filename())

	for _, testCase := range []struct {
		prefix   string
		expected []string
	}{
		{
			prefix:   ".",
			expected: []string{"buf", "google"},
		},
		{
			prefix:   "lsp.",
			expected: []string{"test"},
		},
		{
			prefix:   "(buf.",
			expected: []string{"lsp", "validate"},
		},
		{
			prefix:   "buf.lsp.test.v1alpha1.",
			expected: []string{"SourceLocation", "TestEnum", "TestMessage", "CodeInfo", "Diagnostic", "FileInfo", "SemanticToken"},
		},
		{
			prefix:   "(buf.lsp.test.v1alpha1.",
			expected: []string{"SourceLocation", "TestEnum", "TestMessage", "CodeInfo", "Diagnostic", "FileInfo", "SemanticToken"},
		},
		{
			prefix:   "[(validate.",
			expected: []string{"message", "oneof", "field"},
		},
		{
			prefix:   "[(validate.message).",
			expected: []string{"cel", "disabled"},
		},
		{
			prefix: "[hi",
			// All the known options.
			expected: []string{
				"java_generic_services",
				"php_class_prefix",
				"unverified_lazy",
				"cc_enable_arenas",
				"java_generate_equals_and_hash",
				"deprecated_legacy_json_field_conflicts",
				"map_entry",
				"lazy",
				"csharp_namespace",
				"java_string_check_utf8",
				"ruby_package",
				"swift_prefix",
				"ctype",
				"jstype",
				"cc_generic_services",
				"go_package",
				"java_multiple_files",
				"java_outer_classname",
				"uninterpreted_option",
				"message_set_wire_format",
				"allow_alias",
				"deprecated",
				"optimize_for",
				"php_generic_services",
				"debug_redact",
				"objc_class_prefix",
				"edition_defaults",
				"packed",
				"retention",
				"idempotency_level",
				"php_namespace",
				"php_metadata_namespace",
				"weak",
				"features",
				"py_generic_services",
				"no_standard_descriptor_accessor",
				"targets",
				"java_package",
			},
		},
	} {
		testCase := testCase
		t.Run(testCase.prefix, func(t *testing.T) {
			t.Parallel()
			// TODO: We should never be calling the lock in a test. If we need to expose test functionality, it should
			// be done with a function contained in the server.
			server.lock.Lock()
			defer server.lock.Unlock()
			expectCompletions(t, server, entry, testCase.prefix, testCase.expected)
		})
	}
}

func expectCompletions(t *testing.T, server *server, entry *fileEntry, prefix string, expectedParts []string) {
	t.Helper()
	completions := server.findPrefixCompletions(context.Background(), entry, symbolName{"buf", "lsp", "test", "v1"}, prefix)
	for _, expectedPart := range expectedParts {
		if _, ok := completions[expectedPart]; !ok {
			got := make([]string, 0, len(completions))
			for key := range completions {
				got = append(got, key)
			}
			t.Fatalf("expected %q in completions, got %v", expectedPart, got)
		}
		delete(completions, expectedPart)
	}
	if len(completions) != 0 {
		got := make([]string, 0, len(completions))
		for key := range completions {
			got = append(got, key)
		}
		t.Fatalf("got unexpected completions: %v", got)
	}
}

func newTestServerWithFilePath(t *testing.T, filePath string) (*server, protocol.DocumentURI, error) {
	t.Helper()
	server, err := newTestServer(t)
	if err != nil {
		return nil, "", err
	}
	entry, err := openFile(context.Background(), server, filePath)
	if err != nil {
		return nil, "", err
	}
	return server, entry, nil
}

func openFile(ctx context.Context, server *server, filePath string) (protocol.DocumentURI, error) {
	fileReader, err := os.Open(filePath)
	if err != nil {
		return "", err
	}

	fileData, err := io.ReadAll(fileReader)
	if err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}
	fileURI := uri.File(absPath)
	if err := server.DidOpen(
		ctx,
		&protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:  fileURI,
				Text: string(fileData),
			},
		},
	); err != nil {
		return "", err
	}
	return fileURI, nil
}

func newTestServer(tb testing.TB) (*server, error) {
	tb.Helper()
	use := "test"
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	cacheDirPath := tb.TempDir()
	env := newEnvFunc(tb, cacheDirPath)(use)

	appContainer := app.NewContainer(
		env,
		nil,
		stdout,
		stderr,
		"test",
	)

	logger, err := zaputil.NewLoggerForFlagValues(appContainer.Stderr(), "info", "text")
	if err != nil {
		return nil, err
	}
	verbosePrinter := verbose.NewPrinter(appContainer.Stderr(), "test")

	container, err := appext.NewContainer(appContainer, "test", logger, verbosePrinter)
	if err != nil {
		return nil, err
	}
	omniProvider, err := bufmoduletesting.NewOmniProvider(
		bufmoduletesting.ModuleData{
			Name:    "buf.build/bufbuild/protovalidate",
			DirPath: "./testdata/protovalidate",
		},
	)
	if err != nil {
		return nil, err
	}
	controller, err := bufctl.NewController(
		container.Logger(),
		tracing.NewTracer(container.Tracer()),
		container,
		omniProvider,
		omniProvider,
		omniProvider,
		omniProvider,
		http.DefaultClient,
		httpauth.NewNopAuthenticator(),
		git.ClonerOptions{},
	)
	if err != nil {
		return nil, err
	}

	server, err := newServer(
		context.Background(),
		container.Logger(),
		tracing.NewTracer(container.Tracer()),
		nil,
		controller,
		cacheDirPath,
	)
	if err != nil {
		return nil, err
	}
	if _, err := server.Initialize(context.Background(), &protocol.InitializeParams{}); err != nil {
		return nil, err
	}
	return server, nil
}

func newEnvFunc(tb testing.TB, cacheDirPath string) func(string) map[string]string {
	tb.Helper()
	require.NotEmpty(tb, cacheDirPath)
	return func(use string) map[string]string {
		return map[string]string{
			useEnvVar(use, "CACHE_DIR"): cacheDirPath,
			useEnvVar(use, "HOME"):      tb.TempDir(),
			"PATH":                      os.Getenv("PATH"),
		}
	}
}

func useEnvVar(use string, suffix string) string {
	return strings.ToUpper(use) + "_" + suffix
}
