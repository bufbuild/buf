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
	"bytes"
	"context"
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

var (
	bufYAML = newFileName("buf.yaml", FileVersionV1Beta1, FileVersionV1, FileVersionV2)
	// Originally we thought we were going to move to buf.mod, and had this around for
	// a while, but then reverted back to buf.yaml. We still need to support buf.mod as
	// we released with it, however.
	bufMod           = newFileName("buf.mod", FileVersionV1Beta1, FileVersionV1)
	bufYAMLFileNames = []*fileName{bufYAML, bufMod}
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
	return newBufYAMLFile(fileVersion, moduleConfigs, configuredDepModuleRefs)
}

// GetBufYAMLFileForPrefix gets the buf.yaml file at the given bucket prefix.
//
// The buf.yaml file will be attempted to be read at prefix/buf.yaml.
func GetBufYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (BufYAMLFile, error) {
	return getFileForPrefix(ctx, bucket, prefix, bufYAMLFileNames, readBufYAMLFile)
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
	var err error
	switch filepath.Ext(override) {
	case ".json", ".yaml", ".yml":
		data, err = os.ReadFile(override)
		if err != nil {
			return nil, fmt.Errorf("could not read file: %v", err)
		}
	default:
		data = []byte(override)
	}
	return ReadBufYAMLFile(bytes.NewReader(data))
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
	return getFileVersionForPrefix(ctx, bucket, prefix, bufYAMLFileNames, true, FileVersionV2)
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
	return putFileForPrefix(ctx, bucket, prefix, bufYAMLFile, bufYAML, writeBufYAMLFile)
}

// ReadBufYAMLFile reads the BufYAMLFile from the io.Reader.
func ReadBufYAMLFile(reader io.Reader) (BufYAMLFile, error) {
	return readFile(reader, "config file", readBufYAMLFile)
}

// WriteBufYAMLFile writes the BufYAMLFile to the io.Writer.
func WriteBufYAMLFile(writer io.Writer, bufYAMLFile BufYAMLFile) error {
	return writeFile(writer, "config file", bufYAMLFile, writeBufYAMLFile)
}

// *** PRIVATE ***

type bufYAMLFile struct {
	fileVersion             FileVersion
	moduleConfigs           []ModuleConfig
	configuredDepModuleRefs []bufmodule.ModuleRef
}

func newBufYAMLFile(
	fileVersion FileVersion,
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
	duplicateModuleConfigFullNameStrings := slicesext.Duplicates(
		slicesext.Map(
			moduleConfigs,
			func(moduleConfig ModuleConfig) string {
				if moduleFullName := moduleConfig.ModuleFullName(); moduleFullName != nil {
					return moduleFullName.String()
				}
				return ""
			},
		),
	)
	if len(duplicateModuleConfigFullNameStrings) > 0 {
		return nil, fmt.Errorf("module name %q seen more than once", strings.Join(duplicateModuleConfigFullNameStrings, ", "))
	}
	duplicateDepModuleFullNames := slicesext.Duplicates(
		slicesext.Map(
			configuredDepModuleRefs,
			func(moduleRef bufmodule.ModuleRef) string {
				return moduleRef.ModuleFullName().String()
			},
		),
	)
	if len(duplicateDepModuleFullNames) > 0 {
		return nil, fmt.Errorf(
			"dep with module name %q seen more than once",
			strings.Join(
				duplicateDepModuleFullNames,
				", ",
			),
		)
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
				configuredDepModuleRefs[i].ModuleFullName().String()
		},
	)
	return &bufYAMLFile{
		fileVersion:             fileVersion,
		moduleConfigs:           moduleConfigs,
		configuredDepModuleRefs: configuredDepModuleRefs,
	}, nil
}

func (c *bufYAMLFile) FileVersion() FileVersion {
	return c.fileVersion
}

func (c *bufYAMLFile) ModuleConfigs() []ModuleConfig {
	return slicesext.Copy(c.moduleConfigs)
}

func (c *bufYAMLFile) ConfiguredDepModuleRefs() []bufmodule.ModuleRef {
	return slicesext.Copy(c.configuredDepModuleRefs)
}

func (*bufYAMLFile) isBufYAMLFile() {}
func (*bufYAMLFile) isFile()        {}

