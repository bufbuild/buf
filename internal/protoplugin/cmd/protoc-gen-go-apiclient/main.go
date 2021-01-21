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

	"github.com/bufbuild/buf/internal/pkg/app/appproto"
	"github.com/bufbuild/buf/internal/pkg/protogenutil"
	"google.golang.org/protobuf/compiler/protogen"
)

const (
	contextPackage = protogen.GoImportPath("context")
	pluginName     = "apiclient"
)

func main() {
	appproto.Main(context.Background(), protogenutil.NewNamedPerGoPackageHandler(handle))
}

func handle(helper protogenutil.NamedHelper, plugin *protogen.Plugin, goPackageFileSet *protogenutil.GoPackageFileSet) error {
	services := goPackageFileSet.Services()
	if len(services) == 0 {
		return nil
	}
	g, err := helper.NewPackageGeneratedFile(plugin, goPackageFileSet, pluginName)
	if err != nil {
		return err
	}
	goPackageName := helper.NewGoPackageName(goPackageFileSet.GoPackageName, pluginName)
	contextGoIdentString := g.QualifiedGoIdent(contextPackage.Ident("Context"))
	g.P(`// Provider provides all the types in `, goPackageName, `.`)
	g.P(`type Provider interface {`)
	for _, service := range services {
		providerInterfaceName := service.GoName + "Provider"
		g.P(providerInterfaceName)
	}
	g.P(`}`)
	g.P()
	for _, service := range services {
		apiGoImportPath, err := helper.NewPackageGoImportPath(
			goPackageFileSet,
			"api",
		)
		if err != nil {
			return err
		}
		interfaceName := service.GoName
		interfaceGoIdent := apiGoImportPath.Ident(interfaceName)
		interfaceGoIdentString := g.QualifiedGoIdent(interfaceGoIdent)
		providerInterfaceName := service.GoName + "Provider"
		g.P(`// `, providerInterfaceName, ` provides a client-side `, interfaceName, ` for an address.`)
		g.P(`type `, providerInterfaceName, ` interface {`)
		g.P(`New`, interfaceName, `(ctx `, contextGoIdentString, `, address string) (`, interfaceGoIdentString, `, error)`)
		g.P(`}`)
		g.P()
	}
	return nil
}
