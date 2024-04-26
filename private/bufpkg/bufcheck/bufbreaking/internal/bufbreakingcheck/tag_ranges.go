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

package bufbreakingcheck

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
)

type tagRange interface {
	Start() int
	End() int
}

type simpleTagRange [2]int

func (r simpleTagRange) Start() int { return r[0] }
func (r simpleTagRange) End() int   { return r[1] }

func checkTagRanges[R bufprotosource.TagRange](
	add addFunc,
	rangeKind string,
	element bufprotosource.NamedDescriptor,
	previousRanges []R,
	ranges []R,
) error {
	if len(previousRanges) == 0 {
		return nil // nothing to check
	}
	collapsedRanges := collapseRanges(ranges)
	for _, previousRange := range previousRanges {
		start, end := previousRange.Start(), previousRange.End()
		missingRanges := findMissing(start, end, collapsedRanges)
		if len(missingRanges) > 0 {
			elementKind, maxTag, err := classifyElementRange(element)
			if err != nil {
				return err
			}
			previousString := bufprotosource.TagRangeString(previousRange)
			removedString := missingRangesString(maxTag, missingRanges)
			add(element,
				nil,
				element.Location(),
				`Previously present %s range %q on %s %q is missing values: %s were removed.`,
				rangeKind,
				previousString,
				elementKind,
				element.Name(),
				removedString)
		}
	}
	return nil
}

func collapseRanges[R tagRange](ranges []R) []simpleTagRange {
	if len(ranges) == 0 {
		return nil
	}
	sortedRanges := make([]simpleTagRange, len(ranges))
	for i, curRange := range ranges {
		start, end := curRange.Start(), curRange.End()
		sortedRanges[i] = simpleTagRange{start, end}
	}
	sort.Slice(sortedRanges, func(i, j int) bool {
		if sortedRanges[i].Start() == sortedRanges[j].Start() {
			return sortedRanges[i].End() < sortedRanges[j].End()
		}
		return sortedRanges[i].Start() < sortedRanges[j].Start()
	})
	var j int
	for i := 1; i < len(sortedRanges); i++ {
		if sortedRanges[i].Start() <= sortedRanges[j].End()+1 {
			// overlapping or adjacent, so we can collapse i into j
			if sortedRanges[i].End() > sortedRanges[j].End() {
				sortedRanges[j][1] = sortedRanges[i].End()
			}
			continue
		}
		j++
		if i != j {
			sortedRanges[j] = sortedRanges[i]
		}
	}
	return sortedRanges[:j+1]
}

func findMissing(start, end int, collapsedRanges []simpleTagRange) []simpleTagRange {
	index := sort.Search(len(collapsedRanges), func(i int) bool {
		return collapsedRanges[i].End() >= start
	})
	var entryStart, entryEnd int
	if index < len(collapsedRanges) {
		entryStart, entryEnd = collapsedRanges[index].Start(), collapsedRanges[index].End()
	}
	var missingRanges []simpleTagRange
	if index >= len(collapsedRanges) || entryStart > end {
		// No overlapping ranges; entire span is missing
		return []simpleTagRange{{start, end}}
	}
	for {
		if start < entryStart {
			if end < entryStart {
				missingRanges = append(missingRanges, simpleTagRange{start, end})
			} else {
				missingRanges = append(missingRanges, simpleTagRange{start, entryStart - 1})
			}
		}
		start = entryEnd + 1
		index++
		if index >= len(collapsedRanges) || entryEnd >= end {
			// no further to go or no need to go further
			break
		}
		entryStart, entryEnd = collapsedRanges[index].Start(), collapsedRanges[index].End()
	}
	if end > entryEnd {
		missingRanges = append(missingRanges, simpleTagRange{entryEnd + 1, end})
	}
	return missingRanges
}

func classifyElementRange(element bufprotosource.Descriptor) (elementKind string, maxTag int, err error) {
	switch element := element.(type) {
	case bufprotosource.Message:
		if element.MessageSetWireFormat() {
			return "message", bufprotosource.MessageSetRangeInclusiveMax, nil
		}
		return "message", bufprotosource.MessageRangeInclusiveMax, nil
	case bufprotosource.Enum:
		return "enum", bufprotosource.EnumRangeInclusiveMax, nil
	default:
		return "", 0, fmt.Errorf("only messages and enums have ranges but instead got %T", element)
	}
}

func missingRangesString(maxTag int, missingRanges []simpleTagRange) string {
	removedStrings := make([]string, len(missingRanges))
	for i, missingRange := range missingRanges {
		start := missingRange.Start()
		end := missingRange.End()
		switch {
		case start == end:
			removedStrings[i] = fmt.Sprintf("[%d]", start)
		case missingRange.End() == maxTag:
			removedStrings[i] = fmt.Sprintf("[%d,max]", start)
		default:
			removedStrings[i] = fmt.Sprintf("[%d,%d]", start, end)
		}
	}
	return strings.Join(removedStrings, ", ")
}
