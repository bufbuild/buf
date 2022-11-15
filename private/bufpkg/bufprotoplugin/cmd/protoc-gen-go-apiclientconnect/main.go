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
	"google.golang.org/protobuf/compiler/protogen"
)

const (
	contextPackage       = protogen.GoImportPath("context")
	connectGoPackage     = protogen.GoImportPath("github.com/bufbuild/connect-go")
	connectclientPackage = protogen.GoImportPath("github.com/bufbuild/buf/private/pkg/connectclient")
	zapPackage           = protogen.GoImportPath("go.uber.org/zap")
	pluginName           = "apiclientconnect"
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
	return nil
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
	contextGoIdentString := g.QualifiedGoIdent(contextPackage.Ident("Context"))
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

	httpClientGoIdentString := g.QualifiedGoIdent(connectGoPackage.Ident("HTTPClient"))
	interceptorGoIdentString := g.QualifiedGoIdent(connectGoPackage.Ident("Interceptor"))
	unaryInterceptorFuncGoIdentString := g.QualifiedGoIdent(connectGoPackage.Ident("UnaryInterceptorFunc"))

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
	g.P(`interceptors []`, interceptorGoIdentString)
	g.P(`authInterceptorProvider func(string)`, unaryInterceptorFuncGoIdentString)
	g.P(`}`)
	g.P()

	g.P(`// ProviderOption is an option for a new Provider.`)
	g.P(`type ProviderOption func(*provider)`)
	g.P()

	g.P(`// WithAddressMapper maps the address with the given function.`)
	g.P(`func WithAddressMapper(addressMapper func(string) string) ProviderOption {`)
	g.P(`return func(provider *provider) {`)
	g.P(`provider.addressMapper = addressMapper`)
	g.P(`}`)
	g.P(`}`)
	g.P()

	g.P(`// WithInterceptors adds the slice of interceptors to all clients returned from this provider.`)
	g.P(`func WithInterceptors(interceptors []`, interceptorGoIdentString, `) ProviderOption {`)
	g.P(`return func(provider *provider) {`)
	g.P(`provider.interceptors = interceptors`)
	g.P(`}`)
	g.P(`}`)
	g.P()
	g.P(`// WithAuthInterceptorProvider configures a provider that, when invoked, returns an interceptor that can be added`)
	g.P(`// to a client for setting the auth token`)
	g.P(`func WithAuthInterceptorProvider(authInterceptorProvider func(string) `, unaryInterceptorFuncGoIdentString, `) ProviderOption {`)
	g.P(`return func(provider *provider) {`)
	g.P(`provider.authInterceptorProvider = authInterceptorProvider`)
	g.P(`}`)
	g.P(`}`)
	g.P()

	clientConfigGoIdentString := g.QualifiedGoIdent(connectclientPackage.Ident("Config"))
	newClientConfigGoIdentString := g.QualifiedGoIdent(connectclientPackage.Ident("NewConfig"))
	clientConfigOptionGoIdentString := g.QualifiedGoIdent(connectclientPackage.Ident("ConfigOption"))
	clientConfigWithAddressMapperGoIdentString := g.QualifiedGoIdent(connectclientPackage.Ident("WithAddressMapper"))
	clientConfigWithInterceptorsGoIdentString := g.QualifiedGoIdent(connectclientPackage.Ident("WithInterceptors"))
	clientConfigWithAuthInterceptorGoIdentString := g.QualifiedGoIdent(connectclientPackage.Ident("WithAuthInterceptorProvider"))

	// Bridge to connect.ClientConfig, for alternate stub creation (for a world w/out these generated api helpers)
	g.P(`func (p *provider) ToClientConfig() *`, clientConfigGoIdentString, ` {`)
	g.P(`var opts []`, clientConfigOptionGoIdentString)
	g.P(`if p.addressMapper != nil {`)
	g.P(`opts = append(opts, `, clientConfigWithAddressMapperGoIdentString, `(p.addressMapper))`)
	g.P(`}`)
	g.P(`if len(p.interceptors) > 0 {`)
	g.P(`opts = append(opts, `, clientConfigWithInterceptorsGoIdentString, `(p.interceptors))`)
	g.P(`}`)
	g.P(`if p.authInterceptorProvider != nil {`)
	g.P(`opts = append(opts, `, clientConfigWithAuthInterceptorGoIdentString, `(p.authInterceptorProvider))`)
	g.P(`}`)
	g.P(`return `, newClientConfigGoIdentString, `(p.httpClient, opts...)`)
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

	withInterceptorsGoIdentString := g.QualifiedGoIdent(connectGoPackage.Ident("WithInterceptors"))

	// Iterate over the services and create a constructor function for each
	for _, service := range goPackageFileSet.Services() {
		interfaceName := service.GoName
		structName := protogenutil.GetUnexportGoName(interfaceName)
		interfaceGoIdent := apiGoImportPath.Ident(interfaceName)
		interfaceGoIdentString := g.QualifiedGoIdent(interfaceGoIdent)
		newClientGoIdent := connectGoImportPath.Ident("New" + interfaceName + "Client")
		newClientGoIdentString := g.QualifiedGoIdent(newClientGoIdent)

		g.P(`// New`, interfaceName, ` creates a new `, interfaceName)
		g.P(`func (p *provider) New`, interfaceName, `(ctx `, contextGoIdentString, `, address string) (`, interfaceGoIdentString, `, error) {`)

		g.P(`interceptors := p.interceptors`)
		g.P(`if p.authInterceptorProvider != nil {`)
		g.P(`interceptor := p.authInterceptorProvider(address)`)
		g.P(`interceptors = append(interceptors, interceptor)`)
		g.P(`}`)

		g.P(`if p.addressMapper != nil {`)
		g.P(`address = p.addressMapper(address)`)
		g.P(`}`)

		g.P(`return &`, structName, `Client{`)
		g.P(`logger: p.logger,`)
		g.P(`client: `, newClientGoIdentString, `(`)
		g.P(`p.httpClient,`)
		g.P(`address,`)
		g.P(withInterceptorsGoIdentString, `(interceptors...),`)
		g.P(`),`)
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

		g.P(`type `, structName, `Client struct {`)
		g.P(`logger *`, loggerGoIdentString)
		g.P(`client `, clientGoIdentString)
		g.P(`}`)
		g.P()
		g.P(`func (s *`, structName, `Client) Unwrap() `, clientGoIdentString, ` {`)
		g.P(`return s.client`)
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
				g.P(method.Comments.Leading, `func (s *`, structName, `Client) `, funcName, `(`)
				for _, funcParameterString := range funcParameterStrings {
					g.P(funcParameterString, `,`)
				}
				g.P(`) (`, strings.Join(funcReturnStrings, `, `), `) {`)
			} else {
				g.P(method.Comments.Leading, `func (s *`, structName, `Client) `, funcName, `(`, strings.Join(funcParameterStrings, `, `), `) (`, strings.Join(funcReturnStrings, `, `), `) {`)
			}
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
