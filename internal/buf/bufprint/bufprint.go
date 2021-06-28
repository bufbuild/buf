// Copyright 2020-2021 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/internal/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	registryv1alpha1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/internal/pkg/protoencoding"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
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
// If the empty string is provided, this is interpeted as FormatText.
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
	apiProvider registryv1alpha1apiclient.Provider,
	address string,
	writer io.Writer,
) RepositoryPrinter {
	return newRepositoryPrinter(apiProvider, address, writer)
}

// RepositoryBranchPrinter is a repository branch printer.
type RepositoryBranchPrinter interface {
	PrintRepositoryBranch(ctx context.Context, format Format, repositoryBranch *registryv1alpha1.RepositoryBranch) error
	PrintRepositoryBranches(ctx context.Context, format Format, nextPageToken string, repositoryBranches ...*registryv1alpha1.RepositoryBranch) error
}

// NewRepositoryBranchPrinter returns a new RepositoryBranchPrinter.
func NewRepositoryBranchPrinter(writer io.Writer) RepositoryBranchPrinter {
	return newRepositoryBranchPrinter(writer)
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

// PrintProtoMessageJSON prints the Protobuf message as JSON.
//
// Shared with internal packages.
func PrintProtoMessageJSON(writer io.Writer, message proto.Message) error {
	data, err := protoencoding.NewJSONMarshalerIndent(nil).Marshal(message)
	if err != nil {
		return err
	}
	_, err = writer.Write(append(data, []byte("\n")...))
	return err
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
