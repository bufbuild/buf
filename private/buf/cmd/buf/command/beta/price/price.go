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
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulestat"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/protostat"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	disableSymlinksFlagName    = "disable-symlinks"
	teamPerTypeDollarsPerMonth = float64(0.50)
	proPerTypeDollarsPerMonth  = float64(1)
	tmplCopy                   = `Current BSR pricing:

  - Team: $50/month per 100 types
  - Pro: $100/month per 100 types

Pricing data last updated on March 13, 2023.
See buf.build/pricing for the latest information.

Your sources have:

  - {{.NumMessages}} messages
  - {{.NumEnums}} enums
  - {{.NumMethods}} methods

This adds up to {{.NumTypes}} types.{{if .ChargeableIsRounded}}

We bill in increments of 100 types. Types are totaled across
your whole organization (Team) or instance (Pro). If these
sources were all that were uploaded to your organization or instance,
this would be rounded up to {{.ChargeableTypes}} types.{{end}}

Based on this, these sources will cost:

- ${{.TeamDollarsPerMonth}}/month for Team
- ${{.ProDollarsPerMonth}}/month for Pro
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
	sourceOrModuleRef, err := buffetch.NewRefParser(container.Logger(), buffetch.RefParserWithProtoFileRefAllowed()).GetSourceOrModuleRef(ctx, input)
	if err != nil {
		return err
	}
	storageosProvider := bufcli.NewStorageosProvider(flags.DisableSymlinks)
	runner := command.NewRunner()
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
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
	statsSlice := make([]*protostat.Stats, len(moduleConfigs))
	for i, moduleConfig := range moduleConfigs {
		stats, err := protostat.GetStats(ctx, bufmodulestat.NewFileWalker(moduleConfig.Module()))
		if err != nil {
			return err
		}
		statsSlice[i] = stats
	}
	tmpl, err := template.New("tmpl").Parse(tmplCopy)
	if err != nil {
		return err
	}
	return tmpl.Execute(
		container.Stdout(),
		newTmplData(protostat.MergeStats(statsSlice...)),
	)
}

type tmplData struct {
	*protostat.Stats

	NumTypes            int
	ChargeableTypes     int
	ChargeableIsRounded bool
	TeamDollarsPerMonth string
	ProDollarsPerMonth  string
}

func newTmplData(stats *protostat.Stats) *tmplData {
	tmplData := &tmplData{
		Stats:    stats,
		NumTypes: stats.NumMessages + stats.NumEnums + stats.NumMethods,
	}
	buckets := tmplData.NumTypes / 100
	if tmplData.NumTypes%100 != 0 {
		tmplData.ChargeableIsRounded = true
		buckets++
	}
	tmplData.ChargeableTypes = buckets * 100
	tmplData.TeamDollarsPerMonth = fmt.Sprintf("%.2f", float64(tmplData.ChargeableTypes)*teamPerTypeDollarsPerMonth)
	tmplData.ProDollarsPerMonth = fmt.Sprintf("%.2f", float64(tmplData.ChargeableTypes)*proPerTypeDollarsPerMonth)
	return tmplData
}
