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

type repositoryBranchPrinter struct {
	writer io.Writer
	asJSON bool
}

func newRepositoryBranchPrinter(
	writer io.Writer,
	asJSON bool,
) *repositoryBranchPrinter {
	return &repositoryBranchPrinter{
		writer: writer,
		asJSON: asJSON,
	}
}

func (p *repositoryBranchPrinter) PrintRepositoryBranches(ctx context.Context, messages ...*registryv1alpha1.RepositoryBranch) error {
	if len(messages) == 0 {
		return nil
	}
	var outputRepositoryBranches []outputRepositoryBranch
	for _, repositoryBranch := range messages {
		outputRepositoryBranch := outputRepositoryBranch{
			ID:         repositoryBranch.Id,
			Name:       repositoryBranch.Name,
			CreateTime: repositoryBranch.CreateTime.AsTime(),
		}
		outputRepositoryBranches = append(outputRepositoryBranches, outputRepositoryBranch)
	}
	if p.asJSON {
		return p.printRepositoryBranchesJSON(outputRepositoryBranches)
	}
	return p.printRepositoryBranchesText(outputRepositoryBranches)
}

func (p *repositoryBranchPrinter) printRepositoryBranchesJSON(outputRepositoryBranches []outputRepositoryBranch) error {
	encoder := json.NewEncoder(p.writer)
	for _, outputRepositoryBranch := range outputRepositoryBranches {
		if err := encoder.Encode(outputRepositoryBranch); err != nil {
			return err
		}
	}
	return nil
}

func (p *repositoryBranchPrinter) printRepositoryBranchesText(outputRepositoryBranches []outputRepositoryBranch) error {
	return WithTabWriter(
		p.writer,
		[]string{
			"ID",
			"Name",
			"Created",
		},
		func(tabWriter TabWriter) error {
			for _, outputRepositoryBranch := range outputRepositoryBranches {
				if err := tabWriter.Write(
					outputRepositoryBranch.ID,
					outputRepositoryBranch.Name,
					outputRepositoryBranch.CreateTime.Format(time.RFC3339),
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

type outputRepositoryBranch struct {
	ID         string    `json:"id,omitempty"`
	Name       string    `json:"name,omitempty"`
	CreateTime time.Time `json:"create_time,omitempty"`
}
