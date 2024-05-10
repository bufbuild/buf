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

// Package buflintcheck impelements the check functions.
//
// These are used by buflintbuild to create RuleBuilders.
package buflintcheck

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/internal/buflintvalidate"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protoversion"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	// CommentIgnorePrefix is the comment ignore prefix.
	//
	// Comments with this prefix do not count towards valid comments in the comment checkers.
	// This is also used in buflint when constructing a new Runner, and is passed to the
	// RunnerWithIgnorePrefix option.
	CommentIgnorePrefix = "buf:lint:ignore"
)

var (
	// CheckCommentEnum is a check function.
	CheckCommentEnum = newEnumCheckFunc(checkCommentEnum)
	// CheckCommentEnumValue is a check function.
	CheckCommentEnumValue = newEnumValueCheckFunc(checkCommentEnumValue)
	// CheckCommentField is a check function.
	CheckCommentField = newFieldCheckFunc(checkCommentField)
	// CheckCommentMessage is a check function.
	CheckCommentMessage = newMessageCheckFunc(checkCommentMessage)
	// CheckCommentOneof is a check function.
	CheckCommentOneof = newOneofCheckFunc(checkCommentOneof)
	// CheckCommentService is a check function.
	CheckCommentService = newServiceCheckFunc(checkCommentService)
	// CheckCommentRPC is a check function.
	CheckCommentRPC = newMethodCheckFunc(checkCommentRPC)
)

func checkCommentEnum(add addFunc, value bufprotosource.Enum) error {
	return checkCommentNamedDescriptor(add, value, "Enum")
}

func checkCommentEnumValue(add addFunc, value bufprotosource.EnumValue) error {
	return checkCommentNamedDescriptor(add, value, "Enum value")
}

func checkCommentField(add addFunc, value bufprotosource.Field) error {
	if value.ParentMessage() != nil && value.ParentMessage().IsMapEntry() {
		// Don't check synthetic fields for map entries. They have no comments.
		return nil
	}
	if value.Type() == descriptorpb.FieldDescriptorProto_TYPE_GROUP {
		// Group fields also have no comments: comments in source get
		// attributed to the nested message, not the field.
		return nil
	}
	return checkCommentNamedDescriptor(add, value, "Field")
}

func checkCommentMessage(add addFunc, value bufprotosource.Message) error {
	if value.IsMapEntry() {
		// Don't check synthetic map entries. They have no comments.
		return nil
	}
	return checkCommentNamedDescriptor(add, value, "Message")
}

func checkCommentOneof(add addFunc, value bufprotosource.Oneof) error {
	oneofDescriptor, err := value.AsDescriptor()
	if err == nil && oneofDescriptor.IsSynthetic() {
		// Don't check synthetic oneofs (for proto3-optional fields). They have no comments.
		return nil
	}
	return checkCommentNamedDescriptor(add, value, "Oneof")
}

func checkCommentRPC(add addFunc, value bufprotosource.Method) error {
	return checkCommentNamedDescriptor(add, value, "RPC")
}

func checkCommentService(add addFunc, value bufprotosource.Service) error {
	return checkCommentNamedDescriptor(add, value, "Service")
}

func checkCommentNamedDescriptor(
	add addFunc,
	namedDescriptor bufprotosource.NamedDescriptor,
	typeName string,
) error {
	location := namedDescriptor.Location()
	if location == nil {
		// this will magically skip map entry fields as well as a side-effect, although originally unintended
		return nil
	}
	if !validLeadingComment(location.LeadingComments()) {
		add(namedDescriptor, location, nil, "%s %q should have a non-empty comment for documentation.", typeName, namedDescriptor.Name())
	}
	return nil
}

// CheckDirectorySamePackage is a check function.
var CheckDirectorySamePackage = newDirToFilesCheckFunc(checkDirectorySamePackage)

