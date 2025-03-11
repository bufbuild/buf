// Copyright 2020-2025 Buf Technologies, Inc.
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

// Package slicesext provides extra functionality on top of the slices package.
package slicesext

import (
	"cmp"
	"fmt"
	"slices"
	"sort"
	"strings"
)

// Indexed is a value that had an index within a slice.
type Indexed[T any] struct {
	Value T
	Index int
}

// Filter returns a new slice containing only the values where f returns true.
func Filter[T any](s []T, f func(T) bool) []T {
	sf := make([]T, 0, len(s))
	for _, e := range s {
		if f(e) {
			sf = append(sf, e)
		}
	}
	return sf
}

// FilterError filters the slice to only the values where f returns true.
//
// Returns error the first time f returns error.
func FilterError[T any](s []T, f func(T) (bool, error)) ([]T, error) {
	sf := make([]T, 0, len(s))
	for _, e := range s {
		ok, err := f(e)
		if err != nil {
			return nil, err
		}
		if ok {
			sf = append(sf, e)
		}
	}
	return sf, nil
}

// Map maps the slice.
func Map[T1, T2 any](s []T1, f func(T1) T2) []T2 {
	sm := make([]T2, len(s))
	for i, e := range s {
		sm[i] = f(e)
	}
	return sm
}

// MapError maps the slice.
//
// Returns error the first time f returns error.
func MapError[T1, T2 any](s []T1, f func(T1) (T2, error)) ([]T2, error) {
	sm := make([]T2, len(s))
	for i, e := range s {
		em, err := f(e)
		if err != nil {
			return nil, err
		}
		sm[i] = em
	}
	return sm, nil
}

// Reduce reduces the slice.
func Reduce[T1, T2 any](s []T1, f func(T2, T1) T2, initialValue T2) T2 {
	value := initialValue
	for _, e := range s {
		value = f(value, e)
	}
	return value
}

// ReduceError reduces the slice.
//
// Returns error the first time f returns error.
func ReduceError[T1, T2 any](s []T1, f func(T2, T1) (T2, error), initialValue T2) (T2, error) {
	value := initialValue
	var err error
	for _, e := range s {
		value, err = f(value, e)
		if err != nil {
			return value, err
		}
	}
	return value, nil
}

// Count returns the number of elements in s where f returns true.
func Count[T any](s []T, f func(T) bool) int {
	count := 0
	for _, e := range s {
		if f(e) {
			count++
		}
	}
	return count
}

// CountError returns the number of elements in s where f returns true.
//
// Returns error the first time f returns error.
func CountError[T any](s []T, f func(T) (bool, error)) (int, error) {
	count := 0
	for _, e := range s {
		ok, err := f(e)
		if err != nil {
			return 0, err
		}
		if ok {
			count++
		}
	}
	return count, nil
}

// ToStructMap converts the slice to a map with struct{} values.
func ToStructMap[T comparable](s []T) map[T]struct{} {
	m := make(map[T]struct{}, len(s))
	for _, e := range s {
		m[e] = struct{}{}
	}
	return m
}

// ToStructMapOmitEmpty converts the slice to a map with struct{} values.
//
// Zero values of T are not added to the map.
//
// TODO FUTURE: Make ToStructMap use this logic, remove ToStructMapOmitEmpty, to match other functions.
func ToStructMapOmitEmpty[T comparable](s []T) map[T]struct{} {
	var zero T
	m := make(map[T]struct{}, len(s))
	for _, e := range s {
		if e != zero {
			m[e] = struct{}{}
		}
	}
	return m
}

// ToValuesMap transforms the input slice into a map from f(V) -> V.
//
// If f(V) is the zero value of K, nothing is added to the map.
//
// Duplicate values of type K will result in a single map entry.
func ToValuesMap[K comparable, V any](s []V, f func(V) K) map[K][]V {
	var zero K
	m := make(map[K][]V)
	for _, v := range s {
		k := f(v)
		if k != zero {
			m[k] = append(m[k], v)
		}
	}
	return m
}

