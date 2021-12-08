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
	"fmt"
	"io"
	"time"

	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

type repositoryPrinter struct {
	address string
	writer  io.Writer
}

func newRepositoryPrinter(
	address string,
	writer io.Writer,
) *repositoryPrinter {
	return &repositoryPrinter{
		address: address,
		writer:  writer,
	}
}

func (p *repositoryPrinter) PrintRepository(ctx context.Context, format Format, message *registryv1alpha1.Repository) error {
	outputRepositories, err := p.registryRepositoriesToOutRepositories(ctx, message)
	if err != nil {
		return err
	}
	if len(outputRepositories) != 1 {
		return fmt.Errorf("error converting repositories: expected 1 got %d", len(outputRepositories))
	}
	switch format {
	case FormatText:
		return p.printRepositoriesText(outputRepositories)
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(outputRepositories[0])
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *repositoryPrinter) PrintRepositories(ctx context.Context, format Format, nextPageToken string, messages ...*registryv1alpha1.Repository) error {
	if len(messages) == 0 {
		return nil
	}
	outputRepositories, err := p.registryRepositoriesToOutRepositories(ctx, messages...)
	if err != nil {
		return err
	}
	switch format {
	case FormatText:
		return p.printRepositoriesText(outputRepositories)
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(paginationWrapper{
			NextPage: nextPageToken,
			Results:  outputRepositories,
		})
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *repositoryPrinter) registryRepositoriesToOutRepositories(ctx context.Context, messages ...*registryv1alpha1.Repository) ([]outputRepository, error) {
	var outputRepositories []outputRepository
	for _, repository := range messages {
		outputRepositories = append(outputRepositories, outputRepository{
			ID:         repository.Id,
			Remote:     p.address,
			Owner:      repository.OwnerName,
			Name:       repository.Name,
			CreateTime: repository.CreateTime.AsTime(),
		})
	}
	return outputRepositories, nil
}

func (p *repositoryPrinter) printRepositoriesText(outputRepositories []outputRepository) error {
	return WithTabWriter(
		p.writer,
		[]string{
			"Full name",
			"Created",
		},
		func(tabWriter TabWriter) error {
			for _, outputRepository := range outputRepositories {
				if err := tabWriter.Write(
					outputRepository.Remote+"/"+outputRepository.Owner+"/"+outputRepository.Name,
					outputRepository.CreateTime.Format(time.RFC3339),
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

type outputRepository struct {
	ID         string    `json:"id,omitempty"`
	Remote     string    `json:"remote,omitempty"`
	Owner      string    `json:"owner,omitempty"`
	Name       string    `json:"name,omitempty"`
	CreateTime time.Time `json:"create_time,omitempty"`
}
