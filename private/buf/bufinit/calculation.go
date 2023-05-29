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

package bufinit

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/stringutil"
)

type calculation struct {
	// A map from file path to the fileInfo that represents this file.
	FilePathToFileInfo map[string]*fileInfo
	// A map from -I directories based on detected imports, to the imports contained within them.
	// Based off of the reverse trie lookup - if a file is detected to be potentially within
	// a given directory and it is imported, this is added as an import dir path, while the
	// import is added to the values for import dir path.
	//
	// Example: if "a/b/c.proto" is a file in FilePathToFileInfo, and some file imports "c.proto",
	// then this map will contain "a/b" -> "c.proto".
	//
	// Note that this might add directories that are do not actually make sense as import directories/
	// For example, in bufbuild/buf, we have imports in tests like "2.proto", but there are a lot
	// of "2.protos" in bufbuild/buf, so if we import "2.proto" anywhere, every directory that
	// contains a "2.proto" will be added as an importDirPath.
	//
	// The idea is that if you were to include all these directories, you would have a superset
	// of the directories you need to compile everything.
	//
	// normalpath.Join(importDirPath, importPath) should always be a key in FilePathToFileInfo.
	ImportDirPathToImportPaths map[string]map[string]struct{}
	//ImportPathToImportDirPaths is the inverse of ImportDirPathToImportPaths, that is for a given
	// "c.proto", it will tell you all directories with a "c.proto.
	ImportPathToImportDirPaths map[string]map[string]struct{}
	// A map from an import that is not detected in the fileInfos, to the file paths that contain
	// this missing import.
	//
	// Example: "a/b/c.proto" imports "gogo.proto", but "gogo.proto" is not detected to be a
	// fileInfo that we have, regardless of how many directories we have stripped (so we don't
	// have ".*/gogo.proto").
	MissingImportPathToFilePaths map[string]map[string]struct{}
	// A map from file path to the inferred include path based on the package, if any.
	//
	// As an example, assume we have "a/b/c/d.proto" with "package c.d". we would infer
	// the include path is "a/b". Say instead the package was "a.b.c.d", we would infer
	// the include path is ".". Say instead the package was "e", we would not infer
	// an include path.
	//
	// This map should only be added to if the file path is a subset of the package, and this
	// is validated. Say we have "package c.e" for file "a/e/e.proto" - this should not
	// be in the map as the package-inferred include path does not match the directory stucture.
	//
	// We may want to lint/warn about this as this is a violation of buf's basic lint rules
	// in the first place, but we are trying to onboard repos in their current state the
	// best that we can.
	FilePathToPackageInferredIncludePath map[string]string
}

func newCalculation() *calculation {
	return &calculation{
		FilePathToFileInfo:                   make(map[string]*fileInfo),
		ImportDirPathToImportPaths:           make(map[string]map[string]struct{}),
		ImportPathToImportDirPaths:           make(map[string]map[string]struct{}),
		MissingImportPathToFilePaths:         make(map[string]map[string]struct{}),
		FilePathToPackageInferredIncludePath: make(map[string]string),
	}
}

// All the import dir paths.
func (c *calculation) AllImportDirPaths() []string {
	importDirPaths := make([]string, 0, len(c.ImportDirPathToImportPaths))
	for importDirPath := range c.ImportDirPathToImportPaths {
		importDirPaths = append(importDirPaths, importDirPath)
	}
	sort.Strings(importDirPaths)
	return importDirPaths
}

// All the package-inferred include paths.
func (c *calculation) AllPackageInferredIncludePaths() []string {
	packageInferredIncludePathMap := make(map[string]struct{})
	for _, packageInferredIncludePath := range c.FilePathToPackageInferredIncludePath {
		packageInferredIncludePathMap[packageInferredIncludePath] = struct{}{}
	}
	return stringutil.MapToSortedSlice(packageInferredIncludePathMap)
}

// A list of imports that are not covered by package-inferred include paths.
//
// This is all the files within import dir paths that are not in package-inferred include paths.
func (c *calculation) ImportPathsNotCoveredByPackageInferredIncludePaths() []string {
	importDirPathMap := stringutil.SliceToMap(c.AllImportDirPaths())
	for _, packageInferredIncludePath := range c.FilePathToPackageInferredIncludePath {
		delete(importDirPathMap, packageInferredIncludePath)
	}
	importPathMap := make(map[string]struct{})
	for importDirPath := range importDirPathMap {
		for importPath := range c.ImportDirPathToImportPaths[importDirPath] {
			importPathMap[importPath] = struct{}{}
		}
	}
	return stringutil.MapToSortedSlice(importPathMap)
}

