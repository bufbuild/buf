// Copyright 2020-2022 Buf Technologies, Inc.
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
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/bufbuild/protocompile/ast"
	"go.uber.org/multierr"
)

// formatter writes an *ast.FileNode as a .proto file.
type formatter struct {
	writer      io.Writer
	fileNode    *ast.FileNode
	indent      int
	lastWritten rune

	// Used to determine how the current node's
	// leading comments should be written.
	previousNode ast.Node

	// if true, a space will be written to the output unless the next character
	// written is a newline (don't wait errant trailing spaces)
	pendingSpace bool
	// if true, the formatter is in the middle of printing compact options
	inCompactOptions bool

	// Records all errors that occur during
	// the formatting process. Nearly any
	// non-nil error represents a bug in the
	// implementation.
	err error
}

// newFormatter returns a new formatter for the given file.
func newFormatter(
	writer io.Writer,
	fileNode *ast.FileNode,
) *formatter {
	return &formatter{
		writer:   writer,
		fileNode: fileNode,
	}
}

// Run runs the formatter and writes the file's content to the formatter's writer.
func (f *formatter) Run() error {
	f.writeFile()
	return f.err
}

// P prints a line to the generated output.
func (f *formatter) P(elements ...string) {
	if len(elements) > 0 {
		// We only want to write an indent if we're
		// writing elements (not just a newline).
		f.Indent()
	}
	for _, elem := range elements {
		f.WriteString(elem)
	}
	f.WriteString("\n")
}

// Space adds a space to the generated output.
func (f *formatter) Space() {
	f.pendingSpace = true
}

// In increases the current level of indentation.
func (f *formatter) In() {
	f.indent++
}

// Out reduces the current level of indentation.
func (f *formatter) Out() {
	if f.indent <= 0 {
		// Unreachable.
		f.err = multierr.Append(
			f.err,
			errors.New("internal error: attempted to decrement indentation at zero"),
		)
		return
	}
	f.indent--
}

// Indent writes the number of spaces associated
// with the current level of indentation.
func (f *formatter) Indent() {
	f.WriteString(strings.Repeat("  ", f.indent))
}

// WriteString writes the given element to the generated output.
func (f *formatter) WriteString(elem string) {
	if f.pendingSpace {
		f.pendingSpace = false
		first, _ := utf8.DecodeRuneInString(elem)

		// We don't want "dangling spaces" before certain characters:
		// newlines, commas, semicolons, and close parens/braces.
		// Similarly, we don't want extra/doubled spaces or dangling
		// spaces after certain characters: open parens/braces. So
		// only print the space if the previous and next character
		// don't match above conditions.

		if !strings.ContainsRune("\x00 \t\n<[{(", f.lastWritten) &&
			!strings.ContainsRune("\n;,)]}>", first) {
			if _, err := f.writer.Write([]byte{' '}); err != nil {
				f.err = multierr.Append(f.err, err)
				return
			}
		}
	}
	if len(elem) == 0 {
		return
	}
	f.lastWritten, _ = utf8.DecodeLastRuneInString(elem)
	if _, err := f.writer.Write([]byte(elem)); err != nil {
		f.err = multierr.Append(f.err, err)
	}
}

// SetPreviousNode sets the previously written node. This should
// be called in all of the comment writing functions.
func (f *formatter) SetPreviousNode(node ast.Node) {
	f.previousNode = node
}

// writeFile writes the file node.
func (f *formatter) writeFile() {
	f.writeFileHeader()
	f.writeFileTypes()
	if f.fileNode.EOF != nil {
		info := f.fileNode.NodeInfo(f.fileNode.EOF)
		if info.LeadingComments().Len() > 0 {
			if !f.previousTrailingCommentsWroteNewline() {
				// If the previous node didn't have trailing comments that
				// wrote a newline, we need to add one here.
				f.P()
			}
			f.P()
			f.writeMultilineComments(info.LeadingComments())
			return
		}
	}
	if f.previousNode != nil {
		// If anything was written, we always conclude with
		// a newline.
		f.P()
	}
}

// writeFileHeader writes the header of a .proto file. This includes the syntax,
// package, imports, and options (in that order). The imports and options are
// sorted. All other file elements are handled by f.writeFileTypes.
//
// For example,
//
//	syntax = "proto3";
//
//	package acme.v1.weather;
//
//	import "acme/payment/v1/payment.proto";
//	import "google/type/datetime.proto";
//
//	option cc_enable_arenas = true;
//	option optimize_for = SPEED;
func (f *formatter) writeFileHeader() {
	var (
		packageNode *ast.PackageNode
		importNodes []*ast.ImportNode
		optionNodes []*ast.OptionNode
		typeNodes   []ast.FileElement
	)
	for _, fileElement := range f.fileNode.Decls {
		switch node := fileElement.(type) {
		case *ast.PackageNode:
			packageNode = node
		case *ast.ImportNode:
			importNodes = append(importNodes, node)
		case *ast.OptionNode:
			optionNodes = append(optionNodes, node)
		case *ast.EmptyDeclNode:
			continue
		default:
			typeNodes = append(typeNodes, node)
		}
	}
	if f.fileNode.Syntax == nil && packageNode == nil && importNodes == nil && optionNodes == nil {
		// There aren't any header values, so we can return early.
		return
	}
	if syntaxNode := f.fileNode.Syntax; syntaxNode != nil {
		f.writeSyntax(syntaxNode)
	}
	if packageNode != nil {
		if f.previousNode != nil {
			if !f.startsWithNewline(f.fileNode.NodeInfo(packageNode.Keyword)) {
				f.P()
			}
			f.P()
		}
		f.writePackage(packageNode)
	}
	sort.Slice(importNodes, func(i, j int) bool {
		return importNodes[i].Name.AsString() < importNodes[j].Name.AsString()
	})
	for i, importNode := range importNodes {
		if i == 0 && f.previousNode != nil {
			f.P()
			f.P()
		}
		if i > 0 {
			f.P()
		}
		f.writeImport(importNode)
	}
	sort.Slice(optionNodes, func(i, j int) bool {
		// The default options (e.g. cc_enable_arenas) should always
		// be sorted above custom options (which are identified by a
		// leading '(').
		left := stringForOptionName(optionNodes[i].Name)
		right := stringForOptionName(optionNodes[j].Name)
		if strings.HasPrefix(left, "(") && !strings.HasPrefix(right, "(") {
			// Prefer the default option on the right.
			return false
		}
		if !strings.HasPrefix(left, "(") && strings.HasPrefix(right, "(") {
			// Prefer the default option on the left.
			return true
		}
		// Both options are custom, so we defer to the standard sorting.
		return left < right
	})
	for i, optionNode := range optionNodes {
		if i == 0 && f.previousNode != nil {
			f.P()
			f.P()
		}
		if i > 0 {
			f.P()
		}
		f.writeFileOption(optionNode)
	}
	if len(typeNodes) > 0 {
		f.P()
	}
}

// writeFileTypes writes the types defined in a .proto file. This includes the messages, enums,
// services, etc. All other elements are ignored since they are handled by f.writeFileHeader.
func (f *formatter) writeFileTypes() {
	var writeNewline bool
	for _, fileElement := range f.fileNode.Decls {
		switch node := fileElement.(type) {
		case *ast.PackageNode, *ast.OptionNode, *ast.ImportNode, *ast.EmptyDeclNode:
			// These elements have already been written by f.writeFileHeader.
			continue
		default:
			if writeNewline {
				// File-level nodes should be separated by a newline.
				f.P()
			}
			f.writeNode(node)
			// We want to start writing newlines as soon as we've written
			// a single type.
			writeNewline = true
		}
	}
}

// writeSyntax writes the syntax.
//
// For example,
//
//	syntax = "proto3";
func (f *formatter) writeSyntax(syntaxNode *ast.SyntaxNode) {
	f.writeStart(syntaxNode.Keyword)
	f.Space()
	f.writeInline(syntaxNode.Equals)
	f.Space()
	f.writeInline(syntaxNode.Syntax)
	f.writeLineEnd(syntaxNode.Semicolon)
}

// writePackage writes the package.
//
// For example,
//
//	package acme.weather.v1;
func (f *formatter) writePackage(packageNode *ast.PackageNode) {
	f.writeStart(packageNode.Keyword)
	f.Space()
	f.writeInline(packageNode.Name)
	f.writeLineEnd(packageNode.Semicolon)
}

// writeImport writes an import statement.
//
// For example,
//
//	import "google/protobuf/descriptor.proto";
func (f *formatter) writeImport(importNode *ast.ImportNode) {
	// We don't use f.writeStart here because the imports are sorted
	// and potentially changed order.
	f.writeBodyEndInline(importNode.Keyword)
	f.Space()
	// We don't want to write the "public" and "weak" nodes
	// if they aren't defined. One could be set, but never both.
	switch {
	case importNode.Public != nil:
		f.writeInline(importNode.Public)
		f.Space()
	case importNode.Weak != nil:
		f.writeInline(importNode.Weak)
		f.Space()
	}
	f.writeInline(importNode.Name)
	f.writeLineEnd(importNode.Semicolon)
}

