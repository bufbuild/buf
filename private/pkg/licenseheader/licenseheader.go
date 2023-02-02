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

// Package licenseheader handles license headers.
package licenseheader

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// LicenseTypeNone is the no license type.
	LicenseTypeNone LicenseType = iota + 1
	// LicenseTypeApache is the Apache 2.0 license type.
	LicenseTypeApache
	// LicenseTypeProprietary is the proprietary license type.
	LicenseTypeProprietary
)

var (
	licenseTypeToString = map[LicenseType]string{
		LicenseTypeNone:        "none",
		LicenseTypeApache:      "apache",
		LicenseTypeProprietary: "proprietary",
	}
	stringToLicenseType = map[string]LicenseType{
		"none":        LicenseTypeNone,
		"apache":      LicenseTypeApache,
		"proprietary": LicenseTypeProprietary,
	}

	// does not include LicenseTypeNone
	licenseTypeToTemplateData = map[LicenseType]string{
		LicenseTypeApache: `Copyright {{.YearRange}} {{.CopyrightHolder}}

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.`,
		LicenseTypeProprietary: `Copyright {{.YearRange}} {{.CopyrightHolder}}

All rights reserved.`,
	}

	// if a header contains any of these lowercase phrases, we consider it a license
	licenseMatchingPhrases = []string{
		"copyright",
	}

	extToPrefix = map[string]string{
		".go":    "//",
		".js":    "//",
		".proto": "//",
		".sql":   "---",
		".ts":    "//",
		".tsx":   "//",
		".bazel": "#",
		".bzl":   "#",
		".kt":    "//",
		".swift": "//",
		".java":  "//",
	}
)

// LicenseType is a recognized license type.
type LicenseType int

// String implements fmt.Stringer.
func (f LicenseType) String() string {
	s, ok := licenseTypeToString[f]
	if !ok {
		return strconv.Itoa(int(f))
	}
	return s
}

// ParseLicenseType parses the LicenseType.
func ParseLicenseType(s string) (LicenseType, error) {
	f, ok := stringToLicenseType[s]
	if ok {
		return f, nil
	}
	return 0, fmt.Errorf("unknown LicenseType: %q", s)
}

// Modify modifies the license header for the filename and data.
//
// Returns the modified data, or the unmodified data if no modifications.
// If the filename extension is not handled, returns the unmodified data.
//
// Note this only works with UTF-8 data with lines split by '\n'.
func Modify(
	licenseType LicenseType,
	copyrightHolder string,
	yearRange string,
	filename string,
	data []byte,
) ([]byte, error) {
	prefix, ok := getPrefix(filename)
	if !ok {
		// if we do not have a prefix, we do not know the filename extension,
		// so we just return the unmodified data
		return data, nil
	}
	remainder := getRemainder(string(data), prefix)
	licenseHeader, err := getLicenseHeader(licenseType, copyrightHolder, yearRange, prefix)
	if err != nil {
		return nil, err
	}
	if licenseHeader == "" {
		return []byte(remainder), nil
	}
	return []byte(licenseHeader + "\n\n" + remainder), nil
}

// getPrefix gets the comment prefix for the filename.
func getPrefix(filename string) (string, bool) {
	prefix, ok := extToPrefix[path.Ext(filepath.ToSlash(filename))]
	return prefix, ok
}

// getRemainder gets the remainder of the file beyond the license header.
func getRemainder(data string, prefix string) string {
	if len(data) == 0 {
		return ""
	}
	lines := strings.Split(data, "\n")
	var headerLines []string
	var lastCommentLine int
	for i, line := range lines {
		if !strings.HasPrefix(line, prefix) {
			lastCommentLine = i
			break
		}
		headerLines = append(headerLines, line)
	}
	// we have reached the first non-comment, check if the header lines represent a license or not
	if !doLinesContainALicense(headerLines) {
		// if they are not a license, return everything
		return data
	}
	// if they are, return everything but the license and with front newlines stripped
	return strings.TrimLeft(strings.Join(lines[lastCommentLine:], "\n"), "\n")
}

func getLicenseHeader(
	licenseType LicenseType,
	copyrightHolder string,
	yearRange string,
	prefix string,
) (string, error) {
	if licenseType == LicenseTypeNone {
		return "", nil
	}
	if copyrightHolder == "" {
		return "", errors.New("copyrightHolder required if not using LicenseTypeNone")
	}
	if yearRange == "" {
		return "", errors.New("yearRange required if not using LicenseTypeNone")
	}
	templateData, ok := licenseTypeToTemplateData[licenseType]
	if !ok {
		return "", fmt.Errorf("unrecognized license type: %q", licenseType)
	}
	tmpl, err := template.New("tmpl").Parse(templateData)
	if err != nil {
		return "", err
	}
	buffer := bytes.NewBuffer(nil)
	if err := tmpl.Execute(
		buffer,
		newLicenseData(
			copyrightHolder,
			yearRange,
		),
	); err != nil {
		return "", err
	}
	lines := strings.Split(buffer.String(), "\n")
	for i, line := range lines {
		if line == "" {
			lines[i] = prefix
		} else {
			lines[i] = prefix + " " + line
		}
	}
	return strings.Join(lines, "\n"), nil
}

func doLinesContainALicense(lines []string) bool {
	for _, line := range lines {
		for _, matchingPhrase := range licenseMatchingPhrases {
			if strings.Contains(strings.ToLower(line), matchingPhrase) {
				return true
			}
		}
	}
	return false
}

type licenseData struct {
	CopyrightHolder string
	YearRange       string
}

func newLicenseData(
	copyrightHolder string,
	yearRange string,
) *licenseData {
	return &licenseData{
		CopyrightHolder: copyrightHolder,
		YearRange:       yearRange,
	}
}
