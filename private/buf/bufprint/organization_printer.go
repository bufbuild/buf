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

	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
)

type organizationPrinter struct {
	address string
	writer  io.Writer
}

func newOrganizationPrinter(
	address string,
	writer io.Writer,
) *organizationPrinter {
	return &organizationPrinter{
		address: address,
		writer:  writer,
	}
}

func (p *organizationPrinter) PrintOrganization(ctx context.Context, format Format, organization *ownerv1.Organization) error {
	outOrganization := registryOrganizationToOutputOrganization(p.address, organization)
	switch format {
	case FormatText:
		return p.printOrganizationsText([]outputOrganization{outOrganization})
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(outOrganization)
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *organizationPrinter) PrintOrganizations(ctx context.Context, format Format, nextPageToken string, organizations ...*ownerv1.Organization) error {
	if len(organizations) == 0 {
		return nil
	}
	var outputOrganizations []outputOrganization
	for _, organization := range organizations {
		outputOrganization := registryOrganizationToOutputOrganization(p.address, organization)
		outputOrganizations = append(outputOrganizations, outputOrganization)
	}
	switch format {
	case FormatText:
		return p.printOrganizationsText(outputOrganizations)
	case FormatJSON:
		return json.NewEncoder(p.writer).Encode(paginationWrapper{
			NextPage: nextPageToken,
			Results:  outputOrganizations,
		})
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (p *organizationPrinter) printOrganizationsText(outputOrganizations []outputOrganization) error {
	return WithTabWriter(
		p.writer,
		[]string{
			"Full Name",
			"Create Time",
		},
		func(tabWriter TabWriter) error {
			for _, outputOrganization := range outputOrganizations {
				if err := tabWriter.Write(
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

func registryOrganizationToOutputOrganization(address string, organization *ownerv1.Organization) outputOrganization {
	return outputOrganization{
		ID:         organization.Id,
		Remote:     address,
		Name:       organization.Name,
		CreateTime: organization.CreateTime.AsTime(),
	}
}
