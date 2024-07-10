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
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
)

type commitPrinter struct {
	writer         io.Writer
	moduleFullName bufmodule.ModuleFullName
}

func newCommitPrinter(
	writer io.Writer,
	moduleFullName bufmodule.ModuleFullName,
) *commitPrinter {
	return &commitPrinter{
		writer:         writer,
		moduleFullName: moduleFullName,
	}
}

func (p *commitPrinter) PrintCommitInfo(ctx context.Context, format Format, commit *modulev1.Commit) error {
	outCommit := commitToOutputCommit(commit)
	switch format {
	case FormatText:
		return p.printCommitsInfo([]outputCommit{outCommit})
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(outCommit)
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *commitPrinter) PrintCommits(ctx context.Context, format Format, commits ...*modulev1.Commit) error {
	switch format {
	case FormatText:
		// Print a label name on each line.
		for _, commit := range commits {
			if _, err := fmt.Fprintf(p.writer, "%s:%s\n", p.moduleFullName, commit.Id); err != nil {
				return err
			}
		}
		return nil
	case FormatJSON:
		// Print a json object on each line.
		for _, commit := range commits {
			if err := json.NewEncoder(p.writer).Encode(
				commitToOutputCommit(commit),
			); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *commitPrinter) PrintCommitPage(ctx context.Context, format Format, nextPageToken string, commits []*modulev1.Commit) error {
	if len(commits) == 0 {
		return nil
	}
	var outputCommits []outputCommit
	for _, commit := range commits {
		outputCommit := commitToOutputCommit(commit)
		outputCommits = append(outputCommits, outputCommit)
	}
	switch format {
	case FormatText:
		return p.printCommitsInfo(outputCommits)
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(paginationWrapper{
			NextPage: nextPageToken,
			Results:  outputCommits,
		})
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *commitPrinter) printCommitsInfo(outputCommits []outputCommit) error {
	return WithTabWriter(
		p.writer,
		[]string{
			"Commit",
			"Create Time", // TODO: this should be a constant
		},
		func(tabWriter TabWriter) error {
			for _, outputCommit := range outputCommits {
				if err := tabWriter.Write(
					outputCommit.Commit,
					outputCommit.CreateTime.Format(time.RFC3339),
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

type outputCommit struct {
	Commit     string    `json:"commit,omitempty"`
	CreateTime time.Time `json:"create_time,omitempty"`
}

func commitToOutputCommit(commit *modulev1.Commit) outputCommit {
	return outputCommit{
		Commit:     commit.Id,
		CreateTime: commit.CreateTime.AsTime(),
	}
}
