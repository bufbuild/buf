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

package bufconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	// DefaultBufYAMLFileName is the default buf.yaml file name.
	DefaultBufYAMLFileName = "buf.yaml"

	// Originally we thought we were going to move to buf.mod, and had this around for
	// a while, but then reverted back to buf.yaml. We still need to support buf.mod as
	// we released with it, however.
	oldBufYAMLFileName        = "buf.mod"
	defaultBufYAMLFileVersion = FileVersionV1Beta1
)

var (
	// ordered
	bufYAMLFileNames                       = []string{DefaultBufYAMLFileName, oldBufYAMLFileName}
	bufYAMLFileNameToSupportedFileVersions = map[string]map[FileVersion]struct{}{
		DefaultBufYAMLFileName: {
			FileVersionV1Beta1: struct{}{},
			FileVersionV1:      struct{}{},
			FileVersionV2:      struct{}{},
		},
		oldBufYAMLFileName: {
			FileVersionV1Beta1: struct{}{},
			FileVersionV1:      struct{}{},
		},
	}
)

// BufYAMLFile represents a buf.yaml file.
type BufYAMLFile interface {
	File

	// ModuleConfigs returns the ModuleConfigs for the File.
	//
	// For v1 buf.yaml, this will only have a single ModuleConfig.
	//
	// This will always be non-empty.
	// All ModuleConfigs will have unique ModuleFullNames.
	// Sorted by DirPath.
	ModuleConfigs() []ModuleConfig
	// ConfiguredDepModuleRefs returns the configured dependencies of the Workspace as ModuleRefs.
	//
	// These come from buf.yaml files.
	//
	// The ModuleRefs in this list will be unique by ModuleFullName.
	// Sorted by ModuleFullName.
	ConfiguredDepModuleRefs() []bufmodule.ModuleRef

	isBufYAMLFile()
}

// NewBufYAMLFile returns a new validated BufYAMLFile.
//
// This should generally not be used outside of testing - use GetBufYAMLFileForPrefix instead.
func NewBufYAMLFile(
	fileVersion FileVersion,
	moduleConfigs []ModuleConfig,
	configuredDepModuleRefs []bufmodule.ModuleRef,
) (BufYAMLFile, error) {
	return newBufYAMLFile(fileVersion, nil, moduleConfigs, configuredDepModuleRefs)
}

// GetBufYAMLFileForPrefix gets the buf.yaml file at the given bucket prefix.
//
// The buf.yaml file will be attempted to be read at prefix/buf.yaml.
func GetBufYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (BufYAMLFile, error) {
	return getFileForPrefix(ctx, bucket, prefix, bufYAMLFileNames, bufYAMLFileNameToSupportedFileVersions, readBufYAMLFile)
}

// GetBufYAMLFileForOverride get the buf.yaml file for either the usually-flag-based override.
//
//   - If the override is set and ends in .json, .yaml, or .yml, the override is treated as a
//     **direct file path on disk** and read (ie not via buckets).
//   - If the override is otherwise non-empty, it is treated as raw data.
//
// This function is the result of the endlessly annoying and shortsighted design decision that the
// original author of this repository made to allow overriding configuration files on the command line.
// Of course, the original author never envisioned buf.work.yamls, merging buf.work.yamls into buf.yamls,
// buf.gen.yamls, or anything of the like, and was very concentrated on "because Bazel."
func GetBufYAMLFileForOverride(override string) (BufYAMLFile, error) {
	var data []byte
	var fileName string
	var err error
	switch filepath.Ext(override) {
	case ".json", ".yaml", ".yml":
		data, err = os.ReadFile(override)
		if err != nil {
			return nil, fmt.Errorf("could not read file: %v", err)
		}
		fileName = filepath.Base(fileName)
	default:
		data = []byte(override)
	}
	return readFile(bytes.NewReader(data), fileName, readBufYAMLFile)
}

// GetBufYAMLFileForOverride get the buf.yaml file for either the usually-flag-based override,
// or if the override is not set, falls back to the prefix.
func GetBufYAMLFileForPrefixOrOverride(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	override string,
) (BufYAMLFile, error) {
	if override != "" {
		return GetBufYAMLFileForOverride(override)
	}
	return GetBufYAMLFileForPrefix(ctx, bucket, prefix)
}

// GetBufYAMLFileForPrefix gets the buf.yaml file version at the given bucket prefix.
//
// The buf.yaml file will be attempted to be read at prefix/buf.yaml.
func GetBufYAMLFileVersionForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (FileVersion, error) {
	return getFileVersionForPrefix(ctx, bucket, prefix, bufYAMLFileNames, bufYAMLFileNameToSupportedFileVersions, true, FileVersionV2, defaultBufYAMLFileVersion)
}

// PutBufYAMLFileForPrefix puts the buf.yaml file at the given bucket prefix.
//
// The buf.yaml file will be attempted to be written to prefix/buf.yaml.
// The buf.yaml file will be written atomically.
func PutBufYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.WriteBucket,
	prefix string,
	bufYAMLFile BufYAMLFile,
) error {
	return putFileForPrefix(ctx, bucket, prefix, bufYAMLFile, DefaultBufYAMLFileName, bufYAMLFileNameToSupportedFileVersions, writeBufYAMLFile)
}

// ReadBufYAMLFile reads the BufYAMLFile from the io.Reader.
//
// fileName may be empty.
func ReadBufYAMLFile(reader io.Reader, fileName string) (BufYAMLFile, error) {
	return readFile(reader, fileName, readBufYAMLFile)
}

// WriteBufYAMLFile writes the BufYAMLFile to the io.Writer.
func WriteBufYAMLFile(writer io.Writer, bufYAMLFile BufYAMLFile) error {
	return writeFile(writer, bufYAMLFile, writeBufYAMLFile)
}

// *** PRIVATE ***

type bufYAMLFile struct {
	fileVersion             FileVersion
	objectData              ObjectData
	moduleConfigs           []ModuleConfig
	configuredDepModuleRefs []bufmodule.ModuleRef
}

