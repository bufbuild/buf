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

package bufformat

import (
	"slices"
	"strings"

	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/bufbuild/protocompile/experimental/token"
	"github.com/bufbuild/protocompile/experimental/token/keyword"
)

// fullNameMatcher determines which types should have deprecated options added.
type fullNameMatcher struct {
	prefixes []string
}

// newFullNameMatcher creates a new matcher for the given FQN prefixes.
func newFullNameMatcher(fqnPrefixes ...string) *fullNameMatcher {
	return &fullNameMatcher{prefixes: fqnPrefixes}
}

// matchesPrefix returns true if the given FQN matches any prefix using
// component-aware prefix matching.
func (m *fullNameMatcher) matchesPrefix(fqn string) bool {
	for _, prefix := range m.prefixes {
		if fqnMatchesPrefix(fqn, prefix) {
			return true
		}
	}
	return false
}

// matchesExact returns true if the given FQN matches any prefix exactly.
func (m *fullNameMatcher) matchesExact(fqn string) bool {
	return slices.Contains(m.prefixes, fqn)
}

// fqnMatchesPrefix returns true if fqn equals prefix or starts with
// "prefix.". This is component-aware matching: "foo.bar" matches
// "foo.bar.baz" but not "foo.bart".
func fqnMatchesPrefix(fqn, prefix string) bool {
	if len(prefix) > len(fqn) {
		return false
	}
	if len(prefix) == len(fqn) {
		return fqn == prefix
	}
	return prefix == "" || strings.HasPrefix(fqn, prefix+".")
}

// applyDeprecations walks the file's AST, adding deprecation markers to every
// declaration whose fully-qualified name matches the matcher. It returns true
// if any prefix matched a declaration, regardless of whether a new option was
// added — already-deprecated types still count as matched, so callers can
// distinguish a no-op pass from one that found no matches at all.
//
// The AST is mutated in place. After calling, the file must be re-rendered
// via the printer to materialize the changes.
//
// NOTE: this function performs all mutations directly through the
// experimental/ast, experimental/token, and experimental/seq APIs rather
// than experimental/ast/edit. The edit package's KindAdd only appends to
// the end of a target's decl list and cannot express the positional
// inserts (body deprecation must precede other decls), compact-options
// entries ([deprecated = true] on a field/enum value), or RPC-method-
// body synthesis (turning `rpc Foo() returns (Bar);` into a body) that
// we need. Once the edit package grows these capabilities, this file
// should collapse into a list of edit.Edits.
func applyDeprecations(file *ast.File, matcher *fullNameMatcher) bool {
	if matcher == nil || len(matcher.prefixes) == 0 {
		return false
	}
	var matched bool
	pkg := packageFQN(file)
	if pkg != "" && matcher.matchesPrefix(pkg) {
		matched = true
		if !hasDeprecatedDecl(file.Decls()) {
			insertFileDeprecatedOption(file)
		}
	}
	walkDecls(file, file.Decls(), pkg, matcher, &matched)
	return matched
}

