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
	"time"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
)

type repositoryCommitPrinter struct {
	writer io.Writer
}

func newRepositoryCommitPrinter(
	writer io.Writer,
) *repositoryCommitPrinter {
	return &repositoryCommitPrinter{
		writer: writer,
	}
}

func (p *repositoryCommitPrinter) PrintRepositoryCommit(ctx context.Context, format Format, repositoryCommit *modulev1.Commit) error {
	outCommit := registryCommitToOutputCommit(repositoryCommit)
	switch format {
	case FormatText:
		return p.printRepositoryCommitsText([]outputRepositoryCommit{outCommit})
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(outCommit)
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *repositoryCommitPrinter) PrintRepositoryCommits(ctx context.Context, format Format, nextPageToken string, repositoryCommits ...*modulev1.Commit) error {
	if len(repositoryCommits) == 0 {
		return nil
	}
	var outputRepositoryCommits []outputRepositoryCommit
	for _, repositoryCommit := range repositoryCommits {
		outputRepositoryCommit := registryCommitToOutputCommit(repositoryCommit)
		outputRepositoryCommits = append(outputRepositoryCommits, outputRepositoryCommit)
	}
	switch format {
	case FormatText:
		return p.printRepositoryCommitsText(outputRepositoryCommits)
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(paginationWrapper{
			NextPage: nextPageToken,
			Results:  outputRepositoryCommits,
		})
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *repositoryCommitPrinter) printRepositoryCommitsText(outputRepositoryCommits []outputRepositoryCommit) error {
	return WithTabWriter(
		p.writer,
		[]string{
			"Commit",
			"Create Time", // TODO: this should be a constant
		},
		func(tabWriter TabWriter) error {
			for _, outputRepositoryCommit := range outputRepositoryCommits {
				if err := tabWriter.Write(
					outputRepositoryCommit.Commit,
					outputRepositoryCommit.CreateTime.Format(time.RFC3339),
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

type outputRepositoryCommit struct {
	Commit     string    `json:"commit,omitempty"`
	CreateTime time.Time `json:"create_time,omitempty"`
}

func registryCommitToOutputCommit(repositoryCommit *modulev1.Commit) outputRepositoryCommit {
	return outputRepositoryCommit{
		Commit:     repositoryCommit.Id,
		CreateTime: repositoryCommit.CreateTime.AsTime(),
	}
}
