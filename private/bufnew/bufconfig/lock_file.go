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
)

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

func (f *lockFile) FileVersion() FileVersion {
	return f.fileVersion
}

func (f *lockFile) DepModuleKeys() []bufmodule.ModuleKey {
	return f.depModuleKeys
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
