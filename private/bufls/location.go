// Copyright 2020-2022 Buf Technologies, Inc.
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

package bufls

import (
	"fmt"
	"path/filepath"
)

type location struct {
	path   string
	line   int
	column int
}

func newLocation(
	path string,
	line int,
	column int,
) (*location, error) {
	if filepath.Ext(path) != ".proto" {
		return nil, fmt.Errorf("location path %s must be a .proto file", path)
	}
	if line <= 0 {
		return nil, fmt.Errorf("location line %d must be a positive integer", line)
	}
	if column <= 0 {
		return nil, fmt.Errorf("location column %d must be a positive integer", column)
	}
	return &location{
		path:   path,
		line:   line,
		column: column,
	}, nil
}

func (p *location) Path() string {
	return p.path
}

func (p *location) Line() int {
	return p.line
}

func (p *location) Column() int {
	return p.column
}

func (p *location) String() string {
	return fmt.Sprintf("%s:%d:%d", p.path, p.line, p.column)
}
