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
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"buf.build/go/bufplugin/check"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufcheckserver/internal/bufcheckserverutil"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufcheckserver/internal/buflintvalidate"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal/bufcheckopt"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/protoversion"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"google.golang.org/protobuf/reflect/protoreflect"
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
			var sourcePath protoreflect.SourcePath
			if packageLocation := file.PackageLocation(); packageLocation != nil {
				sourcePath = packageLocation.SourcePath()
			}
			responseWriter.AddAnnotation(
				check.WithFileNameAndSourcePath(file.Path(), sourcePath),
				check.WithMessagef(
					"%s detected within directory %q.",
					messagePrefix,
					dirPath,
				),
			)
		}
	}
	return nil
}

// HandleLintEnumNoAllowAlias is a handle function.
var HandleLintEnumNoAllowAlias = bufcheckserverutil.NewLintEnumRuleHandler(handleLintEnumNoAllowAlias)

func handleLintEnumNoAllowAlias(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	enum bufprotosource.Enum,
) error {
	if enum.AllowAlias() {
		responseWriter.AddProtosourceAnnotation(
			enum.AllowAliasLocation(),
			nil,
			`Enum option "allow_alias" on enum %q must be false.`,
			enum.Name(),
		)
	}
	return nil
}

// HandleLintEnumPascalCase is a handle function.
var HandleLintEnumPascalCase = bufcheckserverutil.NewLintEnumRuleHandler(handleLintEnumPascalCase)

func handleLintEnumPascalCase(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	enum bufprotosource.Enum,
) error {
	name := enum.Name()
	expectedName := stringutil.ToPascalCase(name)
	if name != expectedName {
		responseWriter.AddProtosourceAnnotation(
			enum.NameLocation(),
			nil,
			"Enum name %q should be PascalCase, such as %q.",
			name,
			expectedName,
		)
	}
	return nil
}

// HandleLintEnumFirstValueZero is a handle function.
var HandleLintEnumFirstValueZero = bufcheckserverutil.NewLintEnumRuleHandler(handleLintEnumFirstValueZero)

func handleLintEnumFirstValueZero(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	enum bufprotosource.Enum,
) error {
	if values := enum.Values(); len(values) > 0 {
		if firstEnumValue := values[0]; firstEnumValue.Number() != 0 {
			// proto3 compilation references the number
			responseWriter.AddProtosourceAnnotation(
				firstEnumValue.NumberLocation(),
				nil,
				"First enum value %q should have a numeric value of 0",
				firstEnumValue.Name(),
			)
		}
	}
	return nil
}

// HandleLintEnumValuePrefix is a handle function.
var HandleLintEnumValuePrefix = bufcheckserverutil.NewLintEnumValueRuleHandler(handleLintEnumValuePrefix)

func handleLintEnumValuePrefix(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	enumValue bufprotosource.EnumValue,
) error {
	name := enumValue.Name()
	expectedPrefix := fieldToUpperSnakeCase(enumValue.Enum().Name()) + "_"
	if !strings.HasPrefix(name, expectedPrefix) {
		responseWriter.AddProtosourceAnnotation(
			enumValue.NameLocation(),
			nil,
			"Enum value name %q should be prefixed with %q.",
			name,
			expectedPrefix,
		)
	}
	return nil
}

// HandleLintEnumValueUpperSnakeCase is a handle function.
var HandleLintEnumValueUpperSnakeCase = bufcheckserverutil.NewLintEnumValueRuleHandler(handleLintEnumValueUpperSnakeCase)

func handleLintEnumValueUpperSnakeCase(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	enumValue bufprotosource.EnumValue,
) error {
	name := enumValue.Name()
	expectedName := fieldToUpperSnakeCase(name)
	if name != expectedName {
		responseWriter.AddProtosourceAnnotation(
			enumValue.NameLocation(),
			nil,
			"Enum value name %q should be UPPER_SNAKE_CASE, such as %q.",
			name,
			expectedName,
		)
	}
	return nil
}

// HandleLintEnumZeroValueSuffix is a handle function.
var HandleLintEnumZeroValueSuffix = bufcheckserverutil.NewLintEnumValueRuleHandler(handleLintEnumZeroValueSuffix)

func handleLintEnumZeroValueSuffix(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	enumValue bufprotosource.EnumValue,
) error {
	suffix, err := bufcheckopt.GetEnumZeroValueSuffix(request.Options())
	if err != nil {
		return err
	}
	request.Options()
	if enumValue.Number() != 0 {
		return nil
	}
	name := enumValue.Name()
	if !strings.HasSuffix(name, suffix) {
		responseWriter.AddProtosourceAnnotation(
			enumValue.NameLocation(),
			nil,
			"Enum zero value name %q should be suffixed with %q.",
			name,
			suffix,
		)
	}
	return nil
}