// TODO: port tests from bufmoduleconfig, buflintconfig, bufbreakingconfig
// TODO: We need to validate all paths on ignore, excludes, etc
func readBufYAMLFile(reader io.Reader, allowJSON bool) (BufYAMLFile, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	// We've always required a file version for buf.yaml files.
	fileVersion, err := getFileVersionForData(data, allowJSON, true, FileVersionV2)
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
		// TODO: we do no validation of paths now
		lintConfig, err := getLintConfigForExternalLint(
			fileVersion,
			externalBufYAMLFile.Lint,
			normalpath.NormalizeAndValidate,
		)
		if err != nil {
			return nil, err
		}
		// TODO: we do no validation of paths now
		breakingConfig, err := getBreakingConfigForExternalBreaking(
			fileVersion,
			externalBufYAMLFile.Breaking,
			normalpath.NormalizeAndValidate,
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
			// Always make sure we have at least one ModuleConfig, with the defaults.
			externalModules = []externalBufYAMLFileModuleV2{
				{},
			}
		}
		var moduleConfigs []ModuleConfig
		for _, externalModule := range externalBufYAMLFile.Modules {
			dirPath := externalModule.Directory
			if dirPath == "" {
				dirPath = "."
			}
			dirPath, err := normalpath.NormalizeAndValidate(dirPath)
			if err != nil {
				return nil, fmt.Errorf("invaid module directory: %w", err)
			}
			var moduleFullName bufmodule.ModuleFullName
			if externalModule.Name != "" {
				moduleFullName, err = bufmodule.ParseModuleFullName(externalModule.Name)
				if err != nil {
					return nil, err
				}
			}
			// The only root for v2 buf.yamls must be ".", so we have to make the excludes relative first.
			relExcludes, err := slicesext.MapError(
				externalModule.Excludes,
				func(exclude string) (string, error) {
					exclude, err := normalpath.NormalizeAndValidate(exclude)
					if err != nil {
						// user error
						return "", fmt.Errorf("invalid exclude: %w", err)
					}
					if exclude == dirPath {
						return "", fmt.Errorf("exclude %q is equal to module directory %q", exclude, dirPath)
					}
					if !normalpath.EqualsOrContainsPath(dirPath, exclude, normalpath.Relative) {
						return "", fmt.Errorf("exclude %q does not reside within module directory %q", exclude, dirPath)
					}
					relExclude, err := normalpath.Rel(dirPath, exclude)
					if err != nil {
						return "", err
					}
					return relExclude, nil
				},
			)
			if err != nil {
				return nil, err
			}
			rootToExcludes, err := getRootToExcludes([]string{"."}, relExcludes)
			if err != nil {
				return nil, err
			}
			validateAndTransformPath := func(pathInWorkspace string) (string, error) {
				pathInWorkspace, err := normalpath.NormalizeAndValidate(pathInWorkspace)
				if err != nil {
					// user error
					return "", fmt.Errorf("invalid path: %w", err)
				}
				if !normalpath.EqualsOrContainsPath(dirPath, pathInWorkspace, normalpath.Relative) {
					return "", fmt.Errorf("%q does not reside within module directory %q", pathInWorkspace, dirPath)
				}
				return filepath.Rel(dirPath, pathInWorkspace)
			}
			// TODO: we do no validation of paths now
			lintConfig, err := getLintConfigForExternalLint(
				fileVersion,
				externalModule.Lint,
				validateAndTransformPath,
			)
			if err != nil {
				return nil, err
			}
			// TODO: we do no validation of paths now
			breakingConfig, err := getBreakingConfigForExternalBreaking(
				fileVersion,
				externalModule.Breaking,
				validateAndTransformPath,
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
			moduleConfigs,
			configuredDepModuleRefs,
		)
	default:
		// This is a system error since we've already parsed.
		return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
	}
}

