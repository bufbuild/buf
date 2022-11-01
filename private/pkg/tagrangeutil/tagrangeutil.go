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
	"sort"

	"github.com/bufbuild/buf/private/pkg/protosource"
)

type tagRangeGroup struct {
	ranges []protosource.TagRange
	start  int
	end    int
}

// sortTagRanges sorts tag ranges by their start, end components.
func sortTagRanges(ranges []protosource.TagRange) []protosource.TagRange {
	rangesCopy := make([]protosource.TagRange, len(ranges))
	copy(rangesCopy, ranges)

	sort.Slice(rangesCopy, func(i, j int) bool {
		return rangesCopy[i].Start() < rangesCopy[j].Start() ||
			(rangesCopy[i].Start() == rangesCopy[j].Start() &&
				rangesCopy[i].End() < rangesCopy[j].End())
	})

	return rangesCopy
}

// groupAdjacentTagRanges sorts and groups adjacent tag ranges.
func groupAdjacentTagRanges(ranges []protosource.TagRange) []tagRangeGroup {
	if len(ranges) == 0 {
		return []tagRangeGroup{}
	}

	sortedTagRanges := sortTagRanges(ranges)

	j := 0
	groupedTagRanges := make([]tagRangeGroup, 1, len(ranges))
	groupedTagRanges[j] = tagRangeGroup{
		ranges: sortedTagRanges[0:1],
		start:  sortedTagRanges[0].Start(),
		end:    sortedTagRanges[0].End(),
	}

	for i := 1; i < len(sortedTagRanges); i++ {
		if sortedTagRanges[i].Start() <= sortedTagRanges[i-1].End()+1 {
			if sortedTagRanges[i].End() > groupedTagRanges[j].end {
				groupedTagRanges[j].end = sortedTagRanges[i].End()
			}
			groupedTagRanges[j].ranges = groupedTagRanges[j].ranges[0 : len(groupedTagRanges[j].ranges)+1]
		} else {
			groupedTagRanges = append(groupedTagRanges, tagRangeGroup{
				ranges: sortedTagRanges[i : i+1],
				start:  sortedTagRanges[i].Start(),
				end:    sortedTagRanges[i].End(),
			})
			j++
		}
	}

	return groupedTagRanges
}

// CheckIsSubset checks if supersetRanges is a superset of subsetRanges.
// If so, it returns true and nil. If not, it returns false with a slice of failing ranges from subsetRanges.
func CheckIsSubset(supersetRanges []protosource.TagRange, subsetRanges []protosource.TagRange) (bool, []protosource.TagRange) {
	if len(subsetRanges) == 0 {
		return true, nil
	}

	if len(supersetRanges) == 0 {
		return false, subsetRanges
	}

	supersetTagRangeGroups := groupAdjacentTagRanges(supersetRanges)
	subsetTagRanges := sortTagRanges(subsetRanges)
	missingTagRanges := []protosource.TagRange{}

	for i, j := 0, 0; j < len(subsetTagRanges); j++ {
		for supersetTagRangeGroups[i].end < subsetTagRanges[j].Start() {
			if i++; i == len(supersetTagRangeGroups) {
				missingTagRanges = append(missingTagRanges, subsetTagRanges[j:]...)
				return false, missingTagRanges
			}
		}
		if supersetTagRangeGroups[i].start > subsetTagRanges[j].Start() ||
			supersetTagRangeGroups[i].end < subsetTagRanges[j].End() {
			missingTagRanges = append(missingTagRanges, subsetTagRanges[j])
		}
	}

	if len(missingTagRanges) != 0 {
		return false, missingTagRanges
	}

	return true, nil
}