func checkDirectorySamePackage(add addFunc, dirPath string, files []bufprotosource.File) error {
	pkgMap := make(map[string]struct{})
	for _, file := range files {
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
		for _, file := range files {
			add(file, file.PackageLocation(), nil, "%s detected within directory %q.", messagePrefix, dirPath)
		}
	}
	return nil
}

// CheckEnumNoAllowAlias is a check function.
var CheckEnumNoAllowAlias = newEnumCheckFunc(checkEnumNoAllowAlias)

func checkEnumNoAllowAlias(add addFunc, enum bufprotosource.Enum) error {
	if enum.AllowAlias() {
		add(enum, enum.AllowAliasLocation(), nil, `Enum option "allow_alias" on enum %q must be false.`, enum.Name())
	}
	return nil
}

// CheckEnumPascalCase is a check function.
var CheckEnumPascalCase = newEnumCheckFunc(checkEnumPascalCase)

func checkEnumPascalCase(add addFunc, enum bufprotosource.Enum) error {
	name := enum.Name()
	expectedName := stringutil.ToPascalCase(name)
	if name != expectedName {
		add(enum, enum.NameLocation(), nil, "Enum name %q should be PascalCase, such as %q.", name, expectedName)
	}
	return nil
}

// CheckEnumFirstValueZero is a check function.
var CheckEnumFirstValueZero = newEnumCheckFunc(checkEnumFirstValueZero)

func checkEnumFirstValueZero(add addFunc, enum bufprotosource.Enum) error {
	if values := enum.Values(); len(values) > 0 {
		if firstEnumValue := values[0]; firstEnumValue.Number() != 0 {
			// proto3 compilation references the number
			add(
				firstEnumValue,
				firstEnumValue.NumberLocation(),
				// also check the name location for this comment ignore, as the number location might not have the comment
				// see https://github.com/bufbuild/buf/issues/1186
				// also check the enum for this comment ignore
				// this allows users to set this "globally" for an enum
				// see https://github.com/bufbuild/buf/issues/161
				[]bufprotosource.Location{
					firstEnumValue.NameLocation(),
					firstEnumValue.Enum().Location(),
				},
				"First enum value %q should have a numeric value of 0",
				firstEnumValue.Name(),
			)
		}
	}
	return nil
}

// CheckEnumValuePrefix is a check function.
var CheckEnumValuePrefix = newEnumValueCheckFunc(checkEnumValuePrefix)

func checkEnumValuePrefix(add addFunc, enumValue bufprotosource.EnumValue) error {
	name := enumValue.Name()
	expectedPrefix := fieldToUpperSnakeCase(enumValue.Enum().Name()) + "_"
	if !strings.HasPrefix(name, expectedPrefix) {
		add(
			enumValue,
			enumValue.NameLocation(),
			// also check the enum for this comment ignore
			// this allows users to set this "globally" for an enum
			// this came up in https://github.com/bufbuild/buf/issues/161
			[]bufprotosource.Location{
				enumValue.Enum().Location(),
			},
			"Enum value name %q should be prefixed with %q.",
			name,
			expectedPrefix,
		)
	}
	return nil
}

// CheckEnumValueUpperSnakeCase is a check function.
var CheckEnumValueUpperSnakeCase = newEnumValueCheckFunc(checkEnumValueUpperSnakeCase)

func checkEnumValueUpperSnakeCase(add addFunc, enumValue bufprotosource.EnumValue) error {
	name := enumValue.Name()
	expectedName := fieldToUpperSnakeCase(name)
	if name != expectedName {
		add(
			enumValue,
			enumValue.NameLocation(),
			// also check the enum for this comment ignore
			// this allows users to set this "globally" for an enum
			[]bufprotosource.Location{
				enumValue.Enum().Location(),
			},
			"Enum value name %q should be UPPER_SNAKE_CASE, such as %q.",
			name,
			expectedName,
		)
	}
	return nil
}

