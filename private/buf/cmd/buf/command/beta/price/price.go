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
	"math"
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
	disableSymlinksFlagName = "disable-symlinks"
	teamsDollarsPerType     = float64(0.50)
	proDollarsPerType       = float64(5)
	proDollarsMinimumSpend  = float64(1000)
	tmplCopy                = `Current BSR pricing:

  - Teams: $0.50 per type
  - Pro: $5.00 per type, with a minimum spend of $1,000 per month

Pricing data last updated on November 1, 2023.

Make sure you are on the latest version of the Buf CLI to get the most updated pricing
information, and see buf.build/pricing if in doubt - this command runs completely locally
and does not interact with our servers.

Your sources have:

  - {{.NumMessages}} messages
  - {{.NumEnums}} enums
  - {{.NumMethods}} methods

This adds up to {{.NumTypes}} types.

Based on this, these sources will cost:

- ${{.TeamsDollarsPerMonth}}/month for Teams
- ${{.ProDollarsPerMonth}}/month for Pro

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
	sourceOrModuleRef, err := buffetch.NewRefParser(container.Logger()).GetSourceOrModuleRef(ctx, input)
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
	moduleConfigSet, err := moduleConfigReader.GetModuleConfigSet(
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
	moduleConfigs := moduleConfigSet.ModuleConfigs()
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

	NumTypes             int
	TeamsDollarsPerMonth string
	ProDollarsPerMonth   string
}

func newTmplData(stats *protostat.Stats) *tmplData {
	tmplData := &tmplData{
		Stats:    stats,
		NumTypes: stats.NumMessages + stats.NumEnums + stats.NumMethods,
	}
	tmplData.TeamsDollarsPerMonth = fmt.Sprintf("%.2f", float64(tmplData.NumTypes)*teamsDollarsPerType)
	tmplData.ProDollarsPerMonth = fmt.Sprintf("%.2f", math.Max(float64(tmplData.NumTypes)*proDollarsPerType, proDollarsMinimumSpend))
	return tmplData
}
