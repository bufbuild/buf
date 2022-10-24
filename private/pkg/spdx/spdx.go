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

package spdx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"go.uber.org/multierr"
)

const (
	licenseListURL = "https://raw.githubusercontent.com/spdx/license-list-data/v3.12/json/licenses.json"
)

var (
	httpClient = &http.Client{}

	bannedLowercaseIDs = []string{
		"custom",
		"none",
	}
)

// GetLicenseInfos get the SPDX license information,
// sorted by lowercase of ID and bans "custom", "none"
func GetLicenseInfos(ctx context.Context) (_ []*LicenseInfo, retErr error) {
	request, err := http.NewRequestWithContext(ctx, "GET", licenseListURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, response.Body.Close())
	}()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP status code %d to be %d", response.StatusCode, http.StatusOK)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	licenseInfoList := &licenseInfoList{}
	if err := json.Unmarshal(data, licenseInfoList); err != nil {
		return nil, err
	}
	lowercaseIDMap := make(map[string]struct{})
	for _, licenseInfo := range licenseInfoList.LicenseInfos {
		lowercaseID := strings.ToLower(licenseInfo.ID)
		if _, ok := lowercaseIDMap[lowercaseID]; ok {
			return nil, fmt.Errorf("duplicate lowercase ID: %q", lowercaseID)
		}
		lowercaseIDMap[lowercaseID] = struct{}{}
	}
	for _, bannedLowercaseID := range bannedLowercaseIDs {
		if _, ok := lowercaseIDMap[bannedLowercaseID]; ok {
			return nil, fmt.Errorf("banned lowercase ID: %q", bannedLowercaseID)
		}
	}
	sort.Slice(
		licenseInfoList.LicenseInfos,
		func(i int, j int) bool {
			return strings.ToLower(licenseInfoList.LicenseInfos[i].ID) <
				strings.ToLower(licenseInfoList.LicenseInfos[j].ID)
		},
	)
	return licenseInfoList.LicenseInfos, nil
}

type licenseInfoList struct {
	LicenseInfos []*LicenseInfo `json:"licenses,omitempty" yaml:"licenses,omitempty"`
}

type LicenseInfo struct {
	ID         string `json:"licenseId,omitempty" yaml:"licenseId,omitempty"`
	Name       string `json:"name,omitempty" yaml:"name,omitempty"`
	DetailsURL string `json:"detailsUrl,omitempty" yaml:"detailsUrl,omitempty"`
}
