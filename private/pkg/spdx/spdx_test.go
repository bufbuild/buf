// Copyright 2020-2023 Buf Technologies, Inc.
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

package spdx_test

import (
	"context"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/spdx"
	"github.com/stretchr/testify/require"
)

var (
	bannedLowercaseIDs = []string{
		"custom",
		"none",
	}
)

func TestGetLicenseInfos(t *testing.T) {
	ctx := context.Background()

	licenseInfos, err := spdx.GetLicenseInfos(ctx)
	require.NoError(t, err)

	// Check that there are license infos
	require.NotEmpty(t, licenseInfos)

	// Check that banned lowercase IDs are not present
	for _, bannedID := range bannedLowercaseIDs {
		for _, licenseInfo := range licenseInfos {
			require.NotEqual(t, strings.ToLower(licenseInfo.ID), bannedID)
		}
	}

	// Check that IDs are sorted in lowercase order
	for i := 1; i < len(licenseInfos); i++ {
		require.True(t, strings.ToLower(licenseInfos[i-1].ID) < strings.ToLower(licenseInfos[i].ID))
	}
}

func TestGetLicenseInfosWithError(t *testing.T) {
	ctx := context.Background()

	// Update package path to match new import path
	licenseInfos, err := spdx.GetLicenseInfos(ctx)
	if err != nil {
		t.Fatalf("GetLicenseInfos returned unexpected error: %v", err)
	}

	if len(licenseInfos) == 0 {
		t.Errorf("GetLicenseInfos returned an empty list of licenses")
	}
}