func newBufYAMLFile(
	fileVersion FileVersion,
	objectData ObjectData,
	moduleConfigs []ModuleConfig,
	configuredDepModuleRefs []bufmodule.ModuleRef,
) (*bufYAMLFile, error) {
	if (fileVersion == FileVersionV1Beta1 || fileVersion == FileVersionV1) && len(moduleConfigs) > 1 {
		return nil, fmt.Errorf("had %d ModuleConfigs passed to NewBufYAMLFile for FileVersion %v", len(moduleConfigs), fileVersion)
	}
	if len(moduleConfigs) == 0 {
		return nil, errors.New("had 0 ModuleConfigs passed to NewBufYAMLFile")
	}
	for _, moduleConfig := range moduleConfigs {
		if (fileVersion == FileVersionV1Beta1 || fileVersion == FileVersionV1) && moduleConfig.DirPath() != "." {
			return nil, fmt.Errorf("invalid DirPath %q in NewBufYAMLFile for %v ModuleConfig", moduleConfig.DirPath(), fileVersion)
		}
		if moduleConfig == nil {
			return nil, errors.New("ModuleConfig was nil in NewBufYAMLFile")
		}
		if fileVersion != moduleConfig.LintConfig().FileVersion() {
			return nil, fmt.Errorf("FileVersion %v was passed to NewBufYAMLFile but had LintConfig FileVersion %v", fileVersion, moduleConfig.LintConfig().FileVersion())
		}
		if fileVersion != moduleConfig.BreakingConfig().FileVersion() {
			return nil, fmt.Errorf("FileVersion %v was passed to NewBufYAMLFile but had BreakingConfig FileVersion %v", fileVersion, moduleConfig.BreakingConfig().FileVersion())
		}
	}
	// Zero values are not added to duplicates.
	duplicateModuleConfigDirPaths := slicesext.Duplicates(
		slicesext.Map(
			moduleConfigs,
			func(moduleConfig ModuleConfig) string {
				return moduleConfig.DirPath()
			},
		),
	)
	if len(duplicateModuleConfigDirPaths) > 0 {
		return nil, fmt.Errorf("module directory %q seen more than once", strings.Join(duplicateModuleConfigDirPaths, ", "))
	}
	// Zero values are not added to duplicates.
	if _, err := bufmodule.ModuleFullNameStringToUniqueValue(moduleConfigs); err != nil {
		return nil, err
	}
	if _, err := bufmodule.ModuleFullNameStringToUniqueValue(configuredDepModuleRefs); err != nil {
		return nil, err
	}
	sort.Slice(
		moduleConfigs,
		func(i int, j int) bool {
			return moduleConfigs[i].DirPath() < moduleConfigs[j].DirPath()
		},
	)
	sort.Slice(
		configuredDepModuleRefs,
		func(i int, j int) bool {
			return configuredDepModuleRefs[i].ModuleFullName().String() <
				configuredDepModuleRefs[j].ModuleFullName().String()
		},
	)
	return &bufYAMLFile{
		fileVersion:             fileVersion,
		objectData:              objectData,
		moduleConfigs:           moduleConfigs,
		configuredDepModuleRefs: configuredDepModuleRefs,
	}, nil
}

func (c *bufYAMLFile) FileVersion() FileVersion {
	return c.fileVersion
}

func (*bufYAMLFile) FileType() FileType {
	return FileTypeBufYAML
}

func (c *bufYAMLFile) ObjectData() ObjectData {
	return c.objectData
}

func (c *bufYAMLFile) ModuleConfigs() []ModuleConfig {
	return slicesext.Copy(c.moduleConfigs)
}

func (c *bufYAMLFile) ConfiguredDepModuleRefs() []bufmodule.ModuleRef {
	return slicesext.Copy(c.configuredDepModuleRefs)
}

func (*bufYAMLFile) isBufYAMLFile() {}
func (*bufYAMLFile) isFile()        {}
func (*bufYAMLFile) isFileInfo()    {}

