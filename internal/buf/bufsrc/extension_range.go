// Copyright 2020 Buf Technologies, Inc.
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

package bufsrc

// TODO: this is hand-computed and this is not great, we should figure out what this actually is or make a constant somewhere else
const extensionRangeMaxMinusOne = 536870911

type extensionRange struct {
	locationDescriptor

	start int // inclusive
	end   int // exclusive
}

func newExtensionRange(
	locationDescriptor locationDescriptor,
	start int,
	end int,
) *extensionRange {
	return &extensionRange{
		locationDescriptor: locationDescriptor,
		start:              start,
		end:                end,
	}
}

func (e *extensionRange) Start() int {
	return e.start
}

func (e *extensionRange) End() int {
	return e.end
}