// writeFileOption writes a file option. This function is slightly
// different than f.writeOption because file options are sorted at
// the top of the file, and leading comments are adjusted accordingly.
func (f *formatter) writeFileOption(optionNode *ast.OptionNode) {
	// We don't use f.writeStart here because the options are sorted
	// and potentially changed order.
	f.writeBodyEndInline(optionNode.Keyword)
	f.Space()
	f.writeNode(optionNode.Name)
	f.Space()
	f.writeInline(optionNode.Equals)
	if node, ok := optionNode.Val.(*ast.CompoundStringLiteralNode); ok {
		// Compound string literals are written across multiple lines
		// immediately after the '=', so we don't need a trailing
		// space in the option prefix.
		f.writeCompoundStringLiteralForSingleOption(node)
		f.writeLineEnd(optionNode.Semicolon)
		return
	}
	f.Space()
	f.writeInline(optionNode.Val)
	f.writeLineEnd(optionNode.Semicolon)
}

// writeOption writes an option.
//
// For example,
//
//	option go_package = "github.com/foo/bar";
func (f *formatter) writeOption(optionNode *ast.OptionNode) {
	f.writeOptionPrefix(optionNode)
	if optionNode.Semicolon != nil {
		if node, ok := optionNode.Val.(*ast.CompoundStringLiteralNode); ok {
			// Compound string literals are written across multiple lines
			// immediately after the '=', so we don't need a trailing
			// space in the option prefix.
			f.writeCompoundStringLiteralForSingleOption(node)
			f.writeLineEnd(optionNode.Semicolon)
			return
		}
		f.writeInline(optionNode.Val)
		f.writeLineEnd(optionNode.Semicolon)
		return
	}
	f.writeInline(optionNode.Val)
}

// writeLastCompactOption writes a compact option but preserves its the
// trailing end comments. This is only used for the last compact option
// since it's the only time a trailing ',' will be omitted.
//
// For example,
//
//	[
//	  deprecated = true,
//	  json_name = "something" // Trailing comment on the last element.
//	]
func (f *formatter) writeLastCompactOption(optionNode *ast.OptionNode) {
	f.writeOptionPrefix(optionNode)
	f.writeLineEnd(optionNode.Val)
}

// writeOptionValue writes the option prefix, which makes up all of the
// option's definition, excluding the final token(s).
//
// For example,
//
//	deprecated =
func (f *formatter) writeOptionPrefix(optionNode *ast.OptionNode) {
	if optionNode.Keyword != nil {
		// Compact options don't have the keyword.
		f.writeStart(optionNode.Keyword)
		f.Space()
		f.writeNode(optionNode.Name)
	} else {
		f.writeStart(optionNode.Name)
	}
	f.Space()
	f.writeInline(optionNode.Equals)
	if _, ok := optionNode.Val.(*ast.CompoundStringLiteralNode); ok {
		// Compound string literals are written across multiple lines
		// immediately after the '=', so we don't need a trailing
		// space in the option prefix.
		return
	}
	f.Space()
}

// writeOptionName writes an option name.
//
// For example,
//
//	go_package
//	(custom.thing)
//	(custom.thing).bridge.(another.thing)
func (f *formatter) writeOptionName(optionNameNode *ast.OptionNameNode) {
	for i := 0; i < len(optionNameNode.Parts); i++ {
		if f.inCompactOptions && i == 0 {
			// The leading comments of the first token (either open rune or the
			// name) will have already been written, so we need to handle this
			// case specially.
			fieldReferenceNode := optionNameNode.Parts[0]
			if fieldReferenceNode.Open != nil {
				f.writeNode(fieldReferenceNode.Open)
				if info := f.fileNode.NodeInfo(fieldReferenceNode.Open); info.TrailingComments().Len() > 0 {
					f.writeInlineComments(info.TrailingComments())
				}
				f.writeInline(fieldReferenceNode.Name)
			} else {
				f.writeNode(fieldReferenceNode.Name)
				if info := f.fileNode.NodeInfo(fieldReferenceNode.Name); info.TrailingComments().Len() > 0 {
					f.writeInlineComments(info.TrailingComments())
				}
			}
			if fieldReferenceNode.Close != nil {
				f.writeInline(fieldReferenceNode.Close)
			}
			continue
		}
		if i > 0 {
			// The length of this slice must be exactly len(Parts)-1.
			f.writeInline(optionNameNode.Dots[i-1])
		}
		f.writeNode(optionNameNode.Parts[i])
	}
}

// writeMessage writes the message node.
//
// For example,
//
//	message Foo {
//	  option deprecated = true;
//	  reserved 50 to 100;
//	  extensions 150 to 200;
//
//	  message Bar {
//	    string name = 1;
//	  }
//	  enum Baz {
//	    BAZ_UNSPECIFIED = 0;
//	  }
//	  extend Bar {
//	    string value = 2;
//	  }
//
//	  Bar bar = 1;
//	  Baz baz = 2;
//	}
func (f *formatter) writeMessage(messageNode *ast.MessageNode) {
	var elementWriterFunc func()
	if len(messageNode.Decls) != 0 {
		elementWriterFunc = func() {
			for i, decl := range messageNode.Decls {
				if i > 0 {
					f.P()
				}
				f.writeNode(decl)
			}
		}
	}
	f.writeStart(messageNode.Keyword)
	f.Space()
	f.writeInline(messageNode.Name)
	f.Space()
	f.writeCompositeTypeBody(
		messageNode.OpenBrace,
		messageNode.CloseBrace,
		elementWriterFunc,
	)
}

// writeMessageLiteral writes a message literal.
//
// For example,
//
//	{
//	  foo: 1
//	  foo: 2
//	  foo: 3
//	  bar: <
//	    name:"abc"
//	    id:123
//	  >
//	}
func (f *formatter) writeMessageLiteral(messageLiteralNode *ast.MessageLiteralNode) {
	if f.maybeWriteCompactMessageLiteral(messageLiteralNode, false) {
		return
	}
	var elementWriterFunc func()
	if len(messageLiteralNode.Elements) > 0 {
		elementWriterFunc = func() {
			f.writeMessageLiteralElements(messageLiteralNode)
		}
	}
	f.writeCompositeValueBody(
		messageLiteralNode.Open,
		messageLiteralNode.Close,
		elementWriterFunc,
	)
}

// writeMessageLiteral writes a message literal suitable for
// an element in an array literal.
func (f *formatter) writeMessageLiteralForArray(
	messageLiteralNode *ast.MessageLiteralNode,
) {
	if f.maybeWriteCompactMessageLiteral(messageLiteralNode, true) {
		return
	}
	var elementWriterFunc func()
	if len(messageLiteralNode.Elements) > 0 {
		elementWriterFunc = func() {
			f.writeMessageLiteralElements(messageLiteralNode)
		}
	}
	f.writeBody(
		messageLiteralNode.Open,
		messageLiteralNode.Close,
		elementWriterFunc,
		f.writeOpenBracePrefixForArray,
		f.writeBodyEndInline,
	)
}

func (f *formatter) maybeWriteCompactMessageLiteral(
	messageLiteralNode *ast.MessageLiteralNode,
	inArrayLiteral bool,
) bool {
	if len(messageLiteralNode.Elements) == 0 || len(messageLiteralNode.Elements) > 1 ||
		f.hasInteriorComments(messageLiteralNode) ||
		messageLiteralHasNestedMessageOrArray(messageLiteralNode) {
		return false
	}
	// messages with a single scalar field and no comments can be
	// printed all on one line
	if inArrayLiteral {
		f.Indent()
	}
	f.writeInline(messageLiteralNode.Open)
	fieldNode := messageLiteralNode.Elements[0]
	f.writeInline(fieldNode.Name)
	if fieldNode.Sep != nil {
		f.writeInline(fieldNode.Sep)
	}
	f.Space()
	f.writeInline(fieldNode.Val)
	f.writeInline(messageLiteralNode.Close)
	return true
}

func messageLiteralHasNestedMessageOrArray(messageLiteralNode *ast.MessageLiteralNode) bool {
	for _, elem := range messageLiteralNode.Elements {
		switch elem.Val.(type) {
		case *ast.ArrayLiteralNode, *ast.MessageLiteralNode:
			return true
		}
	}
	return false
}

func arrayLiteralHasNestedMessageOrArray(arrayLiteralNode *ast.ArrayLiteralNode) bool {
	for _, elem := range arrayLiteralNode.Elements {
		switch elem.(type) {
		case *ast.ArrayLiteralNode, *ast.MessageLiteralNode:
			return true
		}
	}
	return false
}

