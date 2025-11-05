// Copyright 2020-2025 Buf Technologies, Inc.
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

package pluginpush

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"buf.build/go/standard/xslices"
	"buf.build/go/standard/xstrings"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const (
	labelFlagName            = "label"
	binaryFlagName           = "binary"
	createFlagName           = "create"
	createVisibilityFlagName = "create-visibility"
	createTypeFlagName       = "create-type"
	sourceControlURLFlagName = "source-control-url"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <remote/owner/plugin>",
		Short: "Push a check plugin to a registry",
		Long:  `The first argument is the plugin full name in the format <remote/owner/plugin>.`,
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
	Labels           []string
	Binary           string
	Create           bool
	CreateVisibility string
	CreateType       string
	SourceControlURL string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindCreateVisibility(flagSet, &f.CreateVisibility, createVisibilityFlagName, createFlagName)
	flagSet.StringSliceVar(
		&f.Labels,
		labelFlagName,
		nil,
		"Associate the label with the plugins pushed. Can be used multiple times.",
	)
	flagSet.StringVar(
		&f.Binary,
		binaryFlagName,
		"",
		"The path to the Wasm binary file to push.",
	)
	flagSet.BoolVar(
		&f.Create,
		createFlagName,
		false,
		fmt.Sprintf(
			"Create the plugin if it does not exist. Defaults to creating a private plugin on the BSR if --%s is not set. Must be used with --%s.",
			createVisibilityFlagName,
			createTypeFlagName,
		),
	)
	flagSet.StringVar(
		&f.CreateType,
		createTypeFlagName,
		"",
		fmt.Sprintf(
			"The plugin's type setting, if created. Can only be set with --%s. Must be one of %s",
			createFlagName,
			xstrings.SliceToString(bufplugin.AllPluginTypeStrings),
		),
	)
	flagSet.StringVar(
		&f.SourceControlURL,
		sourceControlURLFlagName,
		"",
		"The URL for viewing the source code of the pushed plugins (e.g. the specific commit in source control).",
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) (retErr error) {
	if err := validateFlags(flags); err != nil {
		return err
	}
	// We parse the plugin full name from the user-provided argument.
	pluginFullName, err := bufparse.ParseFullName(container.Arg(0))
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	commit, err := upload(ctx, container, flags, pluginFullName)
	if err != nil {
		return err
	}
	// Only one commit is returned.
	if _, err := fmt.Fprintf(container.Stdout(), "%s\n", commit.PluginKey().String()); err != nil {
		return syserror.Wrap(err)
	}
	return nil
}

func upload(
	ctx context.Context,
	container appext.Container,
	flags *flags,
	pluginFullName bufparse.FullName,
) (_ bufplugin.Commit, retErr error) {
	var plugin bufplugin.Plugin
	switch {
	case flags.Binary != "":
		// We create a local plugin reference to the Wasm binary.
		var err error
		plugin, err = bufplugin.NewLocalWasmPlugin(
			pluginFullName,
			flags.Binary,
			nil, // args
			func() ([]byte, error) {
				wasmBinary, err := os.ReadFile(flags.Binary)
				if err != nil {
					return nil, fmt.Errorf("could not read Wasm binary %q: %w", flags.Binary, err)
				}
				return wasmBinary, nil
			},
		)
		if err != nil {
			return nil, err
		}
	default:
		// This should never happen because the flags are validated.
		return nil, syserror.Newf("--%s must be set", binaryFlagName)
	}
	uploader, err := bufcli.NewPluginUploader(container)
	if err != nil {
		return nil, err
	}
	var options []bufplugin.UploadOption
	if flags.Create {
		createPluginVisibility, err := bufplugin.ParsePluginVisibility(flags.CreateVisibility)
		if err != nil {
			return nil, err
		}
		createPluginType, err := bufplugin.ParsePluginType(flags.CreateType)
		if err != nil {
			return nil, err
		}
		options = append(options, bufplugin.UploadWithCreateIfNotExist(
			createPluginVisibility,
			createPluginType,
		))
	}
	if len(flags.Labels) > 0 {
		options = append(options, bufplugin.UploadWithLabels(flags.Labels...))
	}
	if flags.SourceControlURL != "" {
		options = append(options, bufplugin.UploadWithSourceControlURL(flags.SourceControlURL))
	}
	commits, err := uploader.Upload(ctx, []bufplugin.Plugin{plugin}, options...)
	if err != nil {
		return nil, err
	}
	if len(commits) != 1 {
		return nil, syserror.Newf("unexpected number of commits returned from server: %d", len(commits))
	}
	return commits[0], nil
}

func validateFlags(flags *flags) error {
	if err := validateLabelFlags(flags); err != nil {
		return err
	}
	if err := validateTypeFlags(flags); err != nil {
		return err
	}
	if err := validateCreateFlags(flags); err != nil {
		return err
	}
	return nil
}

func validateLabelFlags(flags *flags) error {
	return validateLabelFlagValues(flags)
}

func validateTypeFlags(flags *flags) error {
	var typeFlags []string
	if flags.Binary != "" {
		typeFlags = append(typeFlags, binaryFlagName)
	}
	if len(typeFlags) > 1 {
		usedFlagsErrStr := strings.Join(
			xslices.Map(
				typeFlags,
				func(flag string) string { return fmt.Sprintf("--%s", flag) },
			),
			", ",
		)
		return appcmd.NewInvalidArgumentErrorf("These flags cannot be used in combination with one another: %s", usedFlagsErrStr)
	}
	if len(typeFlags) == 0 {
		return appcmd.NewInvalidArgumentErrorf("--%s must be set", binaryFlagName)
	}
	return nil
}

func validateLabelFlagValues(flags *flags) error {
	if slices.Contains(flags.Labels, "") {
		return appcmd.NewInvalidArgumentErrorf("--%s requires a non-empty string", labelFlagName)
	}
	return nil
}

func validateCreateFlags(flags *flags) error {
	if flags.Create {
		if flags.CreateVisibility == "" {
			return appcmd.NewInvalidArgumentErrorf("--%s must be set if --%s is set", createVisibilityFlagName, createFlagName)
		}
		if _, err := bufplugin.ParsePluginVisibility(flags.CreateVisibility); err != nil {
			return appcmd.WrapInvalidArgumentError(err)
		}
	}
	if flags.Create {
		if flags.CreateType == "" {
			return appcmd.NewInvalidArgumentErrorf("--%s must be set if --%s is set", createTypeFlagName, createFlagName)
		}
		if _, err := bufplugin.ParsePluginType(flags.CreateType); err != nil {
			return appcmd.WrapInvalidArgumentError(err)
		}
	}
	return nil
}