// walkDecls recursively visits the declarations under parentFQN, applying
// deprecation mutations to every matching def.
func walkDecls(
	file *ast.File,
	decls seq.Inserter[ast.DeclAny],
	parentFQN string,
	matcher *fullNameMatcher,
	matched *bool,
) {
	for decl := range seq.Values(decls) {
		def := decl.AsDef()
		if def.IsZero() {
			continue
		}
		name := defName(def)
		fqn := joinFQN(parentFQN, name)

		switch def.Classify() {
		case ast.DefKindMessage, ast.DefKindService:
			if name != "" && matcher.matchesPrefix(fqn) {
				*matched = true
				if !hasDeprecatedDecl(def.Body().Decls()) {
					insertBodyDeprecatedOption(file, def.Body())
				}
			}
			if !def.Body().IsZero() {
				walkDecls(file, def.Body().Decls(), fqn, matcher, matched)
			}
		case ast.DefKindEnum:
			if name != "" && matcher.matchesPrefix(fqn) {
				*matched = true
				if !hasDeprecatedDecl(def.Body().Decls()) {
					insertBodyDeprecatedOption(file, def.Body())
				}
			}
			// Enum values are scoped under the enum's parent, not the enum
			// itself, so we recurse with parentFQN unchanged.
			if !def.Body().IsZero() {
				walkDecls(file, def.Body().Decls(), parentFQN, matcher, matched)
			}
		case ast.DefKindMethod:
			if name != "" && matcher.matchesPrefix(fqn) {
				*matched = true
				if def.Body().IsZero() {
					attachMethodBodyWithDeprecated(file, def)
				} else if !hasDeprecatedDecl(def.Body().Decls()) {
					insertBodyDeprecatedOption(file, def.Body())
				}
			}
		case ast.DefKindField, ast.DefKindGroup:
			if name != "" && matcher.matchesExact(fqn) {
				*matched = true
				if !hasDeprecatedCompactOption(def.Options()) {
					addCompactDeprecated(file, def)
				}
			}
			// Groups behave like messages for nested types.
			if def.Classify() == ast.DefKindGroup && !def.Body().IsZero() {
				walkDecls(file, def.Body().Decls(), fqn, matcher, matched)
			}
		case ast.DefKindEnumValue:
			if name != "" && matcher.matchesExact(fqn) {
				*matched = true
				if !hasDeprecatedCompactOption(def.Options()) {
					addCompactDeprecated(file, def)
				}
			}
		case ast.DefKindOneof, ast.DefKindExtend:
			// Oneofs and extend blocks are containers without their own FQN
			// participation in deprecation, but their nested decls (fields)
			// still need to be visited under the surrounding scope.
			if !def.Body().IsZero() {
				walkDecls(file, def.Body().Decls(), parentFQN, matcher, matched)
			}
		}
	}
}

// packageFQN returns the canonicalized package name from a file's
// `package ...;` declaration, or "" if there is no package.
func packageFQN(file *ast.File) string {
	pkg := file.Package()
	if pkg.IsZero() {
		return ""
	}
	return pkg.Path().Canonicalized()
}

// defName returns the identifier name of a DeclDef, or "" if the def has
// no single-identifier name (e.g. a compound option path).
func defName(def ast.DeclDef) string {
	ident := def.Name().AsIdent()
	if ident.IsZero() {
		return ""
	}
	return ident.Name()
}

// joinFQN concatenates parent and name with a dot, handling empty inputs.
func joinFQN(parent, name string) string {
	if name == "" {
		return parent
	}
	if parent == "" {
		return name
	}
	return parent + "." + name
}

// hasDeprecatedDecl reports whether decls already contains
// `option deprecated = true;`.
func hasDeprecatedDecl(decls seq.Inserter[ast.DeclAny]) bool {
	for decl := range seq.Values(decls) {
		def := decl.AsDef()
		if def.IsZero() || def.Classify() != ast.DefKindOption {
			continue
		}
		if def.Name().IsIdents("deprecated") && exprIsTrue(def.Value()) {
			return true
		}
	}
	return false
}

// hasDeprecatedCompactOption reports whether opts already contains
// `deprecated = true`.
func hasDeprecatedCompactOption(opts ast.CompactOptions) bool {
	if opts.IsZero() {
		return false
	}
	for entry := range seq.Values(opts.Entries()) {
		if entry.Path.IsIdents("deprecated") && exprIsTrue(entry.Value) {
			return true
		}
	}
	return false
}

// exprIsTrue reports whether expr is the literal `true` identifier.
func exprIsTrue(expr ast.ExprAny) bool {
	path := expr.AsPath()
	if path.IsZero() {
		return false
	}
	return path.Path.IsIdents("true")
}

// insertFileDeprecatedOption inserts `option deprecated = true;` at the
// file level, positioned after the leading syntax/package/import/option
// runs. Required because edit.KindAdd would append to the end of the
// file's decl list — and the file-level deprecation pattern is to place
// the option alongside the other file-level options near the top.
//
// (In practice, printer.Legacy() with CanonicalizeFileOrder also moves
// file-level options into position, but we still insert at the right
// place so the AST itself is canonical and so non-format mode prints
// correctly.)
func insertFileDeprecatedOption(file *ast.File) {
	decls := file.Decls()
	insertPos := 0
	for i := range decls.Len() {
		d := decls.At(i)
		if !d.AsSyntax().IsZero() || !d.AsPackage().IsZero() || !d.AsImport().IsZero() {
			insertPos = i + 1
			continue
		}
		if def := d.AsDef(); !def.IsZero() && def.Classify() == ast.DefKindOption {
			insertPos = i + 1
			continue
		}
		break
	}
	decls.Insert(insertPos, newDeprecatedOptionDecl(file).AsAny())
}

