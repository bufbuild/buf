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

package price

import (
	"context"
	"fmt"
	"text/template"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulestat"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/app/applog"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/protostat"
	"github.com/bufbuild/connect-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	disableSymlinksFlagName       = "disable-symlinks"
	teamsDollarsPerType           = float64(0.50)
	proDollarsPerType             = float64(1)
	teamsDollarsPerTypeDiscounted = float64(0.40)
	proDollarsPerTypeDiscounted   = float64(0.80)
	tmplCopy                      = `Current BSR pricing:

  - Teams: $0.50 per type
  - Pro: $1.00 per type

If you sign up before October 15, 2023, we will give you a 20% discount for the first year:

  - Teams: $0.40 per type for the first year
  - Pro: $0.80 per type for the first year

Pricing data last updated on April 4, 2023.

Make sure you are on the latest version of the Buf CLI to get the most updated pricing
information, and see buf.build/pricing if in doubt - this command runs completely locally
and does not interact with our servers.

{{if .IsOrganization}}Your organization has {{.NumPrivateRepositories}} private repositories that you have
permission to access. Within these {{.NumPrivateRepositories}} private repositories, you have:
{{else}}Your sources have:
{{end}}
  - {{.NumMessages}} messages
  - {{.NumEnums}} enums
  - {{.NumMethods}} methods

This adds up to {{.NumTypes}} types.

Based on this, these sources will cost:

- ${{.TeamsDollarsPerMonth}}/month for Teams
- ${{.ProDollarsPerMonth}}/month for Pro

If you sign up before October 15, 2023, for the first year, these sources will cost:

- ${{.TeamsDollarsPerMonthDiscounted}}/month for Teams
- ${{.ProDollarsPerMonthDiscounted}}/month for Pro

These values should be treated as an estimate - we price based on the average number
of private types you have on the BSR during your billing period.
`
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <source>",
		Short: "Get the price for BSR paid plans for a given source or module",
		Long:  bufcli.GetSourceOrModuleLong(`the source or module to get a price for`),
		Args:  cobra.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
			bufcli.NewErrorInterceptor(),
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	DisableSymlinks bool

	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	sourceOrModuleRefs, isOrganization, err := getSourceOrModuleRefsAndIsOrganization(ctx, container, clientConfig, input)
	if err != nil {
		return err
	}
	storageosProvider := bufcli.NewStorageosProvider(flags.DisableSymlinks)
	runner := command.NewRunner()
	moduleReader, err := bufcli.NewModuleReaderAndCreateCacheDirs(container, clientConfig)
	if err != nil {
		return err
	}
	moduleConfigReader, err := bufcli.NewWireModuleConfigReaderForModuleReader(
		container,
		storageosProvider,
		runner,
		clientConfig,
		moduleReader,
	)
	if err != nil {
		return err
	}
	var statsSlice []*protostat.Stats
	for _, sourceOrModuleRef := range sourceOrModuleRefs {
		moduleConfigs, err := moduleConfigReader.GetModuleConfigs(
			ctx,
			container,
			sourceOrModuleRef,
			"",
			nil,
			nil,
			false,
		)
		if err != nil {
			return err
		}
		for _, moduleConfig := range moduleConfigs {
			stats, err := protostat.GetStats(ctx, bufmodulestat.NewFileWalker(moduleConfig.Module()))
			if err != nil {
				return err
			}
			statsSlice = append(statsSlice, stats)
		}
	}
	tmpl, err := template.New("tmpl").Parse(tmplCopy)
	if err != nil {
		return err
	}
	return tmpl.Execute(
		container.Stdout(),
		newTmplData(protostat.MergeStats(statsSlice...), isOrganization, len(sourceOrModuleRefs)),
	)
}

// We need a way to parse the input to see if it is a source, or an organization.
// We don't have a clean way to do this right now, so we just try both.
// This could could be signficantly refactored if this presents an issue down the line.
//
// The return values:
//
//   - The SourceOrModuleRefs. If the input was a source, the length will be 1. If the
//     input was an organization, the length will be the number of private repositories
//     that the user had access to in the org.
//   - A bool that will be true if the input was an organization.
func getSourceOrModuleRefsAndIsOrganization(
	ctx context.Context,
	container applog.Container,
	clientConfig *connectclient.Config,
	input string,
) ([]buffetch.SourceOrModuleRef, bool, error) {
	// TODO: this doesn't fail on an organization
	//
	// GetSourceOrModuleRef assumes you have a source or module, so when it gets to
	// buf.build/acme, given that this isn't a module ref, it assumes this is the
	// directory buf.build/acme.
	//
	// We will likely need to refactor the command to be either source-specific or
	// organization-specific (potentially two commands?) as differentiating between
	// sources, modules, and organizations is at best error-prone, at worst impossible
	// without sending pings (which we don't want to do in the local case).
	sourceOrModuleRef, err := buffetch.NewRefParser(container.Logger()).GetSourceOrModuleRef(ctx, input)
	if err != nil {
		moduleOwner, err := bufmoduleref.ModuleOwnerForString(input)
		if err != nil {
			return nil, false, appcmd.NewInvalidArgumentErrorf("could not parse %q as either a source or an organization", input)
		}
		sourceOrModuleRefs, err := getSourceOrModuleRefsForOrganization(ctx, container, clientConfig, moduleOwner)
		if err != nil {
			return nil, false, err
		}
		return sourceOrModuleRefs, true, nil
	}
	return []buffetch.SourceOrModuleRef{
		sourceOrModuleRef,
	}, false, nil
}

