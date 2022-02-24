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

	"github.com/jhump/protocompile/ast"
)

// messageElements is a collection of a single message's elements.
type messageElements struct {
	Options         []*ast.OptionNode
	Reserved        []*ast.ReservedNode
	ExtensionRanges []*ast.ExtensionRangeNode

	// NestedTypes are defined as an interface so that we maintain
	// the order that messages, enums, and extends are specified.
	NestedTypes []ast.MessageElement

	// Fields are defined as an interface so that we maintain
	// the order that fields, groups, oneof, and maps are
	// specified.
	Fields []ast.MessageElement
}

// enumElements is a collection of a single enum's elements.
type enumElements struct {
	Options    []*ast.OptionNode
	Reserved   []*ast.ReservedNode
	EnumValues []*ast.EnumValueNode
}

// oneOfElements is a collection of a single oneof's elements.
type oneOfElements struct {
	Options []*ast.OptionNode

	// Both fields and groups are consolidated in the same slice
	// to preserve order.
	Fields []ast.OneOfElement
}

// serviceElements is a collection of a single service's elements.
type serviceElements struct {
	Options []*ast.OptionNode
	RPCs    []*ast.RPCNode
}

// elementsForMessage returns all of the given message's elements
// in a single set.
func elementsForMessage(messageNode *ast.MessageNode) (*messageElements, error) {
	return elementsForMessageBody(messageNode.MessageBody)
}

// elementsForGroup returns all of the given group's elements
// in a single set.
func elementsForGroup(groupNode *ast.GroupNode) (*messageElements, error) {
	return elementsForMessageBody(groupNode.MessageBody)
}

// elementsForMessageBody returns the elements associated with the given
// message body.
func elementsForMessageBody(messageBody ast.MessageBody) (*messageElements, error) {
	if len(messageBody.Decls) == 0 {
		return nil, nil
	}
	messageElements := new(messageElements)
	for _, messageElement := range messageBody.Decls {
		switch node := messageElement.(type) {
		case *ast.OptionNode:
			messageElements.Options = append(messageElements.Options, node)
		case *ast.FieldNode, *ast.MapFieldNode, *ast.OneOfNode, *ast.GroupNode:
			messageElements.Fields = append(messageElements.Fields, node)
		case *ast.MessageNode, *ast.EnumNode, *ast.ExtendNode:
			messageElements.NestedTypes = append(messageElements.NestedTypes, node)
		case *ast.ExtensionRangeNode:
			messageElements.ExtensionRanges = append(messageElements.ExtensionRanges, node)
		case *ast.ReservedNode:
			messageElements.Reserved = append(messageElements.Reserved, node)
		case *ast.EmptyDeclNode:
			// Nothing to do here.
			continue
		default:
			return nil, fmt.Errorf("unexpected message element: %T", messageElement)
		}
	}
	return messageElements, nil
}

// elementsForEnum returns all of the given enum's elements in a single set.
func elementsForEnum(enumNode *ast.EnumNode) (*enumElements, error) {
	if len(enumNode.Decls) == 0 {
		return nil, nil
	}
	enumElements := new(enumElements)
	for _, enumElement := range enumNode.Decls {
		switch node := enumElement.(type) {
		case *ast.OptionNode:
			enumElements.Options = append(enumElements.Options, node)
		case *ast.ReservedNode:
			enumElements.Reserved = append(enumElements.Reserved, node)
		case *ast.EnumValueNode:
			enumElements.EnumValues = append(enumElements.EnumValues, node)
		case *ast.EmptyDeclNode:
			// Nothing to do here.
			continue
		default:
			return nil, fmt.Errorf("unexpected enum element: %T", enumElement)
		}
	}
	return enumElements, nil
}

// elementsForOneOf returns all of the given oneof's elements in a single set.
func elementsForOneOf(oneOfNode *ast.OneOfNode) (*oneOfElements, error) {
	if len(oneOfNode.Decls) == 0 {
		return nil, nil
	}
	oneOfElements := new(oneOfElements)
	for _, oneOfElement := range oneOfNode.Decls {
		switch node := oneOfElement.(type) {
		case *ast.OptionNode:
			oneOfElements.Options = append(oneOfElements.Options, node)
		case *ast.FieldNode, *ast.GroupNode:
			oneOfElements.Fields = append(oneOfElements.Fields, node)
		case *ast.EmptyDeclNode:
			// Nothing to do here.
			continue
		default:
			return nil, fmt.Errorf("unexpected oneof element: %T", oneOfElement)
		}
	}
	return oneOfElements, nil
}

// elementsForService returns all of the given service's elements in a single set.
func elementsForService(serviceNode *ast.ServiceNode) (*serviceElements, error) {
	if len(serviceNode.Decls) == 0 {
		return nil, nil
	}
	serviceElements := new(serviceElements)
	for _, serviceElement := range serviceNode.Decls {
		switch node := serviceElement.(type) {
		case *ast.OptionNode:
			serviceElements.Options = append(serviceElements.Options, node)
		case *ast.RPCNode:
			serviceElements.RPCs = append(serviceElements.RPCs, node)
		case *ast.EmptyDeclNode:
			// Nothing to do here.
			continue
		default:
			return nil, fmt.Errorf("unexpected service element: %T", serviceElement)
		}
	}
	return serviceElements, nil
}

// elementsForExtend returns all of the given extend's elements in a single set.
func elementsForExtend(extendNode *ast.ExtendNode) ([]ast.ExtendElement, error) {
	if len(extendNode.Decls) == 0 {
		return nil, nil
	}
	var extendElements []ast.ExtendElement
	for _, extendElement := range extendNode.Decls {
		switch node := extendElement.(type) {
		case *ast.FieldNode, *ast.GroupNode:
			extendElements = append(extendElements, node)
		case *ast.EmptyDeclNode:
			// Nothing to do here.
			continue
		default:
			return nil, fmt.Errorf("unexpected extend element: %T", extendElement)
		}
	}
	return extendElements, nil
}