func readBufYAMLFile(
	data []byte,
	objectData ObjectData,
	allowJSON bool,
) (BufYAMLFile, error) {
	// We've always required a file version for buf.yaml files.
	fileVersion, err := getFileVersionForData(data, allowJSON, true, bufYAMLFileNameToSupportedFileVersions, FileVersionV2, defaultBufYAMLFileVersion)
	if err != nil {
		return nil, err
	}
	switch fileVersion {
	case FileVersionV1Beta1, FileVersionV1:
		var externalBufYAMLFile externalBufYAMLFileV1Beta1V1
		if err := getUnmarshalStrict(allowJSON)(data, &externalBufYAMLFile); err != nil {
			return nil, fmt.Errorf("invalid as version %v: %w", fileVersion, err)
		}
		if fileVersion == FileVersionV1 && len(externalBufYAMLFile.Build.Roots) > 0 {
			return nil, fmt.Errorf("build.roots cannot be set on version %v: %v", fileVersion, externalBufYAMLFile.Build.Roots)
		}
		var moduleFullName bufmodule.ModuleFullName
		if externalBufYAMLFile.Name != "" {
			moduleFullName, err = bufmodule.ParseModuleFullName(externalBufYAMLFile.Name)
			if err != nil {
				return nil, err
			}
		}
		rootToExcludes, err := getRootToExcludes(externalBufYAMLFile.Build.Roots, externalBufYAMLFile.Build.Excludes)
		if err != nil {
			return nil, err
		}
		configuredDepModuleRefs, err := getConfiguredDepModuleRefsForExternalDeps(externalBufYAMLFile.Deps)
		if err != nil {
			return nil, err
		}
		lintConfig, err := getLintConfigForExternalLintV1Beta1V1(
			fileVersion,
			externalBufYAMLFile.Lint,
			".",
			true,
		)
		if err != nil {
			return nil, err
		}
		breakingConfig, err := getBreakingConfigForExternalBreaking(
			fileVersion,
			externalBufYAMLFile.Breaking,
			".",
			true,
		)
		if err != nil {
			return nil, err
		}
		moduleConfig, err := newModuleConfig(
			"",
			moduleFullName,
			rootToExcludes,
			lintConfig,
			breakingConfig,
		)
		if err != nil {
			return nil, err
		}
		return newBufYAMLFile(
			fileVersion,
			objectData,
			[]ModuleConfig{
				moduleConfig,
			},
			configuredDepModuleRefs,
		)
	case FileVersionV2:
		var externalBufYAMLFile externalBufYAMLFileV2
		if err := getUnmarshalStrict(allowJSON)(data, &externalBufYAMLFile); err != nil {
			return nil, fmt.Errorf("invalid as version %v: %w", fileVersion, err)
		}
		externalModules := externalBufYAMLFile.Modules
		if len(externalModules) == 0 {
			externalModules = []externalBufYAMLFileModuleV2{
				{
					Path: ".",
					Name: externalBufYAMLFile.Name,
				},
			}
		} else if externalBufYAMLFile.Name != "" {
			return nil, errors.New("top-level name key cannot be specified if modules are specified, you must specify the name on each individual module, the top-level name key is only for the default case where you have one module at path \".\".")
		}
		// If a module does not have its own lint section, then we use this as the default.
		defaultExternalLintConfig := externalBufYAMLFile.Lint
		defaultExternalBreakingConfig := externalBufYAMLFile.Breaking
		var moduleConfigs []ModuleConfig
		for _, externalModule := range externalModules {
			dirPath := externalModule.Path
			if dirPath == "" {
				dirPath = "."
			}
			dirPath, err := normalpath.NormalizeAndValidate(dirPath)
			if err != nil {
				return nil, fmt.Errorf("invalid module path: %w", err)
			}
			var moduleFullName bufmodule.ModuleFullName
			if externalModule.Name != "" {
				moduleFullName, err = bufmodule.ParseModuleFullName(externalModule.Name)
				if err != nil {
					return nil, err
				}
			}
			// Makes sure that the given path is normalized, validated, and contained within dirPath.
			//
			// Used on excludes, and lint and breaking change paths.
			//
			// We first check that a given path is within a module before passing it to this function
			// if the path came from defaultExternalLintConfig or defaultExternalBreakingConfig.
			// The only root for v2 buf.yamls must be ".", so we have to make the excludes relative first.
			relExcludes, err := slicesext.MapError(
				externalModule.Excludes,
				func(path string) (string, error) {
					path, err := normalpath.NormalizeAndValidate(path)
					if err != nil {
						// user error
						return "", fmt.Errorf("invalid exclude path: %w", err)
					}
					if path == dirPath {
						return "", fmt.Errorf("exclude path %q is equal to module directory %q", path, dirPath)
					}
					if !normalpath.EqualsOrContainsPath(dirPath, path, normalpath.Relative) {
						return "", fmt.Errorf("exclude path %q does not reside within module directory %q", path, dirPath)
					}
					return normalpath.Rel(dirPath, path)
				},
			)
			if err != nil {
				return nil, err
			}
			rootToExcludes, err := getRootToExcludes([]string{"."}, relExcludes)
			if err != nil {
				return nil, err
			}
			externalLintConfig := defaultExternalLintConfig
			lintRequirePathsToBeContainedWithinModuleDirPath := false
			if !externalModule.Lint.isEmpty() {
				externalLintConfig = externalModule.Lint
				// We have a module-specific configuration, all paths must be within the module.
				lintRequirePathsToBeContainedWithinModuleDirPath = true
			}
			lintConfig, err := getLintConfigForExternalLintV2(
				fileVersion,
				externalLintConfig,
				dirPath,
				lintRequirePathsToBeContainedWithinModuleDirPath,
			)
			if err != nil {
				return nil, err
			}
			externalBreakingConfig := defaultExternalBreakingConfig
			breakingRequirePathsToBeContainedWithinModuleDirPath := false
			if !externalModule.Breaking.isEmpty() {
				externalBreakingConfig = externalModule.Breaking
				// We have a module-specific configuration, all paths must be within the module.
				breakingRequirePathsToBeContainedWithinModuleDirPath = true
			}
			breakingConfig, err := getBreakingConfigForExternalBreaking(
				fileVersion,
				externalBreakingConfig,
				dirPath,
				breakingRequirePathsToBeContainedWithinModuleDirPath,
			)
			if err != nil {
				return nil, err
			}
			moduleConfig, err := newModuleConfig(
				dirPath,
				moduleFullName,
				rootToExcludes,
				lintConfig,
				breakingConfig,
			)
			if err != nil {
				return nil, err
			}
			moduleConfigs = append(moduleConfigs, moduleConfig)
		}
		configuredDepModuleRefs, err := getConfiguredDepModuleRefsForExternalDeps(externalBufYAMLFile.Deps)
		if err != nil {
			return nil, err
		}
		return newBufYAMLFile(
			fileVersion,
			objectData,
			moduleConfigs,
			configuredDepModuleRefs,
		)
	default:
		// This is a system error since we've already parsed.
		return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
	}
}

