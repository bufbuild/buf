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

package bufls

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufwire"
	"github.com/bufbuild/buf/private/buf/bufwork"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/jhump/protocompile/ast"
	"github.com/jhump/protocompile/parser"
	"github.com/jhump/protocompile/reporter"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// errBreak is a sentinel error used to break out of an ast.Walk without
// returning an actionable error.
var errBreak = errors.New("break")

type engine struct {
	logger               *zap.Logger
	container            appflag.Container
	moduleConfigReader   bufwire.ModuleConfigReader
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder
	imageBuilder         bufimagebuild.Builder
}

func newEngine(
	logger *zap.Logger,
	container appflag.Container,
	moduleConfigReader bufwire.ModuleConfigReader,
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder,
	imageBuilder bufimagebuild.Builder,
) *engine {
	return &engine{
		logger:               logger,
		container:            container,
		moduleConfigReader:   moduleConfigReader,
		moduleFileSetBuilder: moduleFileSetBuilder,
		imageBuilder:         imageBuilder,
	}
}

func (e *engine) Definition(ctx context.Context, location Location) (_ Location, retErr error) {
	externalPath := location.Path()
	moduleFileSet, image, err := e.buildForExternalPath(ctx, externalPath)
	if err != nil {
		return nil, err
	}
	moduleFile, err := moduleFileForExternalPath(ctx, moduleFileSet, image, externalPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, moduleFile.Close())
	}()
	fileNode, err := parser.Parse(moduleFile.ExternalPath(), moduleFile, reporter.NewHandler(nil))
	if err != nil {
		return nil, err
	}
	// TODO: We currently iterate O(n) in the file to determine what identifier the
	// location points to. We can do a lot better - O(logn) at least and O(1) at best.
	// We might be able to add functionality to protocompile so that it can resolve a
	// node from a span more efficiently.
	ancestorTracker := new(ast.AncestorTracker)
	var node ast.TerminalNode
	var parentNode ast.Node
	var messagePath []*ast.MessageNode
	visitor := &ast.SimpleVisitor{
		DoVisitTerminalNode: func(terminalNode ast.TerminalNode) error {
			info := fileNode.NodeInfo(terminalNode)
			if locationIsWithinNode(location, info) {
				// At this point, the terminal node can only represent
				// a couple different things - any of the keywords, primitive
				// types, or an identifier (e.g. pet.v1.Pet).
				node = terminalNode
				parentNode = ancestorTracker.Parent()
				for _, parent := range ancestorTracker.Path() {
					// Capture all of the messages in the parent path
					// so that we can recover the message's full name
					// (i.e. for nested messages).
					messageNode, ok := parent.(*ast.MessageNode)
					if !ok {
						continue
					}
					messagePath = append(messagePath, messageNode)
				}
				return errBreak
			}
			return nil
		},
	}
	if err := ast.Walk(fileNode, visitor, ancestorTracker.AsWalkOptions()...); err != nil && !errors.Is(err, errBreak) {
		return nil, err
	}
	if node == nil {
		return nil, newCannotResolveLocationError(location)
	}
	identifier, err := resolveIdentifierFromNode(
		location,
		image,
		fileNode,
		node,
		parentNode,
		messagePath,
	)
	if err != nil {
		return nil, err
	}
	return e.findLocationForIdentifier(
		ctx,
		location,
		moduleFileSet,
		image,
		fileNode,
		identifier,
	)
}