// HandleLintFieldLowerSnakeCase is a handle function.
var HandleLintFieldLowerSnakeCase = bufcheckserverutil.NewLintFieldRuleHandler(handleLintFieldLowerSnakeCase)

func handleLintFieldLowerSnakeCase(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	field bufprotosource.Field,
) error {
	message := field.ParentMessage()
	if message != nil && message.IsMapEntry() {
		// this check should always pass anyways but just in case
		return nil
	}
	name := field.Name()
	expectedName := fieldToLowerSnakeCase(name)
	if name != expectedName {
		responseWriter.AddProtosourceAnnotation(
			field.NameLocation(),
			nil,
			"Field name %q should be lower_snake_case, such as %q.",
			name,
			expectedName,
		)
	}
	return nil
}

// HandleLintFieldNoDescriptor is a handle function.
var HandleLintFieldNoDescriptor = bufcheckserverutil.NewLintFieldRuleHandler(handleLintFieldNoDescriptor)

func handleLintFieldNoDescriptor(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	field bufprotosource.Field,
) error {
	name := field.Name()
	if strings.ToLower(strings.Trim(name, "_")) == "descriptor" {
		responseWriter.AddProtosourceAnnotation(
			field.NameLocation(),
			nil,
			`Field name %q cannot be any capitalization of "descriptor" with any number of prefix or suffix underscores.`,
			name,
		)
	}
	return nil
}

// HandleLintFieldNotRequired is a handle function.
var HandleLintFieldNotRequired = bufcheckserverutil.NewLintFieldRuleHandler(handleLintFieldNotRequired)

func handleLintFieldNotRequired(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	field bufprotosource.Field,
) error {
	fieldDescriptor, err := field.AsDescriptor()
	if err != nil {
		return err
	}
	// We use the protoreflect field descriptor to handle editions, where the
	// field is set to required using special "features" options, instead of the
	// label on the descriptor proto.
	if fieldDescriptor.Cardinality() == protoreflect.Required {
		responseWriter.AddProtosourceAnnotation(
			field.NameLocation(),
			nil,
			`Field named %q should not be required.`,
		)
	}
	return nil
}

// HandleLintFileLowerSnakeCase is a handle function.
var HandleLintFileLowerSnakeCase = bufcheckserverutil.NewLintFileRuleHandler(handleLintFileLowerSnakeCase)

func handleLintFileLowerSnakeCase(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	file bufprotosource.File,
) error {
	filename := file.Path()
	base := normalpath.Base(filename)
	ext := normalpath.Ext(filename)
	baseWithoutExt := strings.TrimSuffix(base, ext)
	expectedBaseWithoutExt := stringutil.ToLowerSnakeCase(baseWithoutExt)
	if baseWithoutExt != expectedBaseWithoutExt {
		responseWriter.AddAnnotation(
			check.WithFileName(filename),
			check.WithMessagef(`Filename %q should be lower_snake_case%s, such as "%s%s".`,
				base,
				ext,
				expectedBaseWithoutExt,
				ext,
			),
		)
	}
	return nil
}

// HandleLintImportNoPublic is a handle function.
var HandleLintImportNoPublic = bufcheckserverutil.NewLintFileImportRuleHandler(handleLintImportNoPublic)

func handleLintImportNoPublic(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	fileImport bufprotosource.FileImport,
) error {
	if fileImport.IsPublic() {
		responseWriter.AddProtosourceAnnotation(
			fileImport.Location(),
			nil,
			`Import %q must not be public.`,
			fileImport.Import(),
		)
	}
	return nil
}

// HandleLintImportNoWeak is a handle function.
var HandleLintImportNoWeak = bufcheckserverutil.NewLintFileImportRuleHandler(handleLintImportNoWeak)

func handleLintImportNoWeak(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	fileImport bufprotosource.FileImport,
) error {
	if fileImport.IsWeak() {
		responseWriter.AddProtosourceAnnotation(
			fileImport.Location(),
			nil,
			`Import %q must not be weak.`,
			fileImport.Import(),
		)
	}
	return nil
}

// HandleLintImportUsed is a handle function.
var HandleLintImportUsed = bufcheckserverutil.NewLintFileImportRuleHandler(handleLintImportUsed)

func handleLintImportUsed(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	fileImport bufprotosource.FileImport,
) error {
	if fileImport.IsUnused() {
		responseWriter.AddProtosourceAnnotation(
			fileImport.Location(),
			nil,
			`Import %q is unused.`,
			fileImport.Import(),
		)
	}
	return nil
}