func writeBufYAMLFile(writer io.Writer, bufYAMLFile BufYAMLFile) error {
	switch fileVersion := bufYAMLFile.FileVersion(); fileVersion {
	case FileVersionV1Beta1, FileVersionV1:
		moduleConfigs := bufYAMLFile.ModuleConfigs()
		// Just some extra sanity checking that we've properly validated.
		if len(moduleConfigs) != 1 {
			return syserror.Newf("expected 1 ModuleConfig, got %d", len(moduleConfigs))
		}
		moduleConfig := moduleConfigs[0]
		// Just some extra sanity checking that we've properly validated.
		if moduleConfig.DirPath() != "." {
			return syserror.Newf("expected ModuleConfig DirPath to be . but was %q", moduleConfig.DirPath())
		}
		externalBufYAMLFile := externalBufYAMLFileV1Beta1V1{
			Version: fileVersion.String(),
		}
		// Already sorted.
		externalBufYAMLFile.Deps = slicesext.Map(
			bufYAMLFile.ConfiguredDepModuleRefs(),
			func(moduleRef bufmodule.ModuleRef) string {
				return moduleRef.String()
			},
		)
		if moduleFullName := moduleConfig.ModuleFullName(); moduleFullName != nil {
			externalBufYAMLFile.Name = moduleFullName.String()
		}
		rootToExcludes := moduleConfig.RootToExcludes()
		excludes, ok := rootToExcludes["."]
		switch fileVersion {
		case FileVersionV1:
			if len(rootToExcludes) != 1 {
				return syserror.Newf("had rootToExcludes length %d for NewModuleConfig with FileVersion %v", len(rootToExcludes), fileVersion)
			}
			if !ok {
				return syserror.Newf("had rootToExcludes without key \".\" for NewModuleConfig with FileVersion %v", fileVersion)
			}
			for _, exclude := range excludes {
				// Excludes are defined to be sorted.
				externalBufYAMLFile.Build.Excludes = append(
					externalBufYAMLFile.Build.Excludes,
					// Remember, in buf.yaml files, excludes are not relative to roots.
					normalpath.Join(".", exclude),
				)
			}
		case FileVersionV1Beta1:
			// If "." -> empty, do not add anything.
			if len(rootToExcludes) != 1 || !(ok && len(excludes) == 0) {
				roots := slicesext.MapKeysToSortedSlice(rootToExcludes)
				for _, root := range roots {
					externalBufYAMLFile.Build.Roots = append(
						externalBufYAMLFile.Build.Roots,
						root,
					)
					for _, exclude := range rootToExcludes[root] {
						// Excludes are defined to be sorted.
						externalBufYAMLFile.Build.Excludes = append(
							externalBufYAMLFile.Build.Excludes,
							// Remember, in buf.yaml files, excludes are not relative to roots.
							normalpath.Join(root, exclude),
						)
					}
				}
			}
		default:
			// Unreachable - we're in a v1/v1beta1 case statement above.
			return syserror.Newf("expected v1 or v1beta1, got FileVersion: %v", fileVersion)
		}
		externalBufYAMLFile.Lint = getExternalLintV1Beta1V1ForLintConfig(moduleConfig.LintConfig(), ".")
		externalBufYAMLFile.Breaking = getExternalBreakingForBreakingConfig(moduleConfig.BreakingConfig(), ".")
		data, err := encoding.MarshalYAML(&externalBufYAMLFile)
		if err != nil {
			return err
		}
		_, err = writer.Write(data)
		return err
	case FileVersionV2:
		externalBufYAMLFile := externalBufYAMLFileV2{
			Version: fileVersion.String(),
		}
		// Already sorted.
		externalBufYAMLFile.Deps = slicesext.Map(
			bufYAMLFile.ConfiguredDepModuleRefs(),
			func(moduleRef bufmodule.ModuleRef) string {
				return moduleRef.String()
			},
		)
		// Keep maps of the JSON-marshaled data to the external lint and breaking configs.
		//
		// If both of these maps are of length 0 or 1, we say that the user really just has a
		// single configuration for lint and breaking, and we infer that they only want
		// to have a single top-level lint and breaking config. In this case, we delete
		// all of the per-module lint and breaking configs, and install the sole value
		// from each.
		//
		// We could make other decisions: if there are two or more matching configs, do a default,
		// and then just override the non-matching, but this gets complicated. The current logic
		// takes care of the base case when writing buf.yaml files.
		stringToExternalLint := make(map[string]externalBufYAMLFileLintV2)
		stringToExternalBreaking := make(map[string]externalBufYAMLFileBreakingV1Beta1V1V2)

		for _, moduleConfig := range bufYAMLFile.ModuleConfigs() {
			moduleDirPath := moduleConfig.DirPath()
			joinDirPath := func(importPath string) string {
				return filepath.Join(moduleDirPath, importPath)
			}
			externalModule := externalBufYAMLFileModuleV2{
				Path: moduleDirPath,
			}
			if moduleFullName := moduleConfig.ModuleFullName(); moduleFullName != nil {
				externalModule.Name = moduleFullName.String()
			}
			rootToExcludes := moduleConfig.RootToExcludes()
			if len(rootToExcludes) != 1 {
				return syserror.Newf("had rootToExcludes length %d for NewModuleConfig with FileVersion %v", len(rootToExcludes), fileVersion)
			}
			excludes, ok := rootToExcludes["."]
			if !ok {
				return syserror.Newf("had rootToExcludes without key \".\" for NewModuleConfig with FileVersion %v", fileVersion)
			}
			externalModule.Excludes = slicesext.Map(excludes, joinDirPath)

			externalLint := getExternalLintV2ForLintConfig(moduleConfig.LintConfig(), moduleDirPath)
			externalLintData, err := json.Marshal(externalLint)
			if err != nil {
				return syserror.Wrap(err)
			}
			stringToExternalLint[string(externalLintData)] = externalLint
			externalModule.Lint = externalLint

			externalBreaking := getExternalBreakingForBreakingConfig(moduleConfig.BreakingConfig(), moduleDirPath)
			externalBreakingData, err := json.Marshal(externalBreaking)
			if err != nil {
				return syserror.Wrap(err)
			}
			stringToExternalBreaking[string(externalBreakingData)] = externalBreaking
			externalModule.Breaking = externalBreaking

			externalBufYAMLFile.Modules = append(externalBufYAMLFile.Modules, externalModule)
		}

		if len(stringToExternalLint) <= 1 && len(stringToExternalBreaking) <= 1 {
			externalLint, err := getZeroOrSingleValueForMap(stringToExternalLint)
			if err != nil {
				return syserror.Wrap(err)
			}
			externalBreaking, err := getZeroOrSingleValueForMap(stringToExternalBreaking)
			if err != nil {
				return syserror.Wrap(err)
			}
			externalBufYAMLFile.Lint = externalLint
			externalBufYAMLFile.Breaking = externalBreaking
			for i := 0; i < len(externalBufYAMLFile.Modules); i++ {
				externalBufYAMLFile.Modules[i].Lint = externalBufYAMLFileLintV2{}
				externalBufYAMLFile.Modules[i].Breaking = externalBufYAMLFileBreakingV1Beta1V1V2{}
			}
		}
		if len(externalBufYAMLFile.Modules) == 1 && externalBufYAMLFile.Modules[0].Path == "." && len(externalBufYAMLFile.Modules[0].Excludes) == 0 {
			// We know that lint and breaking will already be top-level from the above if statement.
			externalBufYAMLFile.Name = externalBufYAMLFile.Modules[0].Name
			externalBufYAMLFile.Modules = []externalBufYAMLFileModuleV2{}
		}

		data, err := encoding.MarshalYAML(&externalBufYAMLFile)
		if err != nil {
			return err
		}
		_, err = writer.Write(data)
		return err
	default:
		// This is a system error since we've already parsed.
		return syserror.Newf("unknown FileVersion: %v", fileVersion)
	}
}

