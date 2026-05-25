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

package protostat

import (
	"context"
	"io"

	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/parser"
	"github.com/bufbuild/protocompile/experimental/report"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/bufbuild/protocompile/experimental/source"
)

// Stats represents some statistics about one or more Protobuf files.
//
// Note that as opposed to most structs in this codebase, we do not omitempty for
// the fields for JSON or YAML.
type Stats struct {
	Files                 int `json:"files" yaml:"files"`
	Types                 int `json:"types" yaml:"types"`
	Packages              int `json:"packages" yaml:"packages"`
	Messages              int `json:"messages" yaml:"messages"`
	DeprecatedMessages    int `json:"deprecated_messages" yaml:"deprecated_messages"`
	Fields                int `json:"fields" yaml:"fields"`
	Enums                 int `json:"enums" yaml:"enums"`
	DeprecatedEnums       int `json:"deprecated_enums" yaml:"deprecated_enums"`
	EnumValues            int `json:"evalues" yaml:"evalues"`
	Services              int `json:"services" yaml:"services"`
	RPCs                  int `json:"rpcs" yaml:"rpcs"`
	DeprecatedRPCs        int `json:"deprecated_rpcs" yaml:"deprecated_rpcs"`
	Extensions            int `json:"extensions" yaml:"extensions"`
	FilesWithSyntaxErrors int `json:"-" yaml:"-"`
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
	statsBuilder := newStatsBuilder()
	if err := fileWalker.Walk(
		ctx,
		func(file io.Reader) error {
			data, err := io.ReadAll(file)
			if err != nil {
				return err
			}
			r := &report.Report{Options: report.Options{SuppressWarnings: true}}
			astFile, _ := parser.Parse("", source.NewFile("", string(data)), r)
			if astFile == nil {
				// Parse only returns a nil file in pathological cases; bail
				// without recording stats, matching the legacy I/O-error path.
				return nil
			}
			for _, d := range r.Diagnostics {
				if d.Level() <= report.Error {
					statsBuilder.FilesWithSyntaxErrors++
					break
				}
			}
			examineFile(statsBuilder, astFile)
			return nil
		},
	); err != nil {
		return nil, err
	}
	statsBuilder.Packages = len(statsBuilder.packages)
	return statsBuilder.Stats, nil
}

// MergeStats merged multiple stats objects into one single Stats object.
//
// A new object is returned.
func MergeStats(statsSlice ...*Stats) *Stats {
	resultStats := &Stats{}
	for _, stats := range statsSlice {
		resultStats.Files += stats.Files
		resultStats.FilesWithSyntaxErrors += stats.FilesWithSyntaxErrors
		resultStats.Packages += stats.Packages
		resultStats.Types += stats.Types
		resultStats.Messages += stats.Messages
		resultStats.DeprecatedMessages += stats.DeprecatedMessages
		resultStats.Fields += stats.Fields
		resultStats.Enums += stats.Enums
		resultStats.DeprecatedEnums += stats.DeprecatedEnums
		resultStats.EnumValues += stats.EnumValues
		resultStats.Services += stats.Services
		resultStats.RPCs += stats.RPCs
		resultStats.DeprecatedRPCs += stats.DeprecatedRPCs
		resultStats.Extensions += stats.Extensions
	}
	return resultStats
}

type statsBuilder struct {
	*Stats

	packages map[string]struct{}
}

func newStatsBuilder() *statsBuilder {
	return &statsBuilder{
		Stats:    &Stats{},
		packages: make(map[string]struct{}),
	}
}

func examineFile(statsBuilder *statsBuilder, file *ast.File) {
	statsBuilder.Files++
	for decl := range seq.Values(file.Decls()) {
		if pkg := decl.AsPackage(); !pkg.IsZero() {
			statsBuilder.packages[pkg.Path().Canonicalized()] = struct{}{}
			continue
		}
		def := decl.AsDef()
		if def.IsZero() {
			continue
		}
		switch def.Classify() {
		case ast.DefKindMessage:
			examineMessage(statsBuilder, def.AsMessage().Body, def)
		case ast.DefKindEnum:
			examineEnum(statsBuilder, def.AsEnum().Body, def)
		case ast.DefKindExtend:
			examineExtend(statsBuilder, def.AsExtend().Body)
		case ast.DefKindService:
			statsBuilder.Services++
			for innerDecl := range seq.Values(def.AsService().Body.Decls()) {
				innerDef := innerDecl.AsDef()
				if innerDef.IsZero() || innerDef.Classify() != ast.DefKindMethod {
					continue
				}
				statsBuilder.RPCs++
				statsBuilder.Types++
				if isDeprecated(innerDef) {
					statsBuilder.DeprecatedRPCs++
				}
			}
		}
	}
}

