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
}

// newFormatter returns a new formatter for the given file.
func newFormatter(writer io.Writer, fileNode *ast.FileNode) *formatter {
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

// writeFile writes the file node.
func (f *formatter) writeFile(fileNode *ast.FileNode) error {
	info := f.fileNode.NodeInfo(fileNode)
	if info.LeadingComments().Len() > 0 {
		if err := f.writeComments(info.LeadingComments()); err != nil {
			return err
		}
		// File nodes need a newline between their leading comments
		// and the first child node (e.g. the syntax).
		f.P()
	}
	if err := f.writeFileHeader(fileNode); err != nil {
		return err
	}
	if err := f.writeFileTypes(fileNode); err != nil {
		return err
	}
	return f.writeComments(info.TrailingComments())
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
	if syntaxNode := f.fileNode.Syntax; syntaxNode != nil {
		if err := f.writeSyntax(syntaxNode); err != nil {
			return err
		}
		f.P()
	}
	var (
		packageNode *ast.PackageNode
		optionNodes []*ast.OptionNode
		importNodes []*ast.ImportNode
	)
	for _, fileElement := range f.fileNode.Decls {
		switch node := fileElement.(type) {
		case *ast.PackageNode:
			packageNode = node
		case *ast.OptionNode:
			optionNodes = append(optionNodes, node)
		case *ast.ImportNode:
			importNodes = append(importNodes, node)
		}
	}
	if packageNode != nil {
		if err := f.writeNode(packageNode); err != nil {
			return err
		}
		f.P()
	}
	sort.Slice(importNodes, func(i, j int) bool {
		return importNodes[i].Name.AsString() < importNodes[j].Name.AsString()
	})
	for _, importNode := range importNodes {
		if err := f.writeNode(importNode); err != nil {
			return err
		}
	}
	if len(importNodes) > 0 {
		f.P()
	}
	sort.Slice(optionNodes, func(i, j int) bool {
		return stringForOptionName(optionNodes[i].Name) < stringForOptionName(optionNodes[j].Name)
	})
	for _, optionNode := range optionNodes {
		if err := f.writeNode(optionNode); err != nil {
			return err
		}
	}
	if len(optionNodes) > 0 {
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
		case *ast.PackageNode, *ast.OptionNode, *ast.ImportNode:
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
	f.P(`syntax = "`, syntaxNode.Syntax.AsString(), `";`)
	return nil
}

// writePackage writes the package.
//
// For example,
//
//  package acme.weather.v1;
//
func (f *formatter) writePackage(packageNode *ast.PackageNode) error {
	f.P("package ", string(packageNode.Name.AsIdentifier()), ";")
	return nil
}

// writeImport writes an import statement.
//
// For example,
//
//  import "google/protobuf/descriptor.proto";
//
func (f *formatter) writeImport(importNode *ast.ImportNode) error {
	var label string
	if importNode.Public != nil {
		label = "public "
	}
	if importNode.Weak != nil {
		label = "weak "
	}
	importStatement := fmt.Sprintf(
		"import %s%q;",
		label,
		importNode.Name.AsString(),
	)
	f.P(importStatement)
	return nil
}

// writeOption writes an option.
//
// For example,
//
//  option go_package = "github.com/foo/bar";
//
func (f *formatter) writeOption(optionNode *ast.OptionNode) error {
	option := fmt.Sprintf(
		"option %s = %s;",
		stringForOptionName(optionNode.Name),
		stringForValue(optionNode.Val),
	)
	f.P(option)
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
func (f *formatter) writeMessage(messageNode *ast.MessageNode) error {
	messageElements, err := elementsForMessage(messageNode)
	if err != nil {
		return err
	}
	if messageElements == nil {
		f.P(`message `, messageNode.Name.Val, ` {}`)
		return nil
	}
	f.P(`message `, messageNode.Name.Val, ` {`)
	f.In()
	if err := f.writeMessageElements(messageElements); err != nil {
		return err
	}
	f.Out()
	f.P(`}`)
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
	if enumElements == nil {
		f.P(`enum `, enumNode.Name.Val, ` {}`)
		return nil
	}
	f.P(`enum `, enumNode.Name.Val, ` {`)
	f.In()
	if err := f.writeEnumElements(enumElements); err != nil {
		return err
	}
	f.Out()
	f.P(`}`)
	return nil
}

// writeEnumValue writes the enum value as a single line.
//
// For example,
//
//  FOO_UNSPECIFIED = 1 [deprecated = true];
//
func (f *formatter) writeEnumValue(enumValueNode *ast.EnumValueNode) error {
	enumValue := fmt.Sprintf(
		"%s = %s%s;",
		enumValueNode.Name.Val,
		stringForValue(enumValueNode.Number),
		stringForCompactOptions(enumValueNode.Options),
	)
	f.P(enumValue)
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
	if serviceElements == nil {
		f.P(`service `, serviceNode.Name.Val, ` {}`)
		return nil
	}
	f.P(`service `, serviceNode.Name.Val, ` {`)
	f.In()
	if err := f.writeServiceElements(serviceElements); err != nil {
		return err
	}
	f.Out()
	f.P(`}`)
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
	rpcPrefix := fmt.Sprintf(
		"rpc %s(%s) returns (%s)",
		rpcNode.Name.Val,
		stringForRPCType(rpcNode.Input),
		stringForRPCType(rpcNode.Output),
	)
	if options == nil {
		f.P(rpcPrefix + ";")
		return nil
	}
	f.P(rpcPrefix + " {")
	f.In()
	for _, option := range options {
		if err := f.writeOption(option); err != nil {
			return err
		}
	}
	f.Out()
	f.P(`}`)
	return nil
}

// writeField writes the field node as a single line.
//
// For example,
//
//  repeated string name = 1 [deprecated = true, json_name = "name"];
//
func (f *formatter) writeField(fieldNode *ast.FieldNode) error {
	field := fmt.Sprintf(
		"%s%s %s = %s%s;",
		f.stringForFieldLabel(fieldNode.Label),
		fieldNode.FldType.AsIdentifier(),
		fieldNode.Name.Val,
		strconv.FormatUint(fieldNode.Tag.Val, 10),
		stringForCompactOptions(fieldNode.GetOptions()),
	)
	f.P(field)
	return nil
}

// writeMapField writes the map field node as a single line.
//
// For example,
//
//  map<string,string> pairs = 1 [deprecated = true, json_name = "pairs"];
func (f *formatter) writeMapField(mapFieldNode *ast.MapFieldNode) error {
	mapField := fmt.Sprintf(
		"map<%s,%s> %s = %s%s;",
		mapFieldNode.MapType.KeyType.Val,
		string(mapFieldNode.MapType.ValueType.AsIdentifier()),
		mapFieldNode.Name.Val,
		strconv.FormatUint(mapFieldNode.Tag.Val, 10),
		stringForCompactOptions(mapFieldNode.GetOptions()),
	)
	f.P(mapField)
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
	if oneOfElements == nil {
		f.P(`oneof `, oneOfNode.Name.Val, ` {}`)
		return nil
	}
	f.P(`oneof `, oneOfNode.Name.Val, ` {`)
	f.In()
	if err := f.writeOneOfElements(oneOfElements); err != nil {
		return err
	}
	f.Out()
	f.P(`}`)
	return nil
}

// writeGroup writes the group node.
//
// For example,
//
//  optional group Key = 4 [deprecated = true, json_name = "key"] {
//    optional uint64 id = 1;
//    optional string name = 2;
//  }
//
func (f *formatter) writeGroup(groupNode *ast.GroupNode) error {
	messageElements, err := elementsForGroup(groupNode)
	if err != nil {
		return err
	}
	groupPrefix := fmt.Sprintf(
		"%sgroup %s = %s%s {",
		f.stringForFieldLabel(groupNode.Label),
		groupNode.Name.Val,
		strconv.FormatUint(groupNode.Tag.Val, 10),
		stringForCompactOptions(groupNode.Options),
	)
	if messageElements == nil {
		f.P(groupPrefix + "};")
		return nil
	}
	f.P(groupPrefix)
	f.In()
	if err := f.writeMessageElements(messageElements); err != nil {
		return err
	}
	f.Out()
	f.P(`}`)
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
	if len(extendElements) == 0 {
		f.P("extend ", string(extendNode.Extendee.AsIdentifier()), "{};")
		return nil
	}
	f.P("extend ", string(extendNode.Extendee.AsIdentifier()), " {")
	f.In()
	for _, extendElement := range extendElements {
		if err := f.writeNode(extendElement); err != nil {
			return err
		}
	}
	f.Out()
	f.P(`}`)
	return nil
}

// writeExtensionRange writes the extension range node.
//
// For example,
//
//  extensions 5-10, 100 to max [deprecated = true];
//
func (f *formatter) writeExtensionRange(extensionRangeNode *ast.ExtensionRangeNode) error {
	extension := fmt.Sprintf(
		"extensions %s%s",
		stringForRanges(extensionRangeNode.Ranges),
		stringForCompactOptions(extensionRangeNode.Options),
	)
	f.P(extension)
	return nil
}

// writeExtensionRange writes the extension range node.
//
// For example,
//
//  reserved 5-10, 100 to max;
//
func (f *formatter) writeReserved(reservedNode *ast.ReservedNode) error {
	// Either names or ranges will be set, but never both.
	var reservedValue string
	if len(reservedNode.Names) > 0 {
		for i, name := range reservedNode.Names {
			if i > 0 {
				reservedValue += ", "
			}
			reservedValue += fmt.Sprintf("%q", name.AsString())
		}
	}
	if len(reservedNode.Ranges) > 0 {
		reservedValue = stringForRanges(reservedNode.Ranges)
	}
	f.P("reserved ", reservedValue, ";")
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
	for _, node := range header {
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	if len(header) > 0 && (len(messageElements.NestedTypes) > 0 || len(messageElements.Fields) > 0) {
		// Include a newline between the header and the types and/or fields.
		f.P()
	}
	for _, node := range messageElements.NestedTypes {
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	if len(messageElements.NestedTypes) > 0 && len(messageElements.Fields) > 0 {
		// Include a newline between the types and fields.
		f.P()
	}
	for _, node := range messageElements.Fields {
		if err := f.writeNode(node); err != nil {
			return err
		}
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
	for _, node := range header {
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	if len(header) > 0 && len(enumElements.EnumValues) > 0 {
		// Include a newline between the header and the enum values.
		f.P()
	}
	for _, node := range enumElements.EnumValues {
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	return nil
}

// writeOneOfElements writes the oneOf elements for a single oneof.
func (f *formatter) writeOneOfElements(oneOfElements *oneOfElements) error {
	for _, node := range oneOfElements.Options {
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	if len(oneOfElements.Options) > 0 && len(oneOfElements.Fields) > 0 {
		// Include a newline between the options and the oneOf values.
		f.P()
	}
	for _, node := range oneOfElements.Fields {
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	return nil
}

// writeServiceElements writes the service elements for a single oneof.
func (f *formatter) writeServiceElements(serviceElements *serviceElements) error {
	for _, node := range serviceElements.Options {
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	if len(serviceElements.Options) > 0 && len(serviceElements.RPCs) > 0 {
		// Include a newline between the options and the RPCs.
		f.P()
	}
	for _, node := range serviceElements.RPCs {
		if err := f.writeNode(node); err != nil {
			return err
		}
	}
	return nil
}

// writeNode writes the node, as well as the comments surrounding it.
// Note that this function only includes the nodes that have comments
// attached to them. Other nodes written in-line (e.g. compact options)
// are handled with simpler string conversion functions.
func (f *formatter) writeNode(node ast.Node) error {
	info := f.fileNode.NodeInfo(node)
	if err := f.writeComments(info.LeadingComments()); err != nil {
		return err
	}
	switch element := node.(type) {
	case *ast.EnumNode:
		return f.writeEnum(element)
	case *ast.EnumValueNode:
		return f.writeEnumValue(element)
	case *ast.ExtendNode:
		return f.writeExtend(element)
	case *ast.ExtensionRangeNode:
		return f.writeExtensionRange(element)
	case *ast.FieldNode:
		return f.writeField(element)
	case *ast.GroupNode:
		return f.writeGroup(element)
	case *ast.ImportNode:
		return f.writeImport(element)
	case *ast.MapFieldNode:
		return f.writeMapField(element)
	case *ast.MessageNode:
		return f.writeMessage(element)
	case *ast.OneOfNode:
		return f.writeOneOf(element)
	case *ast.OptionNode:
		return f.writeOption(element)
	case *ast.PackageNode:
		return f.writePackage(element)
	case *ast.ReservedNode:
		return f.writeReserved(element)
	case *ast.RPCNode:
		return f.writeRPC(element)
	case *ast.ServiceNode:
		return f.writeService(element)
	case *ast.SyntaxNode:
		return f.writeSyntax(element)
	case *ast.EmptyDeclNode:
		// Nothing to do here.
	default:
		return fmt.Errorf("unexpected node: %T", node)
	}
	return f.writeComments(info.TrailingComments())
}

// writeComments writes the given comments.
func (f *formatter) writeComments(comments ast.Comments) error {
	if comments.Len() == 0 {
		return nil
	}
	for i := 0; i < comments.Len(); i++ {
		// Preserve the indentation configured on the formatter.
		//
		// f.P will automatically handle newlines, so we make sure
		// to remove the trailing newline from the comment, if any.
		f.P(strings.TrimRight(comments.Index(i).RawText(), "\n"))
	}
	return nil
}

// stringForFieldLabel returns the string representation of this field label, if any.
func (f *formatter) stringForFieldLabel(fieldLabel ast.FieldLabel) string {
	if fieldLabel.Required {
		return "required "
	}
	if fieldLabel.Repeated {
		return "repeated "
	}
	if f.fileNode.Syntax.Syntax.AsString() == "proto2" {
		return "optional "
	}
	// TODO: Where do we handle synthetic oneofs (i.e. proto3 optional)?
	return ""
}

// stringForCompactOptions returns the string representation of the given
// element with compact options.
func stringForCompactOptions(compactOptionsNode *ast.CompactOptionsNode) string {
	if compactOptionsNode == nil || len(compactOptionsNode.Options) == 0 {
		return ""
	}
	result := " ["
	for i, option := range compactOptionsNode.Options {
		if i > 0 {
			// Add a comma between each of the options.
			result += ", "
		}
		result += fmt.Sprintf("%s = %s", stringForOptionName(option.Name), stringForValue(option.Val))
	}
	result += "]"
	return result
}

// stringForRanges returns the string representation of the given range nodes.
//
// For example,
//
//  1 to 100, 200 to max
//
func stringForRanges(rangeNodes []*ast.RangeNode) string {
	var result string
	for i, rangeNode := range rangeNodes {
		if i > 0 {
			result += ", "
		}
		result += stringForRange(rangeNode)
	}
	return result
}

// stringForRange returns the string representation of the given range node.
//
// For example,
//
//  1 to 100
//
func stringForRange(rangeNode *ast.RangeNode) string {
	start := stringForValue(rangeNode.StartVal)
	if rangeNode.To == nil {
		return start
	}
	// Either EndVal or Max will be set, but never both.
	var end string
	switch {
	case rangeNode.EndVal != nil:
		end = stringForValue(rangeNode.EndVal)
	case rangeNode.Max != nil:
		end = "max"
	}
	return fmt.Sprintf("%s to %s", start, end)
}

// stringForOptionName returns the string representation of the given option name node.
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

// stringForRPCType returns the string representation of the given RPC type node.
func stringForRPCType(rpcTypeNode *ast.RPCTypeNode) string {
	var result string
	if rpcTypeNode.Stream != nil {
		result += "stream "
	}
	return result + string(rpcTypeNode.MessageType.AsIdentifier())
}

// stringForValue returns the string representation of the given value node
// (e.g. a compact option value).
func stringForValue(valueNode ast.ValueNode) string {
	switch value := valueNode.(type) {
	case *ast.IdentNode:
		return string(value.AsIdentifier())
	case *ast.CompoundIdentNode:
		return value.Val
	case *ast.StringLiteralNode:
		return fmt.Sprintf("%q", value.Val)
	case *ast.CompoundStringLiteralNode:
		return fmt.Sprintf("%q", value.Val)
	case *ast.UintLiteralNode:
		return strconv.FormatUint(value.Val, 10)
	case *ast.PositiveUintLiteralNode:
		return "+" + strconv.FormatUint(value.Val, 10)
	case *ast.NegativeIntLiteralNode:
		return "-" + strconv.FormatInt(value.Val, 10)
	case *ast.FloatLiteralNode:
		return strconv.FormatFloat(value.Val, 'g', 2, 64)
	case *ast.SpecialFloatLiteralNode:
		// Will be "inf" or "nan".
		return fmt.Sprintf("%q", value.KeywordNode.Val)
	case *ast.SignedFloatLiteralNode:
		return string(value.Sign.Rune) + strconv.FormatFloat(value.Val, 'g', 2, 64)
	case *ast.BoolLiteralNode:
		return strconv.FormatBool(value.Val)
	case *ast.ArrayLiteralNode:
		result := "["
		for i, valueNode := range value.Elements {
			if i > 0 {
				result += ", "
			}
			result += stringForValue(valueNode)
		}
		result += "]"
		return result
	case *ast.MessageLiteralNode:
		result := "{"
		for i, element := range value.Elements {
			if i > 0 {
				result += " " // We only support the ' ' separator.
			}
			result += fmt.Sprintf("%s:%s", stringForFieldReference(element.Name), stringForValue(element.Val))
		}
		result += "}"
		return result
	case *ast.NoSourceNode:
		// Nothing to do here.
		return ""
	}
	return ""
}

// stringForFieldReference returns the string representation of the given field reference.
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