// resolvedIdentifierFromNode returns the full name of the descriptor associated with
// the given node and/or parent node (e.g. pet.v1.Pet).
//
// TODO: The protocompile library already needs to perform the reference resolution
// algorithm (i.e. during the linking phase). We can simplify a lot of this logic by
// depending on protocompile's implementation rather than reimplementing our own
// version here.
func resolveIdentifierFromNode(
	location Location,
	image bufimage.Image,
	fileNode *ast.FileNode,
	node ast.TerminalNode,
	parentNode ast.Node,
	messagePath []*ast.MessageNode,
) (string, error) {
	identNode, ok := node.(*ast.IdentNode)
	if !ok {
		// This node isn't an identifier, so it must be a keyword, a separator
		// (e.g. '.' or ','), or a primitive literal (the other valid values of
		// ast.TerminalNode).
		//
		// There's nothing for us to do (i.e. there isn't anywhere we can jump
		// to to show where the message keyword is defined).
		return "", newCannotResolveLocationError(location)
	}
	var identifier string
	if identNode != nil {
		identifier = string(identNode.AsIdentifier())
	}
	if compoundIdentNode, ok := parentNode.(*ast.CompoundIdentNode); ok {
		// If the parent is a *ast.CompoundIdentNode then it either represents
		// a nested descriptor, or a descriptor from another package.
		//
		// In either case, we use *ast.IdentNode to recognize where the user's
		// cursor is, and include all of the other children up to (and including)
		// that identifier so that it's appropriately scoped.
		//
		// For example, the following cursor positions resolve to the following
		// descriptors:
		//
		//  * foo.v1.[F]oo.Bar => foo.v1.Foo
		//  * foo.v1.Foo.[B]ar => foo.v1.Foo.Bar
		//
		var compoundIdentifier string
		if compoundIdentNode.LeadingDot != nil {
			compoundIdentifier += "."
		}
		for i, component := range compoundIdentNode.Components {
			compoundIdentifier += component.Val
			if component == identNode {
				// This is the component that the user's cursor is on,
				// so we stop here.
				break
			}
			if len(compoundIdentNode.Dots) > i {
				// The length of Dots is always one less than the length
				// of Components.
				compoundIdentifier += "."
			}
		}
		identifier = compoundIdentifier
	}
	if len(identifier) == 0 {
		// Unreachable, but included for additional safety.
		return "", newCannotResolveLocationError(location)
	}
	if strings.HasPrefix(identifier, ".") {
		// If the identifier has a leading dot, then the descriptor must already
		// be fully-qualified. We work with identifiers in terms of their
		// full name (i.e. the protoreflect.FullName representation), so we
		// can simply trim the leading dot, and return early.
		return strings.TrimPrefix(identifier, "."), nil
	}
	// At this point, the identifier might represent a nested descriptor in the same file.
	// Unfortunately, it's not enough to consult the *ast.AncestorTracker used in the
	// ast.Walk - the identifier could represent a nested message in another message defined
	// at the top-level.
	//
	// For example,
	//
	//  package foo.v1;
	//
	//  message Foo {
	//    foo.v1.Bar.Baz baz = 1;
	//  }
	//
	//  message Bar {
	//    message Baz {}
	//  }
	//
	identifierComponents := strings.Split(identifier, ".")
	var resolvedIdentifier bool
	if len(messagePath) > 0 {
		for i := len(messagePath) - 1; i >= 0; i-- {
			// Start from the innermost message so that we preserve
			// Protobuf's scoping precedence rules.
			//
			// All of the messages and enums defined in the current
			// message could define the descriptor we're looking for.
			if resolvedIdentifier {
				break
			}
			var messageElements []ast.MessageElement
			for _, decl := range messagePath[i].Decls {
				switch node := decl.(type) {
				case *ast.EnumNode, *ast.GroupNode, *ast.MessageNode:
					messageElements = append(messageElements, node)
				}
			}
			for _, messageElement := range messageElements {
				if _, ok := findNestedDescriptor(fileNode, messageElement, identifierComponents...); ok {
					// A nested message can be referenced in the same way as
					// a top-level message, so we need to consult the other
					// messages defined in the same scope (but only at the
					// innermost level).
					//
					// For example,
					//
					//  message Foo {
					//    message Bar {
					//      string baz = 1;
					//    }
					//    // This message is referencing the nested Foo.Bar
					//    // message, not the top-level .Bar message.
					//    Bar bar = 1;
					//  }
					//
					//  message Bar {}
					//
					for j := i; j >= 0; j-- {
						// Start from the innermost message so that we format
						// the name correctly. We only go up until the element
						// in the messagePath that finished the resolution.
						identifier = messagePath[j].Name.Val + "." + identifier
					}
					// If the identifier represents a nested message in the
					// same package, then we need to preprend the package prefix.
					//
					// Otherwise, the identifier must already have the package
					// prefix to the package where it's defined.
					resolvedIdentifier = true
					break
				}
			}
		}
		if !resolvedIdentifier {
			// We weren't able to resolve the identifier from a nested message
			// in the messagePath, so we next need to check if the nested message
			// is referenced by its full name (excluding the package prefix). To
			// do so, we start from all the top-level messages and verify if a
			// type with that name exists.
			//
			// For example,
			//
			//  package foo.v1;
			//
			//  message Foo {
			//    Bar.Baz baz = 1;
			//  }
			//
			//  message Bar {
			//    message Baz {}
			//  }
			//
			for _, decl := range fileNode.Decls {
				switch node := decl.(type) {
				case *ast.EnumNode, *ast.MessageNode:
					messageElement, ok := node.(ast.MessageElement)
					if !ok {
						// Unreachable, but included for additional safety.
						return "", fmt.Errorf("expected a message element, got %T", node)
					}
					if _, ok := findNestedDescriptor(fileNode, messageElement, identifierComponents...); ok {
						resolvedIdentifier = true
						break
					}
				}
			}
		}
	}
	packageName := packageNameForFile(fileNode)
	if len(packageName) == 0 {
		// The identifier must already be fully-qualified, or its
		// part of the default, empty package.
		return identifier, nil
	}
	if resolvedIdentifier {
		packageNamePrefix := packageName + "."
		if !strings.HasPrefix(identifier, packageNamePrefix) {
			// The identifier might already contain the package
			// prefix even if the descriptor is defined in the
			// same package.
			//
			// For example,
			//
			//  package foo.v1;
			//
			//  message Foo {
			//    foo.v1.Bar bar = 1;
			//  }
			//
			//  message Bar {}
			//
			identifier = packageNamePrefix + identifier
		}
		return identifier, nil
	}
	// At this point, we know that the identifier isn't defined in the current
	// file, so we continue with the reference resolution algorithm and search
	// for a valid reference in the package hierarchy.
	scopeSplit := strings.Split(packageName, ".")
	for i := len(scopeSplit); i >= 0; i-- {
		candidateScope := strings.Join(scopeSplit[:i], ".")
		var fileDescriptorProtos []*descriptorpb.FileDescriptorProto
		for _, imageFile := range image.Files() {
			fileDescriptorProto := imageFile.Proto()
			if fileDescriptorProto.GetPackage() != candidateScope {
				continue
			}
			fileDescriptorProtos = append(fileDescriptorProtos, fileDescriptorProto)
		}
		if len(fileDescriptorProtos) > 0 {
			for _, fileDescriptorProto := range fileDescriptorProtos {
				foundIdentifier, matchedFirstComponent := isNestedDescriptorFromFile(fileDescriptorProto, identifierComponents...)
				if foundIdentifier {
					descriptorName := strings.Join(identifierComponents, ".")
					return candidateScope + "." + descriptorName, nil
				}
				if matchedFirstComponent {
					// If the first component was matched and we failed to find the entire identifier,
					// this is a failure.
					return "", newCannotResolveLocationError(location)
				}
			}
		}
	}
	return identifier, nil
}