// FilePathsNotCoveredByPackageInferredIncludePaths is a list of file paths not covered by package-inferred include paths.
func (c *calculation) FilePathsNotCoveredByPackageInferredIncludePaths() []string {
	filePathMap := make(map[string]struct{})
	for filePath := range c.FilePathToFileInfo {
		filePathMap[filePath] = struct{}{}
	}
	for filePath := range c.FilePathToPackageInferredIncludePath {
		delete(filePathMap, filePath)
	}
	return stringutil.MapToSortedSlice(filePathMap)
}

// ImportDirPathToImportPathsNotCoveredByPackageInferredIncludePaths is the same as ImportPathsNotCoveredByPackageInferredIncludePaths
// except it gives you a map importDirPath -> [importPaths], instead of just the importPaths.
func (c *calculation) ImportDirPathToImportPathsNotCoveredByPackageInferredIncludePaths() map[string]map[string]struct{} {
	importDirPathToImportPaths := make(map[string]map[string]struct{})
	for importDirPath, importPathMap := range c.ImportDirPathToImportPaths {
		// We make a copy of the importDirPathToImportPaths map, but we don't need to copy each
		// importPathMap as we do not modify it (and won't, concievably).
		importDirPathToImportPaths[importDirPath] = importPathMap
	}
	for _, packageInferredIncludePath := range c.FilePathToPackageInferredIncludePath {
		delete(importDirPathToImportPaths, packageInferredIncludePath)
	}
	return importDirPathToImportPaths
}

// Returns ImportPathToImportDirPaths only for those importPaths where len(importDirPaths) > 1
func (c *calculation) ImportPathToImportDirPathsWithMoreThanOneImportDirPaths() map[string]map[string]struct{} {
	importPathToImportDirPaths := make(map[string]map[string]struct{})
	for importPath, importDirPathMap := range c.ImportPathToImportDirPaths {
		// We make a copy of the importPathToImportDirPaths map, but we don't need to copy each
		// importDirPathsMap as we do not modify it (and won't, concievably).
		importPathToImportDirPaths[importPath] = importDirPathMap
	}
	for importPath, importDirPathMap := range importPathToImportDirPaths {
		if len(importDirPathMap) <= 1 {
			delete(importPathToImportDirPaths, importPath)
		}
	}
	return importPathToImportDirPaths
}

// handles FilePathToFileInfo
func (c *calculation) addFileInfo(fileInfo *fileInfo) error {
	if _, ok := c.FilePathToFileInfo[fileInfo.Path]; ok {
		// we don't expect this from our production of fileInfos, this is a system error
		return fmt.Errorf("calculation: duplicate filePath: %q", fileInfo.Path)
	}
	c.FilePathToFileInfo[fileInfo.Path] = fileInfo
	return nil
}

// handles ImportDirPathToImportPaths
// handles ImportPathToImportDirPaths
func (c *calculation) addImportDirPathAndImportPath(importDirPath string, importPath string) error {
	importPathMap, ok := c.ImportDirPathToImportPaths[importDirPath]
	if !ok {
		importPathMap = make(map[string]struct{})
		c.ImportDirPathToImportPaths[importDirPath] = importPathMap
	}
	importPathMap[importPath] = struct{}{}
	importDirPathMap, ok := c.ImportPathToImportDirPaths[importPath]
	if !ok {
		importDirPathMap = make(map[string]struct{})
		c.ImportPathToImportDirPaths[importPath] = importDirPathMap
	}
	importDirPathMap[importDirPath] = struct{}{}
	return nil
}

// handles FilePathToPackageInferredIncludePath
func (c *calculation) addPackageInferredIncludePath(filePath string, packageInferredIncludePath string) error {
	if _, ok := c.FilePathToPackageInferredIncludePath[filePath]; ok {
		return fmt.Errorf("calculation: duplicate filePath for package-inferred include path: %q", filePath)
	}
	c.FilePathToPackageInferredIncludePath[filePath] = packageInferredIncludePath
	return nil
}

