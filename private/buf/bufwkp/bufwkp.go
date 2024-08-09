// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufwkp

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/bufplugin-go/check"
	"github.com/gofrs/uuid/v5"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	v2ServicePascalCaseRuleSpec = &check.RuleSpec{
		ID: "SERVICE_PASCAL_CASE",
		Categories: []string{
			"BASIC",
			"DEFAULT",
		},
		Purpose: "Checks that services are PascalCase.",
		Type:    check.RuleTypeLint,
		Handler: check.RuleHandlerFunc(handleV2ServicePascalCase),
	}

	v2Spec = &check.Spec{
		Rules: []*check.RuleSpec{
			v2ServicePascalCaseRuleSpec,
		},
		Before: before,
	}
)

type protosourceFilesContextKey struct{}
type againstProtosourceFilesContextKey struct{}

func handleV2ServicePascalCase(
	_ context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
) error {
	return nil
}

func newRuleHandler(
	f func(
		ctx context.Context,
		responseWriter check.ResponseWriter,
		request check.Request,
		files []bufprotosource.File,
		againstFiles []bufprotosource.File,
	) error,
) check.RuleHandler {
	return check.RuleHandlerFunc(
		func(
			ctx context.Context,
			responseWriter check.ResponseWriter,
			request check.Request,
		) error {
			return f(
				ctx,
				responseWriter,
				request,
				// Is this OK with nil?
				ctx.Value(protosourceFilesContextKey{}).([]bufprotosource.File),
				// Is this OK with nil?
				ctx.Value(againstProtosourceFilesContextKey{}).([]bufprotosource.File),
			)
		},
	)
}

func before(
	ctx context.Context,
	request check.Request,
) (context.Context, check.Request, error) {
	protosourceFiles, err := protosourceFilesForFiles(ctx, request.Files())
	if err != nil {
		return nil, nil, err
	}
	againstProtosourceFiles, err := protosourceFilesForFiles(ctx, request.Files())
	if err != nil {
		return nil, nil, err
	}
	if len(protosourceFiles) > 0 {
		ctx = context.WithValue(ctx, protosourceFilesContextKey{}, protosourceFiles)
	}
	if len(againstProtosourceFiles) > 0 {
		ctx = context.WithValue(ctx, againstProtosourceFilesContextKey{}, againstProtosourceFiles)
	}
	return ctx, request, nil
}

func protosourceFilesForFiles(ctx context.Context, files []check.File) ([]bufprotosource.File, error) {
	if len(files) == 0 {
		return nil, nil
	}
	resolver, err := newResolver(files)
	if err != nil {
		return nil, err
	}
	return bufprotosource.NewFiles(ctx, slicesext.Map(files, newInputFile), resolver)
}

func newResolver(files []check.File) (protodesc.Resolver, error) {
	return protodesc.NewFiles(
		&descriptorpb.FileDescriptorSet{
			File: slicesext.Map(files, check.File.FileDescriptorProto),
		},
	)
}

type inputFile struct {
	check.File
}

func newInputFile(file check.File) *inputFile {
	return &inputFile{
		File: file,
	}
}

func (i *inputFile) Path() string {
	return i.File.FileDescriptorProto().GetName()
}

// TODO: We will need to reconcile this on the client-side as right now we rely on ExternalPath
// being passed end-to-end.
func (i *inputFile) ExternalPath() string {
	return i.Path()
}

func (i *inputFile) ModuleFullName() bufmodule.ModuleFullName {
	return nil
}

func (i *inputFile) CommitID() uuid.UUID {
	return uuid.Nil
}
