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
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/jhump/protocompile/ast"
)

// formatter writes an *ast.FileNode as a .proto file.
type formatter struct {
	writer   io.Writer
	fileNode *ast.FileNode
	indent   int

	// Used to determine how the current node's
	// leading comments should be written.
	previousNode ast.Node
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
	return f.writeFile(f.fileNode)
}

// P prints a line to the generated output.
func (f *formatter) P(elements ...string) {
	if len(elements) > 0 {
		// Don't use an indent if we're just writing a newline.
		_, _ = fmt.Fprint(f.writer, strings.Repeat("  ", f.indent))
	}
	for _, elem := range elements {
		_, _ = fmt.Fprint(f.writer, elem)
	}
	_, _ = fmt.Fprintln(f.writer)
}

// Space adds a space to the generated output.
func (f *formatter) Space() {
	f.WriteString(" ")
}

// In increases the current level of indentation.
func (f *formatter) In() {
	f.indent++
}

// Out reduces the current level of indentation.
func (f *formatter) Out() {
	if f.indent > 0 {
		f.indent--
	}
}

// WriteString writes the given element to the generated output.
func (f *formatter) WriteString(elem string) {
	_, _ = fmt.Fprint(f.writer, elem)
}

// SetPreviousNode sets the previously written node. This should
// be called in all of the comment writing functions.
func (f *formatter) SetPreviousNode(node ast.Node) {
	f.previousNode = node
}

// writeFile writes the file node.
func (f *formatter) writeFile(fileNode *ast.FileNode) error {
	if err := f.writeFileHeader(fileNode); err != nil {
		return err
	}
	if err := f.writeFileTypes(fileNode); err != nil {
		return err
	}
	if fileNode.EOF != nil {
		info := f.fileNode.NodeInfo(fileNode.EOF)
		if info.LeadingComments().Len() > 0 {
			f.P()
			f.P()
			return f.writeMultilineComments(info.LeadingComments())
		}
	}
	f.P()
	return nil
}

// writeFileHeader writes the header of a .proto file. This includes the syntax,
// package, imports, and options (in that order). The imports and options are
// sorted. All other file elements are handled by f.writeFileTypes.
//
// For example,
//
//  syntax = "proto3";
//
//  package acme.v1.weather;
//
//  import "acme/payment/v1/payment.proto";
//  import "google/type/datetime.proto";
//
//  option cc_enable_arenas = true;
//  option optimize_for = SPEED;
//
func (f *formatter) writeFileHeader(fileNode *ast.FileNode) error {
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
		return nil
	}
	if syntaxNode := f.fileNode.Syntax; syntaxNode != nil {
		if err := f.writeSyntax(syntaxNode); err != nil {
			return err
		}
	}
	if packageNode != nil {
		if f.previousNode != nil {
			if !f.startsWithNewline(f.fileNode.NodeInfo(packageNode.Keyword)) {
				f.P()
			}
			f.P()
		}
		if err := f.writePackage(packageNode); err != nil {
			return err
		}
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
		if err := f.writeImport(importNode); err != nil {
			return err
		}
	}
	sort.Slice(optionNodes, func(i, j int) bool {
		return stringForOptionName(optionNodes[i].Name) < stringForOptionName(optionNodes[j].Name)
	})
	for i, optionNode := range optionNodes {
		if i == 0 && f.previousNode != nil {
			f.P()
			f.P()
		}
		if i > 0 {
			f.P()
		}
		if err := f.writeFileOption(optionNode); err != nil {
			return err
		}
	}
	if len(typeNodes) > 0 {
		f.P()
	}
	return nil
}

// writeFileTypes writes the types defined in a .proto file. This includes the messages, enums,
// services, etc. All other elements are ignored since they are handled by f.writeFileHeader.
func (f *formatter) writeFileTypes(fileNode *ast.FileNode) error {
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
			if err := f.writeNode(node); err != nil {
				return err
			}
			// We want to start writing newlines as soon as we've written
			// a single type.
			writeNewline = true
		}
	}
	return nil
}

// writeSyntax writes the syntax.
//
// For example,
//
//  syntax = "proto3";
//
func (f *formatter) writeSyntax(syntaxNode *ast.SyntaxNode) error {
	if err := f.writeStart(syntaxNode.Keyword); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(syntaxNode.Equals); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(syntaxNode.Syntax); err != nil {
		return err
	}
	return f.writeLineEnd(syntaxNode.Semicolon)
}

// writePackage writes the package.
//
// For example,
//
//  package acme.weather.v1;
//
func (f *formatter) writePackage(packageNode *ast.PackageNode) error {
	if err := f.writeStart(packageNode.Keyword); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(packageNode.Name); err != nil {
		return err
	}
	return f.writeLineEnd(packageNode.Semicolon)
}

// writeImport writes an import statement.
//
// For example,
//
//  import "google/protobuf/descriptor.proto";
//
func (f *formatter) writeImport(importNode *ast.ImportNode) error {
	// We don't use f.writeStart here because the imports are sorted
	// and potentially changed order.
	if err := f.writeBodyEndInline(importNode.Keyword); err != nil {
		return err
	}
	f.Space()
	// We don't want to write the "public" and "weak" nodes
	// if they aren't defined. One could be set, but never both.
	switch {
	case importNode.Public != nil:
		if err := f.writeInline(importNode.Public); err != nil {
			return err
		}
		f.Space()
	case importNode.Weak != nil:
		if err := f.writeInline(importNode.Weak); err != nil {
			return err
		}
		f.Space()
	}
	if err := f.writeInline(importNode.Name); err != nil {
		return err
	}
	return f.writeLineEnd(importNode.Semicolon)
}

// writeFileOption writes a file option. This function is slightly
// different than f.writeOption because file options are sorted at
// the top of the file, and leading comments are adjusted accordingly.
func (f *formatter) writeFileOption(optionNode *ast.OptionNode) error {
	// We don't use f.writeStart here because the options are sorted
	// and potentially changed order.
	if err := f.writeBodyEndInline(optionNode.Keyword); err != nil {
		return err
	}
	f.Space()
	if err := f.writeNode(optionNode.Name); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(optionNode.Equals); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(optionNode.Val); err != nil {
		return err
	}
	return f.writeLineEnd(optionNode.Semicolon)
}

// writeOption writes an option.
//
// For example,
//
//  option go_package = "github.com/foo/bar";
//
func (f *formatter) writeOption(optionNode *ast.OptionNode) error {
	if err := f.writeOptionPrefix(optionNode); err != nil {
		return err
	}
	f.Space()
	if optionNode.Semicolon != nil {
		if err := f.writeInline(optionNode.Val); err != nil {
			return err
		}
		return f.writeLineEnd(optionNode.Semicolon)
	}
	return f.writeInline(optionNode.Val)
}

