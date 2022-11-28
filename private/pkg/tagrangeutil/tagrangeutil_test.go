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

package tagrangeutil

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bufbuild/buf/private/pkg/protosource"
)

// Implementation of protosource.TagRange for testing.
type testTagRange struct {
	start, end int
}

func (testTagRange) File() protosource.File { return nil }

func (testTagRange) Location() protosource.Location { return nil }

func (r testTagRange) Start() int { return r.start }

func (r testTagRange) End() int { return r.end }

func (testTagRange) Max() bool { return false }

// Parses a string like 5-10,12 into test ranges.
func stringToTestRanges(rangesString string) []protosource.TagRange {
	results := []protosource.TagRange{}
	if rangesString == "" {
		return results
	}
	rangeStrings := strings.Split(rangesString, ",")
	for _, rangeString := range rangeStrings {
		beginString, endString, hasEnd := strings.Cut(rangeString, "-")
		if !hasEnd {
			endString = beginString
		}
		begin, err := strconv.Atoi(beginString)
		if err != nil {
			panic(err)
		}
		end, err := strconv.Atoi(endString)
		if err != nil {
			panic(err)
		}
		results = append(results, testTagRange{begin, end})
	}
	return results
}

func testExpectedSubset(t *testing.T, supersetRangesString string, subsetRangesString string, expectedMissingString string) {
	supersetRanges := stringToTestRanges(supersetRangesString)
	subsetRanges := stringToTestRanges(subsetRangesString)
	expectedMissing := stringToTestRanges(expectedMissingString)
	isSubset, actualMissing := CheckIsSubset(supersetRanges, subsetRanges)

	if isSubset && len(actualMissing) > 0 {
		t.Error("Subset flag should be cleared when missing ranges are found")
	} else if !isSubset && len(actualMissing) == 0 {
		t.Error("Subset flag should be set when missing ranges aren't found")
	}

	assert.Equal(t, len(expectedMissing), len(actualMissing), fmt.Sprint(actualMissing))
	if len(expectedMissing) == len(actualMissing) {
		for i := range actualMissing {
			assert.Equal(t, expectedMissing[i], actualMissing[i])
		}
	}
}

func TestCheckIsSubset(t *testing.T) {
	testExpectedSubset(t, "1", "1", "")
	testExpectedSubset(t, "1-2", "1", "")
	testExpectedSubset(t, "1", "1-2", "1-2")
	testExpectedSubset(t, "5,4,3,2,1", "1-5", "")
	testExpectedSubset(t, "1-5", "1,2,4,5,3", "")
	testExpectedSubset(t, "90-199,200", "100-200", "")
	testExpectedSubset(t, "100-200", "90-199,200", "90-199")
	testExpectedSubset(t, "1-5,200,1000-1001,2000", "1-3,5,1001,3000", "3000")
	testExpectedSubset(t, "1-92,93-95,96-99,100", "1-93,3-100,5-200", "5-200")
	testExpectedSubset(t, "1-92,3-50,5-75,10-90,15-30", "1-20,5-10,30-75,91,92,93", "93")
}