// HandleLintMessagePascalCase is a handle function.
var HandleLintMessagePascalCase = bufcheckserverutil.NewLintMessageRuleHandler(handleLintMessagePascalCase)

func handleLintMessagePascalCase(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	message bufprotosource.Message,
) error {
	if message.IsMapEntry() {
		// map entries should always be pascal case but we don't want to check them anyways
		return nil
	}
	name := message.Name()
	expectedName := stringutil.ToPascalCase(name)
	if name != expectedName {
		responseWriter.AddProtosourceAnnotation(
			message.NameLocation(),
			nil,
			"Message name %q should be PascalCase, such as %q.",
			name,
			expectedName,
		)
	}
	return nil
}

// HandleLintOneofLowerSnakeCase is a handle function.
var HandleLintOneofLowerSnakeCase = bufcheckserverutil.NewLintOneofRuleHandler(handleLintOneofLowerSnakeCase)

func handleLintOneofLowerSnakeCase(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	oneof bufprotosource.Oneof,
) error {
	name := oneof.Name()
	expectedName := fieldToLowerSnakeCase(name)
	if name != expectedName {
		// if this is an implicit oneof for a proto3 optional field, do not error
		// https://github.com/protocolbuffers/protobuf/blob/master/docs/implementing_proto3_presence.md
		if fields := oneof.Fields(); len(fields) == 1 {
			if fields[0].Proto3Optional() {
				return nil
			}
		}
		responseWriter.AddProtosourceAnnotation(
			oneof.NameLocation(),
			nil,
			"Oneof name %q should be lower_snake_case, such as %q.",
			name,
			expectedName,
		)
	}
	return nil
}

// HandleLintPackageDefined is a handle function.
var HandleLintPackageDefined = bufcheckserverutil.NewLintFileRuleHandler(handleLintPackageDefined)

func handleLintPackageDefined(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	file bufprotosource.File,
) error {
	if file.Package() == "" {
		responseWriter.AddAnnotation(
			check.WithFileName(file.Path()),
			check.WithMessage("Files must have a package defined."),
		)
	}
	return nil
}

// HandleLintPackageDirectoryMatch is a handle function.
var HandleLintPackageDirectoryMatch = bufcheckserverutil.NewLintFileRuleHandler(handleLintPackageDirectoryMatch)

func handleLintPackageDirectoryMatch(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	file bufprotosource.File,
) error {
	pkg := file.Package()
	if pkg == "" {
		return nil
	}
	expectedDirPath := strings.ReplaceAll(pkg, ".", "/")
	dirPath := normalpath.Dir(file.Path())
	// need to check case where in root relative directory and no package defined
	// this should be valid although if SENSIBLE is turned on this will be invalid
	if dirPath != expectedDirPath {
		responseWriter.AddProtosourceAnnotation(
			file.PackageLocation(),
			nil,
			`Files with package %q must be within a directory "%s" relative to root but were in directory "%s".`,
			pkg,
			normalpath.Unnormalize(expectedDirPath),
			normalpath.Unnormalize(dirPath),
		)
	}
	return nil
}

// HandleLintPackageLowerSnakeCase is a handle function.
var HandleLintPackageLowerSnakeCase = bufcheckserverutil.NewLintFileRuleHandler(handleLintPackageLowerSnakeCase)

func handleLintPackageLowerSnakeCase(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	file bufprotosource.File,
) error {
	pkg := file.Package()
	if pkg == "" {
		return nil
	}
	split := strings.Split(pkg, ".")
	for i, elem := range split {
		split[i] = stringutil.ToLowerSnakeCase(elem)
	}
	expectedPkg := strings.Join(split, ".")
	if pkg != expectedPkg {
		responseWriter.AddProtosourceAnnotation(
			file.PackageLocation(),
			nil,
			"Package name %q should be lower_snake.case, such as %q.",
			pkg,
			expectedPkg,
		)
	}
	return nil
}

// HandleLintPackageNoImportCycle is a handle function.
//
// Note that imports are not skipped via the helper, as we want to detect import cycles
// even if they are within imports, and report on them. If a non-import is part of an
// import cycle, we report it, even if the import cycle includes imports in it.
var HandleLintPackageNoImportCycle = bufcheckserverutil.NewRuleHandler(handleLintPackageNoImportCycle)