// CheckEnumZeroValueSuffix is a check function.
var CheckEnumZeroValueSuffix = func(
	id string,
	ignoreFunc internal.IgnoreFunc,
	files []bufprotosource.File,
	suffix string,
) ([]bufanalysis.FileAnnotation, error) {
	return newEnumValueCheckFunc(
		func(add addFunc, enumValue bufprotosource.EnumValue) error {
			return checkEnumZeroValueSuffix(add, enumValue, suffix)
		},
	)(id, ignoreFunc, files)
}

func checkEnumZeroValueSuffix(add addFunc, enumValue bufprotosource.EnumValue, suffix string) error {
	if enumValue.Number() != 0 {
		return nil
	}
	name := enumValue.Name()
	if !strings.HasSuffix(name, suffix) {
		add(
			enumValue,
			enumValue.NameLocation(),
			// also check the enum for this comment ignore
			// this allows users to set this "globally" for an enum
			[]bufprotosource.Location{
				enumValue.Enum().Location(),
			},
			"Enum zero value name %q should be suffixed with %q.",
			name,
			suffix,
		)
	}
	return nil
}

// CheckFieldLowerSnakeCase is a check function.
var CheckFieldLowerSnakeCase = newFieldCheckFunc(checkFieldLowerSnakeCase)

func checkFieldLowerSnakeCase(add addFunc, field bufprotosource.Field) error {
	message := field.ParentMessage()
	if message != nil && message.IsMapEntry() {
		// this check should always pass anyways but just in case
		return nil
	}
	name := field.Name()
	expectedName := fieldToLowerSnakeCase(name)
	if name != expectedName {
		var otherLocs []bufprotosource.Location
		if message != nil {
			// also check the message for this comment ignore
			// this allows users to set this "globally" for a message
			otherLocs = []bufprotosource.Location{message.Location()}
		}
		add(
			field,
			field.NameLocation(),
			otherLocs,
			"Field name %q should be lower_snake_case, such as %q.",
			name,
			expectedName,
		)
	}
	return nil
}

// CheckFieldNoDescriptor is a check function.
var CheckFieldNoDescriptor = newFieldCheckFunc(checkFieldNoDescriptor)

func checkFieldNoDescriptor(add addFunc, field bufprotosource.Field) error {
	name := field.Name()
	if strings.ToLower(strings.Trim(name, "_")) == "descriptor" {
		var otherLocs []bufprotosource.Location
		if message := field.ParentMessage(); message != nil {
			// also check the message for this comment ignore
			// this allows users to set this "globally" for a message
			otherLocs = []bufprotosource.Location{message.Location()}
		}
		add(
			field,
			field.NameLocation(),
			otherLocs,
			`Field name %q cannot be any capitalization of "descriptor" with any number of prefix or suffix underscores.`,
			name,
		)
	}
	return nil
}

// CheckFieldNotRequired is a check function.
var CheckFieldNotRequired = newFieldCheckFunc(checkFieldNotRequired)

func checkFieldNotRequired(add addFunc, field bufprotosource.Field) error {
	fieldDescriptor, err := field.AsDescriptor()
	if err != nil {
		return err
	}
	// We use the protoreflect field descriptor to handle editions, where the
	// field is set to required using special "features" options, instead of the
	// label on the descriptor proto.
	if fieldDescriptor.Cardinality() == protoreflect.Required {
		var otherLocs []bufprotosource.Location
		if message := field.ParentMessage(); message != nil {
			// also check the message for this comment ignore
			// this allows users to set this "globally" for a message
			otherLocs = []bufprotosource.Location{message.Location()}
		}
		add(
			field,
			field.NameLocation(),
			otherLocs,
			`Field named %q should not be required.`,
		)
	}
	return nil
}

// CheckFileLowerSnakeCase is a check function.
var CheckFileLowerSnakeCase = newFileCheckFunc(checkFileLowerSnakeCase)

