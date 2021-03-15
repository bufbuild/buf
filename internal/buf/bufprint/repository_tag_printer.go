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
	"encoding/json"
	"io"
	"time"

	registryv1alpha1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/alpha/registry/v1alpha1"
)

type repositoryTagPrinter struct {
	writer io.Writer
	asJSON bool
}

func newRepositoryTagPrinter(
	writer io.Writer,
	asJSON bool,
) *repositoryTagPrinter {
	return &repositoryTagPrinter{
		writer: writer,
		asJSON: asJSON,
	}
}

func (p *repositoryTagPrinter) PrintRepositoryTags(ctx context.Context, messages ...*registryv1alpha1.RepositoryTag) error {
	if len(messages) == 0 {
		return nil
	}
	var outputRepositoryTags []outputRepositoryTag
	for _, repositoryTag := range messages {
		outputRepositoryTag := outputRepositoryTag{
			ID:         repositoryTag.Id,
			Name:       repositoryTag.Name,
			Commit:     repositoryTag.CommitName,
			CreateTime: repositoryTag.CreateTime.AsTime(),
		}
		outputRepositoryTags = append(outputRepositoryTags, outputRepositoryTag)
	}
	if p.asJSON {
		return p.printRepositoryTagsJSON(outputRepositoryTags)
	}
	return p.printRepositoryTagsText(outputRepositoryTags)
}

func (p *repositoryTagPrinter) printRepositoryTagsJSON(outputRepositoryTags []outputRepositoryTag) error {
	encoder := json.NewEncoder(p.writer)
	for _, outputRepositoryTag := range outputRepositoryTags {
		if err := encoder.Encode(outputRepositoryTag); err != nil {
			return err
		}
	}
	return nil
}

func (p *repositoryTagPrinter) printRepositoryTagsText(outputRepositoryTags []outputRepositoryTag) error {
	return WithTabWriter(
		p.writer,
		[]string{
			"ID",
			"Name",
			"Commit",
			"Created",
		},
		func(tabWriter TabWriter) error {
			for _, outputRepositoryTag := range outputRepositoryTags {
				if err := tabWriter.Write(
					outputRepositoryTag.ID,
					outputRepositoryTag.Name,
					outputRepositoryTag.Commit,
					outputRepositoryTag.CreateTime.Format(time.RFC3339),
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

type outputRepositoryTag struct {
	ID         string    `json:"id,omitempty"`
	Name       string    `json:"name,omitempty"`
	Commit     string    `json:"commit,omitempty"`
	CreateTime time.Time `json:"create_time,omitempty"`
}