func handleLintPackageNoImportCycle(
	_ context.Context,
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
) error {
	files := request.ProtosourceFiles()
	packageToDirectlyImportedPackageToFileImports, err := bufprotosource.PackageToDirectlyImportedPackageToFileImports(files...)
	if err != nil {
		return err
	}
	// This is way more algorithmically complex than it needs to be.
	//
	// We're doing a DFS starting at each package. What we should do is start from any package,
	// do the DFS and keep track of the packages hit, and then don't ever do DFS from a given
	// package twice. The problem is is that with the current janky package -> direct -> file imports
	// setup, we would then end up with error messages like "import cycle: a -> b -> c -> b", and
	// attach the error message to a file with package a, and we want to just print "b -> c -> b".
	// So to get this to market, we just do a DFS from each package.
	//
	// This may prove to be too expensive but early testing say it is not so far.
	for pkg := range packageToDirectlyImportedPackageToFileImports {
		// Can equal "" per the function signature of PackageToDirectlyImportedPackageToFileImports
		if pkg == "" {
			continue
		}
		// Go one deep in the potential import cycle so that we can get the file imports
		// we want to potentially attach errors to.
		//
		// We know that pkg is never equal to directlyImportedPackage due to the signature
		// of PackageToDirectlyImportedPackageToFileImports.
		for directlyImportedPackage, fileImports := range packageToDirectlyImportedPackageToFileImports[pkg] {
			// Can equal "" per the function signature of PackageToDirectlyImportedPackageToFileImports
			if directlyImportedPackage == "" {
				continue
			}
			if importCycle := getImportCycleIfExists(
				directlyImportedPackage,
				packageToDirectlyImportedPackageToFileImports,
				map[string]struct{}{
					pkg: {},
				},
				[]string{
					pkg,
				},
			); len(importCycle) > 0 {
				for _, fileImport := range fileImports {
					// We used newFilesWithImportsCheckFunc, meaning that we did not skip imports.
					// We do not want to report errors on imports.
					if fileImport.File().IsImport() {
						continue
					}
					responseWriter.AddProtosourceAnnotation(
						fileImport.Location(),
						nil,
						`Package import cycle: %s`,
						strings.Join(importCycle, ` -> `),
					)
				}
			}
		}
	}
	return nil
}

// HandleLintPackageSameDirectory is a handle function.
var HandleLintPackageSameDirectory = bufcheckserverutil.NewLintPackageToFilesRuleHandler(handleLintPackageSameDirectory)

func handleLintPackageSameDirectory(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	pkg string,
	pkgFiles []bufprotosource.File,
) error {
	dirMap := make(map[string]struct{})
	for _, file := range pkgFiles {
		dirMap[normalpath.Dir(file.Path())] = struct{}{}
	}
	if len(dirMap) > 1 {
		dirs := slicesext.MapKeysToSortedSlice(dirMap)
		for _, file := range pkgFiles {
			var sourcePath protoreflect.SourcePath
			if packageLocation := file.PackageLocation(); packageLocation != nil {
				sourcePath = packageLocation.SourcePath()
			}
			responseWriter.AddAnnotation(
				check.WithFileNameAndSourcePath(file.Path(), sourcePath),
				check.WithMessagef(
					"Multiple directories %q contain files with package %q.",
					strings.Join(dirs, ","),
					pkg,
				),
			)
		}
	}
	return nil
}

var (
	// HandleLintPackageSameCsharpNamespace is a handle function.
	HandleLintPackageSameCsharpNamespace = bufcheckserverutil.NewLintPackageToFilesRuleHandler(handleLintPackageSameCsharpNamespace)
	// HandleLintPackageSameGoPackage is a handle function.
	HandleLintPackageSameGoPackage = bufcheckserverutil.NewLintPackageToFilesRuleHandler(handleLintPackageSameGoPackage)
	// HandleLintPackageSameJavaMultipleFiles is a handle function.
	HandleLintPackageSameJavaMultipleFiles = bufcheckserverutil.NewLintPackageToFilesRuleHandler(handleLintPackageSameJavaMultipleFiles)
	// HandleLintPackageSameJavaPackage is a handle function.
	HandleLintPackageSameJavaPackage = bufcheckserverutil.NewLintPackageToFilesRuleHandler(handleLintPackageSameJavaPackage)
	// HandleLintPackageSamePhpNamespace is a handle function.
	HandleLintPackageSamePhpNamespace = bufcheckserverutil.NewLintPackageToFilesRuleHandler(handleLintPackageSamePhpNamespace)
	// HandleLintPackageSameRubyPackage is a handle function.
	HandleLintPackageSameRubyPackage = bufcheckserverutil.NewLintPackageToFilesRuleHandler(handleLintPackageSameRubyPackage)
	// HandleLintPackageSameSwiftPrefix is a handle function.
	HandleLintPackageSameSwiftPrefix = bufcheckserverutil.NewLintPackageToFilesRuleHandler(handleLintPackageSameSwiftPrefix)
)