// writeMessageLiteralElements writes the message literal's elements.
//
// For example,
//
//	foo: 1
//	foo: 2
func (f *formatter) writeMessageLiteralElements(messageLiteralNode *ast.MessageLiteralNode) {
	for i := 0; i < len(messageLiteralNode.Elements); i++ {
		if i > 0 {
			f.P()
		}
		if sep := messageLiteralNode.Seps[i]; sep != nil {
			f.writeMessageFieldWithSeparator(messageLiteralNode.Elements[i])
			f.writeLineEnd(messageLiteralNode.Seps[i])
			continue
		}
		f.writeNode(messageLiteralNode.Elements[i])
	}
}

// writeMessageField writes the message field node, and concludes the
// line without leaving room for a trailing separator in the parent
// message literal.
func (f *formatter) writeMessageField(messageFieldNode *ast.MessageFieldNode) {
	f.writeMessageFieldPrefix(messageFieldNode)
	f.writeLineEnd(messageFieldNode.Val)
}

// writeMessageFieldWithSeparator writes the message field node,
// but leaves room for a trailing separator in the parent message
// literal.
func (f *formatter) writeMessageFieldWithSeparator(messageFieldNode *ast.MessageFieldNode) {
	f.writeMessageFieldPrefix(messageFieldNode)
	f.writeInline(messageFieldNode.Val)
}

// writeMessageFieldPrefix writes the message field node as a single line.
//
// For example,
//
//	foo:"bar"
func (f *formatter) writeMessageFieldPrefix(messageFieldNode *ast.MessageFieldNode) {
	// The comments need to be written as a multiline comment above
	// the message field name.
	//
	// Note that this is different than how field reference nodes are
	// normally formatted in-line (i.e. as option name components).
	fieldReferenceNode := messageFieldNode.Name
	if fieldReferenceNode.Open != nil {
		f.writeStart(fieldReferenceNode.Open)
		f.writeInline(fieldReferenceNode.Name)
	} else {
		f.writeStart(fieldReferenceNode.Name)
	}
	if fieldReferenceNode.Close != nil {
		f.writeInline(fieldReferenceNode.Close)
	}
	if messageFieldNode.Sep != nil {
		f.writeInline(messageFieldNode.Sep)
	}
	if _, ok := messageFieldNode.Val.(*ast.CompoundStringLiteralNode); ok {
		// Compound string literals are written across multiple lines
		// immediately after the ':', so we don't need a trailing
		// space in the option prefix.
		return
	}
	f.Space()
}

// writeEnum writes the enum node.
//
// For example,
//
//	enum Foo {
//	  option deprecated = true;
//	  reserved 1 to 5;
//
//	  FOO_UNSPECIFIED = 0;
//	}
func (f *formatter) writeEnum(enumNode *ast.EnumNode) {
	var elementWriterFunc func()
	if len(enumNode.Decls) > 0 {
		elementWriterFunc = func() {
			for i, decl := range enumNode.Decls {
				if i > 0 {
					f.P()
				}
				f.writeNode(decl)
			}
		}
	}
	f.writeStart(enumNode.Keyword)
	f.Space()
	f.writeInline(enumNode.Name)
	f.Space()
	f.writeCompositeTypeBody(
		enumNode.OpenBrace,
		enumNode.CloseBrace,
		elementWriterFunc,
	)
}

// writeEnumValue writes the enum value as a single line. If the enum has
// compact options, it will be written across multiple lines.
//
// For example,
//
//	FOO_UNSPECIFIED = 1 [
//	  deprecated = true
//	];
func (f *formatter) writeEnumValue(enumValueNode *ast.EnumValueNode) {
	f.writeStart(enumValueNode.Name)
	f.Space()
	f.writeInline(enumValueNode.Equals)
	f.Space()
	f.writeInline(enumValueNode.Number)
	if enumValueNode.Options != nil {
		f.Space()
		f.writeNode(enumValueNode.Options)
	}
	f.writeLineEnd(enumValueNode.Semicolon)
}

// writeField writes the field node as a single line. If the field has
// compact options, it will be written across multiple lines.
//
// For example,
//
//	repeated string name = 1 [
//	  deprecated = true,
//	  json_name = "name"
//	];
func (f *formatter) writeField(fieldNode *ast.FieldNode) {
	// We need to handle the comments for the field label specially since
	// a label might not be defined, but it has the leading comments attached
	// to it.
	if fieldNode.Label.KeywordNode != nil {
		f.writeStart(fieldNode.Label)
		f.Space()
		f.writeInline(fieldNode.FldType)
	} else {
		// If a label was not written, the multiline comments will be
		// attached to the type.
		if compoundIdentNode, ok := fieldNode.FldType.(*ast.CompoundIdentNode); ok {
			f.writeCompountIdentForFieldName(compoundIdentNode)
		} else {
			f.writeStart(fieldNode.FldType)
		}
	}
	f.Space()
	f.writeInline(fieldNode.Name)
	f.Space()
	f.writeInline(fieldNode.Equals)
	f.Space()
	f.writeInline(fieldNode.Tag)
	if fieldNode.Options != nil {
		f.Space()
		f.writeNode(fieldNode.Options)
	}
	f.writeLineEnd(fieldNode.Semicolon)
}

// writeMapField writes a map field (e.g. 'map<string, string> pairs = 1;').
func (f *formatter) writeMapField(mapFieldNode *ast.MapFieldNode) {
	f.writeNode(mapFieldNode.MapType)
	f.Space()
	f.writeInline(mapFieldNode.Name)
	f.Space()
	f.writeInline(mapFieldNode.Equals)
	f.Space()
	f.writeInline(mapFieldNode.Tag)
	if mapFieldNode.Options != nil {
		f.Space()
		f.writeNode(mapFieldNode.Options)
	}
	f.writeLineEnd(mapFieldNode.Semicolon)
}

// writeMapType writes a map type (e.g. 'map<string, string>').
func (f *formatter) writeMapType(mapTypeNode *ast.MapTypeNode) {
	f.writeStart(mapTypeNode.Keyword)
	f.writeInline(mapTypeNode.OpenAngle)
	f.writeInline(mapTypeNode.KeyType)
	f.writeInline(mapTypeNode.Comma)
	f.Space()
	f.writeInline(mapTypeNode.ValueType)
	f.writeInline(mapTypeNode.CloseAngle)
}

// writeFieldReference writes a field reference (e.g. '(foo.bar)').
func (f *formatter) writeFieldReference(fieldReferenceNode *ast.FieldReferenceNode) {
	if fieldReferenceNode.Open != nil {
		f.writeInline(fieldReferenceNode.Open)
	}
	f.writeInline(fieldReferenceNode.Name)
	if fieldReferenceNode.Close != nil {
		f.writeInline(fieldReferenceNode.Close)
	}
}

// writeExtend writes the extend node.
//
// For example,
//
//	extend google.protobuf.FieldOptions {
//	  bool redacted = 33333;
//	}
func (f *formatter) writeExtend(extendNode *ast.ExtendNode) {
	var elementWriterFunc func()
	if len(extendNode.Decls) > 0 {
		elementWriterFunc = func() {
			for i, decl := range extendNode.Decls {
				if i > 0 {
					f.P()
				}
				f.writeNode(decl)
			}
		}
	}
	f.writeStart(extendNode.Keyword)
	f.Space()
	f.writeInline(extendNode.Extendee)
	f.Space()
	f.writeCompositeTypeBody(
		extendNode.OpenBrace,
		extendNode.CloseBrace,
		elementWriterFunc,
	)
}

// writeService writes the service node.
//
// For example,
//
//	service FooService {
//	  option deprecated = true;
//
//	  rpc Foo(FooRequest) returns (FooResponse) {};
func (f *formatter) writeService(serviceNode *ast.ServiceNode) {
	var elementWriterFunc func()
	if len(serviceNode.Decls) > 0 {
		elementWriterFunc = func() {
			for i, decl := range serviceNode.Decls {
				if i > 0 {
					f.P()
				}
				f.writeNode(decl)
			}
		}
	}
	f.writeStart(serviceNode.Keyword)
	f.Space()
	f.writeInline(serviceNode.Name)
	f.Space()
	f.writeCompositeTypeBody(
		serviceNode.OpenBrace,
		serviceNode.CloseBrace,
		elementWriterFunc,
	)
}

