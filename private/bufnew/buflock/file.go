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
	"sort"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/slicesextended"
)

type file struct {
	fileVersion   FileVersion
	depModuleKeys []bufmodule.ModuleKey
}

func newFile(
	fileVersion FileVersion,
	depModuleKeys []bufmodule.ModuleKey,
) (*file, error) {
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
	return &file{
		fileVersion:   fileVersion,
		depModuleKeys: depModuleKeys,
	}, nil
}

func (f *file) FileVersion() FileVersion {
	return f.fileVersion
}

func (f *file) DepModuleKeys() []bufmodule.ModuleKey {
	return f.depModuleKeys
}

func (*file) isFile() {}

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
	case FileVersionV2:
		var externalFile externalFileV2
		if err := encoding.UnmarshalYAMLStrict(data, &externalFile); err != nil {
			return nil, fmt.Errorf("failed to decode lock file as version %v: %w", fileVersion, err)
		}
		depModuleKeys := make([]bufmodule.ModuleKey, len(externalFile.Deps))
		for i, dep := range externalFile.Deps {
			dep := dep
			moduleFullName, err := bufmodule.ParseModuleFullName(dep.Module)
			if err != nil {
				return nil, fmt.Errorf("failed to decode lock file: invalid module name: %w", err)
			}
			// TODO: We should be able to remove commit. See comment on externalFileDepV2.
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
	case FileVersionV2:
		depModuleKeys := file.DepModuleKeys()
		externalFile := externalFileV2{
			Version: fileVersion.String(),
			Deps:    make([]externalFileDepV2, len(depModuleKeys)),
		}
		for i, depModuleKey := range depModuleKeys {
			digest, err := depModuleKey.Digest()
			if err != nil {
				return fmt.Errorf("failed to encode lock file: digest error: %w", err)
			}
			externalFile.Deps[i] = externalFileDepV2{
				Module: depModuleKey.ModuleFullName().String(),
				// TODO: We should be able to remove commit. See comment on externalFileDepV2.
				Commit: depModuleKey.CommitID(),
				Digest: digest.String(),
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
