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
	PrintOrganization(ctx context.Context, format Format, organization *registryv1alpha1.Organization) error
}

// NewOrganizationPrinter returns a new OrganizationPrinter.
func NewOrganizationPrinter(address string, writer io.Writer) OrganizationPrinter {
	return newOrganizationPrinter(address, writer)
}

// RepositoryPrinter is a repository printer.
type RepositoryPrinter interface {
	PrintRepository(ctx context.Context, format Format, repository *registryv1alpha1.Repository) error
	PrintRepositories(ctx context.Context, format Format, nextPageToken string, repositories ...*registryv1alpha1.Repository) error
}

// NewRepositoryPrinter returns a new RepositoryPrinter.
func NewRepositoryPrinter(
	clientConfig *connectclient.Config,
	address string,
	writer io.Writer,
) RepositoryPrinter {
	return newRepositoryPrinter(clientConfig, address, writer)
}

// RepositoryTagPrinter is a repository tag printer.
type RepositoryTagPrinter interface {
	PrintRepositoryTag(ctx context.Context, format Format, repositoryTag *registryv1alpha1.RepositoryTag) error
	PrintRepositoryTags(ctx context.Context, format Format, nextPageToken string, repositoryTags ...*registryv1alpha1.RepositoryTag) error
}

// NewRepositoryTagPrinter returns a new RepositoryTagPrinter.
func NewRepositoryTagPrinter(writer io.Writer) RepositoryTagPrinter {
	return newRepositoryTagPrinter(writer)
}

// RepositoryCommitPrinter is a repository commit printer.
type RepositoryCommitPrinter interface {
	PrintRepositoryCommit(ctx context.Context, format Format, repositoryCommit *registryv1alpha1.RepositoryCommit) error
	PrintRepositoryCommits(ctx context.Context, format Format, nextPageToken string, repositoryCommits ...*registryv1alpha1.RepositoryCommit) error
}

// NewRepositoryCommitPrinter returns a new RepositoryCommitPrinter.
func NewRepositoryCommitPrinter(writer io.Writer) RepositoryCommitPrinter {
	return newRepositoryCommitPrinter(writer)
}

// RepositoryDraftPrinter is a repository draft printer.
type RepositoryDraftPrinter interface {
	PrintRepositoryDraft(ctx context.Context, format Format, repositoryCommit *registryv1alpha1.RepositoryCommit) error
	PrintRepositoryDrafts(ctx context.Context, format Format, nextPageToken string, repositoryCommits ...*registryv1alpha1.RepositoryCommit) error
}

// NewRepositoryDraftPrinter returns a new RepositoryDraftPrinter.
func NewRepositoryDraftPrinter(writer io.Writer) RepositoryDraftPrinter {
	return newRepositoryDraftPrinter(writer)
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