func handleLintPackageSameCsharpNamespace(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	pkg string,
	pkgFiles []bufprotosource.File,
) error {
	return handleLintPackageSameOptionValue(
		responseWriter,
		pkg,
		pkgFiles,
		bufprotosource.File.CsharpNamespace,
		bufprotosource.File.CsharpNamespaceLocation,
		"csharp_namespace",
	)
}

func handleLintPackageSameGoPackage(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	pkg string,
	pkgFiles []bufprotosource.File,
) error {
	return handleLintPackageSameOptionValue(
		responseWriter,
		pkg,
		pkgFiles,
		bufprotosource.File.GoPackage,
		bufprotosource.File.GoPackageLocation,
		"go_package",
	)
}

func handleLintPackageSameJavaMultipleFiles(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	pkg string,
	pkgFiles []bufprotosource.File,
) error {
	return handleLintPackageSameOptionValue(
		responseWriter,
		pkg,
		pkgFiles,
		func(file bufprotosource.File) string {
			// Return empty string when the option is not present, instead of returning a "true" or "false" value.
			if fileOptions := file.FileDescriptor().GetOptions(); fileOptions == nil || fileOptions.JavaMultipleFiles == nil {
				return ""
			}
			return strconv.FormatBool(file.JavaMultipleFiles())
		},
		bufprotosource.File.JavaMultipleFilesLocation,
		"java_multiple_files",
	)
}

func handleLintPackageSameJavaPackage(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	pkg string,
	pkgFiles []bufprotosource.File,
) error {
	return handleLintPackageSameOptionValue(
		responseWriter,
		pkg,
		pkgFiles,
		bufprotosource.File.JavaPackage,
		bufprotosource.File.JavaPackageLocation,
		"java_package",
	)
}

func handleLintPackageSamePhpNamespace(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	pkg string,
	pkgFiles []bufprotosource.File,
) error {
	return handleLintPackageSameOptionValue(
		responseWriter,
		pkg,
		pkgFiles,
		bufprotosource.File.PhpNamespace,
		bufprotosource.File.PhpNamespaceLocation,
		"php_namespace",
	)
}

func handleLintPackageSameRubyPackage(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	pkg string,
	pkgFiles []bufprotosource.File,
) error {
	return handleLintPackageSameOptionValue(
		responseWriter,
		pkg,
		pkgFiles,
		bufprotosource.File.RubyPackage,
		bufprotosource.File.RubyPackageLocation,
		"ruby_package",
	)
}

func handleLintPackageSameSwiftPrefix(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	pkg string,
	pkgFiles []bufprotosource.File,
) error {
	return handleLintPackageSameOptionValue(
		responseWriter,
		pkg,
		pkgFiles,
		bufprotosource.File.SwiftPrefix,
		bufprotosource.File.SwiftPrefixLocation,
		"swift_prefix",
	)
}

func handleLintPackageSameOptionValue(
	responseWriter bufcheckserverutil.ResponseWriter,
	pkg string,
	pkgFiles []bufprotosource.File,
	getFileOptionValue func(bufprotosource.File) string,
	getFileOptionLocation func(bufprotosource.File) bufprotosource.Location,
	name string,
) error {
	optionValueMap := make(map[string]struct{})
	for _, file := range pkgFiles {
		optionValueMap[getFileOptionValue(file)] = struct{}{}
	}
	if len(optionValueMap) > 1 {
		_, noOptionValue := optionValueMap[""]
		delete(optionValueMap, "")
		optionValues := slicesext.MapKeysToSortedSlice(optionValueMap)
		for _, file := range pkgFiles {
			var message string
			if noOptionValue {
				message = fmt.Sprintf(
					"Files in package %q have both values %q and no value for option %q and all values must be equal.",
					pkg,
					strings.Join(optionValues, ","),
					name,
				)
			} else {
				message = fmt.Sprintf(
					"Files in package %q have multiple values %q for option %q and all values must be equal.",
					pkg,
					strings.Join(optionValues, ","),
					name,
				)
			}
			var sourcePath protoreflect.SourcePath
			if fileOptionLocation := getFileOptionLocation(file); fileOptionLocation != nil {
				sourcePath = fileOptionLocation.SourcePath()
			}
			responseWriter.AddAnnotation(
				check.WithFileNameAndSourcePath(file.Path(), sourcePath),
				check.WithMessage(message),
			)
		}
	}
	return nil
}

// HandleLintPackageVersionSuffix is a handle function.
var HandleLintPackageVersionSuffix = bufcheckserverutil.NewLintFileRuleHandler(handleLintPackageVersionSuffix)

