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

package opt

// Optional is a value that may or may not be present.
type Optional[T comparable] struct {
	// Value is the value of the Optional.
	//
	// If this not equal to the zero value of T, this is considered to be present.
	Value T
}

// NewOptional returns a new Optional for the value.
//
// An Optional can also be constructed with &Optional{Value: value}.
func NewOptional[T comparable](value T) *Optional[T] {
	return &Optional[T]{
		Value: value,
	}
}

// Present returns true if the Value is not equal to the zero value of T.
func (o *Optional[T]) Present() bool {
	var zero T
	return o.Value != zero
}