func getRootToExcludes(roots []string, fullExcludes []string) (map[string][]string, error) {
	if len(roots) == 0 {
		roots = []string{"."}
	}

	rootToExcludes := make(map[string][]string)
	roots, err := normalizeAndCheckPaths(roots, "root")
	if err != nil {
		return nil, err
	}
	for _, root := range roots {
		// we already checked duplicates, but just in case
		if _, ok := rootToExcludes[root]; ok {
			return nil, fmt.Errorf("unexpected duplicate root: %q", root)
		}
		rootToExcludes[root] = []string{}
	}
	if len(fullExcludes) == 0 {
		return rootToExcludes, nil
	}

	// This also verifies that fullExcludes is unique.
	fullExcludes, err = normalizeAndCheckPaths(fullExcludes, "exclude")
	if err != nil {
		return nil, err
	}
	// Verify that no exclude equals a root directly and only directories are specified.
	for _, fullExclude := range fullExcludes {
		if normalpath.Ext(fullExclude) == ".proto" {
			return nil, fmt.Errorf("excludes can only be directories but file %s discovered", fullExclude)
		}
		if _, ok := rootToExcludes[fullExclude]; ok {
			return nil, fmt.Errorf("%s is both a root and exclude, which means the entire root is excluded, which is not valid", fullExclude)
		}
	}

	// Verify that all excludes are within a root.
	rootMap := slicesext.ToStructMap(roots)
	for _, fullExclude := range fullExcludes {
		switch matchingRoots := normalpath.MapAllEqualOrContainingPaths(rootMap, fullExclude, normalpath.Relative); len(matchingRoots) {
		case 0:
			return nil, fmt.Errorf("exclude %s is not contained in any root, which is not valid", fullExclude)
		case 1:
			root := matchingRoots[0]
			exclude, err := normalpath.Rel(root, fullExclude)
			if err != nil {
				return nil, err
			}
			// Just in case.
			exclude, err = normalpath.NormalizeAndValidate(exclude)
			if err != nil {
				return nil, err
			}
			rootToExcludes[root] = append(rootToExcludes[root], exclude)
		default:
			// This should never happen, but just in case.
			return nil, fmt.Errorf("exclude %q was in multiple roots %v", fullExclude, matchingRoots)
		}
	}

	for root, excludes := range rootToExcludes {
		uniqueSortedExcludes := stringutil.SliceToUniqueSortedSliceFilterEmptyStrings(excludes)
		if len(excludes) != len(uniqueSortedExcludes) {
			// This should never happen, but just in case.
			return nil, fmt.Errorf("excludes %v are not unique", excludes)
		}
		rootToExcludes[root] = uniqueSortedExcludes
	}
	return rootToExcludes, nil
}

func getConfiguredDepModuleRefsForExternalDeps(
	externalDeps []string,
) ([]bufmodule.ModuleRef, error) {
	configuredDepModuleRefs := make([]bufmodule.ModuleRef, len(externalDeps))
	for i, externalDep := range externalDeps {
		moduleRef, err := bufmodule.ParseModuleRef(externalDep)
		if err != nil {
			return nil, fmt.Errorf("invalid dep: %w", err)
		}
		configuredDepModuleRefs[i] = moduleRef
	}
	return configuredDepModuleRefs, nil
}

func getLintConfigForExternalLintV1Beta1V1(
	fileVersion FileVersion,
	externalLint externalBufYAMLFileLintV1Beta1V1,
	moduleDirPath string,
	requirePathsToBeContainedWithinModuleDirPath bool,
) (LintConfig, error) {
	var checkConfig CheckConfig
	disabled, err := isLintOrBreakingDisabledBasedOnIgnores("lint.ignore", externalLint.Ignore, moduleDirPath)
	if err != nil {
		return nil, err
	}
	if disabled {
		checkConfig = newDisabledCheckConfig(fileVersion)
	} else {
		ignore, err := getRelPathsForLintOrBreakingExternalPaths("lint.ignore", externalLint.Ignore, moduleDirPath, requirePathsToBeContainedWithinModuleDirPath)
		if err != nil {
			return nil, err
		}
		ignoreOnly := make(map[string][]string)
		for idOrCategory, paths := range externalLint.IgnoreOnly {
			relPaths, err := getRelPathsForLintOrBreakingExternalPaths("lint.ignore_only", paths, moduleDirPath, requirePathsToBeContainedWithinModuleDirPath)
			if err != nil {
				return nil, err
			}
			if len(relPaths) > 0 {
				ignoreOnly[idOrCategory] = relPaths
			}
		}
		checkConfig, err = newEnabledCheckConfig(
			fileVersion,
			externalLint.Use,
			externalLint.Except,
			ignore,
			ignoreOnly,
		)
		if err != nil {
			return nil, err
		}
	}
	return newLintConfig(
		checkConfig,
		externalLint.EnumZeroValueSuffix,
		externalLint.RPCAllowSameRequestResponse,
		externalLint.RPCAllowGoogleProtobufEmptyRequests,
		externalLint.RPCAllowGoogleProtobufEmptyResponses,
		externalLint.ServiceSuffix,
		externalLint.AllowCommentIgnores,
	), nil
}

func getLintConfigForExternalLintV2(
	fileVersion FileVersion,
	externalLint externalBufYAMLFileLintV2,
	moduleDirPath string,
	requirePathsToBeContainedWithinModuleDirPath bool,
) (LintConfig, error) {
	var checkConfig CheckConfig
	disabled, err := isLintOrBreakingDisabledBasedOnIgnores("lint.ignore", externalLint.Ignore, moduleDirPath)
	if err != nil {
		return nil, err
	}
	if disabled {
		checkConfig = newDisabledCheckConfig(fileVersion)
	} else {
		ignore, err := getRelPathsForLintOrBreakingExternalPaths("lint.ignore", externalLint.Ignore, moduleDirPath, requirePathsToBeContainedWithinModuleDirPath)
		if err != nil {
			return nil, err
		}
		ignoreOnly := make(map[string][]string)
		for idOrCategory, paths := range externalLint.IgnoreOnly {
			relPaths, err := getRelPathsForLintOrBreakingExternalPaths("lint.ignore_only", paths, moduleDirPath, requirePathsToBeContainedWithinModuleDirPath)
			if err != nil {
				return nil, err
			}
			if len(relPaths) > 0 {
				ignoreOnly[idOrCategory] = relPaths
			}
		}
		checkConfig, err = newEnabledCheckConfig(
			fileVersion,
			externalLint.Use,
			externalLint.Except,
			ignore,
			ignoreOnly,
		)
		if err != nil {
			return nil, err
		}
	}
	return newLintConfig(
		checkConfig,
		externalLint.EnumZeroValueSuffix,
		externalLint.RPCAllowSameRequestResponse,
		externalLint.RPCAllowGoogleProtobufEmptyRequests,
		externalLint.RPCAllowGoogleProtobufEmptyResponses,
		externalLint.ServiceSuffix,
		!externalLint.DisallowCommentIgnores,
	), nil
}