func handleLintPackageVersionSuffix(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	file bufprotosource.File,
) error {
	pkg := file.Package()
	if pkg == "" {
		return nil
	}
	if _, ok := protoversion.NewPackageVersionForPackage(pkg); !ok {
		responseWriter.AddProtosourceAnnotation(
			file.PackageLocation(),
			nil,
			`Package name %q should be suffixed with a correctly formed version, such as %q.`,
			pkg,
			pkg+".v1",
		)
	}
	return nil
}

// HandleLintProtovalidate is a handle function.
var HandleLintProtovalidate = bufcheckserverutil.NewRuleHandler(handleLintProtovalidate)

// handleLintProtovalidate runs checks all predefined rules, message rules, and field rules.
//
// NOTE: Oneofs also have protovalidate support, but they only have a "required" field, so nothing to lint.
func handleLintProtovalidate(
	ctx context.Context,
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
) error {
	// TODO: addAnnotationFunc is used to set add annotations to responseWriter. A follow-up
	// will be made to refactor the code so we no longer need this.
	addAnnotationFunc := func(
		_ bufprotosource.Descriptor,
		location bufprotosource.Location,
		_ []bufprotosource.Location,
		format string,
		args ...interface{},
	) {
		responseWriter.AddProtosourceAnnotation(
			location,
			nil,
			format,
			args...,
		)
	}
	// We create a new extension resolver using all of the files from the request, including
	// import files. This is because there can be a case where a non-import file uses a predefined
	// rule from an imported file.
	extensionResolver, err := protoencoding.NewResolver(
		slicesext.Map(
			request.ProtosourceFiles(),
			func(protosourceFile bufprotosource.File) protodescriptor.FileDescriptor {
				return protosourceFile.FileDescriptor()
			},
		)...,
	)
	if err != nil {
		return err
	}
	// However, we only want to check non-import files, so we can use NewLintMessageRuleHandler
	// and NewLintFieldRuleHandler utils to check messages and fields respectively.
	if err := bufcheckserverutil.NewLintMessageRuleHandler(
		func(
			_ bufcheckserverutil.ResponseWriter,
			_ bufcheckserverutil.Request,
			message bufprotosource.Message,
		) error {
			return buflintvalidate.CheckMessage(addAnnotationFunc, message)
		},
		// The responseWriter is being passed in through the shared addAnnotationFunc, so we
		// do not pass in responseWriter again. This should be addressed in a refactor.
	).Handle(ctx, nil, request); err != nil {
		return err
	}
	return bufcheckserverutil.NewLintFieldRuleHandler(
		func(
			_ bufcheckserverutil.ResponseWriter,
			_ bufcheckserverutil.Request,
			field bufprotosource.Field,
		) error {
			if err := buflintvalidate.CheckPredefinedRuleExtension(addAnnotationFunc, field, extensionResolver); err != nil {
				return err
			}
			return buflintvalidate.CheckField(addAnnotationFunc, field, extensionResolver)
		},
		// The responseWriter is being passed in through the shared addAnnotationFunc, so we
		// do not pass in responseWriter again. This should be addressed in a refactor.
	).Handle(ctx, nil, request)
}

// HandleLintRPCNoClientStreaming is a handle function.
var HandleLintRPCNoClientStreaming = bufcheckserverutil.NewLintMethodRuleHandler(handleLintRPCNoClientStreaming)

func handleLintRPCNoClientStreaming(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	method bufprotosource.Method,
) error {
	if method.ClientStreaming() {
		responseWriter.AddProtosourceAnnotation(
			method.Location(),
			nil,
			"RPC %q is client streaming.",
			method.Name(),
		)
	}
	return nil
}

// HandleLintRPCNoServerStreaming is a handle function.
var HandleLintRPCNoServerStreaming = bufcheckserverutil.NewLintMethodRuleHandler(handleLintRPCNoServerStreaming)

func handleLintRPCNoServerStreaming(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	method bufprotosource.Method,
) error {
	if method.ServerStreaming() {
		responseWriter.AddProtosourceAnnotation(
			method.Location(),
			nil,
			"RPC %q is server streaming.",
			method.Name(),
		)
	}
	return nil
}

// HandleLintRPCPascalCase is a handle function.
var HandleLintRPCPascalCase = bufcheckserverutil.NewLintMethodRuleHandler(handleLintRPCPascalCase)

func handleLintRPCPascalCase(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	method bufprotosource.Method,
) error {
	name := method.Name()
	expectedName := stringutil.ToPascalCase(name)
	if name != expectedName {
		responseWriter.AddProtosourceAnnotation(
			method.NameLocation(),
			nil,
			"RPC name %q should be PascalCase, such as %q.",
			name,
			expectedName,
		)
	}
	return nil
}

