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

package bufimagemodifyv2

import "google.golang.org/protobuf/types/descriptorpb"

type prefixOverride string

func newPrefixOverride(prefix string) prefixOverride {
	return prefixOverride(prefix)
}

func (p prefixOverride) get() string {
	return string(p)
}

func (p prefixOverride) override() {}

type valueOverride[T string | bool | descriptorpb.FileOptions_OptimizeMode] struct {
	value T
}

func newValueOverride[T string | bool | descriptorpb.FileOptions_OptimizeMode](val T) valueOverride[T] {
	return valueOverride[T]{
		value: val,
	}
}

func (v valueOverride[T]) get() T {
	return v.value
}

func (v valueOverride[T]) override() {}
