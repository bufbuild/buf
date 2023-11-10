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

// Package stringutil implements string utilities.
package stringutil

import (
	"strings"
	"unicode"

	"github.com/bufbuild/buf/private/pkg/slicesextended"
)

// TrimLines splits the output into individual lines and trims the spaces from each line.
//
// This also trims the start and end spaces from the original output.
func TrimLines(output string) string {
	return strings.TrimSpace(strings.Join(SplitTrimLines(output), "\n"))
}

// SplitTrimLines splits the output into individual lines and trims the spaces from each line.
func SplitTrimLines(output string) []string {
	// this should work for windows as well as \r will be trimmed
	split := strings.Split(output, "\n")
	lines := make([]string, len(split))
	for i, line := range split {
		lines[i] = strings.TrimSpace(line)
	}
	return lines
}

// SplitTrimLinesNoEmpty splits the output into individual lines and trims the spaces from each line.
//
// This removes any empty lines.
func SplitTrimLinesNoEmpty(output string) []string {
	// this should work for windows as well as \r will be trimmed
	split := strings.Split(output, "\n")
	lines := make([]string, 0, len(split))
	for _, line := range split {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// MapToSortedSlice transforms m to a sorted slice.
//
// Deprecated: Use slicesextended.MapToSortedSlice instead.
func MapToSortedSlice(m map[string]struct{}) []string {
	return slicesextended.MapToSortedSlice(m)
}

// MapToSlice transforms m to a slice.
//
// Deprecated: Use slicesextended.MapToSlice instead.
func MapToSlice(m map[string]struct{}) []string {
	return slicesextended.MapToSlice(m)
}

// SliceToMap transforms s to a map.
//
// Deprecated: Use slicesextended.ToMap instead.
func SliceToMap(s []string) map[string]struct{} {
	return slicesextended.ToMap(s)
}

// SliceToUniqueSortedSlice returns a sorted copy of s with no duplicates.
//
// Deprecated: Use slicesextended.ToUniqueSorted instead.
func SliceToUniqueSortedSlice(s []string) []string {
	return slicesextended.ToUniqueSorted(s)
}

// SliceToUniqueSortedSliceFilterEmptyStrings returns a sorted copy of s with no duplicates and no empty strings.
//
// Strings with only spaces are considered empty.
func SliceToUniqueSortedSliceFilterEmptyStrings(s []string) []string {
	m := slicesextended.ToMap(s)
	for key := range m {
		if strings.TrimSpace(key) == "" {
			delete(m, key)
		}
	}
	return slicesextended.MapToSortedSlice(m)
}

// SliceToChunks splits s into chunks of the given chunk size.
//
// If s is nil or empty, returns empty.
// If chunkSize is <=0, returns [][]string{s}.
//
// Deprecated: use slicesextended.ToChunks instead.
func SliceToChunks(s []string, chunkSize int) [][]string {
	return slicesextended.ToChunks(s, chunkSize)
}

// SliceElementsEqual returns true if the two slices have equal elements.
//
// Nil and empty slices are treated as equals.
//
// Deprecated: use slicesextended.ElementsEqual instead.
func SliceElementsEqual(one []string, two []string) bool {
	return slicesextended.ElementsEqual(one, two)
}

// SliceElementsContained returns true if superset contains subset.
//
// Nil and empty slices are treated as equals.
//
// Deprecated: use slicesextended.ElementsContained instead.
func SliceElementsContained(superset []string, subset []string) bool {
	return slicesextended.ElementsContained(superset, subset)
}

// JoinSliceQuoted joins the slice with quotes.
func JoinSliceQuoted(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	return `"` + strings.Join(s, `"`+sep+`"`) + `"`
}

// SliceToString prints the slice as [e1,e2].
func SliceToString(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return "[" + strings.Join(s, ",") + "]"
}

// SliceToHumanString prints the slice as "e1, e2, and e3".
func SliceToHumanString(s []string) string {
	switch len(s) {
	case 0:
		return ""
	case 1:
		return s[0]
	case 2:
		return s[0] + " and " + s[1]
	default:
		return strings.Join(s[:len(s)-1], ", ") + ", and " + s[len(s)-1]
	}
}

// SliceToHumanStringQuoted prints the slice as `"e1", "e2", and "e3"`.
func SliceToHumanStringQuoted(s []string) string {
	switch len(s) {
	case 0:
		return ""
	case 1:
		return `"` + s[0] + `"`
	case 2:
		return `"` + s[0] + `" and "` + s[1] + `"`
	default:
		return `"` + strings.Join(s[:len(s)-1], `", "`) + `", and "` + s[len(s)-1] + `"`
	}
}

// SliceToHumanStringOr prints the slice as "e1, e2, or e3".
func SliceToHumanStringOr(s []string) string {
	switch len(s) {
	case 0:
		return ""
	case 1:
		return s[0]
	case 2:
		return s[0] + " or " + s[1]
	default:
		return strings.Join(s[:len(s)-1], ", ") + ", or " + s[len(s)-1]
	}
}

// SliceToHumanStringOrQuoted prints the slice as `"e1", "e2", or "e3"`.
func SliceToHumanStringOrQuoted(s []string) string {
	switch len(s) {
	case 0:
		return ""
	case 1:
		return `"` + s[0] + `"`
	case 2:
		return `"` + s[0] + `" or "` + s[1] + `"`
	default:
		return `"` + strings.Join(s[:len(s)-1], `", "`) + `", or "` + s[len(s)-1] + `"`
	}
}

// SnakeCaseOption is an option for snake_case conversions.
type SnakeCaseOption func(*snakeCaseOptions)

// SnakeCaseWithNewWordOnDigits is a SnakeCaseOption that signifies
// to split on digits, ie foo_bar_1 instead of foo_bar1.
func SnakeCaseWithNewWordOnDigits() SnakeCaseOption {
	return func(snakeCaseOptions *snakeCaseOptions) {
		snakeCaseOptions.newWordOnDigits = true
	}
}

// ToLowerSnakeCase transforms s to lower_snake_case.
func ToLowerSnakeCase(s string, options ...SnakeCaseOption) string {
	return strings.ToLower(toSnakeCase(s, options...))
}

// ToUpperSnakeCase transforms s to UPPER_SNAKE_CASE.
func ToUpperSnakeCase(s string, options ...SnakeCaseOption) string {
	return strings.ToUpper(toSnakeCase(s, options...))
}

// ToPascalCase converts s to PascalCase.
//
// Splits on '-', '_', ' ', '\t', '\n', '\r'.
// Uppercase letters will stay uppercase,
func ToPascalCase(s string) string {
	output := ""
	var previous rune
	for i, c := range strings.TrimSpace(s) {
		if !isDelimiter(c) {
			if i == 0 || isDelimiter(previous) || unicode.IsUpper(c) {
				output += string(unicode.ToUpper(c))
			} else {
				output += string(unicode.ToLower(c))
			}
		}
		previous = c
	}
	return output
}

// IsAlphanumeric returns true for [0-9a-zA-Z].
func IsAlphanumeric(r rune) bool {
	return IsNumeric(r) || IsAlpha(r)
}

// IsAlpha returns true for [a-zA-Z].
func IsAlpha(r rune) bool {
	return IsLowerAlpha(r) || IsUpperAlpha(r)
}

// IsLowerAlpha returns true for [a-z].
func IsLowerAlpha(r rune) bool {
	return 'a' <= r && r <= 'z'
}

// IsUpperAlpha returns true for [A-Z].
func IsUpperAlpha(r rune) bool {
	return 'A' <= r && r <= 'Z'
}

// IsNumeric returns true for [0-9].
func IsNumeric(r rune) bool {
	return '0' <= r && r <= '9'
}

// IsLowerAlphanumeric returns true for [0-9a-z].
func IsLowerAlphanumeric(r rune) bool {
	return IsNumeric(r) || IsLowerAlpha(r)
}

func toSnakeCase(s string, options ...SnakeCaseOption) string {
	snakeCaseOptions := &snakeCaseOptions{}
	for _, option := range options {
		option(snakeCaseOptions)
	}
	output := ""
	s = strings.TrimFunc(s, isDelimiter)
	for i, c := range s {
		if isDelimiter(c) {
			c = '_'
		}
		if i == 0 {
			output += string(c)
		} else if isSnakeCaseNewWord(c, snakeCaseOptions.newWordOnDigits) &&
			output[len(output)-1] != '_' &&
			((i < len(s)-1 && !isSnakeCaseNewWord(rune(s[i+1]), true) && !isDelimiter(rune(s[i+1]))) ||
				(snakeCaseOptions.newWordOnDigits && unicode.IsDigit(c)) ||
				(unicode.IsLower(rune(s[i-1])))) {
			output += "_" + string(c)
		} else if !(isDelimiter(c) && output[len(output)-1] == '_') {
			output += string(c)
		}
	}
	return output
}

func isSnakeCaseNewWord(r rune, newWordOnDigits bool) bool {
	if newWordOnDigits {
		return unicode.IsUpper(r) || unicode.IsDigit(r)
	}
	return unicode.IsUpper(r)
}

func isDelimiter(r rune) bool {
	return r == '.' || r == '-' || r == '_' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

type snakeCaseOptions struct {
	newWordOnDigits bool
}
