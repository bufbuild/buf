// Copyright 2020-2025 Buf Technologies, Inc.
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
	"io/fs"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/google/uuid"
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

// ModuleCycleError is the error returned if a cycle is detected in module dependencies.
type ModuleCycleError struct {
	// Descriptions are the module descriptions that represent the cycle.
	Descriptions []string
}

// Error implements the error interface.
func (m *ModuleCycleError) Error() string {
	if m == nil {
		return ""
	}
	var builder strings.Builder
	_, _ = builder.WriteString("cycle detected in module dependencies:\n")
	for i, description := range m.Descriptions {
		if i == 0 {
			_, _ = builder.WriteString("    ")
		} else {
			_, _ = builder.WriteString(" -> ")
		}
		_, _ = builder.WriteString(description)
		if i != len(m.Descriptions)-1 {
			_, _ = builder.WriteString("\n")
		}
	}
	return builder.String()
}

// DuplicateProtoPathError is the error returned if a .proto file with the same path
// is detected in two or more Modules.
//
// This check is done as part of ModuleReadBucket.Walks, and Module.ModuleDeps.
type DuplicateProtoPathError struct {
	// ProtoPath is the path of the .proto that is duplicated.
	//
	// A well-formed DuplicateProtoPathError will have a normalized and non-empty ProtoPath.
	ProtoPath string
	// ModuleDescriptions are the Module descriptions that contain the ProtoPath.
	//
	// A well-formed DuplicateProtoPathError will have two or more Module descriptions.
	ModuleDescriptions []string
}

// Error implements the error interface.
func (d *DuplicateProtoPathError) Error() string {
	if d == nil {
		return ""
	}
	var builder strings.Builder
	// Writing even if the error is malformed via d.Path being empty.
	_, _ = builder.WriteString(d.ProtoPath)
	_, _ = builder.WriteString(" is contained in multiple modules:\n")
	for i, moduleDescription := range d.ModuleDescriptions {
		_, _ = builder.WriteString("  ")
		_, _ = builder.WriteString(moduleDescription)
		if i != len(d.ModuleDescriptions)-1 {
			_, _ = builder.WriteString("\n")
		}
	}
	return builder.String()
}

// NoProtoFilesError is the error returned if a Module has no .proto files.
//
// This check is done as part of ModuleReadBucket.Walks.
type NoProtoFilesError struct {
	// ModuleDescription is the description of the Module that has no .proto files.
	//
	// A well-formed NoProtoFilesError will have a non-empty ModuleDescription.
	ModuleDescription string
}

// Error implements the error interface.
func (n *NoProtoFilesError) Error() string {
	if n == nil {
		return ""
	}
	var builder strings.Builder
	_, _ = builder.WriteString(`Module "`)
	// Writing even if the error is malformed via d.ModuleDescription being empty.
	_, _ = builder.WriteString(n.ModuleDescription)
	_, _ = builder.WriteString(`" had no .proto files`)
	return builder.String()
}

// DigestMismatchError is the error returned if the Digest of a downloaded Module or Commit
// does not match the expected digest in a buf.lock file.
type DigestMismatchError struct {
	FullName       bufparse.FullName
	CommitID       uuid.UUID
	ExpectedDigest Digest
	ActualDigest   Digest
}

// Error implements the error interface.
func (m *DigestMismatchError) Error() string {
	if m == nil {
		return ""
	}
	var builder strings.Builder
	_, _ = builder.WriteString(`*** Digest verification failed`)
	if m.FullName != nil {
		_, _ = builder.WriteString(` for "`)
		_, _ = builder.WriteString(m.FullName.String())
		if m.CommitID != uuid.Nil {
			_, _ = builder.WriteString(`:`)
			_, _ = builder.WriteString(uuidutil.ToDashless(m.CommitID))
		}
		_, _ = builder.WriteString(`"`)
	}
	_, _ = builder.WriteString(` ***`)
	_, _ = builder.WriteString("\n")
	if m.ExpectedDigest != nil && m.ActualDigest != nil {
		_, _ = builder.WriteString("\t")
		_, _ = builder.WriteString(`Expected digest (from buf.lock): "`)
		_, _ = builder.WriteString(m.ExpectedDigest.String())
		_, _ = builder.WriteString(`"`)
		_, _ = builder.WriteString("\n")
		_, _ = builder.WriteString("\t")
		_, _ = builder.WriteString(`Actual digest: "`)
		_, _ = builder.WriteString(m.ActualDigest.String())
		_, _ = builder.WriteString(`"`)
		_, _ = builder.WriteString("\n")
	}
	_, _ = builder.WriteString("\t")
	_, _ = builder.WriteString(`This may be the result of a hand-edited or corrupted buf.lock file, a corrupted local cache, and/or an attack.`)
	_, _ = builder.WriteString("\n")
	_, _ = builder.WriteString("\t")
	_, _ = builder.WriteString(`To clear your local cache, run "buf registry cc".`)
	return builder.String()
}