// findLocationForIdentifier returns the location of the node identified by the identValueNode based
// on the file node. Note that the retruned location will not always be found in the given fileNode - it
// will often be defined in another file in the module, or one of the module's dependencies.
func (e *engine) findLocationForIdentifier(
	ctx context.Context,
	inputLocation Location,
	moduleFileSet bufmodule.ModuleFileSet,
	image bufimage.Image,
	fileNode *ast.FileNode,
	identifier string,
) (_ Location, retErr error) {
	if len(identifier) == 0 {
		return nil, errors.New("identifier must be non-empty")
	}
	files, err := protodesc.NewFiles(bufimage.ImageToFileDescriptorSet(image))
	if err != nil {
		return nil, err
	}
	identifierFullName := protoreflect.FullName(identifier)
	descriptor, err := files.FindDescriptorByName(identifierFullName)
	if err != nil && !errors.Is(err, protoregistry.NotFound) {
		return nil, err
	}
	if errors.Is(err, protoregistry.NotFound) {
		// TODO: The identifier is either a [custom] option, or one of the well-known types.
		//
		// If the identifier is a WKT, we might want to initialize the local module cache
		// with a synthesized version of the well-known types that we can always jump to
		// (e.g. ~/.cache/buf/v1/wkt).
		return nil, newCannotResolveLocationError(inputLocation)
	}
	// Now that we know where the identifier is defined, parse the
	// file into an AST to locate where it's defined.
	//
	// By default, we assume that the file we've already parsed is
	// the same file that defines the identifier so that we don't
	// unnecessarily parse the same file more than once.
	parentFileNode := fileNode
	parentFilePath := descriptor.ParentFile().Path()
	parentModuleFile, err := moduleFileSet.GetModuleFile(ctx, parentFilePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, parentModuleFile.Close())
	}()
	if fileNode.Name() != parentModuleFile.ExternalPath() {
		// We only need to parse the file if it's different than the input.
		parentFileNode, err = parser.Parse(parentModuleFile.ExternalPath(), parentModuleFile, reporter.NewHandler(nil))
		if err != nil {
			return nil, err
		}
	}
	// We manually iterate over the file's ast.MessageElement values
	// so that we can more clearly track the path from the descriptor's
	// name to the associated ast.NodeInfo.
	//
	// Alternatively, we could use ast.Walk in combination with the
	// *ast.AncestorTracker, but we would need to compare the tracker's
	// path with the target path for every visited message and/or enum node.
	packageNamePrefix := packageNameForFile(parentFileNode) + "."
	descriptorName := strings.TrimPrefix(identifier, packageNamePrefix)
	descriptorNameComponents := strings.Split(descriptorName, ".")
	var nodeInfo *ast.NodeInfo
	for _, decl := range parentFileNode.Decls {
		switch node := decl.(type) {
		case *ast.EnumNode, *ast.MessageNode:
			messageElement, ok := node.(ast.MessageElement)
			if !ok {
				// Unreachable, but included for additional safety.
				return nil, fmt.Errorf("expected a message element, got %T", node)
			}
			if targetNodeInfo, ok := findNestedDescriptor(parentFileNode, messageElement, descriptorNameComponents...); ok {
				nodeInfo = &targetNodeInfo
				break
			}
		}
	}
	if nodeInfo == nil {
		// Should be unreachable, but could be an internal error / bug if we get here.
		return nil, fmt.Errorf("could not find %s in %s", identifier, parentModuleFile.ExternalPath())
	}
	start := nodeInfo.Start()
	return newLocation(
		parentModuleFile.ExternalPath(),
		start.Line,
		start.Col,
	)
}

