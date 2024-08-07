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

package bufprint

import (
	"context"
	"fmt"
	"io"
	"strconv"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/protostat"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"go.uber.org/multierr"
	"google.golang.org/protobuf/proto"
)

const (
	// FormatText is the text format.
	FormatText Format = 1
	// FormatJSON is the JSON format.
	FormatJSON Format = 2
)

var (
	// AllFormatsString is the string representation of all Formats.
	AllFormatsString = stringutil.SliceToString([]string{FormatText.String(), FormatJSON.String()})
)

// Format is a format to print.
type Format int

// ParseFormat parses the format.
//
// If the empty string is provided, this is interpreted as FormatText.
func ParseFormat(s string) (Format, error) {
	switch s {
	case "", "text":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	default:
		return 0, fmt.Errorf("unknown format: %s", s)
	}
}

// String implements fmt.Stringer.
func (f Format) String() string {
	switch f {
	case FormatText:
		return "text"
	case FormatJSON:
		return "json"
	default:
		return strconv.Itoa(int(f))
	}
}

// CuratedPluginPrinter is a printer for curated plugins.
type CuratedPluginPrinter interface {
	PrintCuratedPlugin(ctx context.Context, format Format, plugin *registryv1alpha1.CuratedPlugin) error
	PrintCuratedPlugins(ctx context.Context, format Format, nextPageToken string, plugins ...*registryv1alpha1.CuratedPlugin) error
}

// NewCuratedPluginPrinter returns a new CuratedPluginPrinter.
func NewCuratedPluginPrinter(writer io.Writer) CuratedPluginPrinter {
	return newCuratedPluginPrinter(writer)
}

// OrganizationPrinter is an organization printer.
type OrganizationPrinter interface {
	PrintOrganizationInfo(ctx context.Context, format Format, organization *ownerv1.Organization) error
}

// NewOrganizationPrinter returns a new OrganizationPrinter.
func NewOrganizationPrinter(address string, writer io.Writer) OrganizationPrinter {
	return newOrganizationPrinter(address, writer)
}

// ModulePrinter is a module printer.
type ModulePrinter interface {
	PrintModuleInfo(ctx context.Context, format Format, repository *modulev1.Module) error
}

// NewModulePrinter returns a new ModulePrinter.
func NewModulePrinter(
	clientConfig *connectclient.Config,
	address string,
	writer io.Writer,
) ModulePrinter {
	return newModulePrinter(clientConfig, address, writer)
}

// LabelPrinter is a repository label printer.
type LabelPrinter interface {
	// PrintLabels prints each label on a new line.
	PrintLabels(ctx context.Context, format Format, label ...*modulev1.Label) error
	// PrintLabels prints information about a label.
	PrintLabelInfo(ctx context.Context, format Format, label *modulev1.Label) error
	// PrintLabelPage prints a page of labels.
	PrintLabelPage(ctx context.Context, format Format, nextPageCommand, nextPageToken string, labels []*modulev1.Label) error
}

// NewLabelPrinter returns a new RepositoryLabelPrinter.
func NewLabelPrinter(writer io.Writer, moduleFullName bufmodule.ModuleFullName) LabelPrinter {
	return newLabelPrinter(writer, moduleFullName)
}

// CommitPrinter is a commit printer.
type CommitPrinter interface {
	PrintCommitInfo(ctx context.Context, format Format, commit *modulev1.Commit) error
	PrintCommits(ctx context.Context, format Format, commits ...*modulev1.Commit) error
	PrintCommitPage(ctx context.Context, format Format, nextPageCommand, nextPageToken string, commits []*modulev1.Commit) error
}

// NewCommitPrinter returns a new RepositoryCommitPrinter.
func NewCommitPrinter(writer io.Writer, moduleFullName bufmodule.ModuleFullName) CommitPrinter {
	return newCommitPrinter(writer, moduleFullName)
}

// TokenPrinter is a token printer.
//
// TODO: update to same format as other printers.
type TokenPrinter interface {
	PrintTokens(ctx context.Context, tokens ...*registryv1alpha1.Token) error
}

// NewTokenPrinter returns a new TokenPrinter.
//
// TODO: update to same format as other printers.
func NewTokenPrinter(writer io.Writer, format Format) (TokenPrinter, error) {
	switch format {
	case FormatText:
		return newTokenTextPrinter(writer), nil
	case FormatJSON:
		return newTokenJSONPrinter(writer), nil
	default:
		return nil, fmt.Errorf("unknown format: %v", format)
	}
}

// StatsPrinter is a printer of Stats.
type StatsPrinter interface {
	PrintStats(ctx context.Context, format Format, stats *protostat.Stats) error
}

// NewStatsPrinter returns a new StatsPrinter.
func NewStatsPrinter(writer io.Writer) StatsPrinter {
	return newStatsPrinter(writer)
}

// TabWriter is a tab writer.
type TabWriter interface {
	Write(values ...string) error
}

// WithTabWriter calls a function with a TabWriter.
//
// Shared with internal packages.
func WithTabWriter(
	writer io.Writer,
	header []string,
	f func(TabWriter) error,
) (retErr error) {
	tabWriter := newTabWriter(writer)
	defer func() {
		retErr = multierr.Append(retErr, tabWriter.Flush())
	}()
	if err := tabWriter.Write(header...); err != nil {
		return err
	}
	return f(tabWriter)
}

// printProtoMessageJSON prints the Protobuf message as JSON.
func printProtoMessageJSON(writer io.Writer, message proto.Message) error {
	data, err := protoencoding.NewJSONMarshaler(nil, protoencoding.JSONMarshalerWithIndent()).Marshal(message)
	if err != nil {
		return err
	}
	_, err = writer.Write(append(data, []byte("\n")...))
	return err
}
