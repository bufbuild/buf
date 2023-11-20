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

// Package slicesextended provides extra functionality on top of the slices package.
package slicesextended

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

// ToMap converts the slice to a map.
func ToMap[T comparable](s []T) map[T]struct{} {
	m := make(map[T]struct{}, len(s))
	for _, e := range s {
		m[e] = struct{}{}
	}
	return m
}

// MapToSortedSlice converts the map to a sorted slice.
func MapToSortedSlice[M ~map[T]struct{}, T Ordered](m M) []T {
	s := MapToSlice(m)
	// TODO: Replace with slices.Sort when we only support Go versions >= 1.21.
	sort.Slice(
		s,
		func(i int, j int) bool {
			return s[i] < s[j]
		},
	)
	return s
}

// MapToSlice converts the map to a slice.
func MapToSlice[T comparable](m map[T]struct{}) []T {
	s := make([]T, 0, len(m))
	for e := range m {
		s = append(s, e)
	}
	return s
}

// ToUniqueSorted returns a sorted copy of s with no duplicates.
func ToUniqueSorted[S ~[]T, T Ordered](s S) S {
	return MapToSortedSlice(ToMap(s))
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
	m := ToMap(superset)
	for _, elem := range subset {
		if _, ok := m[elem]; !ok {
			return false
		}
	}
	return true
}
