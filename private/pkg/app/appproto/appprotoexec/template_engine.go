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

package appprotoexec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/aymerick/raymond"
	"github.com/bufbuild/buf/private/pkg/storage"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

const goImportBlockFmt = `
import (
%s
)
`

type templateEngine struct {
	readBucket storage.ReadBucket
}

func newTemplateEngine(readBucket storage.ReadBucket) (*templateEngine, error) {
	return &templateEngine{
		readBucket: readBucket,
	}, nil
}

// Generate executes the given template(s) for all of the files
// contained in the CodeGeneratorRequest.
func (t *templateEngine) Generate(
	ctx context.Context,
	request *pluginpb.CodeGeneratorRequest,
) (*pluginpb.CodeGeneratorResponse, error) {
	// TODO: Temporary hack to get a CodeGeneratorRequest.
	bytes, err := protojson.Marshal(request)
	if err != nil {
		return nil, err
	}
	f, err := os.Create("request.bin")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := f.Write(bytes); err != nil {
		return nil, err
	}
	filesToGenerate := make(map[string]struct{})
	for _, fileToGenerate := range request.FileToGenerate {
		filesToGenerate[fileToGenerate] = struct{}{}
	}
	// Path -> Content
	templates := make(map[*raymond.Template]*raymond.Template)
	if err := t.readBucket.Walk(ctx, "", func(objectInfo storage.ObjectInfo) error {
		// Remove the .tmpl suffix from the path template so that it isn't included
		// in the generated filename.
		pathTemplate, err := raymond.Parse(strings.TrimSuffix(objectInfo.Path(), ".tmpl"))
		if err != nil {
			return err
		}
		contentTemplate, err := raymond.ParseFile(objectInfo.ExternalPath())
		if err != nil {
			return err
		}
		templates[pathTemplate] = contentTemplate
		return nil
	}); err != nil {
		return nil, err
	}
	fileSet := make(map[string]*descriptorpb.FileDescriptorProto)
	for _, file := range request.ProtoFile {
		fileSet[file.GetName()] = file
	}
	var generatedFiles []*pluginpb.CodeGeneratorResponse_File
	for _, file := range request.ProtoFile {
		if _, ok := filesToGenerate[file.GetName()]; !ok {
			continue
		}
		templateData := newTemplateData(file)
		templateHelpers := newTemplateHelpers(file, fileSet)
		for pathTemplate, contentTemplate := range templates {
			// We need to clone the template so that the helper functions are tailored
			// to each file.
			pathTemplateClone := pathTemplate.Clone()
			pathTemplateClone.RegisterHelpers(templateHelpers)
			generatedFilename, err := pathTemplateClone.Exec(templateData)
			if err != nil {
				return nil, err
			}
			contentTemplateClone := contentTemplate.Clone()
			contentTemplateClone.RegisterHelpers(templateHelpers)
			generatedContent, err := contentTemplateClone.Exec(templateData)
			if err != nil {
				return nil, err
			}
			// Now that the content has been processed once, we run through it
			// again to capture any post-processing tags.
			postProcessTemplate, err := raymond.Parse(generatedContent)
			if err != nil {
				return nil, err
			}
			postProcessTemplate.RegisterHelpers(templateHelpers)
			generatedContent, err = postProcessTemplate.Exec(templateData)
			if err != nil {
				return nil, err
			}
			generatedFiles = append(
				generatedFiles,
				&pluginpb.CodeGeneratorResponse_File{
					Name:    &generatedFilename,
					Content: &generatedContent,
				},
			)
		}
	}
	return &pluginpb.CodeGeneratorResponse{
		File: generatedFiles,
	}, nil
}

// TODO: This should probably be moved to its own file.
func newTemplateHelpers(file *descriptorpb.FileDescriptorProto, dependencies map[string]*descriptorpb.FileDescriptorProto) map[string]interface{} {
	goImports := make(map[string]string)
	goGlobals := make(map[string]struct{})
	return map[string]interface{}{
		"goImport": func(importPath string) string {
			if alias, ok := goImports[importPath]; ok {
				return fmt.Sprintf("%s %q", alias, importPath)
			}
			alias := goPackageNameFromImportPath(importPath)
			for i := 2; ; i++ {
				_, taken := goGlobals[alias]
				if !taken && !isGoKeyword(alias) {
					break
				}
				alias = fmt.Sprintf("%s%d", alias, i)
			}
			goImports[importPath] = alias
			goGlobals[alias] = struct{}{}
			return fmt.Sprintf("%s %q", alias, importPath)
		},
		"goImports": func() string {
			// This template function generates a placehodler for
			// another template function. In this case, it adds a
			// sentinel value for post-processing.
			return "{{{renderGoImports}}}"
		},
		"renderGoImports": func() string {
			// This function should not be called by users directly -
			// it's a template function that will take affect in the
			// post-processing phase.
			var importStatements []string
			for importPath, alias := range goImports {
				importStatements = append(importStatements, fmt.Sprintf("  %s %q", alias, importPath))
			}
			importBlock := strings.Join(importStatements, "\n")
			return fmt.Sprintf(goImportBlockFmt, importBlock)
		},
		"goImportAlias": func(importPath string) string {
			if alias, ok := goImports[importPath]; ok {
				return alias
			}
			alias := goPackageNameFromImportPath(importPath)
			for i := 2; ; i++ {
				_, taken := goGlobals[alias]
				if !taken && !isGoKeyword(alias) {
					break
				}
				alias = fmt.Sprintf("%s%d", alias, i)
			}
			goImports[importPath] = alias
			goGlobals[alias] = struct{}{}
			return alias
		},
		"goImportPath": func(filename string) string {
			_, ok := dependencies[filename]
			if !ok {
				return "<unknown>"
			}
			// TODO: We would need to trim the final ';' component, if any.
			// This also doesn't account for modifier flags.
			return "github.com/bufbuild/buf/gen/proto/something"
			// return dependency.GetOptions().GetGoPackage()
		},
		"goType": func(typeName string) string {
			// TODO: We would need to understand where this type was defined,
			// then reference the go_package path and resolve the appropriate
			// alias from there.
			return typeName
		},
		"trimSuffix": func(value string, suffix string) string {
			return strings.TrimSuffix(value, suffix)
		},
	}
}

// goPackageNameFromImportPath returns a Go package name suitable
// for the given import path.
func goPackageNameFromImportPath(importPath string) string {
	packageName := filepath.Base(importPath)
	packageName = strings.TrimSuffix(packageName, "-go")
	return strings.Map(func(c rune) rune {
		switch {
		case unicode.IsLetter(c), unicode.IsDigit(c):
			return c
		default:
			return '_'
		}
	}, packageName)
}

func isGoKeyword(value string) bool {
	_, ok := goKeywords[value]
	return ok
}

// goKeywords is a set of the Go language keywords.
var goKeywords = map[string]struct{}{
	"any":         {},
	"break":       {},
	"case":        {},
	"chan":        {},
	"const":       {},
	"continue":    {},
	"default":     {},
	"defer":       {},
	"else":        {},
	"fallthrough": {},
	"for":         {},
	"func":        {},
	"go":          {},
	"goto":        {},
	"if":          {},
	"import":      {},
	"interface":   {},
	"map":         {},
	"package":     {},
	"range":       {},
	"return":      {},
	"select":      {},
	"struct":      {},
	"switch":      {},
	"type":        {},
	"var":         {},
}