// insertBodyDeprecatedOption inserts `option deprecated = true;` at the
// very top of a body. The legacy formatter places injected deprecation
// markers before any other options in the body; matching that ordering
// keeps existing goldens stable. Body decl order is not canonicalized by
// the printer, so positional insert is required.
func insertBodyDeprecatedOption(file *ast.File, body ast.DeclBody) {
	body.Decls().Insert(0, newDeprecatedOptionDecl(file).AsAny())
}

// attachMethodBodyWithDeprecated synthesizes a body containing
// `option deprecated = true;` and attaches it to a method definition that
// previously ended with a semicolon. The printer ignores the stored
// semicolon when a body is present (see printer/decl.go printMethod), so
// we do not need to clear it — and DeclDef has no public way to do so today.
func attachMethodBodyWithDeprecated(file *ast.File, def ast.DeclDef) {
	stream := file.Stream()
	nodes := file.Nodes()
	openBrace := stream.NewPunct(keyword.LBrace.String())
	closeBrace := stream.NewPunct(keyword.RBrace.String())
	stream.NewFused(openBrace, closeBrace)
	body := nodes.NewDeclBody(openBrace)
	seq.Append(body.Decls(), newDeprecatedOptionDecl(file).AsAny())
	def.SetBody(body)
}

// addCompactDeprecated adds `deprecated = true` to a def's compact options.
// If the def has no compact-options bracket yet, one is synthesized. The
// new entry is prepended (index 0) to match the legacy formatter's golden
// output, which always lists `deprecated` first when adding to a field or
// enum value that already has other compact options.
func addCompactDeprecated(file *ast.File, def ast.DeclDef) {
	stream := file.Stream()
	nodes := file.Nodes()
	opts := def.Options()
	if opts.IsZero() {
		openBracket := stream.NewPunct(keyword.LBracket.String())
		closeBracket := stream.NewPunct(keyword.RBracket.String())
		stream.NewFused(openBracket, closeBracket)
		opts = nodes.NewCompactOptions(openBracket)
		def.SetOptions(opts)
	}
	entries := opts.Entries()
	deprecatedOpt := newDeprecatedOption(file)
	if entries.Len() == 0 {
		seq.Append(entries, deprecatedOpt)
		return
	}
	// Prepend with a comma so the new entry is followed by ", " and the
	// previous first entry becomes the second. Commas in this model are
	// owned by the entry they follow.
	comma := stream.NewPunct(keyword.Comma.String())
	entries.InsertComma(0, deprecatedOpt, comma)
}

// newDeprecatedOptionDecl synthesizes an `option deprecated = true;` decl.
func newDeprecatedOptionDecl(file *ast.File) ast.DeclDef {
	stream := file.Stream()
	nodes := file.Nodes()
	optionType := ast.TypePath{
		Path: nodes.NewPath(
			nodes.NewPathComponent(token.Zero, stream.NewIdent(keyword.Option.String())),
		),
	}.AsAny()
	namePath := nodes.NewPath(
		nodes.NewPathComponent(token.Zero, stream.NewIdent("deprecated")),
	)
	value := ast.ExprPath{
		Path: nodes.NewPath(
			nodes.NewPathComponent(token.Zero, stream.NewIdent(keyword.True.String())),
		),
	}.AsAny()
	return nodes.NewDeclDef(ast.DeclDefArgs{
		Type:      optionType,
		Name:      namePath,
		Equals:    stream.NewPunct(keyword.Assign.String()),
		Value:     value,
		Semicolon: stream.NewPunct(keyword.Semi.String()),
	})
}

// newDeprecatedOption synthesizes a `deprecated = true` compact option entry.
func newDeprecatedOption(file *ast.File) ast.Option {
	stream := file.Stream()
	nodes := file.Nodes()
	return ast.Option{
		Path: nodes.NewPath(
			nodes.NewPathComponent(token.Zero, stream.NewIdent("deprecated")),
		),
		Equals: stream.NewPunct(keyword.Assign.String()),
		Value: ast.ExprPath{
			Path: nodes.NewPath(
				nodes.NewPathComponent(token.Zero, stream.NewIdent(keyword.True.String())),
			),
		}.AsAny(),
	}
}
