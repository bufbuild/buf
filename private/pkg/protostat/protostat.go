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

package protostat

import (
	"context"
	"io"

	"github.com/bufbuild/protocompile/ast"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
)

// Stats represents some statistics about one or more Protobuf files.
type Stats struct {
	NumFiles                 int `json:"num_files,omitempty" yaml:"num_files,omitempty"`
	NumPackages              int `json:"num_packages,omitempty" yaml:"num_packages,omitempty"`
	NumFilesWithSyntaxErrors int `json:"num_files_with_syntax_errors,omitempty" yaml:"num_files_with_syntax_errors,omitempty"`
	NumMessages              int `json:"num_messages,omitempty" yaml:"num_messages,omitempty"`
	NumFields                int `json:"num_fields,omitempty" yaml:"num_fields,omitempty"`
	NumEnums                 int `json:"num_enums,omitempty" yaml:"num_enums,omitempty"`
	NumEnumValues            int `json:"num_enum_values,omitempty" yaml:"num_enum_values,omitempty"`
	NumExtensions            int `json:"num_extensions,omitempty" yaml:"num_extensions,omitempty"`
	NumServices              int `json:"num_services,omitempty" yaml:"num_services,omitempty"`
	NumMethods               int `json:"num_methods,omitempty" yaml:"num_methods,omitempty"`
}

// FileWalker goes through all .proto files for GetStats.
type FileWalker interface {
	// Walk will invoke f for all .proto files for GetStats.
	Walk(ctx context.Context, f func(io.Reader) error) error
}

// GetStats gathers some simple statistics about a set of Protobuf files.
//
// See the packages protostatos and protostatstorage for helpers for the
// os and storage packages.
func GetStats(ctx context.Context, fileWalker FileWalker) (*Stats, error) {
	handler := reporter.NewHandler(
		reporter.NewReporter(
			func(reporter.ErrorWithPos) error {
				// never aborts
				return nil
			},
			nil,
		),
	)
	statsBuilder := newStatsBuilder()
	if err := fileWalker.Walk(
		ctx,
		func(file io.Reader) error {
			// This can return an error and non-nil AST.
			// We do not need the filePath because we do not report errors.
			astRoot, err := parser.Parse("", file, handler)
			if astRoot == nil {
				// No AST implies an I/O error trying to read the
				// file contents. No stats to collect.
				return err
			}
			if err != nil {
				// There was a syntax error, but we still have a partial
				// AST we can examine.
				statsBuilder.NumFilesWithSyntaxErrors++
			}
			examineFile(statsBuilder, astRoot)
			return nil
		},
	); err != nil {
		return nil, err
	}
	statsBuilder.NumPackages = len(statsBuilder.packages)
	return statsBuilder.Stats, nil
}

type statsBuilder struct {
	*Stats

	packages map[ast.Identifier]struct{}
}

func newStatsBuilder() *statsBuilder {
	return &statsBuilder{
		Stats:    &Stats{},
		packages: make(map[ast.Identifier]struct{}),
	}
}

func examineFile(statsBuilder *statsBuilder, fileNode *ast.FileNode) {
	statsBuilder.NumFiles++
	for _, decl := range fileNode.Decls {
		switch decl := decl.(type) {
		case *ast.PackageNode:
			statsBuilder.packages[decl.Name.AsIdentifier()] = struct{}{}
		case *ast.MessageNode:
			examineMessage(statsBuilder, &decl.MessageBody)
		case *ast.EnumNode:
			examineEnum(statsBuilder, decl)
		case *ast.ExtendNode:
			examineExtend(statsBuilder, decl)
		case *ast.ServiceNode:
			statsBuilder.NumServices++
			for _, decl := range decl.Decls {
				_, ok := decl.(*ast.RPCNode)
				if ok {
					statsBuilder.NumMethods++
				}
			}
		}
	}
}

func examineMessage(statsBuilder *statsBuilder, messageBody *ast.MessageBody) {
	statsBuilder.NumMessages++
	for _, decl := range messageBody.Decls {
		switch decl := decl.(type) {
		case *ast.FieldNode, *ast.MapFieldNode:
			statsBuilder.NumFields++
		case *ast.GroupNode:
			statsBuilder.NumFields++
			examineMessage(statsBuilder, &decl.MessageBody)
		case *ast.OneOfNode:
			for _, ooDecl := range decl.Decls {
				switch ooDecl := ooDecl.(type) {
				case *ast.FieldNode:
					statsBuilder.NumFields++
				case *ast.GroupNode:
					statsBuilder.NumFields++
					examineMessage(statsBuilder, &ooDecl.MessageBody)
				}
			}
		case *ast.MessageNode:
			examineMessage(statsBuilder, &decl.MessageBody)
		case *ast.EnumNode:
			examineEnum(statsBuilder, decl)
		case *ast.ExtendNode:
			examineExtend(statsBuilder, decl)
		}
	}
}

func examineEnum(statsBuilder *statsBuilder, enumNode *ast.EnumNode) {
	statsBuilder.NumEnums++
	for _, decl := range enumNode.Decls {
		_, ok := decl.(*ast.EnumValueNode)
		if ok {
			statsBuilder.NumEnumValues++
		}
	}
}

func examineExtend(statsBuilder *statsBuilder, extendNode *ast.ExtendNode) {
	for _, decl := range extendNode.Decls {
		switch decl := decl.(type) {
		case *ast.FieldNode:
			statsBuilder.NumExtensions++
		case *ast.GroupNode:
			statsBuilder.NumExtensions++
			examineMessage(statsBuilder, &decl.MessageBody)
		}
	}
}