func checkFileLowerSnakeCase(add addFunc, file bufprotosource.File) error {
	filename := file.Path()
	base := normalpath.Base(filename)
	ext := normalpath.Ext(filename)
	baseWithoutExt := strings.TrimSuffix(base, ext)
	expectedBaseWithoutExt := stringutil.ToLowerSnakeCase(baseWithoutExt)
	if baseWithoutExt != expectedBaseWithoutExt {
		add(file, nil, nil, `Filename %q should be lower_snake_case%s, such as "%s%s".`, base, ext, expectedBaseWithoutExt, ext)
	}
	return nil
}

var (
	// CheckImportNoPublic is a check function.
	CheckImportNoPublic = newFileImportCheckFunc(checkImportNoPublic)
	// CheckImportNoWeak is a check function.
	CheckImportNoWeak = newFileImportCheckFunc(checkImportNoWeak)
	// CheckImportUsed is a check function.
	CheckImportUsed = newFileImportCheckFunc(checkImportUsed)
)

func checkImportNoPublic(add addFunc, fileImport bufprotosource.FileImport) error {
	return checkImportNoPublicWeak(add, fileImport, fileImport.IsPublic(), "public")
}

func checkImportNoWeak(add addFunc, fileImport bufprotosource.FileImport) error {
	return checkImportNoPublicWeak(add, fileImport, fileImport.IsWeak(), "weak")
}

func checkImportNoPublicWeak(add addFunc, fileImport bufprotosource.FileImport, value bool, name string) error {
	if value {
		add(fileImport, fileImport.Location(), nil, `Import %q must not be %s.`, fileImport.Import(), name)
	}
	return nil
}

func checkImportUsed(add addFunc, fileImport bufprotosource.FileImport) error {
	if fileImport.IsUnused() {
		add(fileImport, fileImport.Location(), nil, `Import %q is unused.`, fileImport.Import())
	}
	return nil
}

// CheckMessagePascalCase is a check function.
var CheckMessagePascalCase = newMessageCheckFunc(checkMessagePascalCase)

func checkMessagePascalCase(add addFunc, message bufprotosource.Message) error {
	if message.IsMapEntry() {
		// map entries should always be pascal case but we don't want to check them anyways
		return nil
	}
	name := message.Name()
	expectedName := stringutil.ToPascalCase(name)
	if name != expectedName {
		add(message, message.NameLocation(), nil, "Message name %q should be PascalCase, such as %q.", name, expectedName)
	}
	return nil
}

// CheckOneofLowerSnakeCase is a check function.
var CheckOneofLowerSnakeCase = newOneofCheckFunc(checkOneofLowerSnakeCase)

func checkOneofLowerSnakeCase(add addFunc, oneof bufprotosource.Oneof) error {
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
		add(
			oneof,
			oneof.NameLocation(),
			// also check the message for this comment ignore
			// this allows users to set this "globally" for a message
			[]bufprotosource.Location{
				oneof.Message().Location(),
			},
			"Oneof name %q should be lower_snake_case, such as %q.",
			name,
			expectedName,
		)
	}
	return nil
}

// CheckPackageDefined is a check function.
var CheckPackageDefined = newFileCheckFunc(checkPackageDefined)

func checkPackageDefined(add addFunc, file bufprotosource.File) error {
	if file.Package() == "" {
		add(file, nil, nil, "Files must have a package defined.")
	}
	return nil
}

// CheckPackageDirectoryMatch is a check function.
var CheckPackageDirectoryMatch = newFileCheckFunc(checkPackageDirectoryMatch)

