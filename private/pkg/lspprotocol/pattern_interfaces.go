// Copyright 2020-2026 Buf Technologies, Inc.
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

// Code generated from the LSP metaModel. DO NOT EDIT.

package lspprotocol

import (
	"fmt"
	"log/slog"
)

// PatternInfo is an interface for types that represent glob patterns.
type PatternInfo interface {
	GetPattern() string
	GetBasePath() string
	isPattern() // marker method
}

// StringPattern implements PatternInfo for string patterns.
type StringPattern struct {
	Pattern string
}

// GetPattern returns the glob pattern string.
func (p StringPattern) GetPattern() string { return p.Pattern }

// GetBasePath returns an empty string for simple patterns.
func (p StringPattern) GetBasePath() string { return "" }
func (p StringPattern) isPattern()          {}

// RelativePatternInfo implements PatternInfo for RelativePattern.
type RelativePatternInfo struct {
	RP       RelativePattern
	BasePath string
}

// GetPattern returns the glob pattern string.
func (p RelativePatternInfo) GetPattern() string { return p.RP.Pattern }

// GetBasePath returns the base path for the pattern.
func (p RelativePatternInfo) GetBasePath() string { return p.BasePath }
func (p RelativePatternInfo) isPattern()          {}

// AsPattern converts GlobPattern to a PatternInfo object.
func (g *GlobPattern) AsPattern() (PatternInfo, error) {
	if g.Value == nil {
		return nil, fmt.Errorf("nil pattern")
	}

	var err error

	switch v := g.Value.(type) {
	case string:
		return StringPattern{Pattern: v}, nil

	case RelativePattern:
		// Handle BaseURI which could be string or DocumentUri
		var basePath string
		switch baseURI := v.BaseURI.Value.(type) {
		case string:
			basePath, err = DocumentURI(baseURI).Path()
			if err != nil {
				slog.Error("Failed to convert URI to path", "uri", baseURI, "error", err)
				return nil, fmt.Errorf("invalid URI: %s", baseURI)
			}

		case DocumentURI:
			basePath, err = baseURI.Path()
			if err != nil {
				slog.Error("Failed to convert DocumentURI to path", "uri", baseURI, "error", err)
				return nil, fmt.Errorf("invalid DocumentURI: %s", baseURI)
			}

		default:
			return nil, fmt.Errorf("unknown BaseURI type: %T", v.BaseURI.Value)
		}

		return RelativePatternInfo{RP: v, BasePath: basePath}, nil

	default:
		return nil, fmt.Errorf("unknown pattern type: %T", g.Value)
	}
}
