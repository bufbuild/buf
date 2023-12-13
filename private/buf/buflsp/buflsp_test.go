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

package buflsp

import (
	"archive/zip"
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
	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"github.com/bufbuild/buf/private/pkg/zaputil"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func TestBufLsp(t *testing.T) {
	t.Parallel()
	lspServer, doc, err := newTestBufLspWith(t, "../../../proto/buftest/buf/lsp/test/v1alpha1/test_cases.proto")
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := lspServer.fileCache[doc.Filename()]
	if !ok {
		t.Fatal("file not in cache")
	}

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
			lspServer.lock.Lock()
			defer lspServer.lock.Unlock()
			expectCompletions(t, lspServer, entry, testCase.prefix, testCase.expected)
		})
	}
}

func expectCompletions(t *testing.T, lspServer *server, entry *fileEntry, prefix string, expectedParts []string) {
	t.Helper()
	completions := lspServer.findPrefixCompletions(context.Background(), entry, symbolName{"buf", "lsp", "test", "v1"}, prefix)
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

func newTestBufLspWith(t *testing.T, fileName string) (*server, protocol.DocumentURI, error) {
	t.Helper()
	lspServer, err := newTestBufLsp(t)
	if err != nil {
		return nil, "", err
	}
	entry, err := openFile(context.Background(), lspServer, fileName)
	if err != nil {
		return nil, "", err
	}
	return lspServer, entry, nil
}

func openFile(ctx context.Context, lspServer *server, fileName string) (protocol.DocumentURI, error) {
	fileReader, err := os.Open(fileName)
	if err != nil {
		return "", err
	}

	fileData, err := io.ReadAll(fileReader)
	if err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(fileName)
	if err != nil {
		return "", err
	}
	fileURI := protocol.DocumentURI("file://" + absPath)
	if err := lspServer.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:  fileURI,
			Text: string(fileData),
		},
	}); err != nil {
		return "", err
	}
	return fileURI, nil
}

func newTestBufLsp(tb testing.TB) (*server, error) {
	tb.Helper()
	use := "test"
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	env := newEnvFunc(tb, "")(use)

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

	container, err := newContainer(appContainer, "test", logger, verbosePrinter)
	if err != nil {
		return nil, err
	}
	omniProvider, err := bufmoduletesting.NewOmniProvider(
		bufmoduletesting.ModuleData{
			Name:       "buf.build/bufbuild/protovalidate",
			PathToData: zipData(tb, "./testdata/protovalidate.zip"),
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
		http.DefaultClient,
		httpauth.NewNopAuthenticator(),
		git.ClonerOptions{},
	)
	if err != nil {
		return nil, err
	}

	lspServer, err := newServer(
		context.Background(),
		nil,
		container,
		controller,
	)
	if err != nil {
		return nil, err
	}
	if _, err := lspServer.Initialize(context.Background(), &protocol.InitializeParams{}); err != nil {
		return nil, err
	}
	return lspServer, nil
}

func zipData(tb testing.TB, filename string) map[string][]byte {
	zipData := map[string][]byte{}
	zipReader, err := zip.OpenReader(filename)
	require.NoError(tb, err)
	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		r, err := file.Open()
		require.NoError(tb, err)
		data, err := ioext.ReadAllAndClose(r)
		require.NoError(tb, err)
		zipData[file.Name] = data
	}
	return zipData
}

type container struct {
	app.Container
	nameContainer    appext.NameContainer
	logContainer     appext.LoggerContainer
	tracerContainer  appext.TracerContainer
	verboseContainer appext.VerboseContainer
}

func newContainer(
	baseContainer app.Container,
	appName string,
	logger *zap.Logger,
	verbosePrinter verbose.Printer,
) (*container, error) {
	nameContainer, err := appext.NewNameContainer(baseContainer, appName)
	if err != nil {
		return nil, err
	}
	return &container{
		Container:        baseContainer,
		nameContainer:    nameContainer,
		logContainer:     appext.NewLoggerContainer(logger),
		tracerContainer:  appext.NewTracerContainer(appName),
		verboseContainer: appext.NewVerboseContainer(verbosePrinter),
	}, nil
}

func (c *container) AppName() string {
	return c.nameContainer.AppName()
}

func (c *container) ConfigDirPath() string {
	return c.nameContainer.ConfigDirPath()
}

func (c *container) CacheDirPath() string {
	return c.nameContainer.CacheDirPath()
}

func (c *container) DataDirPath() string {
	return c.nameContainer.DataDirPath()
}

func (c *container) Port() (uint16, error) {
	return c.nameContainer.Port()
}

func (c *container) Logger() *zap.Logger {
	return c.logContainer.Logger()
}

func (c *container) Tracer() trace.Tracer {
	return c.tracerContainer.Tracer()
}

func (c *container) VerbosePrinter() verbose.Printer {
	return c.verboseContainer.VerbosePrinter()
}

func newEnvFunc(tb testing.TB, cacheDir string) func(string) map[string]string {
	tb.Helper()
	if cacheDir == "" {
		cacheDir = tb.TempDir()
	}
	return func(use string) map[string]string {
		return map[string]string{
			useEnvVar(use, "CACHE_DIR"): cacheDir,
			useEnvVar(use, "HOME"):      tb.TempDir(),
			"PATH":                      os.Getenv("PATH"),
		}
	}
}

func useEnvVar(use string, suffix string) string {
	return strings.ToUpper(use) + "_" + suffix
}