func getBreakingConfigForExternalBreaking(
	fileVersion FileVersion,
	externalBreaking externalBufYAMLFileBreakingV1Beta1V1V2,
	moduleDirPath string,
	requirePathsToBeContainedWithinModuleDirPath bool,
) (BreakingConfig, error) {
	var checkConfig CheckConfig
	disabled, err := isLintOrBreakingDisabledBasedOnIgnores("breaking.ignore", externalBreaking.Ignore, moduleDirPath)
	if err != nil {
		return nil, err
	}
	if disabled {
		checkConfig = newDisabledCheckConfig(fileVersion)
	} else {
		ignore, err := getRelPathsForLintOrBreakingExternalPaths("breaking.ignore", externalBreaking.Ignore, moduleDirPath, requirePathsToBeContainedWithinModuleDirPath)
		if err != nil {
			return nil, err
		}
		ignoreOnly := make(map[string][]string)
		for idOrCategory, paths := range externalBreaking.IgnoreOnly {
			relPaths, err := getRelPathsForLintOrBreakingExternalPaths("breaking.ignore_only", paths, moduleDirPath, requirePathsToBeContainedWithinModuleDirPath)
			if err != nil {
				return nil, err
			}
			if len(relPaths) > 0 {
				ignoreOnly[idOrCategory] = relPaths
			}
		}
		checkConfig, err = newEnabledCheckConfig(
			fileVersion,
			externalBreaking.Use,
			externalBreaking.Except,
			ignore,
			ignoreOnly,
		)
		if err != nil {
			return nil, err
		}
	}
	return newBreakingConfig(
		checkConfig,
		externalBreaking.IgnoreUnstablePackages,
	), nil
}

// isLintOrBreakingDisabledBasedOnIgnores returns true if lint or breaking should be entirely disabled
// based on an ignore path equaling moduleDirPath.
//
// See comments on CheckConfig.Disabled() for why this is a scenario we want to support.
func isLintOrBreakingDisabledBasedOnIgnores(
	fieldName string,
	ignores []string,
	moduleDirPath string,
) (bool, error) {
	for _, ignore := range ignores {
		ignore, err := normalpath.NormalizeAndValidate(ignore)
		if err != nil {
			// user error
			return false, fmt.Errorf("%s: invalid path: %w", fieldName, err)
		}
		if ignore == moduleDirPath {
			return true, nil
		}
	}
	return false, nil
}

// getRelPathsForLintOrBreakingExternalPaths performs the following operation for either
// getLintConfigForExternalLint or getBreakingConfigForExternalBreaking:
//
//   - Normalized and validates the path. If the path is invalid, returns error.
//   - Checks to make sure the path is not equal to the given module directory path. If so, returns error.
//   - If the path is not contained within the module directory path, the path is not added to the
//     returned slice if requirePathsToBeContainedWithinModuleDirPath is false. This can happen when we
//     are transforming a path from the default workspace-wide lint or breaking config. We want to skip these paths.
//     If requirePathsToBeContainedWithinModuleDirPath is true, return error.
//   - Otherwise, adds the path relative to the given module directory path to the returned slice.
//
// isLintOrBreakingDisabledBasedOnIgnores should be called before this function.
func getRelPathsForLintOrBreakingExternalPaths(
	fieldName string,
	paths []string,
	moduleDirPath string,
	requirePathsToBeContainedWithinModuleDirPath bool,
) ([]string, error) {
	relPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		path, err := normalpath.NormalizeAndValidate(path)
		if err != nil {
			// user error
			return nil, fmt.Errorf("%s: invalid path: %w", fieldName, err)
		}
		if !normalpath.EqualsOrContainsPath(moduleDirPath, path, normalpath.Relative) {
			if !requirePathsToBeContainedWithinModuleDirPath {
				continue
			}
			return nil, fmt.Errorf("%s: path %q is not contained within module directory %q", fieldName, path, moduleDirPath)
		}
		relPath, err := normalpath.Rel(moduleDirPath, path)
		if err != nil {
			return nil, err
		}
		relPaths = append(relPaths, relPath)
	}
	return relPaths, nil
}

func getExternalLintV1Beta1V1ForLintConfig(lintConfig LintConfig, moduleDirPath string) externalBufYAMLFileLintV1Beta1V1 {
	joinDirPath := func(importPath string) string {
		return normalpath.Join(moduleDirPath, importPath)
	}
	externalLint := externalBufYAMLFileLintV1Beta1V1{}
	// All already sorted.
	externalLint.Use = lintConfig.UseIDsAndCategories()
	externalLint.Except = lintConfig.ExceptIDsAndCategories()
	externalLint.Ignore = slicesext.Map(lintConfig.IgnorePaths(), joinDirPath)
	externalLint.IgnoreOnly = make(map[string][]string, len(lintConfig.IgnoreIDOrCategoryToPaths()))
	for idOrCategory, importPaths := range lintConfig.IgnoreIDOrCategoryToPaths() {
		externalLint.IgnoreOnly[idOrCategory] = slicesext.Map(importPaths, joinDirPath)
	}
	externalLint.EnumZeroValueSuffix = lintConfig.EnumZeroValueSuffix()
	externalLint.RPCAllowSameRequestResponse = lintConfig.RPCAllowSameRequestResponse()
	externalLint.RPCAllowGoogleProtobufEmptyRequests = lintConfig.RPCAllowGoogleProtobufEmptyRequests()
	externalLint.RPCAllowGoogleProtobufEmptyResponses = lintConfig.RPCAllowGoogleProtobufEmptyResponses()
	externalLint.ServiceSuffix = lintConfig.ServiceSuffix()
	externalLint.AllowCommentIgnores = lintConfig.AllowCommentIgnores()
	return externalLint
}

