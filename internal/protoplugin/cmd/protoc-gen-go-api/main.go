// Copyright 2020-2021 Buf Technologies, Inc.
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

package main

import (
	"context"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/app/appproto"
	"github.com/bufbuild/buf/internal/pkg/protogenutil"
	"google.golang.org/protobuf/compiler/protogen"
)

const (
	contextPackage = protogen.GoImportPath("context")
	pluginName     = "api"
)

func main() {
	appproto.Main(context.Background(), protogenutil.NewNamedPerFileHandler(handle))
}

func handle(helper protogenutil.NamedHelper, plugin *protogen.Plugin, file *protogen.File) error {
	if len(file.Services) == 0 {
		return nil
	}
	g, err := helper.NewGeneratedFile(plugin, file, pluginName)
	if err != nil {
		return err
	}
	contextGoIdentString := g.QualifiedGoIdent(contextPackage.Ident("Context"))
	for _, service := range file.Services {
		interfaceName := service.GoName
		g.P(service.Comments.Leading, `type `, interfaceName, ` interface {`)
		for _, method := range service.Methods {
			if err := protogenutil.ValidateMethodUnary(method); err != nil {
				return err
			}
			requestParameterStrings, err := protogenutil.GetParameterStrings(g, method.Input.Fields)
			if err != nil {
				return err
			}
			responseParameterStrings, err := protogenutil.GetParameterStrings(g, method.Output.Fields)
			if err != nil {
				return err
			}
			funcName := method.GoName
			funcParameterStrings := append([]string{`ctx ` + contextGoIdentString}, requestParameterStrings...)
			funcReturnStrings := append(responseParameterStrings, `err error`)
			if len(funcParameterStrings) > 2 || len(funcReturnStrings) > 2 {
				g.P(method.Comments.Leading, funcName, `(`)
				for _, funcParameterString := range funcParameterStrings {
					g.P(funcParameterString, `,`)
				}
				g.P(`) (`, strings.Join(funcReturnStrings, `, `), `)`)
			} else {
				g.P(method.Comments.Leading, funcName, `(`, strings.Join(funcParameterStrings, `, `), `) (`, strings.Join(funcReturnStrings, `, `), `)`)
			}
		}
		g.P(`}`)
		g.P()
	}
	return nil
}