// ToValuesMapError transforms the input slice into a map from f(V) -> V.
//
// If f(V) is the zero value of K, nothing is added to the map.
//
// Duplicate values of type K will result in a single map entry.
//
// Returns error the first time f returns error.
func ToValuesMapError[K comparable, V any](s []V, f func(V) (K, error)) (map[K][]V, error) {
	var zero K
	m := make(map[K][]V)
	for _, v := range s {
		k, err := f(v)
		if err != nil {
			return nil, err
		}
		if k != zero {
			m[k] = append(m[k], v)
		}
	}
	return m, nil
}

// ToUniqueValuesMap transforms the input slice into a map from f(V) -> V.
//
// If f(V) is the zero value of K, nothing is added to the map.
//
// Duplicate values of type K will result in an error.
func ToUniqueValuesMap[K comparable, V any](s []V, f func(V) K) (map[K]V, error) {
	return ToUniqueValuesMapError(s, func(v V) (K, error) { return f(v), nil })
}

// ToUniqueValuesMapError transforms the input slice into a map from f(V) -> V.
//
// If f(V) is the zero value of K, nothing is added to the map.
//
// Duplicate values of type K will result in an error.
// Otherwise returns error the first time f returns error.
func ToUniqueValuesMapError[K comparable, V any](s []V, f func(V) (K, error)) (map[K]V, error) {
	var zero K
	m := make(map[K]V)
	for _, v := range s {
		k, err := f(v)
		if err != nil {
			return nil, err
		}
		if k != zero {
			if _, ok := m[k]; ok {
				return nil, fmt.Errorf("duplicate key: %v", k)
			}
			m[k] = v
		}
	}
	return m, nil
}

// ToIndexed indexes the slice.
func ToIndexed[T any](s []T) []Indexed[T] {
	si := make([]Indexed[T], len(s))
	for i, e := range s {
		si[i] = Indexed[T]{Value: e, Index: i}
	}
	return si
}

// ToIndexedValuesMap calls ToValuesMap on the indexed values.
func ToIndexedValuesMap[K comparable, V any](values []V, f func(V) K) map[K][]Indexed[V] {
	return ToValuesMap(ToIndexed(values), func(indexedV Indexed[V]) K { return f(indexedV.Value) })
}

// ToIndexedValuesMapError calls ToValuesMapError on the indexed values.
func ToIndexedValuesMapError[K comparable, V any](values []V, f func(V) (K, error)) (map[K][]Indexed[V], error) {
	return ToValuesMapError(ToIndexed(values), func(indexedV Indexed[V]) (K, error) { return f(indexedV.Value) })
}

// ToUniqueIndexedValuesMap calls ToUniqueValuesMap on the indexed values.
func ToUniqueIndexedValuesMap[K comparable, V any](values []V, f func(V) K) (map[K]Indexed[V], error) {
	return ToUniqueValuesMap(ToIndexed(values), func(indexedV Indexed[V]) K { return f(indexedV.Value) })
}

// ToUniqueIndexedValuesMapError calls ToUniqueValuesMapError on the indexed values.
func ToUniqueIndexedValuesMapError[K comparable, V any](values []V, f func(V) (K, error)) (map[K]Indexed[V], error) {
	return ToUniqueValuesMapError(ToIndexed(values), func(indexedV Indexed[V]) (K, error) { return f(indexedV.Value) })
}

// IndexedToValues takes the indexed values and returns them as values.
func IndexedToValues[T any](s []Indexed[T]) []T {
	return Map(s, func(indexedT Indexed[T]) T { return indexedT.Value })
}

// IndexedToSortedValues takes the indexed values and returns them as values sorted by index.
func IndexedToSortedValues[T any](s []Indexed[T]) []T {
	c := make([]Indexed[T], len(s))
	copy(c, s)
	sort.Slice(c, func(i int, j int) bool { return c[i].Index < c[j].Index })
	return IndexedToValues(c)
}

// MapKeysToSortedSlice converts the map's keys to a sorted slice.
func MapKeysToSortedSlice[M ~map[K]V, K cmp.Ordered, V any](m M) []K {
	s := MapKeysToSlice(m)
	slices.Sort(s)
	return s
}