// TODO: port tests from bufconfig
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
		if moduleConfig.DirPath() != "" {
			return syserror.Newf("expected ModuleConfig DirPath to be empty but was %q", moduleConfig.DirPath())
		}
		externalBufYAMLFile := externalBufYAMLFileV1Beta1V1{
			Version: fileVersion.String(),
		}
		// Alredy sorted.
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
		// All already sorted.
		lintConfig := moduleConfig.LintConfig()
		externalBufYAMLFile.Lint.Use = lintConfig.UseIDsAndCategories()
		externalBufYAMLFile.Lint.Except = lintConfig.ExceptIDsAndCategories()
		externalBufYAMLFile.Lint.Ignore = lintConfig.IgnorePaths()
		externalBufYAMLFile.Lint.IgnoreOnly = lintConfig.IgnoreIDOrCategoryToPaths()
		externalBufYAMLFile.Lint.EnumZeroValueSuffix = lintConfig.EnumZeroValueSuffix()
		externalBufYAMLFile.Lint.RPCAllowSameRequestResponse = lintConfig.RPCAllowSameRequestResponse()
		externalBufYAMLFile.Lint.RPCAllowGoogleProtobufEmptyRequests = lintConfig.RPCAllowGoogleProtobufEmptyRequests()
		externalBufYAMLFile.Lint.RPCAllowGoogleProtobufEmptyResponses = lintConfig.RPCAllowGoogleProtobufEmptyResponses()
		externalBufYAMLFile.Lint.ServiceSuffix = lintConfig.ServiceSuffix()
		externalBufYAMLFile.Lint.AllowCommentIgnores = lintConfig.AllowCommentIgnores()
		breakingConfig := moduleConfig.BreakingConfig()
		externalBufYAMLFile.Breaking.Use = breakingConfig.UseIDsAndCategories()
		externalBufYAMLFile.Breaking.Except = breakingConfig.ExceptIDsAndCategories()
		externalBufYAMLFile.Breaking.Ignore = breakingConfig.IgnorePaths()
		externalBufYAMLFile.Breaking.IgnoreOnly = breakingConfig.IgnoreIDOrCategoryToPaths()
		externalBufYAMLFile.Breaking.IgnoreUnstablePackages = breakingConfig.IgnoreUnstablePackages()
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
		// Alredy sorted.
		externalBufYAMLFile.Deps = slicesext.Map(
			bufYAMLFile.ConfiguredDepModuleRefs(),
			func(moduleRef bufmodule.ModuleRef) string {
				return moduleRef.String()
			},
		)
		for _, moduleConfig := range bufYAMLFile.ModuleConfigs() {
			externalModule := externalBufYAMLFileModuleV2{
				Directory: moduleConfig.DirPath(),
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
			externalModule.Excludes = excludes
			// All already sorted.
			lintConfig := moduleConfig.LintConfig()
			externalModule.Lint.Use = lintConfig.UseIDsAndCategories()
			externalModule.Lint.Except = lintConfig.ExceptIDsAndCategories()
			joinDirPath := func(importPath string) string {
				return filepath.Join(moduleConfig.DirPath(), importPath)
			}
			externalModule.Lint.Ignore = slicesext.Map(lintConfig.IgnorePaths(), joinDirPath)
			externalModule.Lint.IgnoreOnly = make(map[string][]string, len(lintConfig.IgnoreIDOrCategoryToPaths()))
			for idOrCategory, importPaths := range lintConfig.IgnoreIDOrCategoryToPaths() {
				externalModule.Lint.IgnoreOnly[idOrCategory] = slicesext.Map(importPaths, joinDirPath)
			}
			externalModule.Lint.EnumZeroValueSuffix = lintConfig.EnumZeroValueSuffix()
			externalModule.Lint.RPCAllowSameRequestResponse = lintConfig.RPCAllowSameRequestResponse()
			externalModule.Lint.RPCAllowGoogleProtobufEmptyRequests = lintConfig.RPCAllowGoogleProtobufEmptyRequests()
			externalModule.Lint.RPCAllowGoogleProtobufEmptyResponses = lintConfig.RPCAllowGoogleProtobufEmptyResponses()
			externalModule.Lint.ServiceSuffix = lintConfig.ServiceSuffix()
			externalModule.Lint.AllowCommentIgnores = lintConfig.AllowCommentIgnores()
			breakingConfig := moduleConfig.BreakingConfig()
			externalModule.Breaking.Use = breakingConfig.UseIDsAndCategories()
			externalModule.Breaking.Except = breakingConfig.ExceptIDsAndCategories()
			externalModule.Breaking.Ignore = slicesext.Map(breakingConfig.IgnorePaths(), joinDirPath)
			externalModule.Breaking.IgnoreOnly = make(map[string][]string, len(breakingConfig.IgnoreIDOrCategoryToPaths()))
			for idOrCategory, importPaths := range breakingConfig.IgnoreIDOrCategoryToPaths() {
				externalModule.Breaking.IgnoreOnly[idOrCategory] = slicesext.Map(importPaths, joinDirPath)
			}
			externalModule.Breaking.IgnoreUnstablePackages = breakingConfig.IgnoreUnstablePackages()
			externalBufYAMLFile.Modules = append(externalBufYAMLFile.Modules, externalModule)
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

func getLintConfigForExternalLint(
	fileVersion FileVersion,
	externalLint externalBufYAMLFileLintV1Beta1V1V2,
	pathTransformFunc func(string) (string, error),
) (LintConfig, error) {
	ignore, err := slicesext.MapError(
		externalLint.Ignore,
		pathTransformFunc,
	)
	if err != nil {
		return nil, err
	}
	ignoreOnly := make(map[string][]string)
	for idOrCategory, specifiedPaths := range externalLint.IgnoreOnly {
		transformedPaths, err := slicesext.MapError(
			specifiedPaths,
			pathTransformFunc,
		)
		if err != nil {
			return nil, err
		}
		ignoreOnly[idOrCategory] = transformedPaths
	}
	return newLintConfig(
		newCheckConfig(
			fileVersion,
			externalLint.Use,
			externalLint.Except,
			ignore,
			ignoreOnly,
		),
		externalLint.EnumZeroValueSuffix,
		externalLint.RPCAllowSameRequestResponse,
		externalLint.RPCAllowGoogleProtobufEmptyRequests,
		externalLint.RPCAllowGoogleProtobufEmptyResponses,
		externalLint.ServiceSuffix,
		externalLint.AllowCommentIgnores,
	), nil
}

func getBreakingConfigForExternalBreaking(
	fileVersion FileVersion,
	externalBreaking externalBufYAMLFileBreakingV1Beta1V1V2,
	pathTransformFunc func(string) (string, error),
) (BreakingConfig, error) {
	ignore, err := slicesext.MapError(
		externalBreaking.Ignore,
		pathTransformFunc,
	)
	if err != nil {
		return nil, err
	}
	ignoreOnly := make(map[string][]string)
	for idOrCategory, specifiedPaths := range externalBreaking.IgnoreOnly {
		transformedPaths, err := slicesext.MapError(
			specifiedPaths,
			pathTransformFunc,
		)
		if err != nil {
			return nil, err
		}
		ignoreOnly[idOrCategory] = transformedPaths
	}
	return newBreakingConfig(
		newCheckConfig(
			fileVersion,
			externalBreaking.Use,
			externalBreaking.Except,
			ignore,
			ignoreOnly,
		),
		externalBreaking.IgnoreUnstablePackages,
	), nil
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
	Lint     externalBufYAMLFileLintV1Beta1V1V2     `json:"lint,omitempty" yaml:"lint,omitempty"`
	Breaking externalBufYAMLFileBreakingV1Beta1V1V2 `json:"breaking,omitempty" yaml:"breaking,omitempty"`
}

// externalBufYAMLFileV2 represents the v2 buf.yaml file.
//
// Note that the lint and breaking ids/categories DID change between versions, make
// sure to deal with this when parsing what to set as defaults, or how to interpret categories.
type externalBufYAMLFileV2 struct {
	Version string                        `json:"version,omitempty" yaml:"version,omitempty"`
	Modules []externalBufYAMLFileModuleV2 `json:"modules,omitempty" yaml:"modules,omitempty"`
	Deps    []string                      `json:"deps,omitempty" yaml:"deps,omitempty"`
}

// externalBufYAMLFileModuleV2 represents a single module configuation within a v2 buf.yaml file.
type externalBufYAMLFileModuleV2 struct {
	Directory string                                 `json:"directory,omitempty" yaml:"directory,omitempty"`
	Name      string                                 `json:"name,omitempty" yaml:"name,omitempty"`
	Excludes  []string                               `json:"excludes,omitempty" yaml:"excludes,omitempty"`
	Lint      externalBufYAMLFileLintV1Beta1V1V2     `json:"lint,omitempty" yaml:"lint,omitempty"`
	Breaking  externalBufYAMLFileBreakingV1Beta1V1V2 `json:"breaking,omitempty" yaml:"breaking,omitempty"`
}

// externalBufYAMLFileBuildV1Beta1V1 represents build configuation within a v1 or
// v1beta1 buf.yaml file, which have the same shape except for roots.
type externalBufYAMLFileBuildV1Beta1V1 struct {
	// Roots are only valid in v1beta! Validate that this is not set for v1.
	Roots    []string `json:"roots,omitempty" yaml:"roots,omitempty"`
	Excludes []string `json:"excludes,omitempty" yaml:"excludes,omitempty"`
}

// externalBufYAMLFileLintV1Beta1V1V2 represents lint configuation within a v1beta1, v1,
// or v2 buf.yaml file, which have the same shape.
//
// Note that the lint and breaking ids/categories DID change between versions, make
// sure to deal with this when parsing what to set as defaults, or how to interpret categories.
type externalBufYAMLFileLintV1Beta1V1V2 struct {
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
