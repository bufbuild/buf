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

package bufconfig

import (
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/slicesextended"
	"go.uber.org/multierr"
)

const (
	// DefaultLockFileName is the default file name you should use for buf.lock Files.
	DefaultLockFileName = "buf.lock"
)

// LockFile represents a buf.lock file.
type LockFile interface {
	// FileVersion returns the file version of the buf.lock file.
	//
	// To migrate a file between versions, use ReadLockFile ->
	// NewLockFile(newFileVersion, file.DepModuleKeys()) ->
	// WriteLockFile.
	FileVersion() FileVersion
	// DepModuleKeys returns the ModuleKeys representing the dependencies as specified in the buf.lock file.
	//
	// Note that ModuleKeys may not have CommitIDs with FileVersionV2.
	// CommitIDs are required for v1beta1 and v1 buf.lock files. Their existence will be verified
	// when calling NewFile or WriteFile for FileVersionV1Beta1 or FileVersionV1, and therefor
	// if FileVersion() is FileVersionV1Beta1 or FileVersionV1, all ModuleKeys will have CommitIDs.
	//
	// All ModuleKeys will have unique ModuleFullNames.
	// ModuleKeys are sorted by ModuleFullName.
	//
	// TODO: We need to add DigestTypes for all the deprecated digests. We then can handle
	// the fact that they're deprecated outside of this package. Another option is to add a
	// buflock.DeprecatedDigestTypeError to return from Digest(), and then handle that downstream.
	DepModuleKeys() []bufmodule.ModuleKey

	isLockFile()
}

// NewLockFile returns a new LockFile.
//
// Note that digests are lazily-loaded; if you need to ensure that all digests are valid, run
// ValidateLockFileDigests().
func NewLockFile(fileVersion FileVersion, depModuleKeys []bufmodule.ModuleKey) (LockFile, error) {
	lockFile, err := newLockFile(fileVersion, depModuleKeys)
	if err != nil {
		return nil, err
	}
	if err := checkV2SupportedYet(lockFile.FileVersion()); err != nil {
		return nil, err
	}
	return lockFile, nil
}

// ReadLockFile reads the File from the io.Reader.
//
// Note that digests are lazily-loaded; if you need to ensure that all digests are valid, run
// ValidateFileDigests().
func ReadLockFile(reader io.Reader) (LockFile, error) {
	lockFile, err := readLockFile(reader)
	if err != nil {
		return nil, err
	}
	if err := checkV2SupportedYet(lockFile.FileVersion()); err != nil {
		return nil, err
	}
	return lockFile, nil
}

// WriteLockFile writes the LockFile to the io.Writer.
func WriteLockFile(writer io.Writer, lockFile LockFile) error {
	if err := checkV2SupportedYet(lockFile.FileVersion()); err != nil {
		return err
	}
	return writeLockFile(writer, lockFile)
}

// ValidateLockFileDigests validates that all Digests on the ModuleKeys are valid, by calling
// each Digest() function.
//
// TODO: should we just ensure this property when returning from NewFile, ReadFile?
func ValidateLockFileDigests(lockFile lockFile) error {
	if err := checkV2SupportedYet(lockFile.FileVersion()); err != nil {
		return err
	}
	var errs []error
	for _, depModuleKey := range lockFile.DepModuleKeys() {
		if _, err := depModuleKey.Digest(); err != nil {
			errs = append(errs, err)
		}
	}
	return multierr.Combine(errs...)
}

// *** PRIVATE ***

type lockFile struct {
	fileVersion   FileVersion
	depModuleKeys []bufmodule.ModuleKey
}

func newLockFile(
	fileVersion FileVersion,
	depModuleKeys []bufmodule.ModuleKey,
) (*lockFile, error) {
	if err := validateNoDuplicateModuleKeysByModuleFullName(depModuleKeys); err != nil {
		return nil, err
	}
	// To make sure we aren't editing input.
	depModuleKeys = slicesextended.Copy(depModuleKeys)
	sort.Slice(
		depModuleKeys,
		func(i int, j int) bool {
			return depModuleKeys[i].ModuleFullName().String() < depModuleKeys[j].ModuleFullName().String()
		},
	)
	lockFile := &lockFile{
		fileVersion:   fileVersion,
		depModuleKeys: depModuleKeys,
	}
	if err := validateV1AndV1Beta1DepsHaveCommits(lockFile); err != nil {
		return nil, err
	}
	return lockFile, nil
}

func (l *lockFile) FileVersion() FileVersion {
	return l.fileVersion
}

func (l *lockFile) DepModuleKeys() []bufmodule.ModuleKey {
	return l.depModuleKeys
}

func (*lockFile) isLockFile() {}

