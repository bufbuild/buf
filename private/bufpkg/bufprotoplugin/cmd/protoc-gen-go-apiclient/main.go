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

package main

import (
	"context"

	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/protogenutil"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"google.golang.org/protobuf/compiler/protogen"
)

const (
	contextPackage    = protogen.GoImportPath("context")
	connectPkgPackage = protogen.GoImportPath("github.com/bufbuild/buf/private/pkg/connect")
	pluginName        = "apiclient"
)

func main() {
	appproto.Main(context.Background(), protogenutil.NewNamedGoPackageHandler(handle))
}

func handle(helper protogenutil.NamedHelper, plugin *protogen.Plugin, goPackageFileSets []*protogenutil.GoPackageFileSet) error {
	for _, goPackageFileSet := range goPackageFileSets {
		if err := handleGoPackage(helper, plugin, goPackageFileSet); err != nil {
			return err
		}
	}
	return handleGlobal(helper, plugin, goPackageFileSets)
}

func handleGoPackage(helper protogenutil.NamedHelper, plugin *protogen.Plugin, goPackageFileSet *protogenutil.GoPackageFileSet) error {
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
	clientConfigGoIdentString := g.QualifiedGoIdent(connectPkgPackage.Ident("ClientConfig"))
	g.P(`// Provider provides all the types in `, goPackageName, `.`)
	g.P(`type Provider interface {`)
	g.P(`ToClientConfig() *`, clientConfigGoIdentString)
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

func handleGlobal(helper protogenutil.NamedHelper, plugin *protogen.Plugin, goPackageFileSets []*protogenutil.GoPackageFileSet) error {
	goPackageFileSetsWithServices := make([]*protogenutil.GoPackageFileSet, 0, len(goPackageFileSets))
	for _, goPackageFileSet := range goPackageFileSets {
		if len(goPackageFileSet.Services()) > 0 {
			goPackageFileSetsWithServices = append(goPackageFileSetsWithServices, goPackageFileSet)
		}
	}
	if len(goPackageFileSetsWithServices) == 0 {
		return nil
	}
	g, err := helper.NewGlobalGeneratedFile(plugin, pluginName)
	if err != nil {
		return err
	}
	g.P(`// Provider provides all Providers.`)
	g.P(`type Provider interface {`)
	for _, goPackageFileSet := range goPackageFileSetsWithServices {
		goImportPath, err := helper.NewPackageGoImportPath(goPackageFileSet, pluginName)
		if err != nil {
			return err
		}
		providerGoIdent := goImportPath.Ident("Provider")
		providerGoIdentString := g.QualifiedGoIdent(providerGoIdent)
		funcName := stringutil.ToPascalCase(goPackageFileSet.ProtoPackage)
		g.P(funcName, `() `, providerGoIdentString)
	}
	g.P(`}`)
	return nil
}
