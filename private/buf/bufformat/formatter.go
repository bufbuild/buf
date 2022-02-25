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

// writeFile writes the file node.
func (f *formatter) writeFile(fileNode *ast.FileNode) error {
	if err := f.writeFileHeader(fileNode); err != nil {
		return err
	}
	if err := f.writeFileTypes(fileNode); err != nil {
		return err
	}
	return nil
}

// writeFileHeader writes the header of a .proto file. This includes the syntax,
// package, imports, and options (in that order). The imports and options are sorted.
// All other file elements are handled by f.writeFileTypes.
//
// For example,
//
//  syntax = "proto3";
//
//  package acme.v1.weather;
//
//  import "google/type/datetime.proto";
//  import "acme/payment/v1/payment.proto";
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
		default:
			typeNodes = append(typeNodes, node)
		}
	}
	if f.fileNode.Syntax == nil && packageNode == nil && importNodes == nil && optionNodes == nil {
		// There aren't any header values, so we can return early.
		return nil
	}
	if syntaxNode := f.fileNode.Syntax; syntaxNode != nil {
		if err := f.writeNode(syntaxNode); err != nil {
			return err
		}
		f.P()
	}
	if packageNode != nil {
		if f.fileNode.Syntax != nil {
			f.P()
		}
		if err := f.writeNode(packageNode); err != nil {
			return err
		}
		f.P()
	}
	if len(importNodes) > 0 && (f.fileNode.Syntax != nil || packageNode != nil) {
		f.P()
	}
	sort.Slice(importNodes, func(i, j int) bool {
		return importNodes[i].Name.AsString() < importNodes[j].Name.AsString()
	})
	for _, importNode := range importNodes {
		if err := f.writeNode(importNode); err != nil {
			return err
		}
		f.P()
	}
	if len(optionNodes) > 0 && (f.fileNode.Syntax != nil || packageNode != nil || len(importNodes) > 0) {
		f.P()
	}
	sort.Slice(optionNodes, func(i, j int) bool {
		return stringForOptionName(optionNodes[i].Name) < stringForOptionName(optionNodes[j].Name)
	})
	for _, optionNode := range optionNodes {
		if err := f.writeNode(optionNode); err != nil {
			return err
		}
		f.P()
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
	if err := f.writeMultiline(syntaxNode.Keyword); err != nil {
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
	return f.writeInline(syntaxNode.Semicolon)
}

// writePackage writes the package.
//
// For example,
//
//  package acme.weather.v1;
//
func (f *formatter) writePackage(packageNode *ast.PackageNode) error {
	if err := f.writeMultiline(packageNode.Keyword); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(packageNode.Name); err != nil {
		return err
	}
	return f.writeInline(packageNode.Semicolon)
}

// writeImport writes an import statement.
//
// For example,
//
//  import "google/protobuf/descriptor.proto";
//
func (f *formatter) writeImport(importNode *ast.ImportNode) error {
	if err := f.writeMultiline(importNode.Keyword); err != nil {
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
	return f.writeInline(importNode.Semicolon)
}

// writeOption writes an option.
//
// For example,
//
//  option go_package = "github.com/foo/bar";
//
func (f *formatter) writeOption(optionNode *ast.OptionNode) error {
	if optionNode.Keyword != nil {
		// Compact options don't have the keyword.
		if err := f.writeMultiline(optionNode.Keyword); err != nil {
			return err
		}
		f.Space()
		if err := f.writeInline(optionNode.Name); err != nil {
			return err
		}
	} else {
		if err := f.writeMultiline(optionNode.Name); err != nil {
			return err
		}
	}
	f.Space()
	if err := f.writeInline(optionNode.Equals); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(optionNode.Val); err != nil {
		return err
	}
	if optionNode.Semicolon != nil {
		return f.writeInline(optionNode.Semicolon)
	}
	return nil
}

// writeOption writes an option name.
//
// For example,
//
//  go_package
//  (custom.thing)
//  (custom.thing).bridge.(another.thing)
//
func (f *formatter) writeOptionName(optionNameNode *ast.OptionNameNode) error {
	for i := 0; i < len(optionNameNode.Parts); i++ {
		if i > 0 {
			// The length of this slice must be exactly len(Parts)-1.
			if err := f.writeInline(optionNameNode.Dots[i-1]); err != nil {
				return err
			}
		}
		if err := f.writeInline(optionNameNode.Parts[i]); err != nil {
			return err
		}
	}
	return nil
}

// writeMessage writes the message node. Messages are formatted in the
// following order:
//
//  * Options
//  * Reserved Ranges
//  * Extension Ranges
//  * Messages, Enums, Extends
//  * Fields, Groups, Oneofs
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
// TODO: The complexity around comment handling needs to be incorporated
// into all of the other composite node implementations (e.g. enum, extend,
// oneof, etc).
func (f *formatter) writeMessage(messageNode *ast.MessageNode) error {
	messageElements, err := elementsForMessage(messageNode)
	if err != nil {
		return err
	}
	if err := f.writeMultiline(messageNode.Keyword); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(messageNode.Name); err != nil {
		return err
	}
	// We need to handle the comments for the '{' specially since
	// the trailing comments need to be written in the message block.
	info := f.fileNode.NodeInfo(messageNode.OpenBrace)
	if info.LeadingComments().Len() > 0 {
		f.Space()
		if err := f.writeInlineComments(info.LeadingComments()); err != nil {
			return err
		}
	}
	f.Space()
	if err := f.writeNode(messageNode.OpenBrace); err != nil {
		return err
	}
	if info.TrailingComments().Len() == 0 && messageElements == nil {
		info = f.fileNode.NodeInfo(messageNode.CloseBrace)
		if info.LeadingComments().Len() == 0 {
			// This is an empty message definition without any comments in
			// the body, so we return early.
			return f.writeInline(messageNode.CloseBrace)
		}
		// Writing comments in this case requires special care - we need
		// to write all of the leading comments in an empty message body,
		// so the comments need to be indented, but not the tokens they're
		// attached to.
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
		if err := f.writeNode(messageNode.CloseBrace); err != nil {
			return err
		}
		return f.writeInlineComments(info.TrailingComments())
	}
	f.P()
	f.In()
	if info.TrailingComments().Len() > 0 {
		if err := f.writeMultilineComments(info.TrailingComments()); err != nil {
			return err
		}
		if messageElements != nil {
			// If there are other elements to write, we need another
			// newline after the trailing comments.
			f.P()
		}
	}
	if messageElements != nil {
		if err := f.writeMessageElements(messageElements); err != nil {
			return err
		}
		f.P()
	}
	f.Out()
	if info := f.fileNode.NodeInfo(messageNode.CloseBrace); info.LeadingComments().Len() > 0 {
		f.P()
	}
	if err := f.writeMultiline(messageNode.CloseBrace); err != nil {
		return err
	}
	return nil
}

// writeMessageElements writes the message elements for a single message.
func (f *formatter) writeMessageElements(messageElements *messageElements) error {
	// Group all of the options, reserved, and extension ranges
	// in a single slice so that they are formatted together.
	var header []ast.MessageElement
	for _, option := range messageElements.Options {
		header = append(header, option)
	}
	for _, reserved := range messageElements.Reserved {
		header = append(header, reserved)
	}
	for _, extensionRange := range messageElements.ExtensionRanges {
		header = append(header, extensionRange)
	}
	for i, node := range header {
		if i > 0 {
			f.P()
		}
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	if len(header) > 0 && (len(messageElements.NestedTypes) > 0 || len(messageElements.Fields) > 0) {
		// Include a newline between the header and the types and/or fields.
		f.P()
		f.P()
	}
	for i, node := range messageElements.NestedTypes {
		if i > 0 {
			f.P()
			f.P()
		}
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	if len(messageElements.NestedTypes) > 0 && len(messageElements.Fields) > 0 {
		// Include a newline between the types and fields.
		f.P()
		f.P()
	}
	for i, node := range messageElements.Fields {
		if i > 0 {
			f.P()
		}
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	return nil
}

// writeMessageLiteral writes a message literal (e.g. '{ foo:1 foo:2 foo:3 bar:<name:"abc" id:123> }').
//
// TODO: This needs to be adapted to be like f.writeMessage.
func (f *formatter) writeMessageLiteral(messageLiteralNode *ast.MessageLiteralNode) error {
	// We need to handle the comments for the '{' specially since
	// the trailing comments need to be written in the message block.
	info := f.fileNode.NodeInfo(messageLiteralNode.Open)
	if info.LeadingComments().Len() > 0 {
		if err := f.writeInlineComments(info.LeadingComments()); err != nil {
			return err
		}
		f.Space()
	}
	if err := f.writeNode(messageLiteralNode.Open); err != nil {
		return err
	}
	if info.TrailingComments().Len() == 0 && len(messageLiteralNode.Elements) == 0 {
		// This is an empty message literal, so we return early.
		return f.writeInline(messageLiteralNode.Close)
	}
	if info.TrailingComments().Len() > 0 || len(messageLiteralNode.Elements) > 0 {
		f.P()
		f.In()
		if info.TrailingComments().Len() > 0 {
			if err := f.writeMultilineComments(info.TrailingComments()); err != nil {
				return err
			}
			f.P()
		}
		for i := 0; i < len(messageLiteralNode.Elements); i++ {
			if i > 0 {
				f.P()
			}
			// Comments are handled in the message field layer.
			if err := f.writeNode(messageLiteralNode.Elements[i]); err != nil {
				return err
			}
			if sep := messageLiteralNode.Seps[i]; sep != nil {
				if err := f.writeInline(messageLiteralNode.Seps[i]); err != nil {
					return err
				}
			}
		}
		f.Out()
		f.P()
	}
	if err := f.writeInlineIndented(messageLiteralNode.Close); err != nil {
		return err
	}
	return nil
}

// writeMessageField writes the message field node as a single line.
//
// For example,
//
//  foo:"bar"
//
func (f *formatter) writeMessageField(messageFieldNode *ast.MessageFieldNode) error {
	if err := f.writeInlineIndented(messageFieldNode.Name); err != nil {
		return err
	}
	if messageFieldNode.Sep != nil {
		if err := f.writeInline(messageFieldNode.Sep); err != nil {
			return err
		}
	}
	if err := f.writeInline(messageFieldNode.Val); err != nil {
		return err
	}
	return nil
}

// writeEnum writes the enum node. Enums are formatted in the
// following order:
//
//  * Options
//  * Reserved Ranges
//  * Enum Values
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
	enumElements, err := elementsForEnum(enumNode)
	if err != nil {
		return err
	}
	if err := f.writeMultiline(enumNode.Keyword); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(enumNode.Name); err != nil {
		return err
	}
	// We need to handle the comments for the '{' specially since
	// the trailing comments need to be written in the enum block.
	info := f.fileNode.NodeInfo(enumNode.OpenBrace)
	if info.LeadingComments().Len() > 0 {
		f.Space()
		if err := f.writeInlineComments(info.LeadingComments()); err != nil {
			return err
		}
	}
	f.Space()
	if err := f.writeNode(enumNode.OpenBrace); err != nil {
		return err
	}
	if info.TrailingComments().Len() == 0 && enumElements == nil {
		// This is an empty enum definition, so we return early.
		return f.writeInline(enumNode.CloseBrace)
	}
	if info.TrailingComments().Len() > 0 || enumElements != nil {
		f.P()
		f.In()
		if info.TrailingComments().Len() > 0 {
			if err := f.writeMultilineComments(info.TrailingComments()); err != nil {
				return err
			}
			f.P()
		}
		if enumElements != nil {
			if err := f.writeEnumElements(enumElements); err != nil {
				return err
			}
		}
		f.Out()
		f.P()
	}
	if err := f.writeInlineIndented(enumNode.CloseBrace); err != nil {
		return err
	}
	return nil
}

// writeEnumElements writes the enum elements for a single enum.
func (f *formatter) writeEnumElements(enumElements *enumElements) error {
	// Group all of the options and reserved ranges in a
	// single slice so that they are formatted together.
	var header []ast.EnumElement
	for _, option := range enumElements.Options {
		header = append(header, option)
	}
	for _, reserved := range enumElements.Reserved {
		header = append(header, reserved)
	}
	for i, node := range header {
		if i > 0 {
			f.P()
		}
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	if len(header) > 0 && len(enumElements.EnumValues) > 0 {
		// Include a newline between the header and the enum values.
		f.P()
		f.P()
	}
	for i, node := range enumElements.EnumValues {
		if i > 0 {
			f.P()
		}
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	return nil
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
	if err := f.writeMultiline(enumValueNode.Name); err != nil {
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
		if err := f.writeInline(enumValueNode.Options); err != nil {
			return err
		}
	}
	if err := f.writeInline(enumValueNode.Semicolon); err != nil {
		return err
	}
	return nil
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
		if err := f.writeMultiline(fieldNode.Label); err != nil {
			return err
		}
		f.Space()
		if err := f.writeInline(fieldNode.FldType); err != nil {
			return err
		}
	} else {
		// If a label was not written, the multiline comments will be
		// attached to the type.
		if err := f.writeMultiline(fieldNode.FldType); err != nil {
			return err
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
		if err := f.writeInline(fieldNode.Options); err != nil {
			return err
		}
	}
	if err := f.writeInline(fieldNode.Semicolon); err != nil {
		return err
	}
	return nil
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
		if err := f.writeInline(mapFieldNode.Options); err != nil {
			return err
		}
	}
	if err := f.writeInline(mapFieldNode.Semicolon); err != nil {
		return err
	}
	return nil
}

// writeMapType writes a map type (e.g. 'map<string, string>').
func (f *formatter) writeMapType(mapTypeNode *ast.MapTypeNode) error {
	if err := f.writeMultiline(mapTypeNode.Keyword); err != nil {
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
	if err := f.writeInline(mapTypeNode.CloseAngle); err != nil {
		return err
	}
	return nil
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
		if err := f.writeInline(fieldReferenceNode.Close); err != nil {
			return err
		}
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
	extendElements, err := elementsForExtend(extendNode)
	if err != nil {
		return err
	}
	if err := f.writeMultiline(extendNode.Keyword); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(extendNode.Extendee); err != nil {
		return err
	}
	// We need to handle the comments for the '{' specially since
	// the trailing comments need to be written in the extend block.
	info := f.fileNode.NodeInfo(extendNode.OpenBrace)
	if info.LeadingComments().Len() > 0 {
		f.Space()
		if err := f.writeInlineComments(info.LeadingComments()); err != nil {
			return err
		}
	}
	f.Space()
	if err := f.writeNode(extendNode.OpenBrace); err != nil {
		return err
	}
	if info.TrailingComments().Len() == 0 && extendElements == nil {
		// This is an empty extend definition, so we return early.
		return f.writeInline(extendNode.CloseBrace)
	}
	if info.TrailingComments().Len() > 0 || extendElements != nil {
		f.P()
		f.In()
		if info.TrailingComments().Len() > 0 {
			if err := f.writeMultilineComments(info.TrailingComments()); err != nil {
				return err
			}
			f.P()
		}
		for i, extendElement := range extendElements {
			if i > 0 {
				f.P()
			}
			if err := f.writeNode(extendElement); err != nil {
				return err
			}
		}
		f.Out()
		f.P()
	}
	if err := f.writeInlineIndented(extendNode.CloseBrace); err != nil {
		return err
	}
	return nil
}

// writeService writes the service node. Services are formatted in
// the following order:
//
//  * Options
//  * RPCs
//
// For example,
//
//  service FooService {
//    option deprecated = true;
//
//    rpc Foo(FooRequest) returns (FooResponse) {};
//
func (f *formatter) writeService(serviceNode *ast.ServiceNode) error {
	serviceElements, err := elementsForService(serviceNode)
	if err != nil {
		return err
	}
	if err := f.writeMultiline(serviceNode.Keyword); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(serviceNode.Name); err != nil {
		return err
	}
	// We need to handle the comments for the '{' specially since
	// the trailing comments need to be written in the service block.
	info := f.fileNode.NodeInfo(serviceNode.OpenBrace)
	if info.LeadingComments().Len() > 0 {
		f.Space()
		if err := f.writeInlineComments(info.LeadingComments()); err != nil {
			return err
		}
	}
	f.Space()
	if err := f.writeNode(serviceNode.OpenBrace); err != nil {
		return err
	}
	if info.TrailingComments().Len() == 0 && serviceElements == nil {
		// This is an empty service definition, so we return early.
		return f.writeInline(serviceNode.CloseBrace)
	}
	if info.TrailingComments().Len() > 0 || serviceElements != nil {
		f.P()
		f.In()
		if info.TrailingComments().Len() > 0 {
			if err := f.writeMultilineComments(info.TrailingComments()); err != nil {
				return err
			}
			f.P()
		}
		if serviceElements != nil {
			if err := f.writeServiceElements(serviceElements); err != nil {
				return err
			}
		}
		f.Out()
		f.P()
	}
	if err := f.writeInlineIndented(serviceNode.CloseBrace); err != nil {
		return err
	}
	return nil
}

// writeServiceElements writes the elements for a single service.
func (f *formatter) writeServiceElements(serviceElements *serviceElements) error {
	for i, node := range serviceElements.Options {
		if i > 0 {
			f.P()
		}
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	if len(serviceElements.Options) > 0 && len(serviceElements.RPCs) > 0 {
		// Include a newline between the options and the RPCs.
		f.P()
		f.P()
	}
	for i, node := range serviceElements.RPCs {
		if i > 0 {
			f.P()
		}
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	return nil
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
	var options []*ast.OptionNode
	for _, rpcElement := range rpcNode.Decls {
		if option, ok := rpcElement.(*ast.OptionNode); ok {
			options = append(options, option)
		}
	}
	if err := f.writeMultiline(rpcNode.Keyword); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(rpcNode.Name); err != nil {
		return err
	}
	f.Space()
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
		return f.writeInline(rpcNode.Semicolon)
	}
	// We need to handle the comments for the '{' specially since
	// the trailing comments need to be written in the RPC block.
	info := f.fileNode.NodeInfo(rpcNode.OpenBrace)
	if info.LeadingComments().Len() > 0 {
		f.Space()
		if err := f.writeInlineComments(info.LeadingComments()); err != nil {
			return err
		}
	}
	f.Space()
	if err := f.writeNode(rpcNode.OpenBrace); err != nil {
		return err
	}
	if info.TrailingComments().Len() == 0 && options == nil {
		// This is an empty RPC definition, so we return early.
		return f.writeInline(rpcNode.CloseBrace)
	}
	if info.TrailingComments().Len() > 0 || options != nil {
		f.P()
		f.In()
		if info.TrailingComments().Len() > 0 {
			if err := f.writeMultilineComments(info.TrailingComments()); err != nil {
				return err
			}
			f.P()
		}
		for i, option := range options {
			if i > 0 {
				f.P()
			}
			if err := f.writeNode(option); err != nil {
				return err
			}
		}
		f.Out()
		f.P()
	}
	if err := f.writeInlineIndented(rpcNode.CloseBrace); err != nil {
		return err
	}
	return nil
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
	if err := f.writeInline(rpcTypeNode.CloseParen); err != nil {
		return err
	}
	return nil
}

// writeOneOf writes the oneof node. OneOfs are formatted in the
// following order:
//
//  * Options
//  * Fields, Groups
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
	oneOfElements, err := elementsForOneOf(oneOfNode)
	if err != nil {
		return err
	}
	if err := f.writeMultiline(oneOfNode.Keyword); err != nil {
		return err
	}
	f.Space()
	if err := f.writeInline(oneOfNode.Name); err != nil {
		return err
	}
	// We need to handle the comments for the '{' specially since
	// the trailing comments need to be written in the oneof block.
	info := f.fileNode.NodeInfo(oneOfNode.OpenBrace)
	if info.LeadingComments().Len() > 0 {
		f.Space()
		if err := f.writeInlineComments(info.LeadingComments()); err != nil {
			return err
		}
	}
	f.Space()
	if err := f.writeNode(oneOfNode.OpenBrace); err != nil {
		return err
	}
	if info.TrailingComments().Len() == 0 && oneOfElements == nil {
		// This is an empty oneof definition, so we return early.
		return f.writeInline(oneOfNode.CloseBrace)
	}
	if info.TrailingComments().Len() > 0 || oneOfElements != nil {
		f.P()
		f.In()
		if info.TrailingComments().Len() > 0 {
			if err := f.writeMultilineComments(info.TrailingComments()); err != nil {
				return err
			}
			f.P()
		}
		if oneOfElements != nil {
			if err := f.writeOneOfElements(oneOfElements); err != nil {
				return err
			}
		}
		f.Out()
		f.P()
	}
	if err := f.writeInlineIndented(oneOfNode.CloseBrace); err != nil {
		return err
	}
	return nil
}

// writeOneOfElements writes the oneOf elements for a single oneof.
func (f *formatter) writeOneOfElements(oneOfElements *oneOfElements) error {
	for i, node := range oneOfElements.Options {
		if i > 0 {
			f.P()
		}
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	if len(oneOfElements.Options) > 0 && len(oneOfElements.Fields) > 0 {
		// Include a newline between the options and the oneOf values.
		f.P()
		f.P()
	}
	for i, node := range oneOfElements.Fields {
		if i > 0 {
			f.P()
		}
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	return nil
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
	messageElements, err := elementsForGroup(groupNode)
	if err != nil {
		return err
	}
	// We need to handle the comments for the group label specially since
	// a label might not be defined, but it has the leading comments attached
	// to it.
	if groupNode.Label.KeywordNode != nil {
		if err := f.writeMultiline(groupNode.Label); err != nil {
			return err
		}
		f.Space()
		if err := f.writeInline(groupNode.Keyword); err != nil {
			return err
		}
	} else {
		// If a label was not written, the multiline comments will be
		// attached to the keyword.
		if err := f.writeMultiline(groupNode.Keyword); err != nil {
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
		if err := f.writeInline(groupNode.Options); err != nil {
			return err
		}
	}
	// We need to handle the comments for the '{' specially since
	// the trailing comments need to be written in the group block.
	info := f.fileNode.NodeInfo(groupNode.OpenBrace)
	if info.LeadingComments().Len() > 0 {
		f.Space()
		if err := f.writeInlineComments(info.LeadingComments()); err != nil {
			return err
		}
	}
	f.Space()
	if err := f.writeNode(groupNode.OpenBrace); err != nil {
		return err
	}
	if info.TrailingComments().Len() == 0 && messageElements == nil {
		info = f.fileNode.NodeInfo(groupNode.CloseBrace)
		if info.LeadingComments().Len() == 0 {
			// This is an empty group definition without any comments in
			// the body, so we return early.
			return f.writeInline(groupNode.CloseBrace)
		}
		// Writing comments in this case requires special care - we need
		// to write all of the leading comments in an empty message body,
		// so the comments need to be indented, but not the tokens they're
		// attached to.
		//
		// For example,
		//
		//  optional group Foo {
		//    // This group might have multiple comments.
		//    // They're attached to the '}'.
		//  }
		f.P()
		f.In()
		if err := f.writeMultilineComments(info.LeadingComments()); err != nil {
			return err
		}
		f.Out()
		if err := f.writeNode(groupNode.CloseBrace); err != nil {
			return err
		}
		return f.writeInlineComments(info.TrailingComments())
	}
	f.P()
	f.In()
	if info.TrailingComments().Len() > 0 {
		if err := f.writeMultilineComments(info.TrailingComments()); err != nil {
			return err
		}
		if messageElements != nil {
			// If there are other elements to write, we need another
			// newline after the trailing comments.
			f.P()
		}
	}
	if messageElements != nil {
		if err := f.writeMessageElements(messageElements); err != nil {
			return err
		}
		f.P()
	}
	f.Out()
	if info := f.fileNode.NodeInfo(groupNode.CloseBrace); info.LeadingComments().Len() > 0 {
		f.P()
	}
	if err := f.writeMultiline(groupNode.CloseBrace); err != nil {
		return err
	}
	return nil
}

// writeExtensionRange writes the extension range node.
//
// For example,
//
//  extensions 5-10, 100 to max [deprecated = true];
//
func (f *formatter) writeExtensionRange(extensionRangeNode *ast.ExtensionRangeNode) error {
	if err := f.writeMultiline(extensionRangeNode.Keyword); err != nil {
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
		if err := f.writeInline(extensionRangeNode.Options); err != nil {
			return err
		}
	}
	if err := f.writeInline(extensionRangeNode.Semicolon); err != nil {
		return err
	}
	return nil
}

// writeReserved writes a reserved node.
//
// For example,
//
//  reserved 5-10, 100 to max;
//
func (f *formatter) writeReserved(reservedNode *ast.ReservedNode) error {
	if err := f.writeMultiline(reservedNode.Keyword); err != nil {
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
	if err := f.writeInline(reservedNode.Semicolon); err != nil {
		return err
	}
	return nil
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
func (f *formatter) writeCompactOptions(compactOptionsNode *ast.CompactOptionsNode) error {
	if err := f.writeInline(compactOptionsNode.OpenBracket); err != nil {
		return err
	}
	f.In()
	for i := 0; i < len(compactOptionsNode.Options); i++ {
		if i > 0 {
			// The length of this slice must be exactly len(Options)-1.
			if err := f.writeInline(compactOptionsNode.Commas[i-1]); err != nil {
				return err
			}
		}
		f.P()
		if err := f.writeNode(compactOptionsNode.Options[i]); err != nil {
			return err
		}
	}
	f.Out()
	if len(compactOptionsNode.Options) > 0 {
		f.P()
	}
	if err := f.writeInlineIndented(compactOptionsNode.CloseBracket); err != nil {
		return err
	}
	return nil
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

// writeArrayLiteral writes an array literal across multiple lines.
//
// For example,
//
//  [
//    "foo",
//    "bar"
//  ]
func (f *formatter) writeArrayLiteral(arrayLiteralNode *ast.ArrayLiteralNode) error {
	if err := f.writeInline(arrayLiteralNode.OpenBracket); err != nil {
		return err
	}
	f.In()
	for i := 0; i < len(arrayLiteralNode.Elements); i++ {
		if i > 0 {
			// The length of this slice must be exactly len(Elements)-1.
			if err := f.writeInline(arrayLiteralNode.Commas[i-1]); err != nil {
				return err
			}
		}
		f.P()
		if err := f.writeInlineIndented(arrayLiteralNode.Elements[i]); err != nil {
			return err
		}
	}
	f.Out()
	if len(arrayLiteralNode.Elements) > 0 {
		f.P()
	}
	if err := f.writeInlineIndented(arrayLiteralNode.CloseBracket); err != nil {
		return err
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
	for _, child := range compoundStringLiteralNode.Children() {
		if err := f.writeInline(child); err != nil {
			return err
		}
	}
	return nil
}

// writeFloatLiteral writes a float literal value (e.g. '42.2').
func (f *formatter) writeFloatLiteral(floatLiteralNode *ast.FloatLiteralNode) error {
	f.WriteString(strconv.FormatFloat(floatLiteralNode.Val, 'g', 2, 64))
	return nil
}

// writeSignedFloatLiteral writes a signed float literal value (e.g. '-42.2').
func (f *formatter) writeSignedFloatLiteral(signedFloatLiteralNode *ast.SignedFloatLiteralNode) error {
	if err := f.writeInline(signedFloatLiteralNode.Sign); err != nil {
		return err
	}
	return f.writeInline(signedFloatLiteralNode.Float)
}

// writeSpecialFloatLiteral writes a special float literal value (e.g. "nan" or "inf").
func (f *formatter) writeSpecialFloatLiteral(specialFloatLiteralNode *ast.SpecialFloatLiteralNode) error {
	f.WriteString(fmt.Sprintf("%q", specialFloatLiteralNode.KeywordNode.Val))
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

// writeNegativeIntLiteral writes a int literal (e.g. '-42').
func (f *formatter) writeNegativeIntLiteral(negativeIntLiteralNode *ast.NegativeIntLiteralNode) error {
	if err := f.writeInline(negativeIntLiteralNode.Minus); err != nil {
		return err
	}
	return f.writeInline(negativeIntLiteralNode.Uint)
}

// writePositiveUintLiteral writes a int literal (e.g. '-42').
func (f *formatter) writePositiveUintLiteral(positiveIntLiteralNode *ast.PositiveUintLiteralNode) error {
	if err := f.writeInline(positiveIntLiteralNode.Plus); err != nil {
		return err
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

// writeMultiline writes the node across multiple lines.
// Multiline nodes are always indented.
func (f *formatter) writeMultiline(node ast.Node) error {
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

// writeInlineIndented writes the node in a single indented line.
//
// This is useful for writing single line declarations that need to
// be indented, such as options, fields, etc.
func (f *formatter) writeInlineIndented(node ast.Node) error {
	_, _ = fmt.Fprint(f.writer, strings.Repeat("  ", f.indent))
	return f.writeInline(node)
}

// writeInline writes the node and its surrounding comments in-line.
//
// This is useful for writing individual nodes like keywords, runes,
// string literals, etc.
//
// For example,
//
//  // This is a leading comment on the syntax keyword.
//  syntax = "proto3" /* This is a leading comment on the ';' */;
//
func (f *formatter) writeInline(node ast.Node) error {
	if _, ok := node.(ast.CompositeNode); ok {
		// We only want to write comments for terminal nodes.
		// Otherwise comments accessible from CompositeNodes
		// will be written twice.
		return f.writeNode(node)
	}
	info := f.fileNode.NodeInfo(node)
	if info.LeadingComments().Len() > 0 {
		if err := f.writeInlineComments(info.LeadingComments()); err != nil {
			return err
		}
		f.Space()
	}
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

// writeMultilineComments writes the given comments as a newline-delimited block.
func (f *formatter) writeMultilineComments(comments ast.Comments) error {
	for i := 0; i < comments.Len(); i++ {
		f.P(strings.TrimSpace(comments.Index(i).RawText()))
	}
	return nil
}

// writeInlineComments writes the given comments in-line. Multiple in-line comments
// are separated by a space.
//
// TODO: We might need to transform '//' comments into C-style comments if the
// node MUST be written in-line.
func (f *formatter) writeInlineComments(comments ast.Comments) error {
	for i := 0; i < comments.Len(); i++ {
		if i > 0 {
			f.Space()
		}
		f.WriteString(strings.TrimSpace(comments.Index(i).RawText()))
	}
	return nil
}

// stringForOptionName returns the string representation of the given option name node.
// This is used for sorting the options at the top of the file.
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
// This is used for sorting the options at the top of the file.
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