func getExternalLintV2ForLintConfig(lintConfig LintConfig, moduleDirPath string) externalBufYAMLFileLintV2 {
	joinDirPath := func(importPath string) string {
		return normalpath.Join(moduleDirPath, importPath)
	}
	externalLint := externalBufYAMLFileLintV2{}
	// All already sorted.
	externalLint.Use = lintConfig.UseIDsAndCategories()
	externalLint.Except = lintConfig.ExceptIDsAndCategories()
	externalLint.Ignore = slicesext.Map(lintConfig.IgnorePaths(), joinDirPath)
	externalLint.IgnoreOnly = make(map[string][]string, len(lintConfig.IgnoreIDOrCategoryToPaths()))
	for idOrCategory, importPaths := range lintConfig.IgnoreIDOrCategoryToPaths() {
		externalLint.IgnoreOnly[idOrCategory] = slicesext.Map(importPaths, joinDirPath)
	}
	externalLint.EnumZeroValueSuffix = lintConfig.EnumZeroValueSuffix()
	externalLint.RPCAllowSameRequestResponse = lintConfig.RPCAllowSameRequestResponse()
	externalLint.RPCAllowGoogleProtobufEmptyRequests = lintConfig.RPCAllowGoogleProtobufEmptyRequests()
	externalLint.RPCAllowGoogleProtobufEmptyResponses = lintConfig.RPCAllowGoogleProtobufEmptyResponses()
	externalLint.ServiceSuffix = lintConfig.ServiceSuffix()
	externalLint.DisallowCommentIgnores = !lintConfig.AllowCommentIgnores()
	return externalLint
}

func getExternalBreakingForBreakingConfig(breakingConfig BreakingConfig, moduleDirPath string) externalBufYAMLFileBreakingV1Beta1V1V2 {
	joinDirPath := func(importPath string) string {
		return normalpath.Join(moduleDirPath, importPath)
	}
	externalBreaking := externalBufYAMLFileBreakingV1Beta1V1V2{}
	// All already sorted.
	externalBreaking.Use = breakingConfig.UseIDsAndCategories()
	externalBreaking.Except = breakingConfig.ExceptIDsAndCategories()
	externalBreaking.Ignore = slicesext.Map(breakingConfig.IgnorePaths(), joinDirPath)
	externalBreaking.IgnoreOnly = make(map[string][]string, len(breakingConfig.IgnoreIDOrCategoryToPaths()))
	for idOrCategory, importPaths := range breakingConfig.IgnoreIDOrCategoryToPaths() {
		externalBreaking.IgnoreOnly[idOrCategory] = slicesext.Map(importPaths, joinDirPath)
	}
	externalBreaking.IgnoreUnstablePackages = breakingConfig.IgnoreUnstablePackages()
	return externalBreaking
}

// externalBufYAMLFileV1Beta1V1 represents the v1 or v1beta1 buf.yaml file, which have
// the same shape EXCEPT build.roots.
//
// Note that the lint and breaking ids/categories DID change between v1beta1 and v1, make
// sure to deal with this when parsing what to set as defaults, or how to interpret categories.
type externalBufYAMLFileV1Beta1V1 struct {
	Version  string                                 `json:"version,omitempty" yaml:"version,omitempty"`
	Name     string                                 `json:"name,omitempty" yaml:"name,omitempty"`
	Deps     []string                               `json:"deps,omitempty" yaml:"deps,omitempty"`
	Build    externalBufYAMLFileBuildV1Beta1V1      `json:"build,omitempty" yaml:"build,omitempty"`
	Lint     externalBufYAMLFileLintV1Beta1V1       `json:"lint,omitempty" yaml:"lint,omitempty"`
	Breaking externalBufYAMLFileBreakingV1Beta1V1V2 `json:"breaking,omitempty" yaml:"breaking,omitempty"`
}

// externalBufYAMLFileV2 represents the v2 buf.yaml file.
//
// Note that the lint and breaking ids/categories DID change between versions, make
// sure to deal with this when parsing what to set as defaults, or how to interpret categories.
type externalBufYAMLFileV2 struct {
	Version  string                                 `json:"version,omitempty" yaml:"version,omitempty"`
	Name     string                                 `json:"name,omitempty" yaml:"name,omitempty"`
	Modules  []externalBufYAMLFileModuleV2          `json:"modules,omitempty" yaml:"modules,omitempty"`
	Deps     []string                               `json:"deps,omitempty" yaml:"deps,omitempty"`
	Lint     externalBufYAMLFileLintV2              `json:"lint,omitempty" yaml:"lint,omitempty"`
	Breaking externalBufYAMLFileBreakingV1Beta1V1V2 `json:"breaking,omitempty" yaml:"breaking,omitempty"`
}

// externalBufYAMLFileModuleV2 represents a single module configuation within a v2 buf.yaml file.
type externalBufYAMLFileModuleV2 struct {
	Path     string                                 `json:"path,omitempty" yaml:"path,omitempty"`
	Name     string                                 `json:"name,omitempty" yaml:"name,omitempty"`
	Excludes []string                               `json:"excludes,omitempty" yaml:"excludes,omitempty"`
	Lint     externalBufYAMLFileLintV2              `json:"lint,omitempty" yaml:"lint,omitempty"`
	Breaking externalBufYAMLFileBreakingV1Beta1V1V2 `json:"breaking,omitempty" yaml:"breaking,omitempty"`
}

// externalBufYAMLFileBuildV1Beta1V1 represents build configuation within a v1 or
// v1beta1 buf.yaml file, which have the same shape except for roots.
type externalBufYAMLFileBuildV1Beta1V1 struct {
	// Roots are only valid in v1beta! Validate that this is not set for v1.
	Roots    []string `json:"roots,omitempty" yaml:"roots,omitempty"`
	Excludes []string `json:"excludes,omitempty" yaml:"excludes,omitempty"`
}

