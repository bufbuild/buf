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

package bufcheckserverhandle

import (
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufcheck/internal/bufcheckopt"
	"github.com/bufbuild/buf/private/buf/bufcheck/internal/bufcheckserver/internal/bufcheckserverutil"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	// HandleLintCommentEnum is a handle function.
	HandleLintCommentEnum = bufcheckserverutil.NewLintEnumRuleHandler(handleLintCommentEnum)
	// HandleLintCommentEnumValue is a handle function.
	HandleLintCommentEnumValue = bufcheckserverutil.NewLintEnumValueRuleHandler(handleLintCommentEnumValue)
	// HandleLintCommentField is a handle function.
	HandleLintCommentField = bufcheckserverutil.NewLintFieldRuleHandler(handleLintCommentField)
	// HandleLintCommentMessage is a handle function.
	HandleLintCommentMessage = bufcheckserverutil.NewLintMessageRuleHandler(handleLintCommentMessage)
	// HandleLintCommentOneof is a handle function.
	HandleLintCommentOneof = bufcheckserverutil.NewLintOneofRuleHandler(handleLintCommentOneof)
	// HandleLintCommentService is a handle function.
	HandleLintCommentService = bufcheckserverutil.NewLintServiceRuleHandler(handleLintCommentService)
	// HandleLintCommentRPC is a handle function.
	HandleLintCommentRPC = bufcheckserverutil.NewLintMethodRuleHandler(handleLintCommentRPC)
)

func handleLintCommentEnum(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	value bufprotosource.Enum,
) error {
	return handleLintCommentNamedDescriptor(responseWriter, request, value, "Enum")
}

func handleLintCommentEnumValue(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	value bufprotosource.EnumValue,
) error {
	return handleLintCommentNamedDescriptor(responseWriter, request, value, "Enum value")
}

func handleLintCommentField(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	value bufprotosource.Field,
) error {
	if value.ParentMessage() != nil && value.ParentMessage().IsMapEntry() {
		// Don't handle synthetic fields for map entries. They have no comments.
		return nil
	}
	if value.Type() == descriptorpb.FieldDescriptorProto_TYPE_GROUP {
		// Group fields also have no comments: comments in source get
		// attributed to the nested message, not the field.
		return nil
	}
	return handleLintCommentNamedDescriptor(responseWriter, request, value, "Field")
}

func handleLintCommentMessage(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	value bufprotosource.Message,
) error {
	if value.IsMapEntry() {
		// Don't handle synthetic map entries. They have no comments.
		return nil
	}
	return handleLintCommentNamedDescriptor(responseWriter, request, value, "Message")
}

func handleLintCommentOneof(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	value bufprotosource.Oneof,
) error {
	oneofDescriptor, err := value.AsDescriptor()
	if err == nil && oneofDescriptor.IsSynthetic() {
		// Don't handle synthetic oneofs (for proto3-optional fields). They have no comments.
		return nil
	}
	return handleLintCommentNamedDescriptor(responseWriter, request, value, "Oneof")
}

func handleLintCommentRPC(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	value bufprotosource.Method,
) error {
	return handleLintCommentNamedDescriptor(responseWriter, request, value, "RPC")
}

func handleLintCommentService(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	value bufprotosource.Service,
) error {
	return handleLintCommentNamedDescriptor(responseWriter, request, value, "Service")
}

func handleLintCommentNamedDescriptor(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	namedDescriptor bufprotosource.NamedDescriptor,
	typeName string,
) error {
	location := namedDescriptor.Location()
	if location == nil {
		// this will magically skip map entry fields as well as a side-effect, although originally unintended
		return nil
	}
	// Note that this does result in us parsing the comment excludes on every call to the rule.
	// This is theoretically inefficient, but shouldn't have any real world impact. The alternative
	// is doing a custom parse of all the options we expect within bufcheckopt, and attaching
	// this result to the bufcheckserverutil.Request, which gets us even further from the bufplugin-go
	// SDK. We want to do what is native as much as possible. If this is a real performance problem,
	// we can update in the future.
	commentExcludes, err := bufcheckopt.GetCommentExcludes(request.Options())
	if err != nil {
		return err
	}
	if !validLeadingComment(commentExcludes, location.LeadingComments()) {
		responseWriter.AddProtosourceAnnotation(
			location,
			nil,
			"%s %q should have a non-empty comment for documentation.",
			typeName,
			namedDescriptor.Name(),
		)
	}
	return nil
}

// HandleLintDirectorySamePackage is a handle function.
var HandleLintDirectorySamePackage = bufcheckserverutil.NewLintDirPathToFilesRuleHandler(handleLintDirectorySamePackage)

func handleLintDirectorySamePackage(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	dirPath string,
	dirFiles []bufprotosource.File,
) error {
	pkgMap := make(map[string]struct{})
	for _, file := range dirFiles {
		// works for no package set as this will result in "" which is a valid map key
		pkgMap[file.Package()] = struct{}{}
	}
	if len(pkgMap) > 1 {
		var messagePrefix string
		if _, ok := pkgMap[""]; ok {
			delete(pkgMap, "")
			if len(pkgMap) > 1 {
				messagePrefix = fmt.Sprintf("Multiple packages %q and file with no package", strings.Join(slicesext.MapKeysToSortedSlice(pkgMap), ","))
			} else {
				// Join works with only one element as well by adding no comma
				messagePrefix = fmt.Sprintf("Package %q and file with no package", strings.Join(slicesext.MapKeysToSortedSlice(pkgMap), ","))
			}
		} else {
			messagePrefix = fmt.Sprintf("Multiple packages %q", strings.Join(slicesext.MapKeysToSortedSlice(pkgMap), ","))
		}
		for _, file := range dirFiles {
			responseWriter.AddProtosourceAnnotation(
				file.PackageLocation(),
				nil,
				"%s detected within directory %q.",
				messagePrefix,
				dirPath,
			)
		}
	}
	return nil
}

// HandleLintServicePascalCase is a handle function.
var HandleLintServicePascalCase = bufcheckserverutil.NewLintServiceRuleHandler(handleLintServicePascalCase)

func handleLintServicePascalCase(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	service bufprotosource.Service,
) error {
	name := service.Name()
	expectedName := stringutil.ToPascalCase(name)
	if name != expectedName {
		responseWriter.AddProtosourceAnnotation(
			service.NameLocation(),
			nil,
			"Service name %q should be PascalCase, such as %q.",
			name,
			expectedName,
		)
	}
	return nil
}

// HandleLintServiceSuffix is a handle function.
var HandleLintServiceSuffix = bufcheckserverutil.NewLintServiceRuleHandler(handleLintServiceSuffix)

func handleLintServiceSuffix(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	service bufprotosource.Service,
) error {
	suffix, err := bufcheckopt.GetServiceSuffix(request.Options())
	if err != nil {
		return err
	}
	name := service.Name()
	if !strings.HasSuffix(name, suffix) {
		responseWriter.AddProtosourceAnnotation(
			service.NameLocation(),
			nil,
			"Service name %q should be suffixed with %q.",
			name,
			suffix,
		)
	}
	return nil
}