// writeRPC writes the RPC node. RPCs are formatted in
// the following order:
//
// For example,
//
//	rpc Foo(FooRequest) returns (FooResponse) {
//	  option deprecated = true;
//	};
func (f *formatter) writeRPC(rpcNode *ast.RPCNode) {
	var elementWriterFunc func()
	if len(rpcNode.Decls) > 0 {
		elementWriterFunc = func() {
			for i, decl := range rpcNode.Decls {
				if i > 0 {
					f.P()
				}
				f.writeNode(decl)
			}
		}
	}
	f.writeStart(rpcNode.Keyword)
	f.Space()
	f.writeInline(rpcNode.Name)
	f.writeInline(rpcNode.Input)
	f.Space()
	f.writeInline(rpcNode.Returns)
	f.Space()
	f.writeInline(rpcNode.Output)
	if rpcNode.OpenBrace == nil {
		// This RPC doesn't have any elements, so we prefer the
		// ';' form.
		//
		//  rpc Ping(PingRequest) returns (PingResponse);
		//
		f.writeLineEnd(rpcNode.Semicolon)
		return
	}
	f.Space()
	f.writeCompositeTypeBody(
		rpcNode.OpenBrace,
		rpcNode.CloseBrace,
		elementWriterFunc,
	)
}

// writeRPCType writes the RPC type node (e.g. (stream foo.Bar)).
func (f *formatter) writeRPCType(rpcTypeNode *ast.RPCTypeNode) {
	f.writeInline(rpcTypeNode.OpenParen)
	if rpcTypeNode.Stream != nil {
		f.writeInline(rpcTypeNode.Stream)
		f.Space()
	}
	f.writeInline(rpcTypeNode.MessageType)
	f.writeInline(rpcTypeNode.CloseParen)
}

// writeOneOf writes the oneof node.
//
// For example,
//
//	oneof foo {
//	  option deprecated = true;
//
//	  string name = 1;
//	  int number = 2;
//	}
func (f *formatter) writeOneOf(oneOfNode *ast.OneOfNode) {
	var elementWriterFunc func()
	if len(oneOfNode.Decls) > 0 {
		elementWriterFunc = func() {
			for i, decl := range oneOfNode.Decls {
				if i > 0 {
					f.P()
				}
				f.writeNode(decl)
			}
		}
	}
	f.writeStart(oneOfNode.Keyword)
	f.Space()
	f.writeInline(oneOfNode.Name)
	f.Space()
	f.writeCompositeTypeBody(
		oneOfNode.OpenBrace,
		oneOfNode.CloseBrace,
		elementWriterFunc,
	)
}

// writeGroup writes the group node.
//
// For example,
//
//	optional group Key = 4 [
//	  deprecated = true,
//	  json_name = "key"
//	] {
//	  optional uint64 id = 1;
//	  optional string name = 2;
//	}
func (f *formatter) writeGroup(groupNode *ast.GroupNode) {
	var elementWriterFunc func()
	if len(groupNode.Decls) > 0 {
		elementWriterFunc = func() {
			for i, decl := range groupNode.Decls {
				if i > 0 {
					f.P()
				}
				f.writeNode(decl)
			}
		}
	}
	// We need to handle the comments for the group label specially since
	// a label might not be defined, but it has the leading comments attached
	// to it.
	if groupNode.Label.KeywordNode != nil {
		f.writeStart(groupNode.Label)
		f.Space()
		f.writeInline(groupNode.Keyword)
	} else {
		// If a label was not written, the multiline comments will be
		// attached to the keyword.
		f.writeStart(groupNode.Keyword)
	}
	f.Space()
	f.writeInline(groupNode.Name)
	f.Space()
	f.writeInline(groupNode.Equals)
	f.Space()
	f.writeInline(groupNode.Tag)
	if groupNode.Options != nil {
		f.Space()
		f.writeNode(groupNode.Options)
	}
	f.Space()
	f.writeCompositeTypeBody(
		groupNode.OpenBrace,
		groupNode.CloseBrace,
		elementWriterFunc,
	)
}

// writeExtensionRange writes the extension range node.
//
// For example,
//
//	extensions 5-10, 100 to max [
//	  deprecated = true
//	];
func (f *formatter) writeExtensionRange(extensionRangeNode *ast.ExtensionRangeNode) {
	f.writeStart(extensionRangeNode.Keyword)
	f.Space()
	for i := 0; i < len(extensionRangeNode.Ranges); i++ {
		if i > 0 {
			// The length of this slice must be exactly len(Ranges)-1.
			f.writeInline(extensionRangeNode.Commas[i-1])
			f.Space()
		}
		f.writeNode(extensionRangeNode.Ranges[i])
	}
	if extensionRangeNode.Options != nil {
		f.Space()
		f.writeNode(extensionRangeNode.Options)
	}
	f.writeLineEnd(extensionRangeNode.Semicolon)
}

// writeReserved writes a reserved node.
//
// For example,
//
//	reserved 5-10, 100 to max;
func (f *formatter) writeReserved(reservedNode *ast.ReservedNode) {
	f.writeStart(reservedNode.Keyword)
	// Either names or ranges will be set, but never both.
	elements := make([]ast.Node, 0, len(reservedNode.Names)+len(reservedNode.Ranges))
	switch {
	case reservedNode.Names != nil:
		for _, nameNode := range reservedNode.Names {
			elements = append(elements, nameNode)
		}
	case reservedNode.Ranges != nil:
		for _, rangeNode := range reservedNode.Ranges {
			elements = append(elements, rangeNode)
		}
	}
	f.Space()
	for i := 0; i < len(elements); i++ {
		if i > 0 {
			// The length of this slice must be exactly len({Names,Ranges})-1.
			f.writeInline(reservedNode.Commas[i-1])
			f.Space()
		}
		f.writeInline(elements[i])
	}
	f.writeLineEnd(reservedNode.Semicolon)
}

// writeRange writes the given range node (e.g. '1 to max').
func (f *formatter) writeRange(rangeNode *ast.RangeNode) {
	f.writeInline(rangeNode.StartVal)
	if rangeNode.To != nil {
		f.Space()
		f.writeInline(rangeNode.To)
	}
	// Either EndVal or Max will be set, but never both.
	switch {
	case rangeNode.EndVal != nil:
		f.Space()
		f.writeInline(rangeNode.EndVal)
	case rangeNode.Max != nil:
		f.Space()
		f.writeInline(rangeNode.Max)
	}
}

// writeCompactOptions writes a compact options node.
//
// For example,
//
//	[
//	  deprecated = true,
//	  json_name = "something"
//	]
func (f *formatter) writeCompactOptions(compactOptionsNode *ast.CompactOptionsNode) {
	f.inCompactOptions = true
	defer func() {
		f.inCompactOptions = false
	}()
	if len(compactOptionsNode.Options) == 1 &&
		!f.hasInteriorComments(compactOptionsNode) {
		// If there's only a single compact scalar option without comments, we can write it
		// in-line. For example:
		//
		//  [deprecated = true]
		//
		// However, this does not include the case when the '[' has trailing comments,
		// or the option name has leading comments. In those cases, we write the option
		// across multiple lines. For example:
		//
		//  [
		//    // This type is deprecated.
		//    deprecated = true
		//  ]
		//
		optionNode := compactOptionsNode.Options[0]
		f.writeInline(compactOptionsNode.OpenBracket)
		f.writeInline(optionNode.Name)
		f.Space()
		f.writeInline(optionNode.Equals)
		if node, ok := optionNode.Val.(*ast.CompoundStringLiteralNode); ok {
			// If there's only a single compact option, the value needs to
			// write its comments (if any) in a way that preserves the closing ']'.
			f.writeCompoundStringLiteralForSingleOption(node)
			f.writeInline(compactOptionsNode.CloseBracket)
			return
		}
		f.Space()
		f.writeInline(optionNode.Val)
		f.writeInline(compactOptionsNode.CloseBracket)
		return
	}
	var elementWriterFunc func()
	if len(compactOptionsNode.Options) > 0 {
		elementWriterFunc = func() {
			for i := 0; i < len(compactOptionsNode.Options); i++ {
				if i > 0 {
					f.P()
				}
				if i == len(compactOptionsNode.Options)-1 {
					// The last element won't have a trailing comma.
					f.writeLastCompactOption(compactOptionsNode.Options[i])
					return
				}
				f.writeNode(compactOptionsNode.Options[i])
				f.writeLineEnd(compactOptionsNode.Commas[i])
			}
		}
	}
	f.writeCompositeValueBody(
		compactOptionsNode.OpenBracket,
		compactOptionsNode.CloseBracket,
		elementWriterFunc,
	)
}

func (f *formatter) hasInteriorComments(n ast.Node) bool {
	cn, ok := n.(ast.CompositeNode)
	if !ok {
		return false
	}
	children := cn.Children()
	for i, child := range children {
		// interior comments mean we ignore leading comments on first
		// token and trailing comments on the last one
		info := f.fileNode.NodeInfo(child)
		if i > 0 && info.LeadingComments().Len() > 0 {
			return true
		}
		if i < len(children)-1 && info.TrailingComments().Len() > 0 {
			return true
		}
	}
	return false
}