func checkPackageDirectoryMatch(add addFunc, file bufprotosource.File) error {
	pkg := file.Package()
	if pkg == "" {
		return nil
	}
	expectedDirPath := strings.ReplaceAll(pkg, ".", "/")
	dirPath := normalpath.Dir(file.Path())
	// need to check case where in root relative directory and no package defined
	// this should be valid although if SENSIBLE is turned on this will be invalid
	if dirPath != expectedDirPath {
		add(file, file.PackageLocation(), nil, `Files with package %q must be within a directory "%s" relative to root but were in directory "%s".`, pkg, normalpath.Unnormalize(expectedDirPath), normalpath.Unnormalize(dirPath))
	}
	return nil
}

// CheckPackageLowerSnakeCase is a check function.
var CheckPackageLowerSnakeCase = newFileCheckFunc(checkPackageLowerSnakeCase)

func checkPackageLowerSnakeCase(add addFunc, file bufprotosource.File) error {
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
		add(file, file.PackageLocation(), nil, "Package name %q should be lower_snake.case, such as %q.", pkg, expectedPkg)
	}
	return nil
}

// CheckPackageNoImportCycle is a check function.
//
// Note that imports are not skipped via the helper, as we want to detect import cycles
// even if they are within imports, and report on them. If a non-import is part of an
// import cycle, we report it, even if the import cycle includes imports in it.
var CheckPackageNoImportCycle = newFilesWithImportsCheckFunc(checkPackageNoImportCycle)

func checkPackageNoImportCycle(add addFunc, files []bufprotosource.File) error {
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
					add(fileImport, fileImport.Location(), nil, `Package import cycle: %s`, strings.Join(importCycle, ` -> `))
				}
			}
		}
	}
	return nil
}

// CheckPackageSameDirectory is a check function.
var CheckPackageSameDirectory = newPackageToFilesCheckFunc(checkPackageSameDirectory)

func checkPackageSameDirectory(add addFunc, pkg string, files []bufprotosource.File) error {
	dirMap := make(map[string]struct{})
	for _, file := range files {
		dirMap[normalpath.Dir(file.Path())] = struct{}{}
	}
	if len(dirMap) > 1 {
		dirs := slicesext.MapKeysToSortedSlice(dirMap)
		for _, file := range files {
			add(file, file.PackageLocation(), nil, "Multiple directories %q contain files with package %q.", strings.Join(dirs, ","), pkg)
		}
	}
	return nil
}

var (
	// CheckPackageSameCsharpNamespace is a check function.
	CheckPackageSameCsharpNamespace = newPackageToFilesCheckFunc(checkPackageSameCsharpNamespace)
	// CheckPackageSameGoPackage is a check function.
	CheckPackageSameGoPackage = newPackageToFilesCheckFunc(checkPackageSameGoPackage)
	// CheckPackageSameJavaMultipleFiles is a check function.
	CheckPackageSameJavaMultipleFiles = newPackageToFilesCheckFunc(checkPackageSameJavaMultipleFiles)
	// CheckPackageSameJavaPackage is a check function.
	CheckPackageSameJavaPackage = newPackageToFilesCheckFunc(checkPackageSameJavaPackage)
	// CheckPackageSamePhpNamespace is a check function.
	CheckPackageSamePhpNamespace = newPackageToFilesCheckFunc(checkPackageSamePhpNamespace)
	// CheckPackageSameRubyPackage is a check function.
	CheckPackageSameRubyPackage = newPackageToFilesCheckFunc(checkPackageSameRubyPackage)
	// CheckPackageSameSwiftPrefix is a check function.
	CheckPackageSameSwiftPrefix = newPackageToFilesCheckFunc(checkPackageSameSwiftPrefix)
)

func checkPackageSameCsharpNamespace(add addFunc, pkg string, files []bufprotosource.File) error {
	return checkPackageSameOptionValue(add, pkg, files, bufprotosource.File.CsharpNamespace, bufprotosource.File.CsharpNamespaceLocation, "csharp_namespace")
}

func checkPackageSameGoPackage(add addFunc, pkg string, files []bufprotosource.File) error {
	return checkPackageSameOptionValue(add, pkg, files, bufprotosource.File.GoPackage, bufprotosource.File.GoPackageLocation, "go_package")
}

