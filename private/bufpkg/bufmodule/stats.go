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

package bufmodule

import (
	"context"

	"github.com/bufbuild/protocompile/ast"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"go.uber.org/multierr"
)

// ComputeStats gathers some simple stats about the size of the given module.
func ComputeStats(ctx context.Context, module Module) (*Stats, error) {
	infos, err := module.SourceFileInfos(ctx)
	if err != nil {
		return nil, err
	}
	handler := reporter.NewHandler(reporter.NewReporter(
		func(reporter.ErrorWithPos) error {
			// never aborts
			return nil
		},
		nil,
	))
	var errs error
	stats := &Stats{packages: map[ast.Identifier]struct{}{}}
	for _, info := range infos {
		file, err := module.GetModuleFile(ctx, info.Path())
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}
		err = func() error {
			defer func() {
				errs = multierr.Append(errs, file.Close())
			}()
			// This can return an error and non-nil AST.
			astRoot, err := parser.Parse(info.Path(), file, handler)
			if astRoot == nil {
				// No AST implies an I/O error trying to read the
				// file contents. No stats to collect.
				return err
			}
			if err != nil {
				// There was a syntax error, but we still have a partial
				// AST we can examine.
				stats.NumFilesWithSyntaxErrors++
			}
			examineFile(astRoot, stats)
			return nil
		}()
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	stats.NumPackages = len(stats.packages)
	stats.packages = nil

	if stats.NumFiles == 0 && errs != nil {
		return nil, errs
	}
	return stats, errs
}

// Stats represents some statistics/metrics about a module.
type Stats struct {
	NumFiles                 int
	NumPackages              int
	NumFilesWithSyntaxErrors int
	NumMessages              int
	NumFields                int
	NumEnums                 int
	NumEnumValues            int
	NumExtensions            int
	NumServices              int
	NumMethods               int

	packages map[ast.Identifier]struct{}
}

func examineFile(f *ast.FileNode, stats *Stats) {
	stats.NumFiles++
	for _, decl := range f.Decls {
		switch decl := decl.(type) {
		case *ast.PackageNode:
			stats.packages[decl.Name.AsIdentifier()] = struct{}{}
		case *ast.MessageNode:
			examineMessage(&decl.MessageBody, stats)
		case *ast.EnumNode:
			examineEnum(decl, stats)
		case *ast.ExtendNode:
			examineExtend(decl, stats)
		case *ast.ServiceNode:
			stats.NumServices++
			for _, decl := range decl.Decls {
				_, ok := decl.(*ast.RPCNode)
				if ok {
					stats.NumMethods++
				}
			}
		}
	}
}

func examineMessage(msg *ast.MessageBody, stats *Stats) {
	stats.NumMessages++
	for _, decl := range msg.Decls {
		switch decl := decl.(type) {
		case *ast.FieldNode, *ast.MapFieldNode:
			stats.NumFields++
		case *ast.GroupNode:
			stats.NumFields++
			examineMessage(&decl.MessageBody, stats)
		case *ast.OneOfNode:
			for _, ooDecl := range decl.Decls {
				switch ooDecl := ooDecl.(type) {
				case *ast.FieldNode:
					stats.NumFields++
				case *ast.GroupNode:
					stats.NumFields++
					examineMessage(&ooDecl.MessageBody, stats)
				}
			}
		case *ast.MessageNode:
			examineMessage(&decl.MessageBody, stats)
		case *ast.EnumNode:
			examineEnum(decl, stats)
		case *ast.ExtendNode:
			examineExtend(decl, stats)
		}
	}
}

func examineEnum(enum *ast.EnumNode, stats *Stats) {
	stats.NumEnums++
	for _, decl := range enum.Decls {
		_, ok := decl.(*ast.EnumValueNode)
		if ok {
			stats.NumEnumValues++
		}
	}
}

func examineExtend(ext *ast.ExtendNode, stats *Stats) {
	for _, decl := range ext.Decls {
		switch decl := decl.(type) {
		case *ast.FieldNode:
			stats.NumExtensions++
		case *ast.GroupNode:
			stats.NumExtensions++
			examineMessage(&decl.MessageBody, stats)
		}
	}
}