// writeArrayLiteral writes an array literal across multiple lines.
//
// For example,
//
//	[
//	  "foo",
//	  "bar"
//	]
func (f *formatter) writeArrayLiteral(arrayLiteralNode *ast.ArrayLiteralNode) {
	if len(arrayLiteralNode.Elements) == 1 &&
		!f.hasInteriorComments(arrayLiteralNode) &&
		!arrayLiteralHasNestedMessageOrArray(arrayLiteralNode) {
		// arrays with a single scalar value and no comments can be
		// printed all on one line
		valueNode := arrayLiteralNode.Elements[0]
		f.writeInline(arrayLiteralNode.OpenBracket)
		f.writeInline(valueNode)
		f.writeInline(arrayLiteralNode.CloseBracket)
		return
	}

	var elementWriterFunc func()
	if len(arrayLiteralNode.Elements) > 0 {
		elementWriterFunc = func() {
			for i := 0; i < len(arrayLiteralNode.Elements); i++ {
				if i > 0 {
					f.P()
				}
				lastElement := i == len(arrayLiteralNode.Elements)-1
				if compositeNode, ok := arrayLiteralNode.Elements[i].(ast.CompositeNode); ok {
					f.writeCompositeValueForArrayLiteral(compositeNode, lastElement)
					if !lastElement {
						f.writeLineEnd(arrayLiteralNode.Commas[i])
					}
					continue
				}
				if lastElement {
					// The last element won't have a trailing comma.
					f.writeBodyEnd(arrayLiteralNode.Elements[i])
					return
				}
				f.writeStart(arrayLiteralNode.Elements[i])
				f.writeLineEnd(arrayLiteralNode.Commas[i])
			}
		}
	}
	f.writeCompositeValueBody(
		arrayLiteralNode.OpenBracket,
		arrayLiteralNode.CloseBracket,
		elementWriterFunc,
	)
}

// writeCompositeForArrayLiteral writes the composite node in a way that's suitable
// for array literals. In general, signed integers and compound strings should have their
// comments written in-line because they are one of many components in a single line.
//
// However, each of these composite types occupy a single line in an array literal,
// so they need their comments to be formatted like a standalone node.
//
// For example,
//
//	option (value) = /* In-line comment for '-42' */ -42;
//
//	option (thing) = {
//	  values: [
//	    // Leading comment on -42.
//	    -42, // Trailing comment on -42.
//	  ]
//	}
//
// The lastElement boolean is used to signal whether or not the composite value
// should be written as the last element (i.e. it doesn't have a trailing comma).
func (f *formatter) writeCompositeValueForArrayLiteral(
	compositeNode ast.CompositeNode,
	lastElement bool,
) {
	switch node := compositeNode.(type) {
	case *ast.CompoundStringLiteralNode:
		f.writeCompoundStringLiteralForArray(node, lastElement)
	case *ast.PositiveUintLiteralNode:
		f.writePositiveUintLiteralForArray(node, lastElement)
	case *ast.NegativeIntLiteralNode:
		f.writeNegativeIntLiteralForArray(node, lastElement)
	case *ast.SignedFloatLiteralNode:
		f.writeSignedFloatLiteralForArray(node, lastElement)
	case *ast.MessageLiteralNode:
		f.writeMessageLiteralForArray(node)
	default:
		f.err = multierr.Append(f.err, fmt.Errorf("unexpected array value node %T", node))
	}
}

// writeCompositeTypeBody writes the body of a composite type, e.g. message, enum, extend, oneof, etc.
func (f *formatter) writeCompositeTypeBody(
	openBrace *ast.RuneNode,
	closeBrace *ast.RuneNode,
	elementWriterFunc func(),
) {
	f.writeBody(
		openBrace,
		closeBrace,
		elementWriterFunc,
		f.writeOpenBracePrefix,
		f.writeBodyEnd,
	)
}

// writeCompositeValueBody writes the body of a composite value, e.g. compact options,
// array literal, etc. We need to handle the ']' different than composite types because
// there could be more tokens following the final ']'.
func (f *formatter) writeCompositeValueBody(
	openBrace *ast.RuneNode,
	closeBrace *ast.RuneNode,
	elementWriterFunc func(),
) {
	f.writeBody(
		openBrace,
		closeBrace,
		elementWriterFunc,
		f.writeOpenBracePrefix,
		f.writeBodyEndInline,
	)
}

// writeBody writes the body of a type or value, e.g. message, enum, compact options, etc.
// The elementWriterFunc is used to write the declarations within the composite type (e.g.
// fields in a message). The openBraceWriterFunc and closeBraceWriterFunc functions are used
// to customize how the '{' and '} nodes are written, respectively.
func (f *formatter) writeBody(
	openBrace *ast.RuneNode,
	closeBrace *ast.RuneNode,
	elementWriterFunc func(),
	openBraceWriterFunc func(ast.Node),
	closeBraceWriterFunc func(ast.Node),
) {
	openBraceWriterFunc(openBrace)
	info := f.fileNode.NodeInfo(openBrace)
	if info.TrailingComments().Len() == 0 && elementWriterFunc == nil {
		info = f.fileNode.NodeInfo(closeBrace)
		if info.LeadingComments().Len() == 0 {
			// This is an empty definition without any comments in
			// the body, so we return early.
			f.writeLineEnd(closeBrace)
			return
		}
		// Writing comments in this case requires special care - we need
		// to write all of the leading comments in an empty type body,
		// so the comments need to be indented, but the tokens they're
		// attached to should not be indented.
		//
		// For example,
		//
		//  message Foo {
		//    // This message might have multiple comments.
		//    // They're attached to the '}'.
		//  }
		f.P()
		f.In()
		f.writeMultilineComments(info.LeadingComments())
		f.Out()
		// The '}' should be indented on its own line but we've already handled
		// the comments, so we manually apply it here.
		f.Indent()
		f.writeNode(closeBrace)
		f.writeTrailingEndComments(info.TrailingComments())
		f.SetPreviousNode(closeBrace)
		return
	}
	f.P()
	f.In()
	if info.TrailingComments().Len() > 0 {
		f.writeMultilineComments(info.TrailingComments())
		if elementWriterFunc != nil {
			// If there are other elements to write, we need another
			// newline after the trailing comments.
			f.P()
		}
	}
	if elementWriterFunc != nil {
		elementWriterFunc()
		if !f.previousTrailingCommentsWroteNewline() {
			// If the previous node didn't have trailing comments that
			// wrote a newline, we need to add one here.
			f.P()
		}
	}
	f.Out()
	closeBraceWriterFunc(closeBrace)
}

// writeOpenBracePrefix writes the open brace with its leading comments in-line.
// This is used for nearly every use case of f.writeBody, excluding the instances
// in array literals.
func (f *formatter) writeOpenBracePrefix(openBrace ast.Node) {
	defer f.SetPreviousNode(openBrace)
	info := f.fileNode.NodeInfo(openBrace)
	if info.LeadingComments().Len() > 0 {
		f.writeInlineComments(info.LeadingComments())
		if info.LeadingWhitespace() != "" {
			f.Space()
		}
	}
	f.writeNode(openBrace)
}

// writeOpenBracePrefixForArray writes the open brace with its leading comments
// on multiple lines. This is only used for message literals in arrays.
func (f *formatter) writeOpenBracePrefixForArray(openBrace ast.Node) {
	defer f.SetPreviousNode(openBrace)
	info := f.fileNode.NodeInfo(openBrace)
	if info.LeadingComments().Len() > 0 {
		f.writeMultilineComments(info.LeadingComments())
	}
	f.Indent()
	f.writeNode(openBrace)
}

// writeCompoundIdent writes a compound identifier (e.g. '.com.foo.Bar').
func (f *formatter) writeCompoundIdent(compoundIdentNode *ast.CompoundIdentNode) {
	if compoundIdentNode.LeadingDot != nil {
		f.writeInline(compoundIdentNode.LeadingDot)
	}
	for i := 0; i < len(compoundIdentNode.Components); i++ {
		if i > 0 {
			// The length of this slice must be exactly len(Components)-1.
			f.writeInline(compoundIdentNode.Dots[i-1])
		}
		f.writeInline(compoundIdentNode.Components[i])
	}
}

// writeCompountIdentForFieldName writes a compound identifier, but handles comments
// specially for field names.
//
// For example,
//
//	message Foo {
//	  // These are comments attached to bar.
//	  bar.v1.Bar bar = 1;
//	}
func (f *formatter) writeCompountIdentForFieldName(compoundIdentNode *ast.CompoundIdentNode) {
	if compoundIdentNode.LeadingDot != nil {
		f.writeStart(compoundIdentNode.LeadingDot)
	}
	for i := 0; i < len(compoundIdentNode.Components); i++ {
		if i == 0 && compoundIdentNode.LeadingDot == nil {
			f.writeStart(compoundIdentNode.Components[i])
			continue
		}
		if i > 0 {
			// The length of this slice must be exactly len(Components)-1.
			f.writeInline(compoundIdentNode.Dots[i-1])
		}
		f.writeInline(compoundIdentNode.Components[i])
	}
}

