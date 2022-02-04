// Copyright 2020-2022 Buf Technologies, Inc.
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
	"encoding/json"
	"fmt"
	"io"

	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

type repositoryTrackCommitPrinter struct {
	writer io.Writer
}

func newRepositoryTrackCommitPrinter(writer io.Writer) *repositoryTrackCommitPrinter {
	return &repositoryTrackCommitPrinter{
		writer: writer,
	}
}

func (p *repositoryTrackCommitPrinter) PrintRepositoryTrackCommit(format Format, message *registryv1alpha1.RepositoryTrackCommit) error {
	outputCommit := registryRepositoryTrackCommitToOutputRepositoryTrackCommit(message)
	switch format {
	case FormatText:
		return p.printRepositoryTrackCommitsText([]outputRepositoryTrackCommit{outputCommit})
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(outputCommit)
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *repositoryTrackCommitPrinter) PrintRepositoryTrackCommits(
	format Format,
	nextPageToken string,
	messages ...*registryv1alpha1.RepositoryTrackCommit,
) error {
	if len(messages) == 0 {
		return nil
	}
	outputCommits := make([]outputRepositoryTrackCommit, len(messages))
	for i, message := range messages {
		outputCommits[i] = registryRepositoryTrackCommitToOutputRepositoryTrackCommit(message)
	}
	switch format {
	case FormatText:
		return p.printRepositoryTrackCommitsText(outputCommits)
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(paginationWrapper{
			NextPage: nextPageToken,
			Results:  outputCommits,
		})
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *repositoryTrackCommitPrinter) printRepositoryTrackCommitsText(output []outputRepositoryTrackCommit) error {
	return WithTabWriter(
		p.writer,
		[]string{"Commit", "Track ID"},
		func(tabWriter TabWriter) error {
			for _, outputCommit := range output {
				if err := tabWriter.Write(
					outputCommit.RepositoryCommit,
					outputCommit.RepositoryTrackID,
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

type outputRepositoryTrackCommit struct {
	RepositoryCommit  string `json:"commit,omitempty"`
	RepositoryTrackID string `json:"track_id,omitempty"`
}

func registryRepositoryTrackCommitToOutputRepositoryTrackCommit(message *registryv1alpha1.RepositoryTrackCommit) outputRepositoryTrackCommit {
	return outputRepositoryTrackCommit{
		RepositoryCommit:  message.GetRepositoryCommit(),
		RepositoryTrackID: message.GetRepositoryTrackId(),
	}
}
