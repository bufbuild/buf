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

package buflintvalidate

import (
	"sync"

	"buf.build/go/protovalidate"
	celpv "buf.build/go/protovalidate/cel"
	"cel.dev/cel-go/cel"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Checker checks protovalidate rules.
//
// A Checker holds state that is reused across checks, so it should be created
// once per set of files being checked and then discarded.
type Checker struct {
	extensionTypeResolver protoencoding.Resolver
	exampleValidator      protovalidate.Validator
	baseCELEnv            *cel.Env

	mu          sync.Mutex
	fileCELEnvs map[string]*cel.Env
}

// NewChecker returns a new Checker.
//
// The extensionTypeResolver must be able to resolve every predefined rule
// extension used by the checked files, including those from imported files.
func NewChecker(extensionTypeResolver protoencoding.Resolver) (*Checker, error) {
	exampleValidator, err := protovalidate.New(
		protovalidate.WithExtensionTypeResolver(extensionTypeResolver),
	)
	if err != nil {
		return nil, err
	}
	baseCELEnv, err := cel.NewEnv(cel.Lib(celpv.NewLibrary()))
	if err != nil {
		return nil, err
	}
	return &Checker{
		extensionTypeResolver: extensionTypeResolver,
		exampleValidator:      exampleValidator,
		baseCELEnv:            baseCELEnv,
		fileCELEnvs:           make(map[string]*cel.Env),
	}, nil
}

// CheckMessage validates that all rules on the message are valid, and any CEL expressions compile.
// It also checks all predefined rule extensions on the messages.
func (c *Checker) CheckMessage(
	// addAnnotationFunc adds an annotation with the descriptor and location for check results.
	addAnnotationFunc func(bufprotosource.Descriptor, bufprotosource.Location, []bufprotosource.Location, string, ...any),
	message bufprotosource.Message,
) error {
	messageDescriptor, err := message.AsDescriptor()
	if err != nil {
		return err
	}
	messageRules, err := protovalidate.ResolveMessageRules(messageDescriptor)
	if err != nil {
		return err
	}
	if messageRules == nil {
		return nil
	}
	checkOneofRulesForMessage(addAnnotationFunc, messageRules, messageDescriptor, message)
	return c.checkCELForMessage(
		addAnnotationFunc,
		messageRules,
		messageDescriptor,
		message,
	)
}

// CheckField validates that all rules on the field are valid, and any CEL expressions compile.
//
// For a set of rules to be valid, it must
//  1. permit _some_ value and all example values, if any
//  2. have a type compatible with the field it validates.
func (c *Checker) CheckField(
	// addAnnotationFunc adds an annotation with the descriptor and location for check results.
	addAnnotationFunc func(bufprotosource.Descriptor, bufprotosource.Location, []bufprotosource.Location, string, ...any),
	field bufprotosource.Field,
) error {
	return c.checkField(addAnnotationFunc, field)
}

// CheckPredefinedRuleExtension checks that a predefined extension is valid, and any CEL expressions compile.
func (c *Checker) CheckPredefinedRuleExtension(
	// addAnnotationFunc adds an annotation with the descriptor and location for check results.
	addAnnotationFunc func(bufprotosource.Descriptor, bufprotosource.Location, []bufprotosource.Location, string, ...any),
	field bufprotosource.Field,
) error {
	return c.checkPredefinedRuleExtension(addAnnotationFunc, field)
}

// celEnvForFile returns a CEL environment with the types from the file and
// its transitive imports registered. This matches the types registered by
// cel.Types for a message in the file, but registering them is expensive, so
// the environment is built once per file.
func (c *Checker) celEnvForFile(fileDescriptor protoreflect.FileDescriptor) (*cel.Env, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if celEnv, ok := c.fileCELEnvs[fileDescriptor.Path()]; ok {
		return celEnv, nil
	}
	celEnv, err := c.baseCELEnv.Extend(cel.TypeDescs(transitiveFileDescriptors(fileDescriptor)...))
	if err != nil {
		return nil, err
	}
	c.fileCELEnvs[fileDescriptor.Path()] = celEnv
	return celEnv, nil
}

func transitiveFileDescriptors(fileDescriptor protoreflect.FileDescriptor) []any {
	seenPaths := map[string]struct{}{
		fileDescriptor.Path(): {},
	}
	typeDescs := []any{fileDescriptor}
	pending := []protoreflect.FileDescriptor{fileDescriptor}
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		imports := current.Imports()
		for i := range imports.Len() {
			importFileDescriptor := imports.Get(i).FileDescriptor
			if _, ok := seenPaths[importFileDescriptor.Path()]; ok {
				continue
			}
			seenPaths[importFileDescriptor.Path()] = struct{}{}
			typeDescs = append(typeDescs, importFileDescriptor)
			pending = append(pending, importFileDescriptor)
		}
	}
	return typeDescs
}
