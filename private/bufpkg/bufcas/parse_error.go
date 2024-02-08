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

package bufcas

import (
	"strings"
)

// ParseError is an error that occurred during parsing.
//
// This is returned by all Parse.* functions in this package.
type ParseError struct {
	// typeString is the user-consumable string representing of the type that was attempted to be parsed.
	//
	// Users cannot rely on this data being structured.
	// Examples: "digest", "digest type".
	typeString string
	// input is the input string that was attempted to be parsed.
	input string
	// err is the underlying error.
	//
	// Err may be a *ParseError itself.
	//
	// This is an error we may give back to the user, use pretty strings that should
	// be read.
	err error
}

// Error implements the error interface.
func (p *ParseError) Error() string {
	var builder strings.Builder
	_, _ = builder.WriteString(`could not parse`)
	if p.typeString != "" {
		_, _ = builder.WriteString(` `)
		_, _ = builder.WriteString(p.typeString)
	}
	if p.input != "" {
		_, _ = builder.WriteString(` "`)
		_, _ = builder.WriteString(p.input)
		_, _ = builder.WriteString(`"`)
	}
	if p.err != nil {
		_, _ = builder.WriteString(`: `)
		_, _ = builder.WriteString(p.err.Error())
	}
	return builder.String()
}

// Unwrap returns the underlying error.
func (p *ParseError) Unwrap() error { return p.err }

// Input returns the input string that was attempted to be parsed.
func (p *ParseError) Input() string {
	return p.input
}
