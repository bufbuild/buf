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
	"github.com/bufbuild/buf/private/pkg/slicesext"
)

type repositoryLabelPrinter struct {
	writer io.Writer
}

func newRepositoryLabelPrinter(
	writer io.Writer,
) *repositoryLabelPrinter {
	return &repositoryLabelPrinter{
		writer: writer,
	}
}

func (p *repositoryLabelPrinter) PrintRepositoryLabel(ctx context.Context, format Format, message *modulev1.Label) error {
	outRepositoryLabel := registryLabelToOutputLabel(message)
	switch format {
	case FormatText:
		return p.printRepositoryLabelsText([]outputRepositoryLabel{outRepositoryLabel})
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(outRepositoryLabel)
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *repositoryLabelPrinter) PrintRepositoryLabels(ctx context.Context, format Format, nextPageToken string, messages ...*modulev1.Label) error {
	if len(messages) == 0 {
		return nil
	}
	outputRepositoryLabels := registryLabelsToOutputLabels(messages)
	switch format {
	case FormatText:
		return p.printRepositoryLabelsText(outputRepositoryLabels)
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(paginationWrapper{
			NextPage: nextPageToken,
			Results:  outputRepositoryLabels,
		})
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *repositoryLabelPrinter) printRepositoryLabelsText(outputRepositoryLabels []outputRepositoryLabel) error {
	archivedLabelCount := slicesext.Count(outputRepositoryLabels, func(label outputRepositoryLabel) bool {
		return label.ArchiveTime != nil
	})
	if archivedLabelCount == 0 {
		return WithTabWriter(
			p.writer,
			[]string{
				"Name",
				"Commit",
				"Create Time",
			},
			func(tabWriter TabWriter) error {
				for _, outputRepositoryLabel := range outputRepositoryLabels {
					if err := tabWriter.Write(
						outputRepositoryLabel.Name,
						outputRepositoryLabel.Commit,
						outputRepositoryLabel.CreateTime.Format(time.RFC3339),
					); err != nil {
						return err
					}
				}
				return nil
			},
		)
	}
	return WithTabWriter(
		p.writer,
		[]string{
			"Name",
			"Commit",
			"Create Time",
			"Archive Time",
		},
		func(tabWriter TabWriter) error {
			for _, outputRepositoryLabel := range outputRepositoryLabels {
				formattedArchiveTime := ""
				if outputRepositoryLabel.ArchiveTime != nil {
					formattedArchiveTime = outputRepositoryLabel.ArchiveTime.Format(time.RFC3339)
				}
				if err := tabWriter.Write(
					outputRepositoryLabel.Name,
					outputRepositoryLabel.Commit,
					outputRepositoryLabel.CreateTime.Format(time.RFC3339),
					formattedArchiveTime,
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

type outputRepositoryLabel struct {
	Name        string     `json:"name,omitempty"`
	Commit      string     `json:"commit,omitempty"`
	CreateTime  time.Time  `json:"create_time,omitempty"`
	ArchiveTime *time.Time `json:"archive_time,omitempty"`
}

func registryLabelToOutputLabel(repositoryLabel *modulev1.Label) outputRepositoryLabel {
	var archiveTime *time.Time
	if repositoryLabel.ArchiveTime != nil {
		timeValue := repositoryLabel.ArchiveTime.AsTime()
		archiveTime = &timeValue
	}
	return outputRepositoryLabel{
		Name:        repositoryLabel.Name,
		Commit:      repositoryLabel.CommitId,
		CreateTime:  repositoryLabel.CreateTime.AsTime(),
		ArchiveTime: archiveTime,
	}
}

func registryLabelsToOutputLabels(labels []*modulev1.Label) []outputRepositoryLabel {
	outputRepositoryLabels := make([]outputRepositoryLabel, len(labels))
	for i, repositoryLabel := range labels {
		outputRepositoryLabels[i] = registryLabelToOutputLabel(repositoryLabel)
	}
	return outputRepositoryLabels
}
