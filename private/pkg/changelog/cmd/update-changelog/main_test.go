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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_release(t *testing.T) {
	tests := []struct {
		name    string
		version string
		date    string
		file    string
		want    string
		wantErr bool
	}{
		{
			name:    "simple",
			version: "v1.0.1",
			date:    "2020-01-02",
			file: `# Changelog

## [Unreleased]

- Change foobar

## [v1.0.0] - 2020-01-01

[Unreleased]: https://github.com/foobar/foo/compare/v1.0.0...HEAD
[v1.0.0]: https://github.com/foobar/foo/compare/v1.0.0...v1.0.0
`,
			want: `# Changelog

## [v1.0.1] - 2020-01-02

- Change foobar

## [v1.0.0] - 2020-01-01

[v1.0.1]: https://github.com/foobar/foo/compare/v1.0.0...v1.0.1
[v1.0.0]: https://github.com/foobar/foo/compare/v1.0.0...v1.0.0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := release(tt.version, tt.date, []byte(tt.file))
			if (err != nil) != tt.wantErr {
				t.Errorf("release() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, string(result))
		})
	}
}

func Test_unrelease(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    string
		wantErr bool
	}{
		{
			name: "simple",
			file: `# Changelog

## [v1.0.1] - 2020-01-02

- Change foobar

## [v1.0.0] - 2020-01-01

[v1.0.1]: https://github.com/foobar/foo/compare/v1.0.0...v1.0.1
[v1.0.0]: https://github.com/foobar/foo/compare/v1.0.0...v1.0.0
`,
			want: `# Changelog

## [Unreleased]

- No changes yet.

## [v1.0.1] - 2020-01-02

- Change foobar

## [v1.0.0] - 2020-01-01

[Unreleased]: https://github.com/foobar/foo/compare/v1.0.1...HEAD
[v1.0.1]: https://github.com/foobar/foo/compare/v1.0.0...v1.0.1
[v1.0.0]: https://github.com/foobar/foo/compare/v1.0.0...v1.0.0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := unrelease([]byte(tt.file))
			if (err != nil) != tt.wantErr {
				t.Errorf("unrelease() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, string(result))
		})
	}
}