// MapKeysToSlice converts the map's keys to a slice.
func MapKeysToSlice[K comparable, V any](m map[K]V) []K {
	s := make([]K, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	return s
}

// MapValuesToSortedSlice converts the map's values to a sorted slice.
//
// Duplicate values will be added. This should generally be used
// in cases where you know there is a 1-1 mapping from K to V.
func MapValuesToSortedSlice[K comparable, V cmp.Ordered](m map[K]V) []V {
	s := MapValuesToSlice(m)
	slices.Sort(s)
	return s
}

// MapValuesToSlice converts the map's values to a slice.
//
// Duplicate values will be added. This should generally be used
// in cases where you know there is a 1-1 mapping from K to V.
func MapValuesToSlice[K comparable, V any](m map[K]V) []V {
	s := make([]V, 0, len(m))
	for _, v := range m {
		s = append(s, v)
	}
	return s
}

// ToUniqueSorted returns a sorted copy of s with no duplicates.
func ToUniqueSorted[S ~[]T, T cmp.Ordered](s S) S {
	return MapKeysToSortedSlice(ToStructMap(s))
}

// ToString prints the slice as [e1,e2,...].
func ToString[S ~[]T, T fmt.Stringer](s S) string {
	if len(s) == 0 {
		return ""
	}
	return "[" + strings.Join(Map(s, T.String), ",") + "]"
}

// Duplicates returns the duplicate values in s.
//
// Values are returned in the order they are found in S.
//
// If an element is the zero value, it is not added to duplicates.
func Duplicates[T comparable](s []T) []T {
	var zero T
	count := make(map[T]int, len(s))
	// Needed instead of var declaration to make tests pass.
	duplicates := make([]T, 0)
	for _, e := range s {
		if e == zero {
			continue
		}
		count[e] = count[e] + 1
		if count[e] == 2 {
			// Only insert the first time this is found.
			duplicates = append(duplicates, e)
		}
	}
	return duplicates
}

// Deduplicate returns the unique values of s.
func Deduplicate[V comparable](s []V) []V {
	seen := make(map[V]struct{})
	result := make([]V, 0, len(s))
	for _, e := range s {
		if _, ok := seen[e]; !ok {
			result = append(result, e)
			seen[e] = struct{}{}
		}
	}
	return result
}

// DeduplicateAny returns the unique values of s when transformed with f.
//
// Earlier occurrences of a value are returned and later occurrences are dropped.
func DeduplicateAny[K comparable, V any](s []V, f func(V) K) []V {
	seen := make(map[K]struct{})
	result := make([]V, 0, len(s))
	for _, e := range s {
		k := f(e)
		if _, ok := seen[k]; !ok {
			result = append(result, e)
			seen[k] = struct{}{}
		}
	}
	return result
}

// ToChunks splits s into chunks of the given chunk size.
//
// If s is nil or empty, returns empty.
// If chunkSize is <=0, returns [][]T{s}.
func ToChunks[T any](s []T, chunkSize int) [][]T {
	var chunks [][]T
	if len(s) == 0 {
		return chunks
	}
	if chunkSize <= 0 {
		return [][]T{s}
	}
	return slices.Collect(slices.Chunk(s, chunkSize))
}

// ElementsContained returns true if superset contains subset.
//
// Nil and empty slices are treated as equals.
func ElementsContained[T comparable](superset []T, subset []T) bool {
	m := ToStructMap(superset)
	for _, elem := range subset {
		if _, ok := m[elem]; !ok {
			return false
		}
	}
	return true
}

// TrimPrefix removes a leading prefix from s, otherwise leaves s as-is.
//
// A slice s is considered to have a prefix p if the elements of p are equal
// to the first len(p) elements of s.
//
// Returns false if p was not a prefix of s.
func TrimPrefix[T comparable](s []T, p []T) ([]T, bool) {
	if len(s) < len(p) {
		return s, false
	}

	for i, x := range p {
		if s[i] != x {
			return s, false
		}
	}

	return s[len(p):], true
}