// HandleLintRPCRequestResponseUnique is a handle function.
var HandleLintRPCRequestResponseUnique = bufcheckserverutil.NewLintFilesRuleHandler(handleLintRPCRequestResponseUnique)

func handleLintRPCRequestResponseUnique(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	files []bufprotosource.File,
) error {
	allowSameRequestResponse, err := bufcheckopt.GetRPCAllowSameRequestResponse(request.Options())
	if err != nil {
		return err
	}
	allowGoogleProtobufEmptyRequests, err := bufcheckopt.GetRPCAllowGoogleProtobufEmptyRequests(request.Options())
	if err != nil {
		return err
	}
	allowGoogleProtobufEmptyResponses, err := bufcheckopt.GetRPCAllowGoogleProtobufEmptyResponses(request.Options())
	if err != nil {
		return err
	}
	allFullNameToMethod, err := bufprotosource.FullNameToMethod(files...)
	if err != nil {
		return err
	}
	// first check if any requests or responses are the same
	// if not, we can treat requests and responses equally for checking if more than
	// one method uses a type
	if !allowSameRequestResponse {
		for _, method := range allFullNameToMethod {
			if method.InputTypeName() == method.OutputTypeName() {
				// if we allow both empty requests and responses, we do not want to add a FileAnnotation
				if !(method.InputTypeName() == "google.protobuf.Empty" && allowGoogleProtobufEmptyRequests && allowGoogleProtobufEmptyResponses) {
					responseWriter.AddProtosourceAnnotation(
						method.Location(),
						nil,
						"RPC %q has the same type %q for the request and response.",
						method.Name(),
						method.InputTypeName(),
					)
				}
			}
		}
	}
	// we have now added errors for the same request and response type if applicable
	// we can now check methods for unique usage of a given type
	requestResponseTypeToFullNameToMethod := make(map[string]map[string]bufprotosource.Method)
	for fullName, method := range allFullNameToMethod {
		for _, requestResponseType := range []string{method.InputTypeName(), method.OutputTypeName()} {
			fullNameToMethod, ok := requestResponseTypeToFullNameToMethod[requestResponseType]
			if !ok {
				fullNameToMethod = make(map[string]bufprotosource.Method)
				requestResponseTypeToFullNameToMethod[requestResponseType] = fullNameToMethod
			}
			fullNameToMethod[fullName] = method
		}
	}
	for requestResponseType, fullNameToMethod := range requestResponseTypeToFullNameToMethod {
		// only this method uses this request or response type, no issue
		if len(fullNameToMethod) == 1 {
			continue
		}
		// if the request or response type is google.protobuf.Empty and we allow this for requests or responses,
		// we have to do a harder check
		if requestResponseType == "google.protobuf.Empty" && (allowGoogleProtobufEmptyRequests || allowGoogleProtobufEmptyResponses) {
			// if both requests and responses can be google.protobuf.Empty, then do not add any error
			// else, we check
			if !(allowGoogleProtobufEmptyRequests && allowGoogleProtobufEmptyResponses) {
				// inside this if statement, one of allowGoogleProtobufEmptyRequests or allowGoogleProtobufEmptyResponses is true
				var requestMethods []bufprotosource.Method
				var responseMethods []bufprotosource.Method
				for _, method := range fullNameToMethod {
					if method.InputTypeName() == "google.protobuf.Empty" {
						requestMethods = append(requestMethods, method)
					}
					if method.OutputTypeName() == "google.protobuf.Empty" {
						responseMethods = append(responseMethods, method)
					}
				}
				if !allowGoogleProtobufEmptyRequests && len(requestMethods) > 1 {
					for _, method := range requestMethods {
						responseWriter.AddProtosourceAnnotation(
							method.Location(),
							nil,
							"%q is used as the request for multiple RPCs.",
							requestResponseType,
						)
					}
				}
				if !allowGoogleProtobufEmptyResponses && len(responseMethods) > 1 {
					for _, method := range responseMethods {
						responseWriter.AddProtosourceAnnotation(
							method.Location(),
							nil,
							"%q is used as the response for multiple RPCs.",
							requestResponseType,
						)
					}
				}
			}
		} else {
			// else, we have a duplicate usage of requestResponseType, add an FileAnnotation to each method
			for _, method := range fullNameToMethod {
				responseWriter.AddProtosourceAnnotation(
					method.Location(),
					nil,
					"%q is used as the request or response type for multiple RPCs.",
					requestResponseType,
				)
			}
		}
	}
	return nil
}