func checkPackageSameJavaMultipleFiles(add addFunc, pkg string, files []bufprotosource.File) error {
	return checkPackageSameOptionValue(
		add,
		pkg,
		files,
		func(file bufprotosource.File) string {
			return strconv.FormatBool(file.JavaMultipleFiles())
		},
		bufprotosource.File.JavaMultipleFilesLocation,
		"java_multiple_files",
	)
}

func checkPackageSameJavaPackage(add addFunc, pkg string, files []bufprotosource.File) error {
	return checkPackageSameOptionValue(add, pkg, files, bufprotosource.File.JavaPackage, bufprotosource.File.JavaPackageLocation, "java_package")
}

func checkPackageSamePhpNamespace(add addFunc, pkg string, files []bufprotosource.File) error {
	return checkPackageSameOptionValue(add, pkg, files, bufprotosource.File.PhpNamespace, bufprotosource.File.PhpNamespaceLocation, "php_namespace")
}

func checkPackageSameRubyPackage(add addFunc, pkg string, files []bufprotosource.File) error {
	return checkPackageSameOptionValue(add, pkg, files, bufprotosource.File.RubyPackage, bufprotosource.File.RubyPackageLocation, "ruby_package")
}

func checkPackageSameSwiftPrefix(add addFunc, pkg string, files []bufprotosource.File) error {
	return checkPackageSameOptionValue(add, pkg, files, bufprotosource.File.SwiftPrefix, bufprotosource.File.SwiftPrefixLocation, "swift_prefix")
}

func checkPackageSameOptionValue(
	add addFunc,
	pkg string,
	files []bufprotosource.File,
	getOptionValue func(bufprotosource.File) string,
	getOptionLocation func(bufprotosource.File) bufprotosource.Location,
	name string,
) error {
	optionValueMap := make(map[string]struct{})
	for _, file := range files {
		optionValueMap[getOptionValue(file)] = struct{}{}
	}
	if len(optionValueMap) > 1 {
		_, noOptionValue := optionValueMap[""]
		delete(optionValueMap, "")
		optionValues := slicesext.MapKeysToSortedSlice(optionValueMap)
		for _, file := range files {
			if noOptionValue {
				add(file, getOptionLocation(file), nil, "Files in package %q have both values %q and no value for option %q and all values must be equal.", pkg, strings.Join(optionValues, ","), name)
			} else {
				add(file, getOptionLocation(file), nil, "Files in package %q have multiple values %q for option %q and all values must be equal.", pkg, strings.Join(optionValues, ","), name)
			}
		}
	}
	return nil
}

// CheckPackageVersionSuffix is a check function.
var CheckPackageVersionSuffix = newFileCheckFunc(checkPackageVersionSuffix)

func checkPackageVersionSuffix(add addFunc, file bufprotosource.File) error {
	pkg := file.Package()
	if pkg == "" {
		return nil
	}
	if _, ok := protoversion.NewPackageVersionForPackage(pkg); !ok {
		add(file, file.PackageLocation(), nil, `Package name %q should be suffixed with a correctly formed version, such as %q.`, pkg, pkg+".v1")
	}
	return nil
}

// CheckProtovalidate is a check function.
var CheckProtovalidate = combine(
	newMessageCheckFunc(checkProtovalidateMessage),
	newFieldCheckFunc(checkProtovalidateField),
	// NOTE: Oneofs also have protovalidate support, but they
	//       only have a "required" field, so nothing to lint.
)

func checkProtovalidateMessage(add addFunc, message bufprotosource.Message) error {
	return buflintvalidate.CheckMessage(add, message)
}

func checkProtovalidateField(add addFunc, field bufprotosource.Field) error {
	return buflintvalidate.CheckField(add, field)
}

// CheckRPCNoClientStreaming is a check function.
var CheckRPCNoClientStreaming = newMethodCheckFunc(checkRPCNoClientStreaming)

