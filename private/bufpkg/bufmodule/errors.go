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

package bufmodule

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

var (
	// ErrNoTargetProtoFiles is the error to return if no target .proto files were found in situations where
	// they were expected to be found.
	//
	// Pre-refactor, we had extremely exacting logic that determined if --path and --exclude-path were valid
	// paths, which almost no CLI tool does. This logic had a heavy burden, when typically this error message
	// is enough (and again, is more than almost any other CLI does - most CLIs silently move on if invalid
	// paths are specified). The pre-refactor logic was the "allowNotExist" logic. Removing the allowNotExist
	// logic was not a breaking change - we do not error in any place that we previously did not.
	//
	// This is used by bufctl.Controller.GetTargetImageWithConfigs, bufworkspace.WorkspaceProvider.GetWorkspaceForBucket, and bufimage.BuildImage.
	//
	// We do assume flag names here, but we're just going with reality.
	ErrNoTargetProtoFiles = errors.New("no .proto files were targeted. This can occur if no .proto files are found in your input, --path points to files that do not exist, or --exclude-path excludes all files.")
)

// ImportNotExistError is the error returned from ModuleDeps() if an import does not exist.
//
// Unwrap() always returns fs.ErrNotExist.
type ImportNotExistError struct {
	fileInfo   FileInfo
	importPath string
}

// Error implements the error interface.
func (i *ImportNotExistError) Error() string {
	if i == nil {
		return ""
	}
	var builder strings.Builder
	if i.fileInfo != nil {
		if externalPath := i.fileInfo.ExternalPath(); externalPath != "" {
			_, _ = builder.WriteString(externalPath)
			_, _ = builder.WriteString(`: `)
		}
	}
	if i.importPath != "" {
		_, _ = builder.WriteString(`import "`)
		_, _ = builder.WriteString(i.importPath)
		_, _ = builder.WriteString(`": `)
	}
	_, _ = builder.WriteString(i.Unwrap().Error())
	return builder.String()
}

// Unwrap returns fs.ErrNotExist.
func (i *ImportNotExistError) Unwrap() error {
	if i == nil {
		return nil
	}
	return fs.ErrNotExist
}

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
	if p == nil {
		return ""
	}
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
func (p *ParseError) Unwrap() error {
	if p == nil {
		return nil
	}
	return p.err
}

// Input returns the input string that was attempted to be parsed.
func (p *ParseError) Input() string {
	if p == nil {
		return ""
	}
	return p.input
}

// *** PRIVATE ***

func newErrNoProtoFiles(moduleID string) error {
	return fmt.Errorf("module %q had no .proto files", moduleID)
}
