// Copyright 2020 Buf Technologies Inc.
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

package internal

import (
	"errors"
	"strconv"
	"strings"

	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/protodesc"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/bufbuild/buf/internal/pkg/util/utilstring"
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

func checkCommentEnum(add addFunc, value protodesc.Enum) error {
	return checkCommentNamedDescriptor(add, value, "Enum")
}

func checkCommentEnumValue(add addFunc, value protodesc.EnumValue) error {
	return checkCommentNamedDescriptor(add, value, "Enum value")
}

func checkCommentField(add addFunc, value protodesc.Field) error {
	return checkCommentNamedDescriptor(add, value, "Field")
}

func checkCommentMessage(add addFunc, value protodesc.Message) error {
	return checkCommentNamedDescriptor(add, value, "Message")
}

func checkCommentOneof(add addFunc, value protodesc.Oneof) error {
	return checkCommentNamedDescriptor(add, value, "Oneof")
}

func checkCommentRPC(add addFunc, value protodesc.Method) error {
	return checkCommentNamedDescriptor(add, value, "RPC")
}

func checkCommentService(add addFunc, value protodesc.Service) error {
	return checkCommentNamedDescriptor(add, value, "Service")
}

func checkCommentNamedDescriptor(
	add addFunc,
	namedDescriptor protodesc.NamedDescriptor,
	typeName string,
) error {
	location := namedDescriptor.Location()
	if location == nil {
		// this will magically skip map entry fields as well as a side-effect, although originally unintended
		return nil
	}
	if strings.TrimSpace(location.LeadingComments()) == "" {
		add(namedDescriptor, location, "%s %q should have a non-empty comment for documentation.", typeName, namedDescriptor.Name())
	}
	return nil
}

// CheckDirectorySamePackage is a check function.
var CheckDirectorySamePackage = newDirToFilesCheckFunc(checkDirectorySamePackage)

func checkDirectorySamePackage(add addFunc, dirPath string, files []protodesc.File) error {
	pkgMap := make(map[string]struct{})
	for _, file := range files {
		// works for no package set as this will result in "" which is a valid map key
		pkgMap[file.Package()] = struct{}{}
	}
	if len(pkgMap) > 1 {
		pkgs := utilstring.MapToSortedSlice(pkgMap)
		for _, file := range files {
			add(file, file.PackageLocation(), "Multiple packages %q detected within directory %q.", strings.Join(pkgs, ","), dirPath)
		}
	}
	return nil
}

// CheckEnumNoAllowAlias is a check function.
var CheckEnumNoAllowAlias = newEnumCheckFunc(checkEnumNoAllowAlias)

func checkEnumNoAllowAlias(add addFunc, enum protodesc.Enum) error {
	if enum.AllowAlias() {
		add(enum, enum.AllowAliasLocation(), `Enum option "allow_alias" on enum %q must be false.`, enum.Name())
	}
	return nil
}

// CheckEnumPascalCase is a check function.
var CheckEnumPascalCase = newEnumCheckFunc(checkEnumPascalCase)

func checkEnumPascalCase(add addFunc, enum protodesc.Enum) error {
	name := enum.Name()
	expectedName := utilstring.ToPascalCase(name)
	if name != expectedName {
		add(enum, enum.NameLocation(), "Enum name %q should be PascalCase, such as %q.", name, expectedName)
	}
	return nil
}

// CheckEnumValuePrefix is a check function.
var CheckEnumValuePrefix = newEnumValueCheckFunc(checkEnumValuePrefix)

func checkEnumValuePrefix(add addFunc, enumValue protodesc.EnumValue) error {
	name := enumValue.Name()
	expectedPrefix := fieldToUpperSnakeCase(enumValue.Enum().Name()) + "_"
	if !strings.HasPrefix(name, expectedPrefix) {
		add(enumValue, enumValue.NameLocation(), "Enum value name %q should be prefixed with %q.", name, expectedPrefix)
	}
	return nil
}

// CheckEnumValueUpperSnakeCase is a check function.
var CheckEnumValueUpperSnakeCase = newEnumValueCheckFunc(checkEnumValueUpperSnakeCase)

func checkEnumValueUpperSnakeCase(add addFunc, enumValue protodesc.EnumValue) error {
	name := enumValue.Name()
	expectedName := fieldToUpperSnakeCase(name)
	if name != expectedName {
		add(enumValue, enumValue.NameLocation(), "Enum value name %q should be UPPER_SNAKE_CASE, such as %q.", name, expectedName)
	}
	return nil
}

// CheckEnumZeroValueSuffix is a check function.
var CheckEnumZeroValueSuffix = func(id string, files []protodesc.File, suffix string) ([]*filev1beta1.FileAnnotation, error) {
	return newEnumValueCheckFunc(
		func(add addFunc, enumValue protodesc.EnumValue) error {
			return checkEnumZeroValueSuffix(add, enumValue, suffix)
		},
	)(id, files)
}

func checkEnumZeroValueSuffix(add addFunc, enumValue protodesc.EnumValue, suffix string) error {
	if enumValue.Number() != 0 {
		return nil
	}
	name := enumValue.Name()
	if !strings.HasSuffix(name, suffix) {
		add(enumValue, enumValue.NameLocation(), "Enum zero value name %q should be suffixed with %q.", name, suffix)
	}
	return nil
}

// CheckFieldLowerSnakeCase is a check function.
var CheckFieldLowerSnakeCase = newFieldCheckFunc(checkFieldLowerSnakeCase)

func checkFieldLowerSnakeCase(add addFunc, field protodesc.Field) error {
	message := field.Message()
	if message == nil {
		// just a sanity check
		return errors.New("field.Message() was nil")
	}
	if message.IsMapEntry() {
		// this check should always pass anyways but just in case
		return nil
	}
	name := field.Name()
	expectedName := fieldToLowerSnakeCase(name)
	if name != expectedName {
		add(field, field.NameLocation(), "Field name %q should be lower_snake_case, such as %q.", name, expectedName)
	}
	return nil
}

// CheckFieldNoDescriptor is a check function.
var CheckFieldNoDescriptor = newFieldCheckFunc(checkFieldNoDescriptor)

func checkFieldNoDescriptor(add addFunc, field protodesc.Field) error {
	name := field.Name()
	if strings.ToLower(strings.Trim(name, "_")) == "descriptor" {
		add(field, field.NameLocation(), `Field name %q cannot be any capitalization of "descriptor" with any number of prefix or suffix underscores.`, name)
	}
	return nil
}

// CheckFileLowerSnakeCase is a check function.
var CheckFileLowerSnakeCase = newFileCheckFunc(checkFileLowerSnakeCase)

func checkFileLowerSnakeCase(add addFunc, file protodesc.File) error {
	filename := file.FilePath()
	base := storagepath.Base(filename)
	ext := storagepath.Ext(filename)
	baseWithoutExt := strings.TrimSuffix(base, ext)
	expectedBaseWithoutExt := utilstring.ToLowerSnakeCase(baseWithoutExt)
	if baseWithoutExt != expectedBaseWithoutExt {
		add(file, nil, `Filename %q should be lower_snake_case%s, such as "%s%s".`, base, ext, expectedBaseWithoutExt, ext)
	}
	return nil
}

var (
	// CheckImportNoPublic is a check function.
	CheckImportNoPublic = newFileImportCheckFunc(checkImportNoPublic)
	// CheckImportNoWeak is a check function.
	CheckImportNoWeak = newFileImportCheckFunc(checkImportNoWeak)
)

func checkImportNoPublic(add addFunc, fileImport protodesc.FileImport) error {
	return checkImportNoPublicWeak(add, fileImport, fileImport.IsPublic(), "public")
}

func checkImportNoWeak(add addFunc, fileImport protodesc.FileImport) error {
	return checkImportNoPublicWeak(add, fileImport, fileImport.IsWeak(), "weak")
}

func checkImportNoPublicWeak(add addFunc, fileImport protodesc.FileImport, value bool, name string) error {
	if value {
		add(fileImport, fileImport.Location(), `Import %q must not be %s.`, fileImport.Import(), name)
	}
	return nil
}

// CheckMessagePascalCase is a check function.
var CheckMessagePascalCase = newMessageCheckFunc(checkMessagePascalCase)

func checkMessagePascalCase(add addFunc, message protodesc.Message) error {
	if message.IsMapEntry() {
		// map entries should always be pascal case but we don't want to check them anyways
		return nil
	}
	name := message.Name()
	expectedName := utilstring.ToPascalCase(name)
	if name != expectedName {
		add(message, message.NameLocation(), "Message name %q should be PascalCase, such as %q.", name, expectedName)
	}
	return nil
}

// CheckOneofLowerSnakeCase is a check function.
var CheckOneofLowerSnakeCase = newOneofCheckFunc(checkOneofLowerSnakeCase)

func checkOneofLowerSnakeCase(add addFunc, oneof protodesc.Oneof) error {
	name := oneof.Name()
	expectedName := fieldToLowerSnakeCase(name)
	if name != expectedName {
		add(oneof, oneof.NameLocation(), "Oneof name %q should be lower_snake_case, such as %q.", name, expectedName)
	}
	return nil
}

// CheckPackageDefined is a check function.
var CheckPackageDefined = newFileCheckFunc(checkPackageDefined)

func checkPackageDefined(add addFunc, file protodesc.File) error {
	if file.Package() == "" {
		add(file, nil, "Files must have a package defined.")
	}
	return nil
}

// CheckPackageDirectoryMatch is a check function.
var CheckPackageDirectoryMatch = newFileCheckFunc(checkPackageDirectoryMatch)

func checkPackageDirectoryMatch(add addFunc, file protodesc.File) error {
	pkg := file.Package()
	if pkg == "" {
		return nil
	}
	expectedDirPath := strings.ReplaceAll(pkg, ".", "/")
	dirPath := storagepath.Dir(file.FilePath())
	// need to check case where in root relative directory and no package defined
	// this should be valid although if SENSIBLE is turned on this will be invalid
	if dirPath != expectedDirPath {
		add(file, file.PackageLocation(), "Files with package %q must be within a directory %q relative to root but were in directory %q.", pkg, storagepath.Unnormalize(expectedDirPath), dirPath)
	}
	return nil
}

// CheckPackageLowerSnakeCase is a check function.
var CheckPackageLowerSnakeCase = newFileCheckFunc(checkPackageLowerSnakeCase)

func checkPackageLowerSnakeCase(add addFunc, file protodesc.File) error {
	pkg := file.Package()
	if pkg == "" {
		return nil
	}
	split := strings.Split(pkg, ".")
	for i, elem := range split {
		split[i] = utilstring.ToLowerSnakeCase(elem)
	}
	expectedPkg := strings.Join(split, ".")
	if pkg != expectedPkg {
		add(file, file.PackageLocation(), "Package name %q should be lower_snake.case, such as %q.", pkg, expectedPkg)
	}
	return nil
}

// CheckPackageSameDirectory is a check function.
var CheckPackageSameDirectory = newPackageToFilesCheckFunc(checkPackageSameDirectory)

func checkPackageSameDirectory(add addFunc, pkg string, files []protodesc.File) error {
	dirMap := make(map[string]struct{})
	for _, file := range files {
		dirMap[storagepath.Dir(file.FilePath())] = struct{}{}
	}
	if len(dirMap) > 1 {
		dirs := utilstring.MapToSortedSlice(dirMap)
		for _, file := range files {
			add(file, file.PackageLocation(), "Multiple directories %q contain files with package %q.", strings.Join(dirs, ","), pkg)
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

func checkPackageSameCsharpNamespace(add addFunc, pkg string, files []protodesc.File) error {
	return checkPackageSameOptionValue(add, pkg, files, protodesc.File.CsharpNamespace, protodesc.File.CsharpNamespaceLocation, "csharp_namespace")
}

func checkPackageSameGoPackage(add addFunc, pkg string, files []protodesc.File) error {
	return checkPackageSameOptionValue(add, pkg, files, protodesc.File.GoPackage, protodesc.File.GoPackageLocation, "go_package")
}

func checkPackageSameJavaMultipleFiles(add addFunc, pkg string, files []protodesc.File) error {
	return checkPackageSameOptionValue(
		add,
		pkg,
		files,
		func(file protodesc.File) string {
			return strconv.FormatBool(file.JavaMultipleFiles())
		},
		protodesc.File.JavaMultipleFilesLocation,
		"java_multiple_files",
	)
}

func checkPackageSameJavaPackage(add addFunc, pkg string, files []protodesc.File) error {
	return checkPackageSameOptionValue(add, pkg, files, protodesc.File.JavaPackage, protodesc.File.JavaPackageLocation, "java_package")
}

func checkPackageSamePhpNamespace(add addFunc, pkg string, files []protodesc.File) error {
	return checkPackageSameOptionValue(add, pkg, files, protodesc.File.PhpNamespace, protodesc.File.PhpNamespaceLocation, "php_namespace")
}

func checkPackageSameRubyPackage(add addFunc, pkg string, files []protodesc.File) error {
	return checkPackageSameOptionValue(add, pkg, files, protodesc.File.RubyPackage, protodesc.File.RubyPackageLocation, "ruby_package")
}

func checkPackageSameSwiftPrefix(add addFunc, pkg string, files []protodesc.File) error {
	return checkPackageSameOptionValue(add, pkg, files, protodesc.File.SwiftPrefix, protodesc.File.SwiftPrefixLocation, "swift_prefix")
}

func checkPackageSameOptionValue(
	add addFunc,
	pkg string,
	files []protodesc.File,
	getOptionValue func(protodesc.File) string,
	getOptionLocation func(protodesc.File) protodesc.Location,
	name string,
) error {
	optionValueMap := make(map[string]struct{})
	for _, file := range files {
		optionValueMap[getOptionValue(file)] = struct{}{}
	}
	if len(optionValueMap) > 1 {
		_, noOptionValue := optionValueMap[""]
		delete(optionValueMap, "")
		optionValues := utilstring.MapToSortedSlice(optionValueMap)
		for _, file := range files {
			if noOptionValue {
				add(file, getOptionLocation(file), "Files in package %q have both values %q and no value for option %q and all values must be equal.", pkg, strings.Join(optionValues, ","), name)
			} else {
				add(file, getOptionLocation(file), "Files in package %q have multiple values %q for option %q and all values must be equal.", pkg, strings.Join(optionValues, ","), name)
			}
		}
	}
	return nil
}

// CheckPackageVersionSuffix is a check function.
var CheckPackageVersionSuffix = newFileCheckFunc(checkPackageVersionSuffix)

func checkPackageVersionSuffix(add addFunc, file protodesc.File) error {
	pkg := file.Package()
	if pkg == "" {
		return nil
	}
	if !packageHasVersionSuffix(pkg) {
		add(file, file.PackageLocation(), `Package name %q should be suffixed with a correctly formed version, such as %q.`, pkg, pkg+".v1")
	}
	return nil
}

// CheckRPCNoClientStreaming is a check function.
var CheckRPCNoClientStreaming = newMethodCheckFunc(checkRPCNoClientStreaming)

func checkRPCNoClientStreaming(add addFunc, method protodesc.Method) error {
	if method.ClientStreaming() {
		add(method, method.Location(), "RPC %q is client streaming.", method.Name())
	}
	return nil
}

// CheckRPCNoServerStreaming is a check function.
var CheckRPCNoServerStreaming = newMethodCheckFunc(checkRPCNoServerStreaming)

func checkRPCNoServerStreaming(add addFunc, method protodesc.Method) error {
	if method.ServerStreaming() {
		add(method, method.Location(), "RPC %q is server streaming.", method.Name())
	}
	return nil
}

// CheckRPCPascalCase is a check function.
var CheckRPCPascalCase = newMethodCheckFunc(checkRPCPascalCase)

func checkRPCPascalCase(add addFunc, method protodesc.Method) error {
	name := method.Name()
	expectedName := utilstring.ToPascalCase(name)
	if name != expectedName {
		add(method, method.NameLocation(), "RPC name %q should be PascalCase, such as %q.", name, expectedName)
	}
	return nil
}

// CheckRPCRequestResponseUnique is a check function.
var CheckRPCRequestResponseUnique = func(
	id string,
	files []protodesc.File,
	allowSameRequestResponse bool,
	allowGoogleProtobufEmptyRequests bool,
	allowGoogleProtobufEmptyResponses bool,
) ([]*filev1beta1.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, files []protodesc.File) error {
			return checkRPCRequestResponseUnique(
				add,
				files,
				allowSameRequestResponse,
				allowGoogleProtobufEmptyRequests,
				allowGoogleProtobufEmptyResponses,
			)
		},
	)(id, files)
}

func checkRPCRequestResponseUnique(
	add addFunc,
	files []protodesc.File,
	allowSameRequestResponse bool,
	allowGoogleProtobufEmptyRequests bool,
	allowGoogleProtobufEmptyResponses bool,
) error {
	allFullNameToMethod, err := protodesc.FullNameToMethod(files...)
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
				if !(method.InputTypeName() == ".google.protobuf.Empty" && allowGoogleProtobufEmptyRequests && allowGoogleProtobufEmptyResponses) {
					add(method, method.Location(), "RPC %q has the same type %q for the request and response.", method.Name(), method.InputTypeName())
				}
			}
		}
	}
	// we have now added errors for the same request and response type if applicable
	// we can now check methods for unique usage of a given type
	requestResponseTypeToFullNameToMethod := make(map[string]map[string]protodesc.Method)
	for fullName, method := range allFullNameToMethod {
		for _, requestResponseType := range []string{method.InputTypeName(), method.OutputTypeName()} {
			fullNameToMethod, ok := requestResponseTypeToFullNameToMethod[requestResponseType]
			if !ok {
				fullNameToMethod = make(map[string]protodesc.Method)
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
		if requestResponseType == ".google.protobuf.Empty" && (allowGoogleProtobufEmptyRequests || allowGoogleProtobufEmptyResponses) {
			// if both requests and responses can be google.protobuf.Empty, then do not add any error
			// else, we check
			if !(allowGoogleProtobufEmptyRequests && allowGoogleProtobufEmptyResponses) {
				// inside this if statement, one of allowGoogleProtobufEmptyRequests or allowGoogleProtobufEmptyResponses is true
				var requestMethods []protodesc.Method
				var responseMethods []protodesc.Method
				for _, method := range fullNameToMethod {
					if method.InputTypeName() == ".google.protobuf.Empty" {
						requestMethods = append(requestMethods, method)
					}
					if method.OutputTypeName() == ".google.protobuf.Empty" {
						responseMethods = append(responseMethods, method)
					}
				}
				if !allowGoogleProtobufEmptyRequests && len(requestMethods) > 1 {
					for _, method := range requestMethods {
						add(method, method.Location(), "%q is used as the request for multiple RPCs.", requestResponseType)
					}
				}
				if !allowGoogleProtobufEmptyResponses && len(responseMethods) > 1 {
					for _, method := range responseMethods {
						add(method, method.Location(), "%q is used as the response for multiple RPCs.", requestResponseType)
					}
				}
			}
		} else {
			// else, we have a duplicate usage of requestResponseType, add an FileAnnotation to each method
			for _, method := range fullNameToMethod {
				add(method, method.Location(), "%q is used as the request or response type for multiple RPCs.", requestResponseType)
			}
		}
	}
	return nil
}

// CheckRPCRequestStandardName is a check function.
var CheckRPCRequestStandardName = func(id string, files []protodesc.File, allowGoogleProtobufEmptyRequests bool) ([]*filev1beta1.FileAnnotation, error) {
	return newMethodCheckFunc(
		func(add addFunc, method protodesc.Method) error {
			return checkRPCRequestStandardName(add, method, allowGoogleProtobufEmptyRequests)
		},
	)(id, files)
}

func checkRPCRequestStandardName(add addFunc, method protodesc.Method, allowGoogleProtobufEmptyRequests bool) error {
	service := method.Service()
	if service == nil {
		return errors.New("method.Service() is nil")
	}
	name := method.InputTypeName()
	if allowGoogleProtobufEmptyRequests && name == ".google.protobuf.Empty" {
		return nil
	}
	if strings.Contains(name, ".") {
		split := strings.Split(name, ".")
		name = split[len(split)-1]
	}
	expectedName1 := utilstring.ToPascalCase(method.Name()) + "Request"
	expectedName2 := utilstring.ToPascalCase(service.Name()) + expectedName1
	if name != expectedName1 && name != expectedName2 {
		add(method, method.InputTypeLocation(), "RPC request type %q should be named %q or %q.", name, expectedName1, expectedName2)
	}
	return nil
}

// CheckRPCResponseStandardName is a check function.
var CheckRPCResponseStandardName = func(id string, files []protodesc.File, allowGoogleProtobufEmptyResponses bool) ([]*filev1beta1.FileAnnotation, error) {
	return newMethodCheckFunc(
		func(add addFunc, method protodesc.Method) error {
			return checkRPCResponseStandardName(add, method, allowGoogleProtobufEmptyResponses)
		},
	)(id, files)
}

func checkRPCResponseStandardName(add addFunc, method protodesc.Method, allowGoogleProtobufEmptyResponses bool) error {
	service := method.Service()
	if service == nil {
		return errors.New("method.Service() is nil")
	}
	name := method.OutputTypeName()
	if allowGoogleProtobufEmptyResponses && name == ".google.protobuf.Empty" {
		return nil
	}
	if strings.Contains(name, ".") {
		split := strings.Split(name, ".")
		name = split[len(split)-1]
	}
	expectedName1 := utilstring.ToPascalCase(method.Name()) + "Response"
	expectedName2 := utilstring.ToPascalCase(service.Name()) + expectedName1
	if name != expectedName1 && name != expectedName2 {
		add(method, method.OutputTypeLocation(), "RPC response type %q should be named %q or %q.", name, expectedName1, expectedName2)
	}
	return nil
}

// CheckServicePascalCase is a check function.
var CheckServicePascalCase = newServiceCheckFunc(checkServicePascalCase)

func checkServicePascalCase(add addFunc, service protodesc.Service) error {
	name := service.Name()
	expectedName := utilstring.ToPascalCase(name)
	if name != expectedName {
		add(service, service.NameLocation(), "Service name %q should be PascalCase, such as %q.", name, expectedName)
	}
	return nil
}

// CheckServiceSuffix is a check function.
var CheckServiceSuffix = func(id string, files []protodesc.File, suffix string) ([]*filev1beta1.FileAnnotation, error) {
	return newServiceCheckFunc(
		func(add addFunc, service protodesc.Service) error {
			return checkServiceSuffix(add, service, suffix)
		},
	)(id, files)
}

func checkServiceSuffix(add addFunc, service protodesc.Service, suffix string) error {
	name := service.Name()
	if !strings.HasSuffix(name, suffix) {
		add(service, service.NameLocation(), "Service name %q should be suffixed with %q.", name, suffix)
	}
	return nil
}
