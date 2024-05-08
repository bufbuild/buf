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

// Package syncext provides extra functionality on top of the sync package.
package syncext

import "sync"

// OnceValue returns a function that invokes f only once and returns the value
// returned by f. The returned function may be called concurrently.
//
// If f panics, the returned function will panic with the same value on every call.
//
// TODO FUTURE: This is directly copied from 1.21 source, remove when no longer need <1.21.
func OnceValue[T any](f func() T) func() T {
	var (
		once   sync.Once
		valid  bool
		p      any
		result T
	)
	g := func() {
		defer func() {
			p = recover()
			if !valid {
				panic(p)
			}
		}()
		result = f()
		f = nil
		valid = true
	}
	return func() T {
		once.Do(g)
		if !valid {
			panic(p)
		}
		return result
	}
}

// OnceValues returns a function that invokes f only once and returns the values
// returned by f. The returned function may be called concurrently.
//
// If f panics, the returned function will panic with the same value on every call.
//
// TODO FUTURE: This is directly copied from 1.21 source, remove when no longer need <1.21.
func OnceValues[T1, T2 any](f func() (T1, T2)) func() (T1, T2) {
	var (
		once  sync.Once
		valid bool
		p     any
		r1    T1
		r2    T2
	)
	g := func() {
		defer func() {
			p = recover()
			if !valid {
				panic(p)
			}
		}()
		r1, r2 = f()
		f = nil
		valid = true
	}
	return func() (T1, T2) {
		once.Do(g)
		if !valid {
			panic(p)
		}
		return r1, r2
	}
}

// OnceValues3 returns a function that invokes f only once and returns the values
// returned by f. The returned function may be called concurrently.
//
// If f panics, the returned function will panic with the same value on every call.
//
// This is copied from sync.OnceValues and extended to for three values.
func OnceValues3[T1, T2, T3 any](f func() (T1, T2, T3)) func() (T1, T2, T3) {
	var (
		once  sync.Once
		valid bool
		p     any
		r1    T1
		r2    T2
		r3    T3
	)
	g := func() {
		defer func() {
			p = recover()
			if !valid {
				panic(p)
			}
		}()
		r1, r2, r3 = f()
		f = nil
		valid = true
	}
	return func() (T1, T2, T3) {
		once.Do(g)
		if !valid {
			panic(p)
		}
		return r1, r2, r3
	}
}