// writeLastCompactOption writes a compact option but preserves its the
// trailing end comments. This is only used for the last compact option
// since it's the only time a trailing ',' will be omitted.
//
// For example,
//
//  [
//    deprecated = true,
//    json_name = "something" // Trailing comment on the last element.
//  ]
//
func (f *formatter) writeLastCompactOption(optionNode *ast.OptionNode) error {
	if err := f.writeOptionPrefix(optionNode); err != nil {
		return err
	}
	f.Space()
	return f.writeLineEnd(optionNode.Val)
}

// writeOptionValue writes the option prefix, which makes up all of the
// option's definition, excluding the final token(s).
//
// For example,
//
//  deprecated =
//
func (f *formatter) writeOptionPrefix(optionNode *ast.OptionNode) error {
	if optionNode.Keyword != nil {
		// Compact options don't have the keyword.
		if err := f.writeStart(optionNode.Keyword); err != nil {
			return err
		}
		f.Space()
		if err := f.writeNode(optionNode.Name); err != nil {
			return err
		}
	} else {
		if err := f.writeStart(optionNode.Name); err != nil {
			return err
		}
	}
	f.Space()
	return f.writeInline(optionNode.Equals)
}

// writeOptionName writes an option name.
//
// For example,
//
//  go_package
//  (custom.thing)
//  (custom.thing).bridge.(another.thing)
//
func (f *formatter) writeOptionName(optionNameNode *ast.OptionNameNode) error {
	for i := 0; i < len(optionNameNode.Parts); i++ {
		if i == 0 {
			// The comments will have already been written as a multiline
			// comment above the option name, so we need to handle this case
			// specially.
			fieldReferenceNode := optionNameNode.Parts[0]
			if fieldReferenceNode.Open != nil {
				if err := f.writeNode(fieldReferenceNode.Open); err != nil {
					return err
				}
				info := f.fileNode.NodeInfo(fieldReferenceNode.Open)
				if info.TrailingComments().Len() > 0 {
					if err := f.writeInlineComments(info.TrailingComments()); err != nil {
						return err
					}
				}
				if err := f.writeInline(fieldReferenceNode.Name); err != nil {
					return err
				}
			} else {
				if err := f.writeNode(fieldReferenceNode.Name); err != nil {
					return err
				}
			}
			if fieldReferenceNode.Close != nil {
				if err := f.writeInline(fieldReferenceNode.Close); err != nil {
					return err
				}
			}
			continue
		}
		// The length of this slice must be exactly len(Parts)-1.
		if err := f.writeInline(optionNameNode.Dots[i-1]); err != nil {
			return err
		}
		if err := f.writeNode(optionNameNode.Parts[i]); err != nil {
			return err
		}
	}
	return nil
}

// writeMessage writes the message node.
//
// For example,
//
//  message Foo {
//    option deprecated = true;
//    reserved 50 to 100;
//    extensions 150 to 200;
//
//    message Bar {
//      string name = 1;
//    }
//    enum Baz {
//      BAZ_UNSPECIFIED = 0;
//    }
//    extend Bar {
//      string value = 2;
//    }
//
//    Bar bar = 1;
//    Baz baz = 2;
//  }
//
func (f *formatter) writeMessage(messageNode *ast.MessageNode) error {
	var elementWriterFunc func() error
	if len(messageNode.Decls) != 0 {
		elementWriterFunc = func() error {
			for i, decl := range messageNode.Decls {
				if i > 0 {
					f.P()
				}
				if err := f.writeNode(decl); err != nil {
					return err
				}
			}
			return nil
		}
	}
	if err := f.writeStart(messageNode.Keyword); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(messageNode.Name); err != nil {
		return err
	}
	f.Space()
	return f.writeCompositeTypeBody(
		messageNode.OpenBrace,
		messageNode.CloseBrace,
		elementWriterFunc,
	)
}

// writeMessageLiteral writes a message literal.
//
// For example,
//  {
//    foo: 1
//    foo: 2
//    foo: 3
//    bar: <
//      name:"abc"
//      id:123
//    >
//  }
//
func (f *formatter) writeMessageLiteral(messageLiteralNode *ast.MessageLiteralNode) error {
	var elementWriterFunc func() error
	if len(messageLiteralNode.Elements) > 0 {
		elementWriterFunc = func() error {
			return f.writeMessageLiteralElements(messageLiteralNode)
		}
	}
	return f.writeCompositeValueBody(
		messageLiteralNode.Open,
		messageLiteralNode.Close,
		elementWriterFunc,
	)
}

// writeMessageLiteral writes a message literal suitable for
// an element in an array literal.
func (f *formatter) writeMessageLiteralForArray(
	messageLiteralNode *ast.MessageLiteralNode,
	lastElement bool,
) error {
	var elementWriterFunc func() error
	if len(messageLiteralNode.Elements) > 0 {
		elementWriterFunc = func() error {
			return f.writeMessageLiteralElements(messageLiteralNode)
		}
	}
	return f.writeBody(
		messageLiteralNode.Open,
		messageLiteralNode.Close,
		elementWriterFunc,
		f.writeOpenBracePrefixForArray,
		f.writeBodyEndInline,
	)
}

// writeMessageLiteralElements writes the message literal's elements.
//
// For example,
//
//  foo: 1
//  foo: 2
//
func (f *formatter) writeMessageLiteralElements(messageLiteralNode *ast.MessageLiteralNode) error {
	for i := 0; i < len(messageLiteralNode.Elements); i++ {
		if i > 0 {
			f.P()
		}
		if sep := messageLiteralNode.Seps[i]; sep != nil {
			if err := f.writeMessageFieldWithSeparator(messageLiteralNode.Elements[i]); err != nil {
				return err
			}
			if err := f.writeLineEnd(messageLiteralNode.Seps[i]); err != nil {
				return err
			}
			continue
		}
		if err := f.writeNode(messageLiteralNode.Elements[i]); err != nil {
			return err
		}
	}
	return nil
}

// writeMessageField writes the message field node, and concludes the
// line without leaving room for a trailing separator in the parent
// message literal.
func (f *formatter) writeMessageField(messageFieldNode *ast.MessageFieldNode) error {
	if err := f.writeMessageFieldPrefix(messageFieldNode); err != nil {
		return err
	}
	f.Space()
	return f.writeLineEnd(messageFieldNode.Val)
}

// writeMessageFieldWithSeparator writes the message field node,
// but leaves room for a trailing separator in the parent message
// literal.
func (f *formatter) writeMessageFieldWithSeparator(messageFieldNode *ast.MessageFieldNode) error {
	if err := f.writeMessageFieldPrefix(messageFieldNode); err != nil {
		return err
	}
	f.Space()
	return f.writeInline(messageFieldNode.Val)
}

