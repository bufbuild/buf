package protodesc

import "github.com/bufbuild/buf/internal/pkg/protodescpb"

type file struct {
	descriptor

	fileDescriptor protodescpb.FileDescriptor

	syntax      Syntax
	fileImports []FileImport
	messages    []Message
	enums       []Enum
	services    []Service

	optimizeMode FileOptionsOptimizeMode
}

func newFile(fileDescriptor protodescpb.FileDescriptor) (*file, error) {
	return newFileBuilder(fileDescriptor).toFile()
}

func (f *file) Syntax() Syntax {
	return f.syntax
}

func (f *file) FileImports() []FileImport {
	return f.fileImports
}

func (f *file) Messages() []Message {
	return f.messages
}

func (f *file) Enums() []Enum {
	return f.enums
}

func (f *file) Services() []Service {
	return f.services
}

func (f *file) CsharpNamespace() string {
	return f.fileDescriptor.GetOptions().GetCsharpNamespace()
}

func (f *file) GoPackage() string {
	return f.fileDescriptor.GetOptions().GetGoPackage()
}

func (f *file) JavaMultipleFiles() bool {
	return f.fileDescriptor.GetOptions().GetJavaMultipleFiles()
}

func (f *file) JavaOuterClassname() string {
	return f.fileDescriptor.GetOptions().GetJavaOuterClassname()
}

func (f *file) JavaPackage() string {
	return f.fileDescriptor.GetOptions().GetJavaPackage()
}

func (f *file) JavaStringCheckUtf8() bool {
	return f.fileDescriptor.GetOptions().GetJavaStringCheckUtf8()
}

func (f *file) ObjcClassPrefix() string {
	return f.fileDescriptor.GetOptions().GetObjcClassPrefix()
}

func (f *file) PhpClassPrefix() string {
	return f.fileDescriptor.GetOptions().GetPhpClassPrefix()
}

func (f *file) PhpNamespace() string {
	return f.fileDescriptor.GetOptions().GetPhpNamespace()
}

func (f *file) PhpMetadataNamespace() string {
	return f.fileDescriptor.GetOptions().GetPhpMetadataNamespace()
}

func (f *file) RubyPackage() string {
	return f.fileDescriptor.GetOptions().GetRubyPackage()
}

func (f *file) SwiftPrefix() string {
	return f.fileDescriptor.GetOptions().GetSwiftPrefix()
}

func (f *file) OptimizeFor() FileOptionsOptimizeMode {
	return f.optimizeMode
}

func (f *file) CcGenericServices() bool {
	return f.fileDescriptor.GetOptions().GetCcGenericServices()
}

func (f *file) JavaGenericServices() bool {
	return f.fileDescriptor.GetOptions().GetJavaGenericServices()
}

func (f *file) PyGenericServices() bool {
	return f.fileDescriptor.GetOptions().GetPyGenericServices()
}

func (f *file) PhpGenericServices() bool {
	return f.fileDescriptor.GetOptions().GetPhpGenericServices()
}

func (f *file) CcEnableArenas() bool {
	return f.fileDescriptor.GetOptions().GetCcEnableArenas()
}

func (f *file) PackageLocation() Location {
	return f.getLocationByPathKey(packagePathKey)
}

func (f *file) CsharpNamespaceLocation() Location {
	return f.getLocationByPathKey(csharpNamespacePathKey)
}

func (f *file) GoPackageLocation() Location {
	return f.getLocationByPathKey(goPackagePathKey)
}

func (f *file) JavaMultipleFilesLocation() Location {
	return f.getLocationByPathKey(javaMultipleFilesPathKey)
}

func (f *file) JavaOuterClassnameLocation() Location {
	return f.getLocationByPathKey(javaOuterClassnamePathKey)
}

func (f *file) JavaPackageLocation() Location {
	return f.getLocationByPathKey(javaPackagePathKey)
}

func (f *file) JavaStringCheckUtf8Location() Location {
	return f.getLocationByPathKey(javaStringCheckUtf8PathKey)
}

func (f *file) ObjcClassPrefixLocation() Location {
	return f.getLocationByPathKey(objcClassPrefixPathKey)
}

func (f *file) PhpClassPrefixLocation() Location {
	return f.getLocationByPathKey(phpClassPrefixPathKey)
}

func (f *file) PhpNamespaceLocation() Location {
	return f.getLocationByPathKey(phpNamespacePathKey)
}

func (f *file) PhpMetadataNamespaceLocation() Location {
	return f.getLocationByPathKey(phpMetadataNamespacePathKey)
}

func (f *file) RubyPackageLocation() Location {
	return f.getLocationByPathKey(rubyPackagePathKey)
}

func (f *file) SwiftPrefixLocation() Location {
	return f.getLocationByPathKey(swiftPrefixPathKey)
}

func (f *file) OptimizeForLocation() Location {
	return f.getLocationByPathKey(optimizeForPathKey)
}

func (f *file) CcGenericServicesLocation() Location {
	return f.getLocationByPathKey(ccGenericServicesPathKey)
}

func (f *file) JavaGenericServicesLocation() Location {
	return f.getLocationByPathKey(javaGenericServicesPathKey)
}

func (f *file) PyGenericServicesLocation() Location {
	return f.getLocationByPathKey(pyGenericServicesPathKey)
}

func (f *file) PhpGenericServicesLocation() Location {
	return f.getLocationByPathKey(phpGenericServicesPathKey)
}

func (f *file) CcEnableArenasLocation() Location {
	return f.getLocationByPathKey(ccEnableArenasPathKey)
}

func (f *file) SyntaxLocation() Location {
	return f.getLocationByPathKey(syntaxPathKey)
}
