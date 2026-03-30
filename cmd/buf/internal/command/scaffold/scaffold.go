// Copyright 2020-2026 Buf Technologies, Inc.
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

package scaffold

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/protocompile/experimental/parser"
	"github.com/bufbuild/protocompile/experimental/report"
	"github.com/bufbuild/protocompile/experimental/source"
)

// NewCommand returns a new scaffold Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	return &appcmd.Command{
		Use:   name,
		Short: "Scaffold buf configuration by analyzing Protobuf files in a git repository",
		Long: `This command generates a buf.yaml with the correct module roots by analyzing .proto files.

The first argument is the root of the git repository to scaffold.
Defaults to "." if no argument is specified.

If a buf.yaml already exists, this command will not overwrite it, and will produce an error.`,
		Args: appcmd.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container)
			},
		),
	}
}

func run(
	ctx context.Context,
	container appext.Container,
) error {
	dirPath := "."
	if container.NumArgs() > 0 {
		dirPath = container.Arg(0)
	}
	if _, err := os.Stat(filepath.Join(dirPath, ".git")); err != nil {
		return fmt.Errorf("%s is not the root of a git repository", dirPath)
	}
	exists, err := bufcli.BufYAMLFileExistsForDirPath(ctx, dirPath)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("buf.yaml already exists in %s, will not overwrite", dirPath)
	}
	fsys := os.DirFS(dirPath)
	fileInfos, err := walkAndParseProtoFiles(fsys)
	if err != nil {
		return err
	}
	if len(fileInfos) == 0 {
		return errors.New("no .proto files found")
	}
	moduleRoots := inferModuleRoots(fileInfos)
	if len(moduleRoots) == 0 {
		return errors.New("could not determine module roots from .proto files")
	}
	moduleConfigs, err := buildModuleConfigs(moduleRoots)
	if err != nil {
		return err
	}
	bufYAMLFile, err := bufconfig.NewBufYAMLFile(
		bufconfig.FileVersionV2,
		moduleConfigs,
		nil,
		nil,
		nil,
		bufconfig.BufYAMLFileWithIncludeDocsLink(),
	)
	if err != nil {
		return err
	}
	return bufcli.PutBufYAMLFileForDirPath(ctx, dirPath, bufYAMLFile)
}

func buildModuleConfigs(moduleRoots []string) ([]bufconfig.ModuleConfig, error) {
	moduleConfigs := make([]bufconfig.ModuleConfig, 0, len(moduleRoots))
	for _, root := range moduleRoots {
		moduleConfig, err := bufconfig.NewModuleConfig(
			root,
			nil,
			map[string][]string{".": {}},
			map[string][]string{".": {}},
			bufconfig.DefaultLintConfigV2,
			bufconfig.DefaultBreakingConfigV2,
		)
		if err != nil {
			return nil, err
		}
		moduleConfigs = append(moduleConfigs, moduleConfig)
	}
	return moduleConfigs, nil
}

type protoFileInfo struct {
	filePath    string
	packageName string
}

// walkAndParseProtoFiles walks the filesystem for .proto files and parses each one.
// Files that fail to parse or have no package declaration are silently skipped.
func walkAndParseProtoFiles(fsys fs.FS) ([]protoFileInfo, error) {
	var fileInfos []protoFileInfo
	err := fs.WalkDir(fsys, ".", func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || path.Ext(filePath) != ".proto" {
			return nil
		}
		data, err := fs.ReadFile(fsys, filePath)
		if err != nil {
			return err
		}
		sourceFile := source.NewFile(filePath, string(data))
		astFile, ok := parser.Parse(filePath, sourceFile, new(report.Report))
		if !ok || astFile == nil {
			return nil
		}
		pkg := astFile.Package()
		if pkg.IsZero() {
			return nil
		}
		packagePath := pkg.Path()
		if packagePath.IsZero() {
			return nil
		}
		fileInfos = append(fileInfos, protoFileInfo{
			filePath:    astFile.Path(),
			packageName: packagePath.Canonicalized(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return fileInfos, nil
}

// inferModuleRoots determines module root directories by deriving each file's
// expected import path from its package declaration, then comparing against its
// file path. The difference is the module root.
//
// Returns sorted, unique module root paths. Returns nil if no roots can be inferred.
func inferModuleRoots(files []protoFileInfo) []string {
	rootMap := make(map[string]struct{})
	for _, file := range files {
		packageDir := strings.ReplaceAll(file.packageName, ".", "/")
		fileName := path.Base(file.filePath)
		expectedImportPath := packageDir + "/" + fileName
		prefix := moduleRootPrefix(file.filePath, expectedImportPath)
		if prefix != "" {
			rootMap[prefix] = struct{}{}
		}
	}
	if len(rootMap) == 0 {
		return nil
	}
	// "." means the entire repo is a single module, no other root can coexist.
	if _, ok := rootMap["."]; ok {
		return []string{"."}
	}
	roots := xslices.MapKeysToSortedSlice(rootMap)
	// Remove roots that are children of other roots. A module root
	// like "proto" already includes all files under "proto/internal",
	// so "proto/internal" as a separate root would be invalid.
	// After sorting, a parent always appears before its children.
	var filtered []string
	for _, root := range roots {
		if !slices.ContainsFunc(filtered, func(parent string) bool {
			return strings.HasPrefix(root, parent+"/")
		}) {
			filtered = append(filtered, root)
		}
	}
	return filtered
}

// moduleRootPrefix returns the prefix of filePath before importPath,
// or "" if filePath does not end with importPath.
func moduleRootPrefix(filePath, importPath string) string {
	if filePath == importPath {
		return "."
	}
	prefix, found := strings.CutSuffix(filePath, "/"+importPath)
	if found {
		return prefix
	}
	return ""
}
