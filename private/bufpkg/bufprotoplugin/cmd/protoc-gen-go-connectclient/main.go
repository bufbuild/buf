// Copyright 2020-2022 Buf Technologies, Inc.
//
// All rights reserved.

package main

import (
	"context"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/protogenutil"
	"google.golang.org/protobuf/compiler/protogen"
)

const (
	contextPackage = protogen.GoImportPath("context")
	connectPackage = protogen.GoImportPath("github.com/bufbuild/connect-go")

	pluginName = "connectclient"
)

func main() {
	appproto.Main(context.Background(), protogenutil.NewNamedPerGoPackageHandler(handle))
}

func handle(helper protogenutil.NamedHelper, plugin *protogen.Plugin, goPackageFileSet *protogenutil.GoPackageFileSet) error {
	if len(goPackageFileSet.Services()) == 0 {
		return nil
	}
	if err := generatePackageFile(helper, plugin, goPackageFileSet); err != nil {
		return err
	}
	for _, file := range goPackageFileSet.Files {
		if len(file.Services) > 0 {
			if err := generateServiceFile(helper, plugin, file); err != nil {
				return err
			}
		}
	}
	return nil
}

func generatePackageFile(helper protogenutil.NamedHelper, plugin *protogen.Plugin, goPackageFileSet *protogenutil.GoPackageFileSet) error {
	g, err := helper.NewPackageGeneratedFile(plugin, goPackageFileSet, pluginName)
	if err != nil {
		return err
	}
	services := goPackageFileSet.Services()
	apiGoImportPath, err := helper.NewPackageGoImportPath(goPackageFileSet, "api")
	if err != nil {
		return err
	}

	for _, service := range services {
		interfaceName := service.GoName
		apiInterfaceGoIdent := apiGoImportPath.Ident(interfaceName)
		apiInterfaceGoIdentString := g.QualifiedGoIdent(apiInterfaceGoIdent)
		g.P(`func `, "New"+service.GoName+"Client", `(`)
		g.P(`client `, g.QualifiedGoIdent(connectPackage.Ident("HTTPClient")), `,`)
		g.P(`address string,`)
		g.P(`options ...`, g.QualifiedGoIdent(connectPackage.Ident("ClientOption")), `,`)
		g.P(`) `, apiInterfaceGoIdentString, ` {`)
		g.P(`return `, "new"+service.GoName+"Client", `(client, address, options...)`)
		g.P(`}`)
		g.P()
	}
	return nil
}

func generateServiceFile(helper protogenutil.NamedHelper, plugin *protogen.Plugin, file *protogen.File) error {
	g, err := helper.NewGeneratedFile(plugin, file, pluginName)
	if err != nil {
		return err
	}
	connectrpc, err := helper.NewGoImportPath(file, "connect")
	if err != nil {
		return err
	}
	connectPkgGoIdentString := g.QualifiedGoIdent(connectPackage.Ident("ClientOption"))
	contextGoIdentString := g.QualifiedGoIdent(contextPackage.Ident("Context"))

	for _, service := range file.Services {
		interfaceName := service.GoName
		structName := protogenutil.GetUnexportGoName(interfaceName) + "Client"

		g.P(`type `, structName, ` struct {`)
		g.P(`client `, connectrpc.Ident(interfaceName+"Client"))
		g.P(`}`)
		g.P()
		g.P(`func new`, interfaceName, `Client(`)
		g.P(`httpClient `, g.QualifiedGoIdent(connectPackage.Ident("HTTPClient")), `,`)
		g.P(`address string,`)
		g.P(`options ...`, connectPkgGoIdentString, `,`)
		g.P(`) *`, protogenutil.GetUnexportGoName(interfaceName+"Client"), ` {`)
		g.P(`return &`, structName, `{`)
		g.P(`client: `, connectrpc.Ident(`New`+interfaceName+`Client`), `(`)
		g.P(`httpClient,`)
		g.P(`address,`)
		g.P(`options...,`)
		g.P(`),`)
		g.P(`}`)
		g.P(`}`)
		g.P()

		for _, method := range service.Methods {
			if err := protogenutil.ValidateMethodUnary(method); err != nil {
				return err
			}
			requestParameterStrings, responseParameterStrings, err := protogenutil.GetRequestAndResponseParameterStrings(g, method.Input.Fields, method.Output.Fields)
			if err != nil {
				return err
			}
			funcName := method.GoName
			funcParameterStrings := append([]string{`ctx ` + contextGoIdentString}, requestParameterStrings...)
			funcReturnStrings := append(responseParameterStrings, `_ error`)
			if len(funcParameterStrings) > 2 || len(funcReturnStrings) > 2 {
				g.P(method.Comments.Leading, `func (s *`, structName, `) `, funcName, `(`)
				for _, funcParameterString := range funcParameterStrings {
					g.P(funcParameterString, `,`)
				}
				g.P(`) (`, strings.Join(funcReturnStrings, `, `), `) {`)
			} else {
				g.P(method.Comments.Leading, `func (s *`, structName, `) `, funcName, `(`, strings.Join(funcParameterStrings, `, `), `) (`, strings.Join(funcReturnStrings, `, `), `) {`)
			}
			requestGoIdentString := g.QualifiedGoIdent(method.Input.GoIdent)
			if len(funcReturnStrings) == 1 {
				g.P(`_, err := s.client.`, funcName, `(`)
			} else {
				g.P(`response, err := s.client.`, funcName, `(`)
			}
			g.P(`ctx,`)
			g.P(g.QualifiedGoIdent(connectPackage.Ident("NewRequest")), `(`)
			g.P(`&`, requestGoIdentString, `{`)
			for _, field := range method.Input.Fields {
				g.P(field.GoName, `: `, protogenutil.GetUnexportGoName(field.GoName), `,`)
			}
			g.P(`}),`)
			g.P(`)`)

			g.P(`if err != nil {`)
			errorReturnString, err := protogenutil.GetParameterErrorReturnString(
				g,
				method.Output.Fields,
				`err`,
			)
			if err != nil {
				return err
			}
			g.P(errorReturnString)
			g.P(`}`)
			returnValueStrings := make([]string, 0, len(method.Output.Fields))
			for _, field := range method.Output.Fields {
				returnValueStrings = append(returnValueStrings, "response.Msg."+field.GoName)
			}
			returnValueStrings = append(returnValueStrings, "nil")
			g.P(`return `, strings.Join(returnValueStrings, ", "))
			g.P(`}`)
			g.P()
		}
		g.P()
	}

	return nil
}