// examineMessage examines a message or group body and updates stats. The def is the
// owning ast.DeclDef (message or group), used for the deprecation check.
func examineMessage(statsBuilder *statsBuilder, body ast.DeclBody, def ast.DeclDef) {
	statsBuilder.Messages++
	statsBuilder.Types++
	if isDeprecated(def) {
		statsBuilder.DeprecatedMessages++
	}
	for decl := range seq.Values(body.Decls()) {
		innerDef := decl.AsDef()
		if innerDef.IsZero() {
			continue
		}
		switch innerDef.Classify() {
		case ast.DefKindField:
			statsBuilder.Fields++
		case ast.DefKindGroup:
			statsBuilder.Fields++
			examineMessage(statsBuilder, innerDef.AsGroup().Body, innerDef)
		case ast.DefKindOneof:
			for ooDecl := range seq.Values(innerDef.AsOneof().Body.Decls()) {
				ooDef := ooDecl.AsDef()
				if ooDef.IsZero() {
					continue
				}
				switch ooDef.Classify() {
				case ast.DefKindField:
					statsBuilder.Fields++
				case ast.DefKindGroup:
					statsBuilder.Fields++
					examineMessage(statsBuilder, ooDef.AsGroup().Body, ooDef)
				}
			}
		case ast.DefKindMessage:
			examineMessage(statsBuilder, innerDef.AsMessage().Body, innerDef)
		case ast.DefKindEnum:
			examineEnum(statsBuilder, innerDef.AsEnum().Body, innerDef)
		case ast.DefKindExtend:
			examineExtend(statsBuilder, innerDef.AsExtend().Body)
		}
	}
}

func examineEnum(statsBuilder *statsBuilder, body ast.DeclBody, def ast.DeclDef) {
	statsBuilder.Enums++
	statsBuilder.Types++
	if isDeprecated(def) {
		statsBuilder.DeprecatedEnums++
	}
	for decl := range seq.Values(body.Decls()) {
		innerDef := decl.AsDef()
		if !innerDef.IsZero() && innerDef.Classify() == ast.DefKindEnumValue {
			statsBuilder.EnumValues++
		}
	}
}

func examineExtend(statsBuilder *statsBuilder, body ast.DeclBody) {
	for decl := range seq.Values(body.Decls()) {
		innerDef := decl.AsDef()
		if innerDef.IsZero() {
			continue
		}
		switch innerDef.Classify() {
		case ast.DefKindField:
			statsBuilder.Extensions++
		case ast.DefKindGroup:
			statsBuilder.Extensions++
			examineMessage(statsBuilder, innerDef.AsGroup().Body, innerDef)
		}
	}
}

// isDeprecated reports whether def is marked `deprecated = true`, either via
// a compact option or a body-level `option deprecated = true;`.
func isDeprecated(def ast.DeclDef) bool {
	for entry := range seq.Values(def.Options().Entries()) {
		if entry.Path.IsIdents("deprecated") && exprIsTrue(entry.Value) {
			return true
		}
	}
	body := def.Body()
	if body.IsZero() {
		return false
	}
	for decl := range seq.Values(body.Decls()) {
		inner := decl.AsDef()
		if inner.IsZero() || inner.Classify() != ast.DefKindOption {
			continue
		}
		opt := inner.AsOption()
		if opt.Path.IsIdents("deprecated") && exprIsTrue(opt.Value) {
			return true
		}
	}
	return false
}

// exprIsTrue reports whether expr is the identifier `true`.
func exprIsTrue(expr ast.ExprAny) bool {
	path := expr.AsPath()
	if path.IsZero() {
		return false
	}
	return path.Path.IsIdents("true")
}
