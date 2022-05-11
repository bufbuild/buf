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
	"strings"

	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/protogenutil"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"google.golang.org/protobuf/compiler/protogen"
)

const (
	contextPackage      = protogen.GoImportPath("context")
	connectGoPackage    = protogen.GoImportPath("github.com/bufbuild/connect-go")
	buftransportPackage = protogen.GoImportPath("github.com/bufbuild/buf/private/bufpkg/buftransport")
	zapPackage          = protogen.GoImportPath("go.uber.org/zap")
	pluginName          = "apiclientconnect"
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
	globalAPIClientGoImportPath, err := helper.NewGlobalGoImportPath("apiclient")
	if err != nil {
		return err
	}
	providerSchemeGoIdent := globalAPIClientGoImportPath.Ident("ProviderScheme")
	providerSchemeGoIdentString := g.QualifiedGoIdent(providerSchemeGoIdent)
	contextGoIdentString := g.QualifiedGoIdent(contextPackage.Ident("Context"))
	httpClientGoIdentString := g.QualifiedGoIdent(connectGoPackage.Ident("HTTPClient"))
	withGRPCIdentString := g.QualifiedGoIdent(connectGoPackage.Ident("WithGRPC"))
	loggerGoIdentString := g.QualifiedGoIdent(zapPackage.Ident("Logger"))
	apiclientGoImportPath, err := helper.NewPackageGoImportPath(
		goPackageFileSet,
		"apiclient",
	)
	if err != nil {
		return err
	}
	providerGoIdent := apiclientGoImportPath.Ident("Provider")
	providerGoIdentString := g.QualifiedGoIdent(providerGoIdent)

	// NewProvider constructor function
	g.P(`// NewProvider returns a new Provider.`)
	g.P(`func NewProvider(`)
	g.P(`logger *`, loggerGoIdentString, `,`)
	g.P(`httpClient `, httpClientGoIdentString, `,`)
	g.P(`options `, `...ProviderOption`, `,`)
	g.P(`) `, providerGoIdentString, `{`)
	g.P(`provider := &provider{`)
	g.P(`logger: logger,`)
	g.P(`httpClient: httpClient,`)
	g.P(`}`)
	g.P(`for _, option := range options {`)
	g.P(`option(provider)`)
	g.P(`}`)
	g.P(`return provider`)
	g.P(`}`)
	g.P()

	// provider struct definition
	g.P(`type provider struct {`)
	g.P(`logger *`, loggerGoIdentString)
	g.P(`httpClient `, httpClientGoIdentString)
	g.P(`addressMapper func(string) string`)
	g.P(`contextModifierProvider func(string) (func (`, contextGoIdentString, `) `, contextGoIdentString, `, error)`)
	g.P(`scheme `, providerSchemeGoIdentString)
	g.P(`}`)
	g.P()

	// ProviderOption type
	g.P(`// ProviderOption is an option for a new Provider.`)
	g.P(`type ProviderOption func(*provider)`)
	g.P()

	// WithAddressMapper functional option
	g.P(`// WithAddressMapper maps the address with the given function.`)
	g.P(`func WithAddressMapper(addressMapper func(string) string) ProviderOption {`)
	g.P(`return func(provider *provider) {`)
	g.P(`provider.addressMapper = addressMapper`)
	g.P(`}`)
	g.P(`}`)
	g.P()

	// WithContextModifierProvider functional option
	// We create the contextModifier once for each address instead of passing address to a contextProvider directly
	// so that we do not have to do the same address logic for each RPC call, and only do it when we create the provider.
	// For example, we might read .netrc to create the contextProvider - we do not want to have to do this on every RPC call.
	g.P(`// WithContextModifierProvider provides a function that  modifies the context before every RPC invocation.`)
	g.P(`// Applied before the address mapper.`)
	g.P(`func WithContextModifierProvider(contextModifierProvider func(address string) (func(`, contextGoIdentString, `) `, contextGoIdentString, `, error)) ProviderOption {`)
	g.P(`return func(provider *provider) {`)
	g.P(`provider.contextModifierProvider = contextModifierProvider`)
	g.P(`}`)
	g.P(`}`)
	g.P()

	// WithScheme functional option
	g.P(`// WithScheme prepends the given scheme to the underlying transport address`)
	g.P(`func WithScheme(scheme `, providerSchemeGoIdentString, `) ProviderOption {`)
	g.P(`return func(provider *provider) {`)
	g.P(`provider.scheme = scheme`)
	g.P(`}`)
	g.P(`}`)
	g.P()

	// buildAddress helper function
	g.P(`// buildAddress modifies the given address with any additional options for transport such as the scheme and any subdomains`)
	g.P(`func (p *provider) buildAddress(address string) string {`)
	g.P(`if p.addressMapper != nil {`)
	g.P(`address = p.addressMapper(address)`)
	g.P(`}`)
	g.P(`if p.scheme != 0 {`)
	g.P(`address = p.scheme.String()+"://"+address`)
	g.P(`}`)
	g.P(`return address`)
	g.P(`}`)
	g.P()

	// Import path for the api named_go_package
	apiGoImportPath, err := helper.NewPackageGoImportPath(
		goPackageFileSet,
		"api",
	)
	if err != nil {
		return err
	}
	// Import path for the connect named_go_package
	connectGoImportPath, err := helper.NewPackageGoImportPath(
		goPackageFileSet,
		"connect",
	)
	if err != nil {
		return err
	}

	// Iterate over the services and create a constructor function for each
	for _, service := range goPackageFileSet.Services() {
		interfaceName := service.GoName
		interfaceGoIdent := apiGoImportPath.Ident(interfaceName)
		interfaceGoIdentString := g.QualifiedGoIdent(interfaceGoIdent)
		structName := protogenutil.GetUnexportGoName(interfaceName)
		newClientGoIdent := connectGoImportPath.Ident("New" + interfaceName + "Client")
		newClientGoIdentString := g.QualifiedGoIdent(newClientGoIdent)

		g.P(`func (p *provider) New`, interfaceName, `(ctx `, contextGoIdentString, `, baseURL string) (`, interfaceGoIdentString, `, error) {`)
		g.P(`var contextModifier func(context.Context) context.Context`)
		g.P(`var err error`)
		g.P(`if p.contextModifierProvider != nil {`)
		g.P(`contextModifier, err = p.contextModifierProvider(baseURL)`)
		g.P(`if err != nil {`)
		g.P(`return  nil, err`)
		g.P(`}`)
		g.P(`}`)
		g.P(`return &`, structName, `{`)
		g.P(`logger: p.logger,`)
		g.P(`client: `, newClientGoIdentString, `(`)
		g.P(`p.httpClient,`)
		g.P(`p.buildAddress(baseURL),`)
		g.P(withGRPCIdentString, `(),`)
		g.P(`),`)
		g.P(`contextModifier: contextModifier,`)
		g.P(`}, nil`)
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
	contextGoIdentString := g.QualifiedGoIdent(contextPackage.Ident("Context"))
	loggerGoIdentString := g.QualifiedGoIdent(zapPackage.Ident("Logger"))

	// Import path for the connect named_go_package
	connectGoImportPath, err := helper.NewGoImportPath(
		file,
		"connect",
	)
	if err != nil {
		return err
	}

	for _, service := range file.Services {
		interfaceName := service.GoName
		structName := protogenutil.GetUnexportGoName(interfaceName)
		clientGoIdent := connectGoImportPath.Ident(interfaceName + "Client")
		clientGoIdentString := g.QualifiedGoIdent(clientGoIdent)
		newRequestGoIdent := connectGoPackage.Ident("NewRequest")
		newRequestGoIdentString := g.QualifiedGoIdent(newRequestGoIdent)

		g.P(`type `, structName, ` struct {`)
		g.P(`logger *`, loggerGoIdentString)
		g.P(`client `, clientGoIdentString)
		g.P(`contextModifier func (`, contextGoIdentString, `) `, contextGoIdentString)
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
			g.P(`if s.contextModifier != nil{`)
			g.P(`ctx = s.contextModifier(ctx)`)
			g.P(`}`)

			requestGoIdentString := g.QualifiedGoIdent(method.Input.GoIdent)
			if len(funcReturnStrings) == 1 {
				g.P(`_, err := s.client.`, funcName, `(`)
			} else {
				g.P(`response, err := s.client.`, funcName, `(`)
			}
			g.P(`ctx,`)
			g.P(newRequestGoIdentString, `(`)
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
	httpClientGoIdentString := g.QualifiedGoIdent(connectGoPackage.Ident("HTTPClient"))
	loggerGoIdentString := g.QualifiedGoIdent(zapPackage.Ident("Logger"))
	globalAPIClientGoImportPath, err := helper.NewGlobalGoImportPath("apiclient")
	if err != nil {
		return err
	}
	globalProviderGoIdent := globalAPIClientGoImportPath.Ident("Provider")
	globalProviderGoIdentString := g.QualifiedGoIdent(globalProviderGoIdent)

	g.P(`// NewProvider returns a new provider.`)
	g.P(`func NewProvider(`)
	g.P(`logger *`, loggerGoIdentString, `,`)
	g.P(`httpClient `, httpClientGoIdentString, `,`)
	g.P(`options `, `...ProviderOption`, `,`)
	g.P(`) `, globalProviderGoIdentString, `{`)
	g.P(`providerOptions := &providerOptions{}`)
	g.P(`for _, option := range options {`)
	g.P(`option(providerOptions)`)
	g.P(`}`)
	g.P(`return &provider{`)
	for _, goPackageFileSet := range goPackageFileSetsWithServices {
		apiclientconnectGoImportPath, err := helper.NewPackageGoImportPath(goPackageFileSet, "apiclientconnect")
		if err != nil {
			return err
		}
		newProviderGoIdent := apiclientconnectGoImportPath.Ident("NewProvider")
		newProviderGoIdentString := g.QualifiedGoIdent(newProviderGoIdent)
		optionsName := protogenutil.GetUnexportGoName(
			stringutil.ToPascalCase(goPackageFileSet.ProtoPackage),
		) + "ProviderOptions"
		providerName := protogenutil.GetUnexportGoName(
			stringutil.ToPascalCase(goPackageFileSet.ProtoPackage),
		) + "Provider"
		g.P(providerName, `:`, newProviderGoIdentString, `(`)
		g.P(`logger,`)
		g.P(`httpClient,`)
		g.P(`providerOptions.`, optionsName, `...,`)
		g.P(`),`)
	}
	g.P(`}`)
	g.P(`}`)
	g.P()
	g.P(`type provider struct {`)
	for _, goPackageFileSet := range goPackageFileSetsWithServices {
		goImportPath, err := helper.NewPackageGoImportPath(goPackageFileSet, "apiclient")
		if err != nil {
			return err
		}
		providerGoIdent := goImportPath.Ident("Provider")
		providerGoIdentString := g.QualifiedGoIdent(providerGoIdent)
		providerName := protogenutil.GetUnexportGoName(
			stringutil.ToPascalCase(goPackageFileSet.ProtoPackage),
		) + "Provider"
		g.P(providerName, ` `, providerGoIdentString)
	}
	g.P(`}`)
	g.P()
	g.P(`// ProviderOption is an option for a new Provider.`)
	g.P(`type ProviderOption func(*providerOptions)`)
	g.P()
	for _, goPackageFileSet := range goPackageFileSetsWithServices {
		apiclientGoImportPath, err := helper.NewPackageGoImportPath(goPackageFileSet, "apiclient")
		if err != nil {
			return err
		}
		providerGoIdent := apiclientGoImportPath.Ident("Provider")
		providerGoIdentString := g.QualifiedGoIdent(providerGoIdent)
		providerName := protogenutil.GetUnexportGoName(
			stringutil.ToPascalCase(goPackageFileSet.ProtoPackage),
		) + "Provider"
		funcName := stringutil.ToPascalCase(goPackageFileSet.ProtoPackage)
		g.P(`func (p *provider) `, funcName, `() `, providerGoIdentString, `{`)
		g.P(`return p.`, providerName)
		g.P(`}`)
		g.P()
	}
	g.P(`type providerOptions struct {`)
	for _, goPackageFileSet := range goPackageFileSetsWithServices {
		goImportPath, err := helper.NewPackageGoImportPath(goPackageFileSet, "apiclientconnect")
		if err != nil {
			return err
		}
		providerOptionGoIdent := goImportPath.Ident("ProviderOption")
		providerOptionGoIdentString := g.QualifiedGoIdent(providerOptionGoIdent)
		optionsName := protogenutil.GetUnexportGoName(
			stringutil.ToPascalCase(goPackageFileSet.ProtoPackage),
		) + "ProviderOptions"
		g.P(optionsName, `[]`, providerOptionGoIdentString)
	}
	g.P(`}`)
	return nil
}