func getSourceOrModuleRefsForOrganization(
	ctx context.Context,
	container applog.Container,
	clientConfig *connectclient.Config,
	moduleOwner bufmoduleref.ModuleOwner,
) ([]buffetch.SourceOrModuleRef, error) {
	organizationID, err := getOrganizationID(ctx, clientConfig, moduleOwner)
	if err != nil {
		return nil, err
	}
	privateRepositoryFullNames, err := getOrganizationPrivateRepositoryFullNames(ctx, clientConfig, moduleOwner, organizationID)
	if err != nil {
		return nil, err
	}
	sourceOrModuleRefs := make([]buffetch.SourceOrModuleRef, len(privateRepositoryFullNames))
	for i, privateRepositoryFullName := range privateRepositoryFullNames {
		sourceOrModuleRef, err := buffetch.NewRefParser(container.Logger()).GetSourceOrModuleRef(ctx, privateRepositoryFullName)
		if err != nil {
			return nil, err
		}
		sourceOrModuleRefs[i] = sourceOrModuleRef
	}
	return sourceOrModuleRefs, nil
}

func getOrganizationID(
	ctx context.Context,
	clientConfig *connectclient.Config,
	moduleOwner bufmoduleref.ModuleOwner,
) (string, error) {
	organizationService := connectclient.Make(
		clientConfig,
		moduleOwner.Remote(),
		registryv1alpha1connect.NewOrganizationServiceClient,
	)
	response, err := organizationService.GetOrganizationByName(
		ctx,
		connect.NewRequest(
			&registryv1alpha1.GetOrganizationByNameRequest{
				Name: moduleOwner.Owner(),
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return "", bufcli.NewOrganizationNotFoundError(moduleOwner.OwnerString())
		}
		return "", err
	}
	return response.Msg.Organization.Id, nil
}

func getOrganizationPrivateRepositoryFullNames(
	ctx context.Context,
	clientConfig *connectclient.Config,
	moduleOwner bufmoduleref.ModuleOwner,
	organizationId string,
) ([]string, error) {
	repositoryService := connectclient.Make(
		clientConfig,
		moduleOwner.Remote(),
		registryv1alpha1connect.NewRepositoryServiceClient,
	)
	var privateRepositoryFullNames []string
	var pageToken string
	for {
		response, err := repositoryService.ListOrganizationRepositories(
			ctx,
			connect.NewRequest(
				&registryv1alpha1.ListOrganizationRepositoriesRequest{
					OrganizationId: organizationId,
					PageToken:      pageToken,
				},
			),
		)
		if err != nil {
			return nil, err
		}
		for _, repository := range response.Msg.Repositories {
			if repository.Visibility == registryv1alpha1.Visibility_VISIBILITY_PRIVATE {
				privateRepositoryFullNames = append(
					privateRepositoryFullNames,
					moduleOwner.OwnerString()+"/"+repository.Name,
				)
			}
		}
		if response.Msg.NextPageToken == "" {
			break
		}
		pageToken = response.Msg.NextPageToken
	}
	return privateRepositoryFullNames, nil
}

type tmplData struct {
	*protostat.Stats

	IsOrganization                 bool
	NumPrivateRepositories         int
	NumTypes                       int
	TeamsDollarsPerMonth           string
	ProDollarsPerMonth             string
	TeamsDollarsPerMonthDiscounted string
	ProDollarsPerMonthDiscounted   string
}

func newTmplData(stats *protostat.Stats, isOrganization bool, numPrivateRepositories int) *tmplData {
	tmplData := &tmplData{
		Stats:                  stats,
		IsOrganization:         isOrganization,
		NumPrivateRepositories: numPrivateRepositories,
		NumTypes:               stats.NumMessages + stats.NumEnums + stats.NumMethods,
	}
	tmplData.TeamsDollarsPerMonth = fmt.Sprintf("%.2f", float64(tmplData.NumTypes)*teamsDollarsPerType)
	tmplData.ProDollarsPerMonth = fmt.Sprintf("%.2f", float64(tmplData.NumTypes)*proDollarsPerType)
	tmplData.TeamsDollarsPerMonthDiscounted = fmt.Sprintf("%.2f", float64(tmplData.NumTypes)*teamsDollarsPerTypeDiscounted)
	tmplData.ProDollarsPerMonthDiscounted = fmt.Sprintf("%.2f", float64(tmplData.NumTypes)*proDollarsPerTypeDiscounted)
	return tmplData
}