// handles MissingImportPathToFilePaths
func (c *calculation) addMissingImportPathAndFilePath(missingImportPath string, filePath string) error {
	filePathMap, ok := c.MissingImportPathToFilePaths[missingImportPath]
	if !ok {
		filePathMap = make(map[string]struct{})
		c.MissingImportPathToFilePaths[missingImportPath] = filePathMap
	}
	filePathMap[filePath] = struct{}{}
	return nil
}

// TODO: could validate that every filePath in MissingImportPathToFilePaths is in FilePathToFileInfo.
// TODO: could valudate that every filePath in FilePathToPackageInferredIncludePath is in FilePathToFileInfo.
func (c *calculation) postValidate() error {
	for importDirPath, importPathMap := range c.ImportDirPathToImportPaths {
		for importPath := range importPathMap {
			joinedPath := normalpath.Join(importDirPath, importPath)
			if _, ok := c.FilePathToFileInfo[joinedPath]; !ok {
				return fmt.Errorf("calculation: had importDirPath %q and importPath %q but no corresponding filePath", importDirPath, importPath)
			}
		}
	}
	for filePath, packageInferredIncludePath := range c.FilePathToPackageInferredIncludePath {
		if _, err := normalpath.Rel(packageInferredIncludePath, filePath); err != nil {
			return fmt.Errorf("calculation: package-inferred include path %q was not a base of filePath %q: %w", packageInferredIncludePath, filePath, err)
		}
	}
	return nil
}

func (c *calculation) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		struct {
			FilePathToFileInfo                                                map[string]*fileInfo           `json:"file_path_to_file_info"`
			ImportDirPathToImportPaths                                        map[string]map[string]struct{} `json:"import_dir_path_to_import_paths"`
			ImportPathToImportDirPaths                                        map[string]map[string]struct{} `json:"import_path_to_import_dir_paths"`
			MissingImportPathToFilePaths                                      map[string]map[string]struct{} `json:"missing_import_path_to_file_paths"`
			FilePathToPackageInferredIncludePath                              map[string]string              `json:"file_path_to_package_inferred_include_path"`
			ImportDirPathToImportPathsNotCoveredByPackageInferredIncludePaths map[string]map[string]struct{} `json:"import_dir_path_to_import_paths_not_covered_by_package_inferred_include_paths"`
			ImportPathToImportDirPathsWithMoreThanOneImportDirPaths           map[string]map[string]struct{} `json:"import_path_to_import_dir_paths_with_more_than_one_import_dir_paths"`
			AllImportDirPaths                                                 []string                       `json:"all_import_dir_paths"`
			AllPackageInferredIncludePaths                                    []string                       `json:"all_package_inferred_include_paths"`
			ImportPathsNotCoveredByPackageInferredIncludePaths                []string                       `json:"import_paths_not_covered_by_package_inferred_include_paths"`
			FilePathsNotCoveredByPackageInferredIncludePaths                  []string                       `json:"file_paths_not_covered_by_package_inferred_include_paths"`
		}{
			FilePathToFileInfo:                                                c.FilePathToFileInfo,
			ImportDirPathToImportPaths:                                        c.ImportDirPathToImportPaths,
			ImportPathToImportDirPaths:                                        c.ImportPathToImportDirPaths,
			MissingImportPathToFilePaths:                                      c.MissingImportPathToFilePaths,
			FilePathToPackageInferredIncludePath:                              c.FilePathToPackageInferredIncludePath,
			ImportDirPathToImportPathsNotCoveredByPackageInferredIncludePaths: c.ImportDirPathToImportPathsNotCoveredByPackageInferredIncludePaths(),
			ImportPathToImportDirPathsWithMoreThanOneImportDirPaths:           c.ImportPathToImportDirPathsWithMoreThanOneImportDirPaths(),
			AllImportDirPaths:                                                 c.AllImportDirPaths(),
			AllPackageInferredIncludePaths:                                    c.AllPackageInferredIncludePaths(),
			ImportPathsNotCoveredByPackageInferredIncludePaths:                c.ImportPathsNotCoveredByPackageInferredIncludePaths(),
			FilePathsNotCoveredByPackageInferredIncludePaths:                  c.FilePathsNotCoveredByPackageInferredIncludePaths(),
		},
	)
}