// writeMessageFieldPrefix writes the message field node as a single line.
//
// For example,
//
//  foo:"bar"
//
func (f *formatter) writeMessageFieldPrefix(messageFieldNode *ast.MessageFieldNode) error {
	// The comments need to be written as a multiline comment above
	// the message field name.
	//
	// Note that this is different than how field reference nodes are
	// normally formatted in-line (i.e. as option name components).
	fieldReferenceNode := messageFieldNode.Name
	if fieldReferenceNode.Open != nil {
		if err := f.writeStart(fieldReferenceNode.Open); err != nil {
			return err
		}
		if err := f.writeInline(fieldReferenceNode.Name); err != nil {
			return err
		}
	} else {
		if err := f.writeStart(fieldReferenceNode.Name); err != nil {
			return err
		}
	}
	if fieldReferenceNode.Close != nil {
		if err := f.writeInline(fieldReferenceNode.Close); err != nil {
			return err
		}
	}
	if messageFieldNode.Sep != nil {
		if err := f.writeInline(messageFieldNode.Sep); err != nil {
			return err
		}
	}
	return nil
}

// writeEnum writes the enum node.
//
// For example,
//
//  enum Foo {
//    option deprecated = true;
//    reserved 1 to 5;
//
//    FOO_UNSPECIFIED = 0;
//  }
//
func (f *formatter) writeEnum(enumNode *ast.EnumNode) error {
	var elementWriterFunc func() error
	if len(enumNode.Decls) > 0 {
		elementWriterFunc = func() error {
			for i, decl := range enumNode.Decls {
				if i > 0 {
					f.P()
				}
				if err := f.writeNode(decl); err != nil {
					return err
				}
			}
			return nil
		}
	}
	if err := f.writeStart(enumNode.Keyword); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(enumNode.Name); err != nil {
		return err
	}
	f.Space()
	return f.writeCompositeTypeBody(
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
//  FOO_UNSPECIFIED = 1 [
//    deprecated = true
//  ];
//
func (f *formatter) writeEnumValue(enumValueNode *ast.EnumValueNode) error {
	if err := f.writeStart(enumValueNode.Name); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(enumValueNode.Equals); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(enumValueNode.Number); err != nil {
		return err
	}
	if enumValueNode.Options != nil {
		f.Space()
		if err := f.writeNode(enumValueNode.Options); err != nil {
			return err
		}
	}
	return f.writeLineEnd(enumValueNode.Semicolon)
}

// writeField writes the field node as a single line. If the field has
// compact options, it will be written across multiple lines.
//
// For example,
//
//  repeated string name = 1 [
//    deprecated = true,
//    json_name = "name"
//  ];
//
func (f *formatter) writeField(fieldNode *ast.FieldNode) error {
	// We need to handle the comments for the field label specially since
	// a label might not be defined, but it has the leading comments attached
	// to it.
	if fieldNode.Label.KeywordNode != nil {
		if err := f.writeStart(fieldNode.Label); err != nil {
			return err
		}
		f.Space()
		if err := f.writeInline(fieldNode.FldType); err != nil {
			return err
		}
	} else {
		// If a label was not written, the multiline comments will be
		// attached to the type.
		if compoundIdentNode, ok := fieldNode.FldType.(*ast.CompoundIdentNode); ok {
			if err := f.writeCompountIdentForFieldName(compoundIdentNode); err != nil {
				return err
			}
		} else {
			if err := f.writeStart(fieldNode.FldType); err != nil {
				return err
			}
		}
	}
	f.Space()
	if err := f.writeInline(fieldNode.Name); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(fieldNode.Equals); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(fieldNode.Tag); err != nil {
		return err
	}
	if fieldNode.Options != nil {
		f.Space()
		if err := f.writeNode(fieldNode.Options); err != nil {
			return err
		}
	}
	return f.writeLineEnd(fieldNode.Semicolon)
}

// writeMapField writes a map field (e.g. 'map<string, string> pairs = 1;').
func (f *formatter) writeMapField(mapFieldNode *ast.MapFieldNode) error {
	if err := f.writeNode(mapFieldNode.MapType); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(mapFieldNode.Name); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(mapFieldNode.Equals); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(mapFieldNode.Tag); err != nil {
		return err
	}
	if mapFieldNode.Options != nil {
		f.Space()
		if err := f.writeNode(mapFieldNode.Options); err != nil {
			return err
		}
	}
	return f.writeLineEnd(mapFieldNode.Semicolon)
}

// writeMapType writes a map type (e.g. 'map<string, string>').
func (f *formatter) writeMapType(mapTypeNode *ast.MapTypeNode) error {
	if err := f.writeStart(mapTypeNode.Keyword); err != nil {
		return err
	}
	if err := f.writeInline(mapTypeNode.OpenAngle); err != nil {
		return err
	}
	if err := f.writeInline(mapTypeNode.KeyType); err != nil {
		return err
	}
	if err := f.writeInline(mapTypeNode.Comma); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(mapTypeNode.ValueType); err != nil {
		return err
	}
	return f.writeInline(mapTypeNode.CloseAngle)
}

// writeFieldReference writes a field reference (e.g. '(foo.bar)').
func (f *formatter) writeFieldReference(fieldReferenceNode *ast.FieldReferenceNode) error {
	if fieldReferenceNode.Open != nil {
		if err := f.writeInline(fieldReferenceNode.Open); err != nil {
			return err
		}
	}
	if err := f.writeInline(fieldReferenceNode.Name); err != nil {
		return err
	}
	if fieldReferenceNode.Close != nil {
		return f.writeInline(fieldReferenceNode.Close)
	}
	return nil
}

// writeExtend writes the extend node.
//
// For example,
//
//  extend google.protobuf.FieldOptions {
//    bool redacted = 33333;
//  }
func (f *formatter) writeExtend(extendNode *ast.ExtendNode) error {
	var elementWriterFunc func() error
	if len(extendNode.Decls) > 0 {
		elementWriterFunc = func() error {
			for i, decl := range extendNode.Decls {
				if i > 0 {
					f.P()
				}
				if err := f.writeNode(decl); err != nil {
					return err
				}
			}
			return nil
		}
	}
	if err := f.writeStart(extendNode.Keyword); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(extendNode.Extendee); err != nil {
		return err
	}
	f.Space()
	return f.writeCompositeTypeBody(
		extendNode.OpenBrace,
		extendNode.CloseBrace,
		elementWriterFunc,
	)
}

// writeService writes the service node.
//
// For example,
//
//  service FooService {
//    option deprecated = true;
//
//    rpc Foo(FooRequest) returns (FooResponse) {};
//
func (f *formatter) writeService(serviceNode *ast.ServiceNode) error {
	var elementWriterFunc func() error
	if len(serviceNode.Decls) > 0 {
		elementWriterFunc = func() error {
			for i, decl := range serviceNode.Decls {
				if i > 0 {
					f.P()
				}
				if err := f.writeNode(decl); err != nil {
					return err
				}
			}
			return nil
		}
	}
	if err := f.writeStart(serviceNode.Keyword); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(serviceNode.Name); err != nil {
		return err
	}
	f.Space()
	return f.writeCompositeTypeBody(
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
//  rpc Foo(FooRequest) returns (FooResponse) {
//    option deprecated = true;
//  };
//
func (f *formatter) writeRPC(rpcNode *ast.RPCNode) error {
	var elementWriterFunc func() error
	if len(rpcNode.Decls) > 0 {
		elementWriterFunc = func() error {
			for i, decl := range rpcNode.Decls {
				if i > 0 {
					f.P()
				}
				if err := f.writeNode(decl); err != nil {
					return err
				}
			}
			return nil
		}
	}
	if err := f.writeStart(rpcNode.Keyword); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(rpcNode.Name); err != nil {
		return err
	}
	if err := f.writeInline(rpcNode.Input); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(rpcNode.Returns); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(rpcNode.Output); err != nil {
		return err
	}
	if rpcNode.OpenBrace == nil {
		// This RPC doesn't have any elements, so we prefer the
		// ';' form.
		//
		//  rpc Ping(PingRequest) returns (PingResponse);
		//
		return f.writeInline(rpcNode.Semicolon)
	}
	f.Space()
	return f.writeCompositeTypeBody(
		rpcNode.OpenBrace,
		rpcNode.CloseBrace,
		elementWriterFunc,
	)
}

// writeRPCType writes the RPC type node (e.g. (stream foo.Bar)).
func (f *formatter) writeRPCType(rpcTypeNode *ast.RPCTypeNode) error {
	if err := f.writeInline(rpcTypeNode.OpenParen); err != nil {
		return err
	}
	if rpcTypeNode.Stream != nil {
		if err := f.writeInline(rpcTypeNode.Stream); err != nil {
			return err
		}
		f.Space()
	}
	if err := f.writeInline(rpcTypeNode.MessageType); err != nil {
		return err
	}
	return f.writeInline(rpcTypeNode.CloseParen)
}

// writeOneOf writes the oneof node.
//
// For example,
//
//  oneof foo {
//    option deprecated = true;
//
//    string name = 1;
//    int number = 2;
//  }
//
func (f *formatter) writeOneOf(oneOfNode *ast.OneOfNode) error {
	var elementWriterFunc func() error
	if len(oneOfNode.Decls) > 0 {
		elementWriterFunc = func() error {
			for i, decl := range oneOfNode.Decls {
				if i > 0 {
					f.P()
				}
				if err := f.writeNode(decl); err != nil {
					return err
				}
			}
			return nil
		}
	}
	if err := f.writeStart(oneOfNode.Keyword); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(oneOfNode.Name); err != nil {
		return err
	}
	f.Space()
	return f.writeCompositeTypeBody(
		oneOfNode.OpenBrace,
		oneOfNode.CloseBrace,
		elementWriterFunc,
	)
}

// writeGroup writes the group node.
//
// For example,
//
//  optional group Key = 4 [
//    deprecated = true,
//    json_name = "key"
//  ] {
//    optional uint64 id = 1;
//    optional string name = 2;
//  }
//
func (f *formatter) writeGroup(groupNode *ast.GroupNode) error {
	var elementWriterFunc func() error
	if len(groupNode.Decls) > 0 {
		elementWriterFunc = func() error {
			for i, decl := range groupNode.Decls {
				if i > 0 {
					f.P()
				}
				if err := f.writeNode(decl); err != nil {
					return err
				}
			}
			return nil
		}
	}
	// We need to handle the comments for the group label specially since
	// a label might not be defined, but it has the leading comments attached
	// to it.
	if groupNode.Label.KeywordNode != nil {
		if err := f.writeStart(groupNode.Label); err != nil {
			return err
		}
		f.Space()
		if err := f.writeInline(groupNode.Keyword); err != nil {
			return err
		}
	} else {
		// If a label was not written, the multiline comments will be
		// attached to the keyword.
		if err := f.writeStart(groupNode.Keyword); err != nil {
			return err
		}
	}
	f.Space()
	if err := f.writeInline(groupNode.Name); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(groupNode.Equals); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(groupNode.Tag); err != nil {
		return err
	}
	if groupNode.Options != nil {
		f.Space()
		if err := f.writeNode(groupNode.Options); err != nil {
			return err
		}
	}
	f.Space()
	return f.writeCompositeTypeBody(
		groupNode.OpenBrace,
		groupNode.CloseBrace,
		elementWriterFunc,
	)
}

// writeExtensionRange writes the extension range node.
//
// For example,
//
//  extensions 5-10, 100 to max [
//    deprecated = true
//  ];
//
func (f *formatter) writeExtensionRange(extensionRangeNode *ast.ExtensionRangeNode) error {
	if err := f.writeStart(extensionRangeNode.Keyword); err != nil {
		return err
	}
	f.Space()
	for i := 0; i < len(extensionRangeNode.Ranges); i++ {
		if i > 0 {
			// The length of this slice must be exactly len(Ranges)-1.
			if err := f.writeInline(extensionRangeNode.Commas[i-1]); err != nil {
				return err
			}
			f.Space()
		}
		if err := f.writeNode(extensionRangeNode.Ranges[i]); err != nil {
			return err
		}
	}
	if extensionRangeNode.Options != nil {
		f.Space()
		if err := f.writeNode(extensionRangeNode.Options); err != nil {
			return err
		}
	}
	return f.writeLineEnd(extensionRangeNode.Semicolon)
}

// writeReserved writes a reserved node.
//
// For example,
//
//  reserved 5-10, 100 to max;
//
func (f *formatter) writeReserved(reservedNode *ast.ReservedNode) error {
	if err := f.writeStart(reservedNode.Keyword); err != nil {
		return err
	}
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
			if err := f.writeInline(reservedNode.Commas[i-1]); err != nil {
				return err
			}
			f.Space()
		}
		if err := f.writeInline(elements[i]); err != nil {
			return err
		}
	}
	return f.writeLineEnd(reservedNode.Semicolon)
}

// writeRange writes the given range node (e.g. '1 to max').
func (f *formatter) writeRange(rangeNode *ast.RangeNode) error {
	if err := f.writeInline(rangeNode.StartVal); err != nil {
		return err
	}
	if rangeNode.To != nil {
		f.Space()
		if err := f.writeInline(rangeNode.To); err != nil {
			return err
		}
	}
	// Either EndVal or Max will be set, but never both.
	switch {
	case rangeNode.EndVal != nil:
		f.Space()
		if err := f.writeInline(rangeNode.EndVal); err != nil {
			return err
		}
	case rangeNode.Max != nil:
		f.Space()
		if err := f.writeInline(rangeNode.Max); err != nil {
			return err
		}
	}
	return nil
}

// writeCompactOptions writes a compact options node.
//
// For example,
//
//  [
//    deprecated = true,
//    json_name = "something"
//  ]
//
func (f *formatter) writeCompactOptions(compactOptionsNode *ast.CompactOptionsNode) error {
	var elementWriterFunc func() error
	if len(compactOptionsNode.Options) > 0 {
		elementWriterFunc = func() error {
			for i := 0; i < len(compactOptionsNode.Options); i++ {
				if i > 0 {
					f.P()
				}
				if i == len(compactOptionsNode.Options)-1 {
					// The last element won't have a trailing comma.
					return f.writeLastCompactOption(compactOptionsNode.Options[i])
				}
				if err := f.writeNode(compactOptionsNode.Options[i]); err != nil {
					return err
				}
				// The length of this slice must be exactly len(Options)-1.
				if err := f.writeLineEnd(compactOptionsNode.Commas[i]); err != nil {
					return err
				}
			}
			return nil
		}
	}
	return f.writeCompositeValueBody(
		compactOptionsNode.OpenBracket,
		compactOptionsNode.CloseBracket,
		elementWriterFunc,
	)
}

// writeArrayLiteral writes an array literal across multiple lines.
//
// For example,
//
//  [
//    "foo",
//    "bar"
//  ]
//
func (f *formatter) writeArrayLiteral(arrayLiteralNode *ast.ArrayLiteralNode) error {
	var elementWriterFunc func() error
	if len(arrayLiteralNode.Elements) > 0 {
		elementWriterFunc = func() error {
			for i := 0; i < len(arrayLiteralNode.Elements); i++ {
				if i > 0 {
					f.P()
				}
				lastElement := i == len(arrayLiteralNode.Elements)-1
				if compositeNode, ok := arrayLiteralNode.Elements[i].(ast.CompositeNode); ok {
					if err := f.writeCompositeValueForArrayLiteral(compositeNode, lastElement); err != nil {
						return err
					}
					if !lastElement {
						if err := f.writeLineEnd(arrayLiteralNode.Commas[i]); err != nil {
							return err
						}
					}
					continue
				}
				if lastElement {
					// The last element won't have a trailing comma.
					return f.writeBodyEnd(arrayLiteralNode.Elements[i])
				}
				if err := f.writeStart(arrayLiteralNode.Elements[i]); err != nil {
					return err
				}
				// The length of this slice must be exactly len(Elements)-1.
				if err := f.writeLineEnd(arrayLiteralNode.Commas[i]); err != nil {
					return err
				}
			}
			return nil
		}
	}
	return f.writeCompositeValueBody(
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
//  option (value) = /* In-line comment for '-42' */ -42;
//
//  option (thing) = {
//    values: [
//      // Leading comment on -42.
//      -42, // Trailing comment on -42.
//    ]
//  }
//
// The lastElement boolean is used to signal whether or not the composite value
// should be written as the last element (i.e. it doesn't have a trailing comma).
func (f *formatter) writeCompositeValueForArrayLiteral(
	compositeNode ast.CompositeNode,
	lastElement bool,
) error {
	switch node := compositeNode.(type) {
	case *ast.CompoundStringLiteralNode:
		return f.writeCompoundStringLiteralForArray(node, lastElement)
	case *ast.PositiveUintLiteralNode:
		return f.writePositiveUintLiteralForArray(node, lastElement)
	case *ast.NegativeIntLiteralNode:
		return f.writeNegativeIntLiteralForArray(node, lastElement)
	case *ast.SignedFloatLiteralNode:
		return f.writeSignedFloatLiteralForArray(node, lastElement)
	case *ast.MessageLiteralNode:
		return f.writeMessageLiteralForArray(node, lastElement)
	default:
		return fmt.Errorf("unexpected array value node %T", node)
	}
}

// writeCompositeTypeBody writes the body of a composite type, e.g. message, enum, extend, oneof, etc.
func (f *formatter) writeCompositeTypeBody(
	openBrace *ast.RuneNode,
	closeBrace *ast.RuneNode,
	elementWriterFunc func() error,
) error {
	return f.writeBody(
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
	elementWriterFunc func() error,
) error {
	return f.writeBody(
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
	elementWriterFunc func() error,
	openBraceWriterFunc func(ast.Node) error,
	closeBraceWriterFunc func(ast.Node) error,
) error {
	if err := openBraceWriterFunc(openBrace); err != nil {
		return err
	}
	info := f.fileNode.NodeInfo(openBrace)
	if info.TrailingComments().Len() == 0 && elementWriterFunc == nil {
		info = f.fileNode.NodeInfo(closeBrace)
		if info.LeadingComments().Len() == 0 {
			// This is an empty definition without any comments in
			// the body, so we return early.
			return f.writeLineEnd(closeBrace)
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
		if err := f.writeMultilineComments(info.LeadingComments()); err != nil {
			return err
		}
		f.Out()
		// The '}' should be indented on its own line but we've already handled
		// the comments, so we manually apply it here.
		_, _ = fmt.Fprint(f.writer, strings.Repeat("  ", f.indent))
		if err := f.writeNode(closeBrace); err != nil {
			return err
		}
		return f.writeTrailingEndComments(info.TrailingComments())
	}
	f.P()
	f.In()
	if info.TrailingComments().Len() > 0 {
		if err := f.writeMultilineComments(info.TrailingComments()); err != nil {
			return err
		}
		if elementWriterFunc != nil {
			// If there are other elements to write, we need another
			// newline after the trailing comments.
			f.P()
		}
	}
	if elementWriterFunc != nil {
		if err := elementWriterFunc(); err != nil {
			return err
		}
		f.P()
	}
	f.Out()
	if info := f.fileNode.NodeInfo(closeBrace); info.LeadingComments().Len() > 0 {
		f.P()
	}
	return closeBraceWriterFunc(closeBrace)
}

// writeOpenBracePrefix writes the open brace with its leading comments in-line.
// This is used for nearly every use case of f.writeBody, excluding the instances
// in array literals.
func (f *formatter) writeOpenBracePrefix(openBrace ast.Node) error {
	info := f.fileNode.NodeInfo(openBrace)
	if info.LeadingComments().Len() > 0 {
		if err := f.writeInlineComments(info.LeadingComments()); err != nil {
			return err
		}
		f.Space()
	}
	return f.writeNode(openBrace)
}

// writeOpenBracePrefixForArray writes the open brace with its leading comments
// on multiple lines. This is only used for message literals in arrays.
func (f *formatter) writeOpenBracePrefixForArray(openBrace ast.Node) error {
	info := f.fileNode.NodeInfo(openBrace)
	if info.LeadingComments().Len() > 0 {
		if err := f.writeMultilineComments(info.LeadingComments()); err != nil {
			return err
		}
	}
	_, _ = fmt.Fprint(f.writer, strings.Repeat("  ", f.indent))
	return f.writeNode(openBrace)
}

// writeCompoundIdent writes a compound identifier (e.g. '.com.foo.Bar').
func (f *formatter) writeCompoundIdent(compoundIdentNode *ast.CompoundIdentNode) error {
	if compoundIdentNode.LeadingDot != nil {
		if err := f.writeInline(compoundIdentNode.LeadingDot); err != nil {
			return err
		}
	}
	for i := 0; i < len(compoundIdentNode.Components); i++ {
		if i > 0 {
			// The length of this slice must be exactly len(Components)-1.
			if err := f.writeInline(compoundIdentNode.Dots[i-1]); err != nil {
				return err
			}
		}
		if err := f.writeInline(compoundIdentNode.Components[i]); err != nil {
			return err
		}
	}
	return nil
}

// writeCompountIdentForFieldName writes a compound identifier, but handles comments
// specially for field names.
//
// For example,
//
//  message Foo {
//    // These are comments attached to bar.
//    bar.v1.Bar bar = 1;
//  }
//
func (f *formatter) writeCompountIdentForFieldName(compoundIdentNode *ast.CompoundIdentNode) error {
	if compoundIdentNode.LeadingDot != nil {
		if err := f.writeStart(compoundIdentNode.LeadingDot); err != nil {
			return err
		}
	}
	for i := 0; i < len(compoundIdentNode.Components); i++ {
		if i == 0 && compoundIdentNode.LeadingDot == nil {
			if err := f.writeStart(compoundIdentNode.Components[i]); err != nil {
				return err
			}
			continue
		}
		if i > 0 {
			// The length of this slice must be exactly len(Components)-1.
			if err := f.writeInline(compoundIdentNode.Dots[i-1]); err != nil {
				return err
			}
		}
		if err := f.writeInline(compoundIdentNode.Components[i]); err != nil {
			return err
		}
	}
	return nil
}

// writeFieldLabel writes the field label node.
//
// For example,
//
//  optional
//  repeated
//  required
//
func (f *formatter) writeFieldLabel(fieldLabel ast.FieldLabel) error {
	f.WriteString(fieldLabel.Val)
	return nil
}

// writeBoolLiteral writes a bool literal (e.g. 'true').
func (f *formatter) writeBoolLiteral(boolLiteralNode *ast.BoolLiteralNode) error {
	f.WriteString(strconv.FormatBool(boolLiteralNode.Val))
	return nil
}

// writeCompoundStringLiteral writes a compound string literal value (e.g. "one " "  string").
func (f *formatter) writeCompoundStringLiteral(compoundStringLiteralNode *ast.CompoundStringLiteralNode) error {
	for i, child := range compoundStringLiteralNode.Children() {
		if i > 0 {
			f.Space()
		}
		if err := f.writeInline(child); err != nil {
			return err
		}
	}
	return nil
}

// writeCompoundStringLiteralForArray writes a compound string literal value,
// but writes its comments suitable for an element in an array literal.
//
// The lastElement boolean is used to signal whether or not the value should
// be written as the last element (i.e. it doesn't have a trailing comma).
func (f *formatter) writeCompoundStringLiteralForArray(
	compoundStringLiteralNode *ast.CompoundStringLiteralNode,
	lastElement bool,
) error {
	for i, child := range compoundStringLiteralNode.Children() {
		if i > 0 {
			f.Space()
		}
		if i == 0 {
			if err := f.writeStart(child); err != nil {
				return err
			}
			continue
		}
		if lastElement && i == len(compoundStringLiteralNode.Children())-1 {
			return f.writeLineEnd(child)
		}
		if err := f.writeInline(child); err != nil {
			return err
		}
	}
	return nil
}

// writeFloatLiteral writes a float literal value (e.g. '42.2').
func (f *formatter) writeFloatLiteral(floatLiteralNode *ast.FloatLiteralNode) error {
	f.WriteString(strconv.FormatFloat(floatLiteralNode.Val, 'g', -1, 64))
	return nil
}

// writeSignedFloatLiteral writes a signed float literal value (e.g. '-42.2').
func (f *formatter) writeSignedFloatLiteral(signedFloatLiteralNode *ast.SignedFloatLiteralNode) error {
	if err := f.writeInline(signedFloatLiteralNode.Sign); err != nil {
		return err
	}
	return f.writeLineEnd(signedFloatLiteralNode.Float)
}

// writeSignedFloatLiteralForArray writes a signed float literal value, but writes
// its comments suitable for an element in an array literal.
//
// The lastElement boolean is used to signal whether or not the value should
// be written as the last element (i.e. it doesn't have a trailing comma).
func (f *formatter) writeSignedFloatLiteralForArray(
	signedFloatLiteralNode *ast.SignedFloatLiteralNode,
	lastElement bool,
) error {
	if err := f.writeStart(signedFloatLiteralNode.Sign); err != nil {
		return err
	}
	if lastElement {
		return f.writeLineEnd(signedFloatLiteralNode.Float)
	}
	return f.writeInline(signedFloatLiteralNode.Float)
}

// writeSpecialFloatLiteral writes a special float literal value (e.g. "nan" or "inf").
func (f *formatter) writeSpecialFloatLiteral(specialFloatLiteralNode *ast.SpecialFloatLiteralNode) error {
	f.WriteString(specialFloatLiteralNode.KeywordNode.Val)
	return nil
}

// writeStringLiteral writes a string literal value (e.g. "foo").
func (f *formatter) writeStringLiteral(stringLiteralNode *ast.StringLiteralNode) error {
	f.WriteString(fmt.Sprintf("%q", stringLiteralNode.Val))
	return nil
}

// writeStringValue writes a string value (e.g. "foo").
func (f *formatter) writeStringValue(stringValueNode ast.StringValueNode) error {
	f.WriteString(fmt.Sprintf("%q", stringValueNode.AsString()))
	return nil
}

// writeUintLiteral writes a uint literal (e.g. '42').
func (f *formatter) writeUintLiteral(uintLiteralNode *ast.UintLiteralNode) error {
	f.WriteString(strconv.FormatUint(uintLiteralNode.Val, 10))
	return nil
}

// writeNegativeIntLiteral writes a negative int literal (e.g. '-42').
func (f *formatter) writeNegativeIntLiteral(negativeIntLiteralNode *ast.NegativeIntLiteralNode) error {
	if err := f.writeInline(negativeIntLiteralNode.Minus); err != nil {
		return err
	}
	return f.writeLineEnd(negativeIntLiteralNode.Uint)
}

// writeNegativeIntLiteralForArray writes a negative int literal value, but writes
// its comments suitable for an element in an array literal.
//
// The lastElement boolean is used to signal whether or not the value should
// be written as the last element (i.e. it doesn't have a trailing comma).
func (f *formatter) writeNegativeIntLiteralForArray(
	negativeIntLiteralNode *ast.NegativeIntLiteralNode,
	lastElement bool,
) error {
	if err := f.writeStart(negativeIntLiteralNode.Minus); err != nil {
		return err
	}
	if lastElement {
		return f.writeLineEnd(negativeIntLiteralNode.Uint)
	}
	return f.writeInline(negativeIntLiteralNode.Uint)
}

// writePositiveUintLiteral writes a positive uint literal (e.g. '+42').
func (f *formatter) writePositiveUintLiteral(positiveIntLiteralNode *ast.PositiveUintLiteralNode) error {
	if err := f.writeInline(positiveIntLiteralNode.Plus); err != nil {
		return err
	}
	return f.writeLineEnd(positiveIntLiteralNode.Uint)
}

// writePositiveUintLiteralForArray writes a positive uint literal value, but writes
// its comments suitable for an element in an array literal.
//
// The lastElement boolean is used to signal whether or not the value should
// be written as the last element (i.e. it doesn't have a trailing comma).
func (f *formatter) writePositiveUintLiteralForArray(
	positiveIntLiteralNode *ast.PositiveUintLiteralNode,
	lastElement bool,
) error {
	if err := f.writeStart(positiveIntLiteralNode.Plus); err != nil {
		return err
	}
	if lastElement {
		return f.writeLineEnd(positiveIntLiteralNode.Uint)
	}
	return f.writeInline(positiveIntLiteralNode.Uint)
}

// writeIdent writes an identifier (e.g. 'foo').
func (f *formatter) writeIdent(identNode *ast.IdentNode) error {
	f.WriteString(identNode.Val)
	return nil
}

// writeKeyword writes a keyword (e.g. 'syntax').
func (f *formatter) writeKeyword(keywordNode *ast.KeywordNode) error {
	f.WriteString(keywordNode.Val)
	return nil
}

// writeRune writes a rune (e.g. '=').
func (f *formatter) writeRune(runeNode *ast.RuneNode) error {
	f.WriteString(string(runeNode.Rune))
	return nil
}

// writeNode writes the node by dispatching to a function tailored to its concrete type.
//
// Comments are handled in each respective write function so that it can determine whether
// to write the comments in-line or not.
func (f *formatter) writeNode(node ast.Node) error {
	switch element := node.(type) {
	case *ast.ArrayLiteralNode:
		return f.writeArrayLiteral(element)
	case *ast.BoolLiteralNode:
		return f.writeBoolLiteral(element)
	case *ast.CompactOptionsNode:
		return f.writeCompactOptions(element)
	case *ast.CompoundIdentNode:
		return f.writeCompoundIdent(element)
	case *ast.CompoundStringLiteralNode:
		return f.writeCompoundStringLiteral(element)
	case *ast.EnumNode:
		return f.writeEnum(element)
	case *ast.EnumValueNode:
		return f.writeEnumValue(element)
	case *ast.ExtendNode:
		return f.writeExtend(element)
	case *ast.ExtensionRangeNode:
		return f.writeExtensionRange(element)
	case ast.FieldLabel:
		return f.writeFieldLabel(element)
	case *ast.FieldNode:
		return f.writeField(element)
	case *ast.FieldReferenceNode:
		return f.writeFieldReference(element)
	case *ast.FloatLiteralNode:
		return f.writeFloatLiteral(element)
	case *ast.GroupNode:
		return f.writeGroup(element)
	case *ast.IdentNode:
		return f.writeIdent(element)
	case *ast.ImportNode:
		return f.writeImport(element)
	case *ast.KeywordNode:
		return f.writeKeyword(element)
	case *ast.MapFieldNode:
		return f.writeMapField(element)
	case *ast.MapTypeNode:
		return f.writeMapType(element)
	case *ast.MessageNode:
		return f.writeMessage(element)
	case *ast.MessageFieldNode:
		return f.writeMessageField(element)
	case *ast.MessageLiteralNode:
		return f.writeMessageLiteral(element)
	case *ast.NegativeIntLiteralNode:
		return f.writeNegativeIntLiteral(element)
	case *ast.OneOfNode:
		return f.writeOneOf(element)
	case *ast.OptionNode:
		return f.writeOption(element)
	case *ast.OptionNameNode:
		return f.writeOptionName(element)
	case *ast.PackageNode:
		return f.writePackage(element)
	case *ast.PositiveUintLiteralNode:
		return f.writePositiveUintLiteral(element)
	case *ast.RangeNode:
		return f.writeRange(element)
	case *ast.ReservedNode:
		return f.writeReserved(element)
	case *ast.RPCNode:
		return f.writeRPC(element)
	case *ast.RPCTypeNode:
		return f.writeRPCType(element)
	case *ast.RuneNode:
		return f.writeRune(element)
	case *ast.ServiceNode:
		return f.writeService(element)
	case *ast.SignedFloatLiteralNode:
		return f.writeSignedFloatLiteral(element)
	case *ast.SpecialFloatLiteralNode:
		return f.writeSpecialFloatLiteral(element)
	case *ast.StringLiteralNode:
		return f.writeStringLiteral(element)
	case ast.StringValueNode:
		return f.writeStringValue(element)
	case *ast.SyntaxNode:
		return f.writeSyntax(element)
	case *ast.UintLiteralNode:
		return f.writeUintLiteral(element)
	case *ast.EmptyDeclNode:
		// Nothing to do here.
		return nil
	}
	return fmt.Errorf("unexpected node: %T", node)
}

// writeStart writes the node across as the start of a line.
// Start nodes have their leading comments written across
// multiple lines, but their trailing comments must be written
// in-line to preserve the line structure.
//
// For example,
//
//  // Leading comment on 'message'.
//  // Spread across multiple lines.
//  message /* This is a trailing comment on 'message' */ Foo {}
//
// Newlines are preserved, so that any logical grouping of elements
// is maintained in the formatted result.
//
// For example,
//
//  // Type represents a set of different types.
//  enum Type {
//    // Unspecified is the naming convention for default enum values.
//    TYPE_UNSPECIFIED = 0;
//
//    // The following elements are the real values.
//    TYPE_ONE = 1;
//    TYPE_TWO = 2;
//  }
//
// Start nodes are always indented according to the formatter's
// current level of indentation (e.g. nested messages, fields, etc).
//
// Note that this is one of the most complex component of the formatter - it
// controls how each node should be separated from one another and preserves
// newlines in the original source.
func (f *formatter) writeStart(node ast.Node) error {
	defer f.SetPreviousNode(node)
	info := f.fileNode.NodeInfo(node)
	var (
		nodeNewlineCount               = newlineCount(info.LeadingWhitespace())
		previousNodeHasTrailingComment = f.hasTrailingComment(f.previousNode)
	)
	if length := info.LeadingComments().Len(); length > 0 {
		firstCommentNewlineCount := newlineCount(info.LeadingComments().Index(0).LeadingWhitespace())
		if !previousNodeHasTrailingComment && firstCommentNewlineCount > 1 || previousNodeHasTrailingComment && firstCommentNewlineCount > 0 {
			// If leading comments are defined, the whitespace we care about
			// is attached to the first comment.
			//
			// If the previous node has a trailing comment, then we expect
			// to see one fewer newline characters.
			f.P()
		}
		if err := f.writeMultilineComments(info.LeadingComments()); err != nil {
			return err
		}
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
			// characters are required because C-style comments can
			// be written in-line.
			f.P()
		}
	} else if !previousNodeHasTrailingComment && nodeNewlineCount > 1 || previousNodeHasTrailingComment && nodeNewlineCount > 0 {
		// If leading comments are not attached to this node, we still
		// want to check whether or not there are any newlines before it.
		f.P()
	}
	_, _ = fmt.Fprint(f.writer, strings.Repeat("  ", f.indent))
	if err := f.writeNode(node); err != nil {
		return err
	}
	if info.TrailingComments().Len() > 0 {
		if err := f.writeInlineComments(info.TrailingComments()); err != nil {
			return err
		}
	}
	return nil
}

// writeInline writes the node and its surrounding comments in-line.
//
// This is useful for writing individual nodes like keywords, runes,
// string literals, etc.
//
// For example,
//
//  // This is a leading comment on the syntax keyword.
//  syntax = /* This is a leading comment on 'proto3' */" proto3";
//
func (f *formatter) writeInline(node ast.Node) error {
	if _, ok := node.(ast.CompositeNode); ok {
		// We only want to write comments for terminal nodes.
		// Otherwise comments accessible from CompositeNodes
		// will be written twice.
		return f.writeNode(node)
	}
	defer f.SetPreviousNode(node)
	info := f.fileNode.NodeInfo(node)
	if info.LeadingComments().Len() > 0 {
		if err := f.writeInlineComments(info.LeadingComments()); err != nil {
			return err
		}
	}
	if err := f.writeNode(node); err != nil {
		return err
	}
	if info.TrailingComments().Len() > 0 {
		if err := f.writeInlineComments(info.TrailingComments()); err != nil {
			return err
		}
	}
	return nil
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
//  message Foo {
//    string bar = 1;
//
//  // Leading comment on '}'.
//  } // Trailing comment on '}.
//
func (f *formatter) writeBodyEnd(node ast.Node) error {
	if _, ok := node.(ast.CompositeNode); ok {
		// We only want to write comments for terminal nodes.
		// Otherwise comments accessible from CompositeNodes
		// will be written twice.
		return f.writeNode(node)
	}
	defer f.SetPreviousNode(node)
	info := f.fileNode.NodeInfo(node)
	if info.LeadingComments().Len() > 0 {
		if err := f.writeMultilineComments(info.LeadingComments()); err != nil {
			return err
		}
	}
	_, _ = fmt.Fprint(f.writer, strings.Repeat("  ", f.indent))
	if err := f.writeNode(node); err != nil {
		return err
	}
	if info.TrailingComments().Len() > 0 {
		f.Space()
		if err := f.writeTrailingEndComments(info.TrailingComments()); err != nil {
			return err
		}
	}
	return nil
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
//  message Foo {
//    string bar = 1 [
//      deprecated = true
//
//    // Leading comment on ']'.
//    ] /* Trailing comment on ']' */ ;
//  }
//
func (f *formatter) writeBodyEndInline(node ast.Node) error {
	if _, ok := node.(ast.CompositeNode); ok {
		// We only want to write comments for terminal nodes.
		// Otherwise comments accessible from CompositeNodes
		// will be written twice.
		return f.writeNode(node)
	}
	defer f.SetPreviousNode(node)
	info := f.fileNode.NodeInfo(node)
	if info.LeadingComments().Len() > 0 {
		if err := f.writeMultilineComments(info.LeadingComments()); err != nil {
			return err
		}
	}
	_, _ = fmt.Fprint(f.writer, strings.Repeat("  ", f.indent))
	if err := f.writeNode(node); err != nil {
		return err
	}
	if info.TrailingComments().Len() > 0 {
		f.Space()
		if err := f.writeInlineComments(info.TrailingComments()); err != nil {
			return err
		}
	}
	return nil
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
//  // This is a leading comment on the syntax keyword.
//  syntax = " proto3" /* This is a leading comment on the ';'; // This is a trailing comment on the ';'.
//
func (f *formatter) writeLineEnd(node ast.Node) error {
	if _, ok := node.(ast.CompositeNode); ok {
		// We only want to write comments for terminal nodes.
		// Otherwise comments accessible from CompositeNodes
		// will be written twice.
		return f.writeNode(node)
	}
	defer f.SetPreviousNode(node)
	info := f.fileNode.NodeInfo(node)
	if info.LeadingComments().Len() > 0 {
		if err := f.writeInlineComments(info.LeadingComments()); err != nil {
			return err
		}
	}
	if err := f.writeNode(node); err != nil {
		return err
	}
	if info.TrailingComments().Len() > 0 {
		f.Space()
		if err := f.writeTrailingEndComments(info.TrailingComments()); err != nil {
			return err
		}
	}
	return nil
}

// writeMultilineComments writes the given comments as a newline-delimited block.
// This is useful for both the beginning of a type (e.g. message, field, etc), as
// well as the trailing comments attached to the beginning of a body block (e.g.
// '{', '[', '<', etc).
//
// For example,
//
//  // This is a comment spread across
//  // multiple lines.
//  message Foo {}
//
func (f *formatter) writeMultilineComments(comments ast.Comments) error {
	for i := 0; i < comments.Len(); i++ {
		f.P(strings.TrimSpace(comments.Index(i).RawText()))
	}
	return nil
}

// writeInlineComments writes the given comments in-line. Standard comments are
// transformed to C-style comments so that we can safely write the comment in-line.
//
// Nearly all of these comments will already be C-style comments. The only cases we're
// preventing are when the type is defined across multiple lines.
//
// For example, given the following:
//
//  extend . google. // in-line comment
//   protobuf .
//    ExtensionRangeOptions {
//     optional string label = 20000;
//    }
//
// The formatted result is shown below:
//
//  extend .google.protobuf./* in-line comment */ExtensionRangeOptions {
//    optional string label = 20000;
//  }
//
func (f *formatter) writeInlineComments(comments ast.Comments) error {
	for i := 0; i < comments.Len(); i++ {
		if i > 0 {
			f.Space()
		}
		text := comments.Index(i).RawText()
		if strings.HasPrefix(text, "//") {
			text = strings.TrimSpace(strings.TrimPrefix(text, "//"))
			text = "/* " + text + " */"
		}
		f.WriteString(text)
	}
	return nil
}

// writeTrailingEndComments writes the given comments at the end of a line and
// preserves the comment style. This is useful or writing comments attached to
// things like ';' and other tokens that conclude a type definition on a single
// line.
func (f *formatter) writeTrailingEndComments(comments ast.Comments) error {
	for i := 0; i < comments.Len(); i++ {
		if i > 0 {
			f.Space()
		}
		f.WriteString(strings.TrimSpace(comments.Index(i).RawText()))
	}
	return nil
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
//  message Foo {
//    string name = 1; // Like this.
//  }
//
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