// externalBufYAMLFileLintV1Beta1V1 represents lint configuation within a v1beta1 or v1
// buf.yaml file, which have the same shape.
//
// Note that the lint and breaking ids/categories DID change between versions, make
// sure to deal with this when parsing what to set as defaults, or how to interpret categories.
type externalBufYAMLFileLintV1Beta1V1 struct {
	Use    []string `json:"use,omitempty" yaml:"use,omitempty"`
	Except []string `json:"except,omitempty" yaml:"except,omitempty"`
	// Ignore are the paths to ignore.
	Ignore []string `json:"ignore,omitempty" yaml:"ignore,omitempty"`
	/// IgnoreOnly are the ID/category to paths to ignore.
	IgnoreOnly                           map[string][]string `json:"ignore_only,omitempty" yaml:"ignore_only,omitempty"`
	EnumZeroValueSuffix                  string              `json:"enum_zero_value_suffix,omitempty" yaml:"enum_zero_value_suffix,omitempty"`
	RPCAllowSameRequestResponse          bool                `json:"rpc_allow_same_request_response,omitempty" yaml:"rpc_allow_same_request_response,omitempty"`
	RPCAllowGoogleProtobufEmptyRequests  bool                `json:"rpc_allow_google_protobuf_empty_requests,omitempty" yaml:"rpc_allow_google_protobuf_empty_requests,omitempty"`
	RPCAllowGoogleProtobufEmptyResponses bool                `json:"rpc_allow_google_protobuf_empty_responses,omitempty" yaml:"rpc_allow_google_protobuf_empty_responses,omitempty"`
	ServiceSuffix                        string              `json:"service_suffix,omitempty" yaml:"service_suffix,omitempty"`
	AllowCommentIgnores                  bool                `json:"allow_comment_ignores,omitempty" yaml:"allow_comment_ignores,omitempty"`
}

// Suppressing unused warning. Keeping this function around for now.
var _ = externalBufYAMLFileLintV1Beta1V1.isEmpty

func (el externalBufYAMLFileLintV1Beta1V1) isEmpty() bool {
	return len(el.Use) == 0 &&
		len(el.Except) == 0 &&
		len(el.Ignore) == 0 &&
		len(el.IgnoreOnly) == 0 &&
		el.EnumZeroValueSuffix == "" &&
		!el.RPCAllowSameRequestResponse &&
		!el.RPCAllowGoogleProtobufEmptyRequests &&
		!el.RPCAllowGoogleProtobufEmptyResponses &&
		el.ServiceSuffix == "" &&
		!el.AllowCommentIgnores
}

// externalBufYAMLFileLintV2 represents lint configuation within a  v2 buf.yaml file.
//
// Note that the lint and breaking ids/categories DID change between versions, make
// sure to deal with this when parsing what to set as defaults, or how to interpret categories.
type externalBufYAMLFileLintV2 struct {
	Use    []string `json:"use,omitempty" yaml:"use,omitempty"`
	Except []string `json:"except,omitempty" yaml:"except,omitempty"`
	// Ignore are the paths to ignore.
	Ignore []string `json:"ignore,omitempty" yaml:"ignore,omitempty"`
	/// IgnoreOnly are the ID/category to paths to ignore.
	IgnoreOnly                           map[string][]string `json:"ignore_only,omitempty" yaml:"ignore_only,omitempty"`
	EnumZeroValueSuffix                  string              `json:"enum_zero_value_suffix,omitempty" yaml:"enum_zero_value_suffix,omitempty"`
	RPCAllowSameRequestResponse          bool                `json:"rpc_allow_same_request_response,omitempty" yaml:"rpc_allow_same_request_response,omitempty"`
	RPCAllowGoogleProtobufEmptyRequests  bool                `json:"rpc_allow_google_protobuf_empty_requests,omitempty" yaml:"rpc_allow_google_protobuf_empty_requests,omitempty"`
	RPCAllowGoogleProtobufEmptyResponses bool                `json:"rpc_allow_google_protobuf_empty_responses,omitempty" yaml:"rpc_allow_google_protobuf_empty_responses,omitempty"`
	ServiceSuffix                        string              `json:"service_suffix,omitempty" yaml:"service_suffix,omitempty"`
	DisallowCommentIgnores               bool                `json:"disallow_comment_ignores,omitempty" yaml:"disallow_comment_ignores,omitempty"`
}

func (el externalBufYAMLFileLintV2) isEmpty() bool {
	return len(el.Use) == 0 &&
		len(el.Except) == 0 &&
		len(el.Ignore) == 0 &&
		len(el.IgnoreOnly) == 0 &&
		el.EnumZeroValueSuffix == "" &&
		!el.RPCAllowSameRequestResponse &&
		!el.RPCAllowGoogleProtobufEmptyRequests &&
		!el.RPCAllowGoogleProtobufEmptyResponses &&
		el.ServiceSuffix == "" &&
		!el.DisallowCommentIgnores
}

// externalBufYAMLFileBreakingV1Beta1V1V2 represents breaking configuation within a v1beta1, v1,
// or v2 buf.yaml file, which have the same shape.
//
// Note that the lint and breaking ids/categories DID change between versions, make
// sure to deal with this when parsing what to set as defaults, or how to interpret categories.
type externalBufYAMLFileBreakingV1Beta1V1V2 struct {
	Use    []string `json:"use,omitempty" yaml:"use,omitempty"`
	Except []string `json:"except,omitempty" yaml:"except,omitempty"`
	// Ignore are the paths to ignore.
	Ignore []string `json:"ignore,omitempty" yaml:"ignore,omitempty"`
	/// IgnoreOnly are the ID/category to paths to ignore.
	IgnoreOnly             map[string][]string `json:"ignore_only,omitempty" yaml:"ignore_only,omitempty"`
	IgnoreUnstablePackages bool                `json:"ignore_unstable_packages,omitempty" yaml:"ignore_unstable_packages,omitempty"`
}

func (eb externalBufYAMLFileBreakingV1Beta1V1V2) isEmpty() bool {
	return len(eb.Use) == 0 &&
		len(eb.Except) == 0 &&
		len(eb.Ignore) == 0 &&
		len(eb.IgnoreOnly) == 0 &&
		!eb.IgnoreUnstablePackages
}

func getZeroOrSingleValueForMap[K comparable, V any](m map[K]V) (V, error) {
	var zero V
	if len(m) > 1 {
		return zero, syserror.Newf("map was of length %d empty in getZeroOrSingleValueForMap", len(m))
	}
	for _, v := range m {
		return v, nil
	}
	// len(m) == 0
	return zero, nil
}
