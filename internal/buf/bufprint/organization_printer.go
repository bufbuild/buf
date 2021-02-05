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

type organizationPrinter struct {
	address string
	writer  io.Writer
	asJSON  bool
}

func newOrganizationPrinter(
	address string,
	writer io.Writer,
	asJSON bool,
) *organizationPrinter {
	return &organizationPrinter{
		address: address,
		writer:  writer,
		asJSON:  asJSON,
	}
}

func (p *organizationPrinter) PrintOrganizations(ctx context.Context, messages ...*registryv1alpha1.Organization) error {
	if len(messages) == 0 {
		return nil
	}
	var outputOrganizations []outputOrganization
	for _, organization := range messages {
		outputOrganization := outputOrganization{
			ID:         organization.Id,
			Remote:     p.address,
			Name:       organization.Name,
			CreateTime: organization.CreateTime.AsTime(),
		}
		outputOrganizations = append(outputOrganizations, outputOrganization)
	}
	if p.asJSON {
		return p.printOrganizationsJSON(outputOrganizations)
	}
	return p.printOrganizationsText(outputOrganizations)
}

func (p *organizationPrinter) printOrganizationsJSON(outputOrganizations []outputOrganization) error {
	encoder := json.NewEncoder(p.writer)
	for _, outputOrganization := range outputOrganizations {
		if err := encoder.Encode(outputOrganization); err != nil {
			return err
		}
	}
	return nil
}

func (p *organizationPrinter) printOrganizationsText(outputOrganizations []outputOrganization) error {
	return WithTabWriter(
		p.writer,
		[]string{
			"ID",
			"Full name",
			"Created",
		},
		func(tabWriter TabWriter) error {
			for _, outputOrganization := range outputOrganizations {
				if err := tabWriter.Write(
					outputOrganization.ID,
					outputOrganization.Remote+"/"+outputOrganization.Name,
					outputOrganization.CreateTime.Format(time.RFC3339),
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

type outputOrganization struct {
	ID         string    `json:"id,omitempty"`
	Remote     string    `json:"remote,omitempty"`
	Name       string    `json:"name,omitempty"`
	CreateTime time.Time `json:"create_time,omitempty"`
}