// writeFieldLabel writes the field label node.
//
// For example,
//
//	optional
//	repeated
//	required
func (f *formatter) writeFieldLabel(fieldLabel ast.FieldLabel) {
	f.WriteString(fieldLabel.Val)
}

// writeCompoundStringLiteral writes a compound string literal value.
//
// For example,
//
//	"one,"
//	"two,"
//	"three"
func (f *formatter) writeCompoundStringLiteral(compoundStringLiteralNode *ast.CompoundStringLiteralNode) {
	f.In()
	for _, child := range compoundStringLiteralNode.Children() {
		if !f.previousTrailingCommentsWroteNewline() {
			// If the previous node didn't have trailing comments that
			// wrote a newline, we need to add one here.
			f.P()
		}
		f.writeBodyEnd(child)
	}
	f.Out()
}

// writeCompoundStringLiteralForSingleOption writes a compound string literal value,
// but writes its comments suitable for a single value option.
//
// The last element is written with in-line comments so that the closing ';' or ']'
// can exist on the same line.
//
// For example,
//
//	option (custom) =
//	  "one,"
//	  "two,"
//	  "three";
func (f *formatter) writeCompoundStringLiteralForSingleOption(compoundStringLiteralNode *ast.CompoundStringLiteralNode) {
	f.In()
	for i, child := range compoundStringLiteralNode.Children() {
		if !f.previousTrailingCommentsWroteNewline() {
			// If the previous node didn't have trailing comments that
			// wrote a newline, we need to add one here.
			f.P()
		}
		if i == len(compoundStringLiteralNode.Children())-1 {
			f.writeBodyEndInline(child)
			break
		}
		f.writeBodyEnd(child)
	}
	f.Out()
}

// writeCompoundStringLiteralForArray writes a compound string literal value,
// but writes its comments suitable for an element in an array literal.
//
// The lastElement boolean is used to signal whether or not the value should
// be written as the last element (i.e. it doesn't have a trailing comma).
func (f *formatter) writeCompoundStringLiteralForArray(
	compoundStringLiteralNode *ast.CompoundStringLiteralNode,
	lastElement bool,
) {
	for i, child := range compoundStringLiteralNode.Children() {
		if i > 0 {
			if !f.previousTrailingCommentsWroteNewline() {
				// If the previous node didn't have trailing comments that
				// wrote a newline, we need to add one here.
				f.P()
			}
		}
		if !lastElement && i == len(compoundStringLiteralNode.Children())-1 {
			f.writeBodyEndInline(child)
			return
		}
		f.writeBodyEnd(child)
	}
}

// writeFloatLiteral writes a float literal value (e.g. '42.2').
func (f *formatter) writeFloatLiteral(floatLiteralNode *ast.FloatLiteralNode) {
	f.WriteString(strconv.FormatFloat(floatLiteralNode.Val, 'g', -1, 64))
}

// writeSignedFloatLiteral writes a signed float literal value (e.g. '-42.2').
func (f *formatter) writeSignedFloatLiteral(signedFloatLiteralNode *ast.SignedFloatLiteralNode) {
	f.writeInline(signedFloatLiteralNode.Sign)
	f.writeInline(signedFloatLiteralNode.Float)
}

// writeSignedFloatLiteralForArray writes a signed float literal value, but writes
// its comments suitable for an element in an array literal.
//
// The lastElement boolean is used to signal whether or not the value should
// be written as the last element (i.e. it doesn't have a trailing comma).
func (f *formatter) writeSignedFloatLiteralForArray(
	signedFloatLiteralNode *ast.SignedFloatLiteralNode,
	lastElement bool,
) {
	f.writeStart(signedFloatLiteralNode.Sign)
	if lastElement {
		f.writeLineEnd(signedFloatLiteralNode.Float)
		return
	}
	f.writeInline(signedFloatLiteralNode.Float)
}

// writeSpecialFloatLiteral writes a special float literal value (e.g. "nan" or "inf").
func (f *formatter) writeSpecialFloatLiteral(specialFloatLiteralNode *ast.SpecialFloatLiteralNode) {
	f.WriteString(specialFloatLiteralNode.KeywordNode.Val)
}

// writeStringLiteral writes a string literal value (e.g. "foo").
// Note that the raw string is written as-is so that it preserves
// the quote style used in the original source.
func (f *formatter) writeStringLiteral(stringLiteralNode *ast.StringLiteralNode) {
	info := f.fileNode.TokenInfo(stringLiteralNode.Token())
	f.WriteString(info.RawText())
}

// writeUintLiteral writes a uint literal (e.g. '42').
func (f *formatter) writeUintLiteral(uintLiteralNode *ast.UintLiteralNode) {
	f.WriteString(strconv.FormatUint(uintLiteralNode.Val, 10))
}

// writeNegativeIntLiteral writes a negative int literal (e.g. '-42').
func (f *formatter) writeNegativeIntLiteral(negativeIntLiteralNode *ast.NegativeIntLiteralNode) {
	f.writeInline(negativeIntLiteralNode.Minus)
	f.writeInline(negativeIntLiteralNode.Uint)
}

// writeNegativeIntLiteralForArray writes a negative int literal value, but writes
// its comments suitable for an element in an array literal.
//
// The lastElement boolean is used to signal whether or not the value should
// be written as the last element (i.e. it doesn't have a trailing comma).
func (f *formatter) writeNegativeIntLiteralForArray(
	negativeIntLiteralNode *ast.NegativeIntLiteralNode,
	lastElement bool,
) {
	f.writeStart(negativeIntLiteralNode.Minus)
	if lastElement {
		f.writeLineEnd(negativeIntLiteralNode.Uint)
		return
	}
	f.writeInline(negativeIntLiteralNode.Uint)
}

// writePositiveUintLiteral writes a positive uint literal (e.g. '+42').
func (f *formatter) writePositiveUintLiteral(positiveIntLiteralNode *ast.PositiveUintLiteralNode) {
	f.writeInline(positiveIntLiteralNode.Plus)
	f.writeInline(positiveIntLiteralNode.Uint)
}

// writePositiveUintLiteralForArray writes a positive uint literal value, but writes
// its comments suitable for an element in an array literal.
//
// The lastElement boolean is used to signal whether or not the value should
// be written as the last element (i.e. it doesn't have a trailing comma).
func (f *formatter) writePositiveUintLiteralForArray(
	positiveIntLiteralNode *ast.PositiveUintLiteralNode,
	lastElement bool,
) {
	f.writeStart(positiveIntLiteralNode.Plus)
	if lastElement {
		f.writeLineEnd(positiveIntLiteralNode.Uint)
		return
	}
	f.writeInline(positiveIntLiteralNode.Uint)
}

// writeIdent writes an identifier (e.g. 'foo').
func (f *formatter) writeIdent(identNode *ast.IdentNode) {
	f.WriteString(identNode.Val)
}

// writeKeyword writes a keyword (e.g. 'syntax').
func (f *formatter) writeKeyword(keywordNode *ast.KeywordNode) {
	f.WriteString(keywordNode.Val)
}

// writeRune writes a rune (e.g. '=').
func (f *formatter) writeRune(runeNode *ast.RuneNode) {
	f.WriteString(string(runeNode.Rune))
}

