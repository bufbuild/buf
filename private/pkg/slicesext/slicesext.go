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

// Package slicesext provides extra functionality on top of the slices package.
package slicesext

import "sort"

// Ordered matches cmp.Ordered until we only support Go versions >= 1.21.
//
// TODO: remove and replace with cmp.Ordered when we only support Go versions >= 1.21.
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~string
}

// Filter filters the slice to only the values where f returns true.
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

// Reduce reduces the slice.
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

// Copy returns a copy of the slice.
func Copy[T any](s []T) []T {
	sc := make([]T, len(s))
	copy(sc, s)
	return sc
}

// ToStructMap converts the slice to a map with struct{} values.
func ToStructMap[T comparable](s []T) map[T]struct{} {
	m := make(map[T]struct{}, len(s))
	for _, e := range s {
		m[e] = struct{}{}
	}
	return m
}

// ToValuesMap transforms the input slice into a map from f(V) -> V.
//
// If f(V) is the zero value of K, nothing is added to the map.
//
// Duplicate values of type K will result in a single map entry.
func ToValuesMapV[K comparable, V any](s []V, f func(V) K) map[K]V {
	var zero K
	m := make(map[K]V)
	for _, v := range s {
		k := f(v)
		if k != zero {
			m[k] = v
		}
	}
	return m
}

// MapKeysToSortedSlice converts the map's keys to a sorted slice.
func MapKeysToSortedSlice[M ~map[K]V, K Ordered, V any](m M) []K {
	s := MapKeysToSlice(m)
	// TODO: Replace with slices.Sort when we only support Go versions >= 1.21.
	sort.Slice(
		s,
		func(i int, j int) bool {
			return s[i] < s[j]
		},
	)
	return s
}

// MapKeysToSlice converts the map's keys to a slice.
func MapKeysToSlice[K comparable, V any](m map[K]V) []K {
	s := make([]K, 0, len(m))
	for e := range m {
		s = append(s, e)
	}
	return s
}

// ToUniqueSorted returns a sorted copy of s with no duplicates.
func ToUniqueSorted[S ~[]T, T Ordered](s S) S {
	return MapKeysToSortedSlice(ToStructMap(s))
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
	c := make([]T, len(s))
	copy(c, s)
	// https://github.com/golang/go/wiki/SliceTricks#batching-with-minimal-allocation
	for chunkSize < len(c) {
		c, chunks = c[chunkSize:], append(chunks, c[0:chunkSize:chunkSize])
	}
	return append(chunks, c)
}

// ElementsEqual returns true if the two slices have equal elements.
//
// Nil and empty slices are treated as equals.
func ElementsEqual[T comparable](one []T, two []T) bool {
	if len(one) != len(two) {
		return false
	}
	for i, elem := range one {
		if two[i] != elem {
			return false
		}
	}
	return true
}

// ElementsContained returns true if superset contains subset.
//
// Nil and empty slices are treated as equals.
func ElementsContained(superset []string, subset []string) bool {
	m := ToStructMap(superset)
	for _, elem := range subset {
		if _, ok := m[elem]; !ok {
			return false
		}
	}
	return true
}