func readLockFile(reader io.Reader) (LockFile, error) {
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
		var externalLockFile externalLockFileV1OrV1Beta1
		if err := encoding.UnmarshalYAMLStrict(data, &externalLockFile); err != nil {
			return nil, fmt.Errorf("failed to decode lock file as version %v: %w", fileVersion, err)
		}
		depModuleKeys := make([]bufmodule.ModuleKey, len(externalLockFile.Deps))
		for i, dep := range externalLockFile.Deps {
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
		return newLockFile(fileVersion, depModuleKeys)
	case FileVersionV2:
		var externalLockFile externalLockFileV2
		if err := encoding.UnmarshalYAMLStrict(data, &externalLockFile); err != nil {
			return nil, fmt.Errorf("failed to decode lock file as version %v: %w", fileVersion, err)
		}
		depModuleKeys := make([]bufmodule.ModuleKey, len(externalLockFile.Deps))
		for i, dep := range externalLockFile.Deps {
			dep := dep
			moduleFullName, err := bufmodule.ParseModuleFullName(dep.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to decode lock file: invalid module name: %w", err)
			}
			depModuleKey, err := bufmodule.NewModuleKey(
				moduleFullName,
				"",
				func() (bufcas.Digest, error) {
					return bufcas.ParseDigest(dep.Digest)
				},
			)
			if err != nil {
				return nil, err
			}
			depModuleKeys[i] = depModuleKey
		}
		return newLockFile(fileVersion, depModuleKeys)
	default:
		// This is a system error since we've already parsed.
		return nil, fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
}

func writeLockFile(writer io.Writer, lockFile LockFile) error {
	if err := validateV1AndV1Beta1DepsHaveCommits(lockFile); err != nil {
		return err
	}
	switch fileVersion := lockFile.FileVersion(); fileVersion {
	case FileVersionV1Beta1, FileVersionV1:
		depModuleKeys := lockFile.DepModuleKeys()
		externalLockFile := externalLockFileV1OrV1Beta1{
			Version: fileVersion.String(),
			Deps:    make([]externalLockFileDepV1OrV1Beta1, len(depModuleKeys)),
		}
		for i, depModuleKey := range depModuleKeys {
			digest, err := depModuleKey.Digest()
			if err != nil {
				return fmt.Errorf("failed to encode lock file: digest error: %w", err)
			}
			externalLockFile.Deps[i] = externalLockFileDepV1OrV1Beta1{
				Remote:     depModuleKey.ModuleFullName().Registry(),
				Owner:      depModuleKey.ModuleFullName().Owner(),
				Repository: depModuleKey.ModuleFullName().Name(),
				Commit:     depModuleKey.CommitID(),
				Digest:     digest.String(),
			}
		}
		// No need to sort - depModuleKeys is already sorted by ModuleFullName
		data, err := encoding.MarshalYAML(&externalLockFile)
		if err != nil {
			return fmt.Errorf("failed to encode lock file: %w", err)
		}
		_, err = writer.Write(append(lockFileHeader, data...))
		return err
	case FileVersionV2:
		depModuleKeys := lockFile.DepModuleKeys()
		externalLockFile := externalLockFileV2{
			Version: fileVersion.String(),
			Deps:    make([]externalLockFileDepV2, len(depModuleKeys)),
		}
		for i, depModuleKey := range depModuleKeys {
			digest, err := depModuleKey.Digest()
			if err != nil {
				return fmt.Errorf("failed to encode lock file: digest error: %w", err)
			}
			externalLockFile.Deps[i] = externalLockFileDepV2{
				Name:   depModuleKey.ModuleFullName().String(),
				Digest: digest.String(),
			}
		}
		// No need to sort - depModuleKeys is already sorted by ModuleFullName
		data, err := encoding.MarshalYAML(&externalLockFile)
		if err != nil {
			return fmt.Errorf("failed to encode lock file: %w", err)
		}
		_, err = writer.Write(append(lockFileHeader, data...))
		return err
	default:
		// This is a system error since we've already parsed.
		return fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
}

func validateNoDuplicateModuleKeysByModuleFullName(moduleKeys []bufmodule.ModuleKey) error {
	moduleFullNameStringMap := make(map[string]struct{})
	for _, moduleKey := range moduleKeys {
		moduleFullNameString := moduleKey.ModuleFullName().String()
		if _, ok := moduleFullNameStringMap[moduleFullNameString]; ok {
			return fmt.Errorf("duplicate module %q attempted to be added to lock file", moduleFullNameString)
		}
		moduleFullNameStringMap[moduleFullNameString] = struct{}{}
	}
	return nil
}

func validateV1AndV1Beta1DepsHaveCommits(lockFile LockFile) error {
	switch fileVersion := lockFile.FileVersion(); fileVersion {
	case FileVersionV1Beta1, FileVersionV1:
		for _, depModuleKey := range lockFile.DepModuleKeys() {
			if depModuleKey.CommitID() == "" {
				// This is a system error.
				return fmt.Errorf(
					"%s lock files require commits, however we did not have a commit for module %q",
					fileVersion.String(),
					depModuleKey.ModuleFullName().String(),
				)
			}
		}
		return nil
	case FileVersionV2:
		// We do not need commits in v2.
		return nil
	default:
		// This is a system error since we've already parsed.
		return fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
}
