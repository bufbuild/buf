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
	"encoding/json"
	"fmt"
	"io"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	"github.com/bufbuild/buf/private/pkg/slicesext"
)

type repositoryDraftPrinter struct {
	writer io.Writer
}

func newRepositoryDraftPrinter(
	writer io.Writer,
) *repositoryDraftPrinter {
	return &repositoryDraftPrinter{
		writer: writer,
	}
}

type outputRepositoryDraft struct {
	Name   string `json:"name,omitempty"`
	Commit string `json:"commit,omitempty"`
}

func (p *repositoryDraftPrinter) PrintRepositoryDrafts(ctx context.Context, format Format, nextPageToken string, drafts ...*modulev1.Label) error {
	if len(drafts) == 0 {
		return nil
	}
	outputRepositoryDrafs := slicesext.Map(
		drafts,
		func(label *modulev1.Label) outputRepositoryDraft {
			return outputRepositoryDraft{
				Name:   label.GetName(),
				Commit: label.GetCommitId(),
			}
		},
	)
	switch format {
	case FormatText:
		return p.printRepositoryDraftsText(outputRepositoryDrafs)
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(paginationWrapper{
			NextPage: nextPageToken,
			Results:  outputRepositoryDrafs,
		})
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *repositoryDraftPrinter) printRepositoryDraftsText(outputRepositoryDrafts []outputRepositoryDraft) error {
	return WithTabWriter(
		p.writer,
		[]string{
			"Name",
			"Commit",
		},
		func(tabWriter TabWriter) error {
			for _, draft := range outputRepositoryDrafts {
				if err := tabWriter.Write(
					draft.Name,
					draft.Commit,
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}