// HandleLintRPCRequestStandardName is a handle function.
var HandleLintRPCRequestStandardName = bufcheckserverutil.NewLintMethodRuleHandler(handleLintRPCRequestStandardName)

func handleLintRPCRequestStandardName(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	method bufprotosource.Method,
) error {
	allowGoogleProtobufEmptyRequests, err := bufcheckopt.GetRPCAllowGoogleProtobufEmptyRequests(request.Options())
	if err != nil {
		return err
	}
	service := method.Service()
	if service == nil {
		return errors.New("method.Service() is nil")
	}
	name := method.InputTypeName()
	if allowGoogleProtobufEmptyRequests && name == "google.protobuf.Empty" {
		return nil
	}
	if strings.Contains(name, ".") {
		split := strings.Split(name, ".")
		name = split[len(split)-1]
	}
	expectedName1 := stringutil.ToPascalCase(method.Name()) + "Request"
	expectedName2 := stringutil.ToPascalCase(service.Name()) + expectedName1
	if name != expectedName1 && name != expectedName2 {
		responseWriter.AddProtosourceAnnotation(
			method.InputTypeLocation(),
			nil,
			"RPC request type %q should be named %q or %q.",
			name,
			expectedName1,
			expectedName2,
		)
	}

	return nil
}

// HandleLintRPCResponseStandardName is a handle function.
var HandleLintRPCResponseStandardName = bufcheckserverutil.NewLintMethodRuleHandler(handleLintRPCResponseStandardName)

func handleLintRPCResponseStandardName(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	method bufprotosource.Method,
) error {
	allowGoogleProtobufEmptyResponses, err := bufcheckopt.GetRPCAllowGoogleProtobufEmptyResponses(request.Options())
	if err != nil {
		return err
	}
	service := method.Service()
	if service == nil {
		return errors.New("method.Service() is nil")
	}
	name := method.OutputTypeName()
	if allowGoogleProtobufEmptyResponses && name == "google.protobuf.Empty" {
		return nil
	}
	if strings.Contains(name, ".") {
		split := strings.Split(name, ".")
		name = split[len(split)-1]
	}
	expectedName1 := stringutil.ToPascalCase(method.Name()) + "Response"
	expectedName2 := stringutil.ToPascalCase(service.Name()) + expectedName1
	if name != expectedName1 && name != expectedName2 {
		responseWriter.AddProtosourceAnnotation(
			method.OutputTypeLocation(),
			nil,
			"RPC response type %q should be named %q or %q.",
			name,
			expectedName1,
			expectedName2,
		)
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

// HandleLintStablePackageNoImportUnstable is a handle function.
var HandleLintStablePackageNoImportUnstable = bufcheckserverutil.NewLintFilesRuleHandler(handleLintStablePackageNoImportUnstable)

func handleLintStablePackageNoImportUnstable(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	files []bufprotosource.File,
) error {
	filePathToFile, err := bufprotosource.FilePathToFile(files...)
	if err != nil {
		return err
	}
	for _, file := range files {
		packageVersion, ok := protoversion.NewPackageVersionForPackage(file.Package())
		if !ok {
			// No package, or no version on package - unstable to determine if stable.
			continue
		}
		if packageVersion.StabilityLevel() != protoversion.StabilityLevelStable {
			// If package is not stable, no failure.
			continue
		}
		// Package is stable. Check imports.
		for _, fileImport := range file.FileImports() {
			if importedFile, ok := filePathToFile[fileImport.Import()]; ok {
				importedFilePackageVersion, ok := protoversion.NewPackageVersionForPackage(importedFile.Package())
				if !ok {
					continue
				}
				if importedFilePackageVersion.StabilityLevel() != protoversion.StabilityLevelStable {
					responseWriter.AddProtosourceAnnotation(
						fileImport.Location(),
						nil,
						`This file is in stable package %q, so it should not depend on %q from unstable package %q.`,
						file.Package(),
						fileImport.Import(),
						importedFile.Package(),
					)
				}
			}
		}
	}
	return nil
}

// HandleLintSyntaxSpecified is a handle function.
var HandleLintSyntaxSpecified = bufcheckserverutil.NewLintFileRuleHandler(handleLintSyntaxSpecified)

func handleLintSyntaxSpecified(
	responseWriter bufcheckserverutil.ResponseWriter,
	_ bufcheckserverutil.Request,
	file bufprotosource.File,
) error {
	if file.Syntax() == bufprotosource.SyntaxUnspecified {
		responseWriter.AddAnnotation(
			check.WithFileName(file.Path()),
			check.WithMessage(`Files must have a syntax explicitly specified. If no syntax is specified, the file defaults to "proto2".`),
		)
	}
	return nil
}