// buildForExternalPath returns the ModuleFileSet that defines the ModuleFile identified by
// the given path, as well as the Image that contains the transitive closure of files that
// can be reached from the path.
func (e *engine) buildForExternalPath(
	ctx context.Context,
	externalPath string,
) (_ bufmodule.ModuleFileSet, _ bufimage.Image, retErr error) {
	refParser := buffetch.NewRefParser(
		e.logger,
		buffetch.RefParserWithProtoFileRefAllowed(),
	)
	sourceRef, err := refParser.GetSourceRef(ctx, externalPath)
	if err != nil {
		return nil, nil, err
	}
	moduleConfigs, err := e.moduleConfigReader.GetModuleConfigs(
		ctx,
		e.container,
		sourceRef,
		"",
		nil,
		nil,
		false,
	)
	if err != nil {
		return nil, nil, err
	}
	if len(moduleConfigs) == 0 {
		// Unreachable - we should always have at least one module.
		return nil, nil, fmt.Errorf("could not build module for %s", externalPath)
	}
	// We only want to build a single ModuleFileSet and Image (for performance).
	// The Module that defines the externalPath as a target file is able to reach
	// all of the references we need, so we only need to build that one.
	for _, moduleConfig := range moduleConfigs {
		fileInfos, err := moduleConfig.Module().TargetFileInfos(ctx)
		if err != nil {
			return nil, nil, err
		}
		var found bool
		for _, fileInfo := range fileInfos {
			if fileInfo.ExternalPath() == externalPath {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		moduleFileSet, err := e.moduleFileSetBuilder.Build(
			ctx,
			moduleConfig.Module(),
			bufmodulebuild.WithWorkspace(moduleConfig.Workspace()),
		)
		if err != nil {
			return nil, nil, err
		}
		image, fileAnnotations, err := e.imageBuilder.Build(
			ctx,
			moduleFileSet,
		)
		if err != nil {
			return nil, nil, err
		}
		if len(fileAnnotations) > 0 {
			return nil, nil, fileAnnotationsToError(fileAnnotations)
		}
		return moduleFileSet, image, nil
	}
	// Unreacable - if we got this far, we should always find the module.
	return nil, nil, fmt.Errorf("could not find %s in any module", externalPath)
}

// findNestedDescriptor returns the ast.NodeInfo associated with the given
// identifierComponents, if any. This function recursively searches through
// the given messageElement, popping the first identifier component off the
// list to approach the base case.
//
// We use the ast.MessageElement type here so that it permits *ast.MessageNode,
// *ast.EnumNode, and *ast.GroupNode values. We validate that those types are the
// only ones permitted.
func findNestedDescriptor(fileNode *ast.FileNode, messageElement ast.MessageElement, identifierComponents ...string) (ast.NodeInfo, bool) {
	if len(identifierComponents) == 0 {
		return ast.NodeInfo{}, false
	}
	targetName := identifierComponents[0]
	if len(identifierComponents) == 1 {
		var name *ast.IdentNode
		switch node := messageElement.(type) {
		case *ast.EnumNode:
			name = node.Name
		case *ast.MessageNode:
			name = node.Name
		case *ast.GroupNode:
			name = node.Name
		}
		if name.Val != targetName {
			return ast.NodeInfo{}, false
		}
		return fileNode.NodeInfo(name), true
	}
	// We need to recurse into the nested message definitions,
	// which could either be a standard nested message, or a
	// group (for "proto2").
	var name string
	var messageBody ast.MessageBody
	switch node := messageElement.(type) {
	case *ast.GroupNode:
		name = node.Name.Val
		messageBody = node.MessageBody
	case *ast.MessageNode:
		name = node.Name.Val
		messageBody = node.MessageBody
	}
	if name != targetName {
		return ast.NodeInfo{}, false
	}
	for _, messageDecl := range messageBody.Decls {
		switch nestedNode := messageDecl.(type) {
		case *ast.EnumNode, *ast.GroupNode, *ast.MessageNode:
			if nodeInfo, ok := findNestedDescriptor(fileNode, nestedNode, identifierComponents[1:]...); ok {
				return nodeInfo, true
			}
		}
	}
	return ast.NodeInfo{}, false
}

// isNestedDescriptorFromFile is behaviorally similar to findNestedDescriptor, but it's tailored
// to the upstream *descriptorpb.FileDescriptorProto type contained within the bufimage.Image.
//
// We only need to search for the identifier at the top-level in this case.
func isNestedDescriptorFromFile(fileDescriptorProto *descriptorpb.FileDescriptorProto, identifierComponents ...string) (bool, bool) {
	if len(identifierComponents) == 0 {
		return false, false
	}
	name := identifierComponents[0]
	for _, descriptorProto := range fileDescriptorProto.GetMessageType() {
		if descriptorProto.GetName() == name {
			if len(identifierComponents) == 1 {
				return true, true
			}
			return isNestedDescriptorFromMessage(descriptorProto, identifierComponents[1:]...), true
		}
	}
	for _, enumDescriptorProto := range fileDescriptorProto.GetEnumType() {
		if len(identifierComponents) == 1 && enumDescriptorProto.GetName() == name {
			// The enum can only match if it's the last component we're looking for.
			return true, true
		}
	}
	return false, false
}

// isNestedDescriptorFromMessage acts the same as isNestedDescriptorFromFile, but is used
// for *descriptorpb.DescriptorProto types.
func isNestedDescriptorFromMessage(descriptorProto *descriptorpb.DescriptorProto, identifierComponents ...string) bool {
	if len(identifierComponents) == 0 {
		return false
	}
	name := identifierComponents[0]
	for _, nestedDescriptorProto := range descriptorProto.GetNestedType() {
		if nestedDescriptorProto.GetName() == name {
			if len(identifierComponents) == 1 {
				return true
			}
			return isNestedDescriptorFromMessage(nestedDescriptorProto, identifierComponents[1:]...)
		}
	}
	for _, enumDescriptorProto := range descriptorProto.GetEnumType() {
		if len(identifierComponents) == 1 && enumDescriptorProto.GetName() == name {
			// The enum can only match if it's the last component we're looking for.
			return true
		}
	}
	return false
}

// moduleFileForExternalPath returns the ModuleFile associated with the given
// externalPath in the ModuleFileSet. We use the Image here so that we only
// iterate over the reachable files.
func moduleFileForExternalPath(
	ctx context.Context,
	moduleFileSet bufmodule.ModuleFileSet,
	image bufimage.Image,
	externalPath string,
) (bufmodule.ModuleFile, error) {
	for _, fileInfo := range image.Files() {
		if externalPath != fileInfo.ExternalPath() {
			continue
		}
		moduleFile, err := moduleFileSet.GetModuleFile(
			ctx,
			fileInfo.Path(),
		)
		if err != nil {
			return nil, err
		}
		return moduleFile, nil
	}
	// TODO: https://github.com/bufbuild/buf/issues/1056
	//
	// This will only happen if a buf.work.yaml exists in a parent
	// directory, but it does not contain the target file.
	//
	// This is also a problem for other commands that interact
	// with buffetch.ProtoFileRef.
	return nil, fmt.Errorf(
		"input %s was not found - is the directory containing this file defined in your %s?",
		externalPath,
		bufwork.ExternalConfigV1FilePath,
	)
}

// packageNameForFile returns the package name defined in the given *ast.FileNode,
// if any.
func packageNameForFile(fileNode *ast.FileNode) string {
	for _, fileElement := range fileNode.Decls {
		packageNode, ok := fileElement.(*ast.PackageNode)
		if ok {
			return string(packageNode.Name.AsIdentifier())
		}
	}
	return ""
}

// locationIsWithinNode returns true if the given location is contained
// within the node.
func locationIsWithinNode(location Location, nodeInfo ast.NodeInfo) bool {
	var (
		start = nodeInfo.Start()
		end   = nodeInfo.End()
	)
	// This is an "open range", so the locaton.Column() must be strictly
	// less than the end.Col.
	return location.Line() >= start.Line && location.Line() <= end.Line && location.Column() >= start.Col && location.Column() < end.Col
}

// fileAnnotationsToError maps the given fileAnnotations into an error.
func fileAnnotationsToError(fileAnnotations []bufanalysis.FileAnnotation) error {
	buffer := bytes.NewBuffer(nil)
	if err := bufanalysis.PrintFileAnnotations(
		buffer,
		fileAnnotations,
		bufanalysis.FormatText.String(),
	); err != nil {
		return err
	}
	// It's important that we trim the trailing newline so that the CLI
	// (and other clients) don't receive a trailing newline in their
	// error messages.
	return errors.New(strings.TrimSuffix(buffer.String(), "\n"))
}

// newCannotResolveLocationError returns an error that describes that the location
// could not be resolved.
func newCannotResolveLocationError(location Location) error {
	return fmt.Errorf("could not resolve definition for location %s", location)
}
