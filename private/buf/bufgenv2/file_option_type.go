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

package bufgenv2

import (
	"strconv"
)

const (
	// FileOptionTypeValue is the file option that we specify a value for.
	FileOptionTypeValue FileOptionType = iota + 1
	// FileOptionTypePrefix is the file option that we specify a prefix for.
	FileOptionTypePrefix
)

var (
	fileOptionTypeToString = map[FileOptionType]string{
		FileOptionTypeValue:  "value",
		FileOptionTypePrefix: "prefix",
	}
)

// FileOptionType is a type of file option we can manage.
type FileOptionType int

// String implements fmt.Stringer.
func (f FileOptionType) String() string {
	s, ok := fileOptionTypeToString[f]
	if !ok {
		return strconv.Itoa(int(f))
	}
	return s
}
