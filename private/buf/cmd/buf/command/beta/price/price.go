// Copyright 2020-2024 Buf Technologies, Inc.
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
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/protostat"
	"github.com/bufbuild/buf/private/pkg/protostat/protostatstorage"
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
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <source>",
		Short: "Get the price for BSR paid plans for a given source or module",
		Long:  bufcli.GetSourceOrModuleLong(`the source or module to get a price for`),
		Args:  appcmd.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
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
	container appext.Container,
	flags *flags,
) error {
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	controller, err := bufcli.NewController(
		container,
		bufctl.WithDisableSymlinks(flags.DisableSymlinks),
	)
	if err != nil {
		return err
	}
	workspace, err := controller.GetWorkspace(
		ctx,
		input,
	)
	if err != nil {
		return err
	}
	stats, err := protostat.GetStats(
		ctx,
		protostatstorage.NewFileWalker(
			bufmodule.ModuleReadBucketToStorageReadBucket(
				bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFilesForTargetModules(
					workspace,
				),
			),
		),
	)
	if err != nil {
		return err
	}
	tmpl, err := template.New("tmpl").Parse(tmplCopy)
	if err != nil {
		return err
	}
	return tmpl.Execute(
		container.Stdout(),
		newTmplData(stats),
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