func checkRPCNoClientStreaming(add addFunc, method bufprotosource.Method) error {
	if method.ClientStreaming() {
		add(
			method,
			method.Location(),
			// also check the service for this comment ignore
			// this allows users to set this "globally" for a service
			[]bufprotosource.Location{
				method.Service().Location(),
			},
			"RPC %q is client streaming.",
			method.Name(),
		)
	}
	return nil
}

// CheckRPCNoServerStreaming is a check function.
var CheckRPCNoServerStreaming = newMethodCheckFunc(checkRPCNoServerStreaming)

func checkRPCNoServerStreaming(add addFunc, method bufprotosource.Method) error {
	if method.ServerStreaming() {
		add(
			method,
			method.Location(),
			// also check the service for this comment ignore
			// this allows users to set this "globally" for a service
			[]bufprotosource.Location{
				method.Service().Location(),
			},
			"RPC %q is server streaming.",
			method.Name(),
		)
	}
	return nil
}

// CheckRPCPascalCase is a check function.
var CheckRPCPascalCase = newMethodCheckFunc(checkRPCPascalCase)

func checkRPCPascalCase(add addFunc, method bufprotosource.Method) error {
	name := method.Name()
	expectedName := stringutil.ToPascalCase(name)
	if name != expectedName {
		add(
			method,
			method.NameLocation(),
			// also check the service for this comment ignore
			// this allows users to set this "globally" for a service
			[]bufprotosource.Location{
				method.Service().Location(),
			},
			"RPC name %q should be PascalCase, such as %q.",
			name,
			expectedName,
		)
	}
	return nil
}

// CheckRPCRequestResponseUnique is a check function.
var CheckRPCRequestResponseUnique = func(
	id string,
	ignoreFunc internal.IgnoreFunc,
	files []bufprotosource.File,
	allowSameRequestResponse bool,
	allowGoogleProtobufEmptyRequests bool,
	allowGoogleProtobufEmptyResponses bool,
) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, files []bufprotosource.File) error {
			return checkRPCRequestResponseUnique(
				add,
				files,
				allowSameRequestResponse,
				allowGoogleProtobufEmptyRequests,
				allowGoogleProtobufEmptyResponses,
			)
		},
	)(id, ignoreFunc, files)
}

func checkRPCRequestResponseUnique(
	add addFunc,
	files []bufprotosource.File,
	allowSameRequestResponse bool,
	allowGoogleProtobufEmptyRequests bool,
	allowGoogleProtobufEmptyResponses bool,
) error {
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
					add(
						method,
						method.Location(),
						// also check the service for this comment ignore
						// this allows users to set this "globally" for a service
						[]bufprotosource.Location{
							method.Service().Location(),
						},
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
						add(
							method,
							method.Location(),
							// also check the service for this comment ignore
							// this allows users to set this "globally" for a service
							[]bufprotosource.Location{
								method.Service().Location(),
							},
							"%q is used as the request for multiple RPCs.",
							requestResponseType,
						)
					}
				}
				if !allowGoogleProtobufEmptyResponses && len(responseMethods) > 1 {
					for _, method := range responseMethods {
						add(
							method,
							method.Location(),
							// also check the service for this comment ignore
							// this allows users to set this "globally" for a service
							[]bufprotosource.Location{
								method.Service().Location(),
							},
							"%q is used as the response for multiple RPCs.",
							requestResponseType,
						)
					}
				}
			}
		} else {
			// else, we have a duplicate usage of requestResponseType, add an FileAnnotation to each method
			for _, method := range fullNameToMethod {
				add(
					method,
					method.Location(),
					// also check the service for this comment ignore
					// this allows users to set this "globally" for a service
					[]bufprotosource.Location{
						method.Service().Location(),
					},
					"%q is used as the request or response type for multiple RPCs.",
					requestResponseType,
				)
			}
		}
	}
	return nil
}