// writeNode writes the node by dispatching to a function tailored to its concrete type.
//
// Comments are handled in each respective write function so that it can determine whether
// to write the comments in-line or not.
func (f *formatter) writeNode(node ast.Node) {
	switch element := node.(type) {
	case *ast.ArrayLiteralNode:
		f.writeArrayLiteral(element)
	case *ast.CompactOptionsNode:
		f.writeCompactOptions(element)
	case *ast.CompoundIdentNode:
		f.writeCompoundIdent(element)
	case *ast.CompoundStringLiteralNode:
		f.writeCompoundStringLiteral(element)
	case *ast.EnumNode:
		f.writeEnum(element)
	case *ast.EnumValueNode:
		f.writeEnumValue(element)
	case *ast.ExtendNode:
		f.writeExtend(element)
	case *ast.ExtensionRangeNode:
		f.writeExtensionRange(element)
	case ast.FieldLabel:
		f.writeFieldLabel(element)
	case *ast.FieldNode:
		f.writeField(element)
	case *ast.FieldReferenceNode:
		f.writeFieldReference(element)
	case *ast.FloatLiteralNode:
		f.writeFloatLiteral(element)
	case *ast.GroupNode:
		f.writeGroup(element)
	case *ast.IdentNode:
		f.writeIdent(element)
	case *ast.ImportNode:
		f.writeImport(element)
	case *ast.KeywordNode:
		f.writeKeyword(element)
	case *ast.MapFieldNode:
		f.writeMapField(element)
	case *ast.MapTypeNode:
		f.writeMapType(element)
	case *ast.MessageNode:
		f.writeMessage(element)
	case *ast.MessageFieldNode:
		f.writeMessageField(element)
	case *ast.MessageLiteralNode:
		f.writeMessageLiteral(element)
	case *ast.NegativeIntLiteralNode:
		f.writeNegativeIntLiteral(element)
	case *ast.OneOfNode:
		f.writeOneOf(element)
	case *ast.OptionNode:
		f.writeOption(element)
	case *ast.OptionNameNode:
		f.writeOptionName(element)
	case *ast.PackageNode:
		f.writePackage(element)
	case *ast.PositiveUintLiteralNode:
		f.writePositiveUintLiteral(element)
	case *ast.RangeNode:
		f.writeRange(element)
	case *ast.ReservedNode:
		f.writeReserved(element)
	case *ast.RPCNode:
		f.writeRPC(element)
	case *ast.RPCTypeNode:
		f.writeRPCType(element)
	case *ast.RuneNode:
		f.writeRune(element)
	case *ast.ServiceNode:
		f.writeService(element)
	case *ast.SignedFloatLiteralNode:
		f.writeSignedFloatLiteral(element)
	case *ast.SpecialFloatLiteralNode:
		f.writeSpecialFloatLiteral(element)
	case *ast.StringLiteralNode:
		f.writeStringLiteral(element)
	case *ast.SyntaxNode:
		f.writeSyntax(element)
	case *ast.UintLiteralNode:
		f.writeUintLiteral(element)
	case *ast.EmptyDeclNode:
		// Nothing to do here.
	default:
		f.err = multierr.Append(f.err, fmt.Errorf("unexpected node: %T", node))
	}
}

// writeStart writes the node across as the start of a line.
// Start nodes have their leading comments written across
// multiple lines, but their trailing comments must be written
// in-line to preserve the line structure.
//
// For example,
//
//	// Leading comment on 'message'.
//	// Spread across multiple lines.
//	message /* This is a trailing comment on 'message' */ Foo {}
//
// Newlines are preserved, so that any logical grouping of elements
// is maintained in the formatted result.
//
// For example,
//
//	// Type represents a set of different types.
//	enum Type {
//	  // Unspecified is the naming convention for default enum values.
//	  TYPE_UNSPECIFIED = 0;
//
//	  // The following elements are the real values.
//	  TYPE_ONE = 1;
//	  TYPE_TWO = 2;
//	}
//
// Start nodes are always indented according to the formatter's
// current level of indentation (e.g. nested messages, fields, etc).
//
// Note that this is one of the most complex component of the formatter - it
// controls how each node should be separated from one another and preserves
// newlines in the original source.
func (f *formatter) writeStart(node ast.Node) {
	defer f.SetPreviousNode(node)
	info := f.fileNode.NodeInfo(node)
	var (
		nodeNewlineCount               = newlineCount(info.LeadingWhitespace())
		previousNodeHasTrailingComment = f.hasTrailingComment(f.previousNode)
		previousNodeIsOpenBrace        = isOpenBrace(f.previousNode)
	)
	if length := info.LeadingComments().Len(); length > 0 {
		// If leading comments are defined, the whitespace we care about
		// is attached to the first comment.
		firstCommentNewlineCount := newlineCount(info.LeadingComments().Index(0).LeadingWhitespace())
		if !previousNodeIsOpenBrace && (!previousNodeHasTrailingComment && firstCommentNewlineCount > 1 ||
			previousNodeHasTrailingComment && firstCommentNewlineCount > 0) {
			// If the previous node is an open brace, this is the first element
			// in the body of a composite type, so we don't want to write a
			// newline, regardless of the previous node's comments. This makes
			// it so that trailing newlines are removed.
			//
			// Otherwise, if the previous node has a trailing comment, then we
			// expect to see one fewer newline characters.
			f.P()
		}
		f.writeMultilineComments(info.LeadingComments())
		var (
			lastCommentIsCStyle = cStyleComment(info.LeadingComments().Index(length - 1))
			nodeNewlineCount    = newlineCount(info.LeadingWhitespace())
		)
		if lastCommentIsCStyle && nodeNewlineCount > 1 || !lastCommentIsCStyle && nodeNewlineCount > 0 {
			// At this point, we're looking at the lines between
			// a comment and the node its attached to.
			//
			// If the last comment is a standard comment, a single newline
			// character is sufficient to warrant a separation of the
			// two.
			//
			// If the last comment is a C-style comment, multiple newline
			// characters are required because C-style comments don't consume
			// a newline.
			f.P()
		}
	} else if !previousNodeIsOpenBrace && (!previousNodeHasTrailingComment && nodeNewlineCount > 1 ||
		previousNodeHasTrailingComment && nodeNewlineCount > 0) {
		// If the previous node is an open brace, this is the first element
		// in the body of a composite type, so we don't want to write a
		// newline. This makes it so that trailing newlines are removed.
		//
		// For example,
		//
		//  message Foo {
		//
		//    string bar = 1;
		//  }
		//
		// Is formatted into the following:
		//
		//  message Foo {
		//    string bar = 1;
		//  }
		f.P()
	}
	f.Indent()
	f.writeNode(node)
	if info.TrailingComments().Len() > 0 {
		f.writeInlineComments(info.TrailingComments())
	}
}

// writeInline writes the node and its surrounding comments in-line.
//
// This is useful for writing individual nodes like keywords, runes,
// string literals, etc.
//
// For example,
//
//	// This is a leading comment on the syntax keyword.
//	syntax = /* This is a leading comment on 'proto3' */" proto3";
func (f *formatter) writeInline(node ast.Node) {
	if _, ok := node.(ast.CompositeNode); ok {
		// We only want to write comments for terminal nodes.
		// Otherwise comments accessible from CompositeNodes
		// will be written twice.
		f.writeNode(node)
		return
	}
	defer f.SetPreviousNode(node)
	info := f.fileNode.NodeInfo(node)
	if info.LeadingComments().Len() > 0 {
		f.writeInlineComments(info.LeadingComments())
		if info.LeadingWhitespace() != "" {
			f.Space()
		}
	}
	f.writeNode(node)
	if info.TrailingComments().Len() > 0 {
		f.writeInlineComments(info.TrailingComments())
	}
}

// writeBodyEnd writes the node as the end of a body.
// Leading comments are written above the token across
// multiple lines, whereas the trailing comments are
// written in-line and preserve their format.
//
// Body end nodes are always indented according to the
// formatter's current level of indentation (e.g. nested
// messages).
//
// This is useful for writing a node that concludes a
// composite node: ']', '}', '>', etc.
//
// For example,
//
//	message Foo {
//	  string bar = 1;
//	  // Leading comment on '}'.
//	} // Trailing comment on '}.
func (f *formatter) writeBodyEnd(node ast.Node) {
	if _, ok := node.(ast.CompositeNode); ok {
		// We only want to write comments for terminal nodes.
		// Otherwise comments accessible from CompositeNodes
		// will be written twice.
		f.writeNode(node)
		return
	}
	defer f.SetPreviousNode(node)
	info := f.fileNode.NodeInfo(node)
	if info.LeadingComments().Len() > 0 {
		f.P()
		f.In()
		f.writeMultilineComments(info.LeadingComments())
		f.Out()
	}
	f.Indent()
	f.writeNode(node)
	if info.TrailingComments().Len() > 0 {
		f.writeTrailingEndComments(info.TrailingComments())
	}
}

// writeBodyEndInline writes the node as the end of a body.
// Leading comments are written above the token across
// multiple lines, whereas the trailing comments are
// written in-line and adapt their comment style if they
// exist.
//
// Body end nodes are always indented according to the
// formatter's current level of indentation (e.g. nested
// messages).
//
// This is useful for writing a node that concludes either
// compact options or an array literal.
//
// This is behaviorally similar to f.writeStart, but it ignores
// the preceding newline logic because these body ends should
// always be compact.
//
// For example,
//
//	message Foo {
//	  string bar = 1 [
//	    deprecated = true
//
//	  // Leading comment on ']'.
//	  ] /* Trailing comment on ']' */ ;
//	}
func (f *formatter) writeBodyEndInline(node ast.Node) {
	if _, ok := node.(ast.CompositeNode); ok {
		// We only want to write comments for terminal nodes.
		// Otherwise comments accessible from CompositeNodes
		// will be written twice.
		f.writeNode(node)
		return
	}
	defer f.SetPreviousNode(node)
	info := f.fileNode.NodeInfo(node)
	if info.LeadingComments().Len() > 0 {
		if f.previousHasTrailingComments() {
			// line between prior node's trailing comments and upcoming leading comments
			f.P()
		}
		f.In()
		f.writeMultilineComments(info.LeadingComments())
		f.Out()
	}
	f.Indent()
	f.writeNode(node)
	if info.TrailingComments().Len() > 0 {
		f.writeInlineComments(info.TrailingComments())
	}
}

