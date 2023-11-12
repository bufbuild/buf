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

package buflock

import (
	"errors"
	"fmt"
	"io"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"go.uber.org/multierr"
)

// File represents a buf.lock file.
type File interface {
	// FileVersion returns the file version of the buf.lock file.
	//
	// To migrate a file between versions, use ReadFile -> NewFile(newVersion, file.DepModuleKeys()) -> WriteFile.
	FileVersion() FileVersion
	// DepModuleKeys returns the ModuleKeys representing the dependencies as specified in the buf.lock file.
	//
	// All ModuleKeys will have unique ModuleFullNames.
	// ModuleKeys are sorted by ModuleFullName.
	//
	// TODO: We need to add DigestTypes for all the deprecated digests. We then can handle
	// the fact that they're deprecated outside of this package.
	DepModuleKeys() []bufmodule.ModuleKey

	isFile()
}

// NewFile returns a new File.
//
// Note that digests are lazily-loaded; if you need to ensure that all digests are valid, run
// ValidateFileDigests().
func NewFile(fileVersion FileVersion, depModuleKeys []bufmodule.ModuleKey) (File, error) {
	return newFile(fileVersion, depModuleKeys)
}

// ReadFile reads the File from the io.Reader.
//
// Note that digests are lazily-loaded; if you need to ensure that all digests are valid, run
// ValidateFileDigests().
func ReadFile(reader io.Reader) (File, error) {
	return readFile(reader)
}

// WriteFile writes the File to the io.Writer.
func WriteFile(writer io.Writer, file File) error {
	return writeFile(writer, file)
}

// ValidateFileDigests validates that all Digests on the ModuleKeys are valid, by calling
// each Digest() function.
//
// TODO: should we just ensure this property when returning from NewFile, ReadFile?
func ValidateFileDigests(file File) error {
	var errs []error
	for _, depModuleKey := range file.DepModuleKeys() {
		if _, err := depModuleKey.Digest(); err != nil {
			errs = append(errs, err)
		}
	}
	return multierr.Combine(errs...)
}

// *** PRIVATE ***

func readFile(reader io.Reader) (File, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var externalFileVersion externalFileVersion
	if err := encoding.UnmarshalYAMLNonStrict(data, &externalFileVersion); err != nil {
		return nil, fmt.Errorf("failed to decode lock file as YAML: %w", err)
	}
	fileVersion, err := parseFileVersion(externalFileVersion.Version)
	if err != nil {
		return nil, err
	}
	switch fileVersion {
	case FileVersionV1Beta1, FileVersionV1:
		var externalFile externalFileV1OrV1Beta1
		if err := encoding.UnmarshalYAMLStrict(data, &externalFile); err != nil {
			return nil, fmt.Errorf("failed to decode lock file as version %v: %w", fileVersion, err)
		}
		depModuleKeys := make([]bufmodule.ModuleKey, len(externalFile.Deps))
		for i, dep := range externalFile.Deps {
			dep := dep
			moduleFullName, err := bufmodule.NewModuleFullName(
				dep.Remote,
				dep.Owner,
				dep.Repository,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to decode lock file: invalid module name: %w", err)
			}
			if dep.Commit == "" {
				return nil, errors.New("failed to decode lock file: no commit specified")
			}
			depModuleKey, err := bufmodule.NewModuleKey(
				moduleFullName,
				dep.Commit,
				func() (bufcas.Digest, error) {
					return bufcas.ParseDigest(dep.Digest)
				},
			)
			if err != nil {
				return nil, err
			}
			depModuleKeys[i] = depModuleKey
		}
		return newFile(fileVersion, depModuleKeys)
	default:
		// This is a system error since we've already parsed.
		return nil, fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
}

func writeFile(writer io.Writer, file File) error {
	switch fileVersion := file.FileVersion(); fileVersion {
	case FileVersionV1Beta1, FileVersionV1:
		depModuleKeys := file.DepModuleKeys()
		externalFile := externalFileV1OrV1Beta1{
			Version: fileVersion.String(),
			Deps:    make([]externalFileDepV1OrV1Beta1, len(depModuleKeys)),
		}
		for i, depModuleKey := range depModuleKeys {
			digest, err := depModuleKey.Digest()
			if err != nil {
				return fmt.Errorf("failed to encode lock file: digest error: %w", err)
			}
			externalFile.Deps[i] = externalFileDepV1OrV1Beta1{
				Remote:     depModuleKey.ModuleFullName().Registry(),
				Owner:      depModuleKey.ModuleFullName().Owner(),
				Repository: depModuleKey.ModuleFullName().Name(),
				Commit:     depModuleKey.CommitID(),
				Digest:     digest.String(),
			}
		}
		// No need to sort - depModuleKeys is already sorted by ModuleFullName
		data, err := encoding.MarshalYAML(&externalFile)
		if err != nil {
			return fmt.Errorf("failed to encode lock file: %w", err)
		}
		_, err = writer.Write(append(fileHeader, data...))
		return err
	default:
		// This is a system error since we've already parsed.
		return fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
}
