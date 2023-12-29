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

package bufmoduleapi

import "github.com/bufbuild/buf/private/pkg/slicesext"

type indexedValue[T any] struct {
	Index int
	Value T
}

func newIndexedValue[T any](index int, value T) *indexedValue[T] {
	return &indexedValue[T]{
		Index: index,
		Value: value,
	}
}

func getKeyToIndexedValues[K comparable, V any](values []V, f func(V) K) map[K][]*indexedValue[V] {
	keyToIndexedValues := make(map[K][]*indexedValue[V])
	for i, value := range values {
		keyToIndexedValues[f(value)] = append(
			keyToIndexedValues[f(value)],
			newIndexedValue(i, value),
		)
	}
	return keyToIndexedValues
}

func getValuesForIndexedValues[T any](indexedValues []*indexedValue[T]) []T {
	return slicesext.Map(indexedValues, func(indexedValue *indexedValue[T]) T { return indexedValue.Value })
}