// writeLineEnd writes the node so that it ends a line.
//
// This is useful for writing individual nodes like ';' and other
// tokens that conclude the end of a single line. In this case, we
// don't want to transform the trailing comment's from '//' to C-style
// because it's not necessary.
//
// For example,
//
//	// This is a leading comment on the syntax keyword.
//	syntax = " proto3" /* This is a leading comment on the ';'; // This is a trailing comment on the ';'.
func (f *formatter) writeLineEnd(node ast.Node) {
	if _, ok := node.(ast.CompositeNode); ok {
		// We only want to write comments for terminal nodes.
		// Otherwise comments accessible from CompositeNodes
		// will be written twice.
		f.writeNode(node)
		return
	}
	defer f.SetPreviousNode(node)
	info := f.fileNode.NodeInfo(node)
	if info.LeadingComments().Len() > 0 {
		f.writeInlineComments(info.LeadingComments())
		if info.LeadingWhitespace() != "" {
			f.Space()
		}
	}
	f.writeNode(node)
	if info.TrailingComments().Len() > 0 {
		f.Space()
		f.writeTrailingEndComments(info.TrailingComments())
	}
}

// writeMultilineComments writes the given comments as a newline-delimited block.
// This is useful for both the beginning of a type (e.g. message, field, etc), as
// well as the trailing comments attached to the beginning of a body block (e.g.
// '{', '[', '<', etc).
//
// For example,
//
//	// This is a comment spread across
//	// multiple lines.
//	message Foo {}
func (f *formatter) writeMultilineComments(comments ast.Comments) {
	var previousComment ast.Comment
	for i := 0; i < comments.Len(); i++ {
		comment := comments.Index(i)
		if i > 0 {
			var (
				previousCommentIsCStyle = cStyleComment(previousComment)
				precedingNewlineCount   = newlineCount(comment.LeadingWhitespace())
			)
			if previousCommentIsCStyle && precedingNewlineCount > 1 ||
				!previousCommentIsCStyle && precedingNewlineCount > 0 {
				// Newlines between blocks of comments should be preserved.
				//
				// For example,
				//
				//  // This is a license header
				//  // spread across multiple lines.
				//
				//  // Package pet.v1 defines a PetStore API.
				//  package pet.v1;
				//
				f.P()
			}
		}
		f.P(strings.TrimSpace(comment.RawText()))
		previousComment = comments.Index(i)
	}
}

// writeInlineComments writes the given comments in-line. Standard comments are
// transformed to C-style comments so that we can safely write the comment in-line.
//
// Nearly all of these comments will already be C-style comments. The only cases we're
// preventing are when the type is defined across multiple lines.
//
// For example, given the following:
//
//	extend . google. // in-line comment
//	 protobuf .
//	  ExtensionRangeOptions {
//	   optional string label = 20000;
//	  }
//
// The formatted result is shown below:
//
//	extend .google.protobuf./* in-line comment */ExtensionRangeOptions {
//	  optional string label = 20000;
//	}
func (f *formatter) writeInlineComments(comments ast.Comments) {
	for i := 0; i < comments.Len(); i++ {
		if i > 0 || comments.Index(i).LeadingWhitespace() != "" || f.lastWritten == ';' || f.lastWritten == '}' {
			f.Space()
		}
		text := comments.Index(i).RawText()
		if strings.HasPrefix(text, "//") {
			text = strings.TrimSpace(strings.TrimPrefix(text, "//"))
			text = "/* " + text + " */"
		}
		f.WriteString(text)
	}
}

// writeTrailingEndComments writes the given comments at the end of a line and
// preserves the comment style. This is useful or writing comments attached to
// things like ';' and other tokens that conclude a type definition on a single
// line.
//
// If there is a newline between this trailing comment and the previous node, the
// comments are written immediately underneath the node on a newline.
//
// For example,
//
//	enum Type {
//	  TYPE_UNSPECIFIED = 0;
//	}
//	// This comment is attached to the '}'
//	// So is this one.
func (f *formatter) writeTrailingEndComments(comments ast.Comments) {
	for i := 0; i < comments.Len(); i++ {
		comment := comments.Index(i)
		if i == 0 && newlineCount(comment.LeadingWhitespace()) > 0 {
			// If the first comment has a newline before it, we need to preserve
			// the trailing comments in a block immediately under the previous
			// node.
			f.P()
			f.writeMultilineComments(comments)
			return
		}
		if i > 0 || comments.Index(i).LeadingWhitespace() != "" {
			f.Space()
		}
		f.WriteString(strings.TrimSpace(comments.Index(i).RawText()))
	}
}

// startsWithNewline returns true if the formatter needs to insert a newline
// before the node is printed. We need to use this in special circumstances,
// namely the file header types, when we always want to include a newline
// between the syntax, package, and import/option blocks.
func (f *formatter) startsWithNewline(info ast.NodeInfo) bool {
	nodeNewlineCount := newlineCount(info.LeadingWhitespace())
	if info.LeadingComments().Len() > 0 {
		// If leading comments are defined, the whitespace we care about
		// is attached to the first comment.
		nodeNewlineCount = newlineCount(info.LeadingComments().Index(0).LeadingWhitespace())
	}
	previousNodeHasTrailingComment := f.hasTrailingComment(f.previousNode)
	return !previousNodeHasTrailingComment && nodeNewlineCount > 1 || previousNodeHasTrailingComment && nodeNewlineCount > 0
}

// hasTrailingComment returns true if the given node has a standard
// trailing comment.
//
// For example,
//
//	message Foo {
//	  string name = 1; // Like this.
//	}
func (f *formatter) hasTrailingComment(node ast.Node) bool {
	if node == nil {
		return false
	}
	info := f.fileNode.NodeInfo(node)
	length := info.TrailingComments().Len()
	if length == 0 {
		return false
	}
	lastComment := info.TrailingComments().Index(length - 1)
	return strings.HasPrefix(lastComment.RawText(), "//")
}

// previousTrailingCommentsWroteNewline returns true if the previous node's
// trailing comments wrote a newline. We need to use this whenever we otherwise
// need to add a newline (e.g. at the end of a composite type's body, and
// for EOF comments).
func (f *formatter) previousTrailingCommentsWroteNewline() bool {
	previousTrailingComments := f.fileNode.NodeInfo(f.previousNode).TrailingComments()
	if previousTrailingComments.Len() > 0 {
		return newlineCount(previousTrailingComments.Index(0).LeadingWhitespace()) > 0
	}
	return false
}

// previousHasTrailingComments returns true if the previous node included
// trailing comments
func (f *formatter) previousHasTrailingComments() bool {
	previousTrailingComments := f.fileNode.NodeInfo(f.previousNode).TrailingComments()
	return previousTrailingComments.Len() > 0
}

// stringForOptionName returns the string representation of the given option name node.
// This is used for sorting file-level options.
func stringForOptionName(optionNameNode *ast.OptionNameNode) string {
	var result string
	for j, part := range optionNameNode.Parts {
		if j > 0 {
			// Add a dot between each of the parts.
			result += "."
		}
		result += stringForFieldReference(part)
	}
	return result
}

// stringForFieldReference returns the string representation of the given field reference.
// This is used for sorting file-level options.
func stringForFieldReference(fieldReference *ast.FieldReferenceNode) string {
	var result string
	if fieldReference.Open != nil {
		result += "("
	}
	result += string(fieldReference.Name.AsIdentifier())
	if fieldReference.Close != nil {
		result += ")"
	}
	return result
}

// isOpenBrace returns true if the given node represents one of the
// possible open brace tokens, namely '{', '[', or '<'.
func isOpenBrace(node ast.Node) bool {
	if node == nil {
		return false
	}
	runeNode, ok := node.(*ast.RuneNode)
	if !ok {
		return false
	}
	return runeNode.Rune == '{' || runeNode.Rune == '[' || runeNode.Rune == '<'
}

// newlineCount returns the number of newlines in the given value.
// This is useful for determining whether or not we should preserve
// the newline between nodes.
//
// The newlines don't need to be adjacent to each other - all of the
// tokens between them are other whitespace characters, so we can
// safely ignore them.
func newlineCount(value string) int {
	return strings.Count(value, "\n")
}

// cStyleComment returns true if the given comment is a C-Style comment,
// such that it starts with /*.
func cStyleComment(comment ast.Comment) bool {
	return strings.HasPrefix(comment.RawText(), "/*")
}
