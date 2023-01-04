// Copyright 2020-2023 Buf Technologies, Inc.
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

package protogenutil

import (
	"fmt"
	"path"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

const namedHelperGoPackageOptionKey = "named_go_package"

type namedHelper struct {
	pluginNameToGoPackage map[string]string
}

func newNamedHelper() *namedHelper {
	return &namedHelper{
		pluginNameToGoPackage: make(map[string]string),
	}
}

func (h *namedHelper) NewGoPackageName(
	baseGoPackageName protogen.GoPackageName,
	pluginName string,
) protogen.GoPackageName {
	return protogen.GoPackageName(string(baseGoPackageName) + pluginName)
}

func (h *namedHelper) NewGoImportPath(
	file *protogen.File,
	pluginName string,
) (protogen.GoImportPath, error) {
	return h.newGoImportPath(
		path.Dir(file.GeneratedFilenamePrefix),
		file.GoPackageName,
		pluginName,
	)
}

func (h *namedHelper) NewPackageGoImportPath(
	goPackageFileSet *GoPackageFileSet,
	pluginName string,
) (protogen.GoImportPath, error) {
	return h.newGoImportPath(
		goPackageFileSet.GeneratedDir,
		goPackageFileSet.GoPackageName,
		pluginName,
	)
}

func (h *namedHelper) NewGlobalGoImportPath(
	pluginName string,
) (protogen.GoImportPath, error) {
	goPackage, ok := h.pluginNameToGoPackage[pluginName]
	if !ok {
		return "", fmt.Errorf("no %s specified for plugin %s", namedHelperGoPackageOptionKey, pluginName)
	}
	return protogen.GoImportPath(goPackage), nil
}

func (h *namedHelper) NewGeneratedFile(
	plugin *protogen.Plugin,
	file *protogen.File,
	pluginName string,
) (*protogen.GeneratedFile, error) {
	goImportPath, err := h.NewGoImportPath(file, pluginName)
	if err != nil {
		return nil, err
	}
	goPackageName := h.NewGoPackageName(file.GoPackageName, pluginName)
	generatedFilePath := path.Dir(file.GeneratedFilenamePrefix) +
		"/" + string(goPackageName) +
		"/" + path.Base(file.GeneratedFilenamePrefix) +
		".pb.go"

	generatedFile := plugin.NewGeneratedFile(generatedFilePath, goImportPath)
	printGeneratedFileNamedHelperHeader(generatedFile, goPackageName, pluginName)
	return generatedFile, nil
}

func (h *namedHelper) NewPackageGeneratedFile(
	plugin *protogen.Plugin,
	goPackageFileSet *GoPackageFileSet,
	pluginName string,
) (*protogen.GeneratedFile, error) {
	goImportPath, err := h.NewPackageGoImportPath(goPackageFileSet, pluginName)
	if err != nil {
		return nil, err
	}
	goPackageName := h.NewGoPackageName(goPackageFileSet.GoPackageName, pluginName)
	fileBaseName := string(goPackageName)
	// make sure this file name would not overlap with any actual file name
	for _, file := range goPackageFileSet.Files {
		if path.Base(file.GeneratedFilenamePrefix) == fileBaseName {
			fileBaseName = fileBaseName + "_pkg"
			// do not break, just for the malicious case, where there is a file
			// packagename_pkg.proto, packagename_pkg_pkg.proto, etc
		}
	}
	generatedFilePath := goPackageFileSet.GeneratedDir +
		"/" + string(goPackageName) +
		"/" + fileBaseName +
		".pb.go"

	generatedFile := plugin.NewGeneratedFile(generatedFilePath, goImportPath)
	printGeneratedFileNamedHelperHeader(generatedFile, goPackageName, pluginName)
	return generatedFile, nil
}

func (h *namedHelper) NewGlobalGeneratedFile(
	plugin *protogen.Plugin,
	pluginName string,
) (*protogen.GeneratedFile, error) {
	goImportPath, err := h.NewGlobalGoImportPath(pluginName)
	if err != nil {
		return nil, err
	}
	goPackageName := h.NewGoPackageName("", pluginName)
	generatedFilePath := string(goPackageName) + ".pb.go"

	generatedFile := plugin.NewGeneratedFile(generatedFilePath, goImportPath)
	printGeneratedFileNamedHelperHeader(generatedFile, goPackageName, pluginName)
	return generatedFile, nil
}

func (h *namedHelper) newGoImportPath(
	generatedDir string,
	baseGoPackageName protogen.GoPackageName,
	pluginName string,
) (protogen.GoImportPath, error) {
	goPackage, ok := h.pluginNameToGoPackage[pluginName]
	if !ok {
		return "", fmt.Errorf("no %s specified for plugin %s", namedHelperGoPackageOptionKey, pluginName)
	}
	return protogen.GoImportPath(goPackage +
		"/" + generatedDir +
		"/" + string(h.NewGoPackageName(baseGoPackageName, pluginName))), nil
}

func (h *namedHelper) handleOption(key string, value string) error {
	if key != namedHelperGoPackageOptionKey {
		return nil
	}
	split := strings.Split(value, "=")
	if len(split) != 2 {
		return fmt.Errorf("unknown value for %s: %s", namedHelperGoPackageOptionKey, value)
	}
	h.pluginNameToGoPackage[split[0]] = split[1]
	return nil
}

func printGeneratedFileNamedHelperHeader(
	generatedFile *protogen.GeneratedFile,
	goPackageName protogen.GoPackageName,
	pluginName string,
) {
	generatedFile.P("// Code generated by protoc-gen-go-", pluginName, ". DO NOT EDIT.")
	generatedFile.P()
	generatedFile.P("package ", goPackageName)
	generatedFile.P()
}
