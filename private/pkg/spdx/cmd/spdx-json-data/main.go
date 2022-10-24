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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/spdx"
	"github.com/spf13/cobra"
)

const (
	programName = "spdx-json-data"
)

func main() {
	appcmd.Main(context.Background(), newCommand())
}

func newCommand() *appcmd.Command {
	return &appcmd.Command{
		Use:  programName,
		Args: cobra.NoArgs,
		Run:  run,
	}
}

func run(ctx context.Context, container app.Container) error {
	licenseInfos, err := spdx.GetLicenseInfos(ctx)
	if err != nil {
		return err
	}
	data, err := getJSONFileData(ctx, licenseInfos)
	if err != nil {
		return err
	}
	_, err = container.Stdout().Write(data)
	return err
}

func getJSONFileData(
	ctx context.Context,
	licenseInfos []*spdx.LicenseInfo,
) ([]byte, error) {
	licenses := make(map[string]licenseInfo)
	for _, info := range licenseInfos {
		detailsFileName := strings.TrimPrefix(info.DetailsURL, "./")
		licenses[strings.ToLower(info.ID)] = licenseInfo{
			ID:         info.ID,
			Name:       info.Name,
			DetailsURL: fmt.Sprintf("https://spdx.org/licenses/%s", detailsFileName),
		}
	}
	return json.MarshalIndent(licenses, "", "\t")
}

type licenseInfo struct {
	ID         string `json:"licenseId,omitempty" yaml:"licenseId,omitempty"`
	Name       string `json:"name,omitempty" yaml:"name,omitempty"`
	DetailsURL string `json:"detailsUrl,omitempty" yaml:"detailsUrl,omitempty"`
}
