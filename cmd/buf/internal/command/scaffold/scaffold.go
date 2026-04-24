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
	"io"
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
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/incremental"
	"github.com/bufbuild/protocompile/experimental/incremental/queries"
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
	bucket, err := storageos.NewProvider(storageos.ProviderWithSymlinks()).NewReadWriteBucket(dirPath)
	if err != nil {
		return err
	}
	if _, err := bufconfig.GetBufYAMLFileVersionForPrefix(ctx, bucket, "."); err == nil {
		return fmt.Errorf("buf.yaml already exists in %s, will not overwrite", dirPath)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	astFiles, err := walkAndParseProtoFiles(ctx, bucket)
	if err != nil {
		return err
	}
	if len(astFiles) == 0 {
		return errors.New("no .proto files found")
	}
	moduleRoots := inferModuleRoots(astFiles)
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
	controller, err := bufcli.NewController(container)
	if err != nil {
		return err
	}
	// Write buf.yaml to disk, then validate it compiles. If validation
	// fails, remove the file so we don't leave a broken config behind.
	if err := bufconfig.PutBufYAMLFileForPrefix(ctx, bucket, ".", bufYAMLFile); err != nil {
		return err
	}
	if _, err := controller.GetImage(ctx, dirPath); err != nil {
		return errors.Join(
			fmt.Errorf("generated buf.yaml does not compile: %w", err),
			bucket.Delete(ctx, bufconfig.DefaultBufYAMLFileName),
		)
	}
	return nil
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

// walkAndParseProtoFiles walks the bucket for .proto files and parses them in parallel.
// Files that fail to parse are silently skipped.
func walkAndParseProtoFiles(ctx context.Context, bucket storage.ReadBucket) ([]*ast.File, error) {
	sourceMap := source.NewMap(nil)
	var paths []string
	if err := storage.WalkReadObjects(
		ctx,
		storage.FilterReadBucket(bucket, storage.MatchPathExt(".proto")),
		"",
		func(readObject storage.ReadObject) error {
			data, err := io.ReadAll(readObject)
			if err != nil {
				return err
			}
			filePath := readObject.Path()
			sourceMap.Add(filePath, string(data))
			paths = append(paths, filePath)
			return nil
		},
	); err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}
	astQueries := make([]incremental.Query[*ast.File], 0, len(paths))
	for _, filePath := range paths {
		astQueries = append(astQueries, queries.AST{
			Opener: sourceMap,
			Path:   filePath,
		})
	}
	results, _, err := incremental.Run(ctx, incremental.New(), astQueries...)
	if err != nil {
		return nil, err
	}
	var astFiles []*ast.File
	for _, result := range results {
		if result.Fatal != nil || result.Value == nil {
			continue
		}
		astFiles = append(astFiles, result.Value)
	}
	return astFiles, nil
}

// inferModuleRoots determines module root directories from parsed AST files.
//
// It first collects all import paths across files, then for each file, tries to
// match its disk path against the import set using tail-to-front suffix matching.
// Files that are never imported fall back to package-name-based inference.
//
// Returns sorted, unique module root paths. Returns nil if no roots can be inferred.
func inferModuleRoots(files []*ast.File) []string {
	// Collect all import paths across all files.
	importPaths := make(map[string]struct{})
	for _, file := range files {
		for imp := range file.Imports() {
			importPath := imp.ImportPath()
			if importPath.IsZero() {
				continue
			}
			literal := importPath.AsLiteral()
			if literal.Token.IsZero() {
				continue
			}
			importPaths[literal.AsString().Text()] = struct{}{}
		}
	}
	// For each file, find its module root by matching against importPaths,
	// falling back to the package declaration.
	roots := make(map[string]struct{})
	for _, file := range files {
		if root, ok := inferRootFromImports(file.Path(), importPaths); ok {
			roots[root] = struct{}{}
			continue
		}
		if root, ok := inferRootFromPackage(file); ok {
			roots[root] = struct{}{}
		}
	}
	if len(roots) == 0 {
		return nil
	}
	// "." means the entire repo is a single module, no other root can coexist.
	if _, ok := roots["."]; ok {
		return []string{"."}
	}
	sortedRoots := xslices.MapKeysToSortedSlice(roots)
	// Remove roots that are children of other roots. A module root
	// like "proto" already includes all files under "proto/internal",
	// so "proto/internal" as a separate root would be invalid.
	// After sorting, a parent always appears before its children.
	var filtered []string
	for _, root := range sortedRoots {
		if !slices.ContainsFunc(filtered, func(parent string) bool {
			return strings.HasPrefix(root, parent+"/")
		}) {
			filtered = append(filtered, root)
		}
	}
	return filtered
}

// inferRootFromImports matches a file's path against known import paths by
// checking progressively shorter suffixes. For "proto/foo/v1/bar.proto", it
// checks "proto/foo/v1/bar.proto", then "foo/v1/bar.proto", then "v1/bar.proto",
// then "bar.proto". The first match is the import path, and the prefix is the root.
func inferRootFromImports(filePath string, importPaths map[string]struct{}) (string, bool) {
	if _, ok := importPaths[filePath]; ok {
		return ".", true
	}
	suffix := filePath
	for {
		_, after, found := strings.Cut(suffix, "/")
		if !found {
			return "", false
		}
		suffix = after
		if _, ok := importPaths[suffix]; ok {
			return strings.TrimSuffix(filePath, "/"+suffix), true
		}
	}
}

// inferRootFromPackage infers a module root from the file's package declaration.
// For a file at "proto/foo/v1/bar.proto" with "package foo.v1", the expected
// import path is "foo/v1/bar.proto", and the root is "proto".
func inferRootFromPackage(file *ast.File) (string, bool) {
	pkg := file.Package()
	if pkg.IsZero() {
		return "", false
	}
	packagePath := pkg.Path()
	if packagePath.IsZero() {
		return "", false
	}
	packageDir := strings.ReplaceAll(packagePath.Canonicalized(), ".", "/")
	expectedImportPath := packageDir + "/" + path.Base(file.Path())
	if file.Path() == expectedImportPath {
		return ".", true
	}
	prefix, found := strings.CutSuffix(file.Path(), "/"+expectedImportPath)
	if found {
		return prefix, true
	}
	return "", false
}