// CheckRPCRequestStandardName is a check function.
var CheckRPCRequestStandardName = func(
	id string,
	ignoreFunc internal.IgnoreFunc,
	files []bufprotosource.File,
	allowGoogleProtobufEmptyRequests bool,
) ([]bufanalysis.FileAnnotation, error) {
	return newMethodCheckFunc(
		func(add addFunc, method bufprotosource.Method) error {
			return checkRPCRequestStandardName(add, method, allowGoogleProtobufEmptyRequests)
		},
	)(id, ignoreFunc, files)
}

func checkRPCRequestStandardName(add addFunc, method bufprotosource.Method, allowGoogleProtobufEmptyRequests bool) error {
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
		add(
			method,
			method.InputTypeLocation(),
			// also check the method and service for this comment ignore
			// this came up in https://github.com/bufbuild/buf/issues/242
			[]bufprotosource.Location{
				method.Location(),
				method.Service().Location(),
			},
			"RPC request type %q should be named %q or %q.",
			name,
			expectedName1,
			expectedName2,
		)
	}
	return nil
}

// CheckRPCResponseStandardName is a check function.
var CheckRPCResponseStandardName = func(
	id string,
	ignoreFunc internal.IgnoreFunc,
	files []bufprotosource.File,
	allowGoogleProtobufEmptyResponses bool,
) ([]bufanalysis.FileAnnotation, error) {
	return newMethodCheckFunc(
		func(add addFunc, method bufprotosource.Method) error {
			return checkRPCResponseStandardName(add, method, allowGoogleProtobufEmptyResponses)
		},
	)(id, ignoreFunc, files)
}

func checkRPCResponseStandardName(add addFunc, method bufprotosource.Method, allowGoogleProtobufEmptyResponses bool) error {
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
		add(
			method,
			method.OutputTypeLocation(),
			// also check the method and service for this comment ignore
			// this came up in https://github.com/bufbuild/buf/issues/242
			[]bufprotosource.Location{
				method.Location(),
				method.Service().Location(),
			},
			"RPC response type %q should be named %q or %q.",
			name,
			expectedName1,
			expectedName2,
		)
	}
	return nil
}

// CheckServicePascalCase is a check function.
var CheckServicePascalCase = newServiceCheckFunc(checkServicePascalCase)

func checkServicePascalCase(add addFunc, service bufprotosource.Service) error {
	name := service.Name()
	expectedName := stringutil.ToPascalCase(name)
	if name != expectedName {
		add(service, service.NameLocation(), nil, "Service name %q should be PascalCase, such as %q.", name, expectedName)
	}
	return nil
}

// CheckServiceSuffix is a check function.
var CheckServiceSuffix = func(
	id string,
	ignoreFunc internal.IgnoreFunc,
	files []bufprotosource.File,
	suffix string,
) ([]bufanalysis.FileAnnotation, error) {
	return newServiceCheckFunc(
		func(add addFunc, service bufprotosource.Service) error {
			return checkServiceSuffix(add, service, suffix)
		},
	)(id, ignoreFunc, files)
}

func checkServiceSuffix(add addFunc, service bufprotosource.Service, suffix string) error {
	name := service.Name()
	if !strings.HasSuffix(name, suffix) {
		add(service, service.NameLocation(), nil, "Service name %q should be suffixed with %q.", name, suffix)
	}
	return nil
}

// CheckSyntaxSpecified is a check function.
var CheckSyntaxSpecified = newFileCheckFunc(checkSyntaxSpecified)

func checkSyntaxSpecified(add addFunc, file bufprotosource.File) error {
	if file.Syntax() == bufprotosource.SyntaxUnspecified {
		add(file, file.SyntaxLocation(), nil, `Files must have a syntax explicitly specified. If no syntax is specified, the file defaults to "proto2".`)
	}
	return nil
}
