// Copyright 2020 Buf Technologies, Inc.
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

package protoc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufcoreutil"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/bufmod"
	"github.com/bufbuild/buf/internal/buf/cmd/internal"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

const (
	includeDirPathsFlagName       = "proto_path"
	includeImportsFlagName        = "include_imports"
	includeSourceInfoFlagName     = "include_source_info"
	printFreeFieldNumbersFlagName = "print_free_field_numbers"
	outputFlagName                = "descriptor_set_out"

	pluginFakeFlagName = "protoc_plugin_fake"
)

// NewCommand returns a new Command
func NewCommand(use string, builder appflag.Builder) *appcmd.Command {
	controller := newController()
	return &appcmd.Command{
		Use:   use,
		Short: "Run protoc logic.",
		Run: builder.NewRunFunc(
			func(ctx context.Context, container applog.Container) error {
				return controller.Run(ctx, container)
			},
		),
		BindFlags:     controller.Bind,
		NormalizeFlag: controller.NormalizeFlag,
	}
}

func newController() *controller {
	return &controller{
		pluginNameToValue: make(map[string]*pluginValue),
	}
}

type controller struct {
	includeDirPaths       []string
	includeImports        bool
	includeSourceInfo     bool
	printFreeFieldNumbers bool
	output                string

	pluginFake        []string
	pluginNameToValue map[string]*pluginValue
}

func (c *controller) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVarP(
		&c.includeDirPaths,
		includeDirPathsFlagName,
		"I",
		[]string{"."},
		`The include directory paths.

This is equivalent to roots in Buf.`,
	)
	flagSet.BoolVar(
		&c.includeImports,
		includeImportsFlagName,
		false,
		`Include imports in the resulting FileDescriptorSet.`,
	)
	flagSet.BoolVar(
		&c.includeSourceInfo,
		includeSourceInfoFlagName,
		false,
		`Include source info in the resulting FileDescriptorSet.`,
	)
	flagSet.BoolVar(
		&c.printFreeFieldNumbers,
		printFreeFieldNumbersFlagName,
		false,
		`Print the free field numbers of all messages.`,
	)
	flagSet.StringVarP(
		&c.output,
		outputFlagName,
		"o",
		"",
		fmt.Sprintf(
			`The location to write the FileDescriptorSet. Must be one of format %s.`,
			buffetch.ImageFormatsString,
		),
	)
	flagSet.StringSliceVar(
		&c.pluginFake,
		pluginFakeFlagName,
		nil,
		`If you are calling this, you should not be.`,
	)
	_ = flagSet.MarkHidden(pluginFakeFlagName)
}

func (c *controller) Run(ctx context.Context, container applog.Container) (retErr error) {
	internal.WarnExperimental(container)
	pluginNameToOut, pluginNameToOpt, err := c.getOutAndOpt()
	if err != nil {
		return err
	}
	container.Logger().Debug("generate_flags", zap.Any("out", pluginNameToOut), zap.Any("opt", pluginNameToOpt))

	filePaths := app.Args(container)
	if len(filePaths) == 0 {
		return errors.New("no input files specified")
	}

	module, err := bufmod.NewIncludeBuilder(container.Logger()).BuildForIncludes(
		ctx,
		c.includeDirPaths,
		bufmod.WithPaths(filePaths...),
	)
	if err != nil {
		return err
	}
	var buildOptions []bufbuild.BuildOption
	if !c.includeSourceInfo {
		buildOptions = append(buildOptions, bufbuild.WithExcludeSourceCodeInfo())
	}
	image, fileAnnotations, err := bufbuild.NewBuilder(container.Logger()).Build(
		ctx,
		module,
		buildOptions...,
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := bufanalysis.PrintFileAnnotations(container.Stderr(), fileAnnotations, false); err != nil {
			return err
		}
		return errors.New("")
	}

	if c.printFreeFieldNumbers {
		s, err := bufcoreutil.FreeMessageRangeStrings(ctx, module, image)
		if err != nil {
			return err
		}
		if _, err := container.Stdout().Write([]byte(strings.Join(s, "\n") + "\n")); err != nil {
			return err
		}
	} else {
		if c.output == "" {
			return fmt.Errorf("--%s is required", outputFlagName)
		}
	}
	if c.output != "" {
		return internal.NewBufwireImageWriter(container.Logger()).PutImage(ctx,
			container,
			c.output,
			image,
			true,
			!c.includeImports,
		)
	}
	return nil
}

func (c *controller) NormalizeFlag(flagSet *pflag.FlagSet, name string) string {
	if name != "descriptor_set_out" && strings.HasSuffix(name, "_out") {
		c.pluginFakeParse(name, "_out", true)
		return pluginFakeFlagName
	}
	if strings.HasSuffix(name, "_opt") {
		c.pluginFakeParse(name, "_opt", false)
		return pluginFakeFlagName
	}
	return name
}

func (c *controller) pluginFakeParse(name string, suffix string, isOut bool) {
	pluginName := strings.TrimSuffix(name, suffix)
	pluginValue, ok := c.pluginNameToValue[pluginName]
	if !ok {
		pluginValue = newPluginValue()
		c.pluginNameToValue[pluginName] = pluginValue
	}
	index := len(c.pluginFake)
	if isOut {
		pluginValue.OutIndex = index
		pluginValue.OutCount++
	} else {
		pluginValue.OptIndex = index
		pluginValue.OptCount++
	}
}

func (c *controller) getOutAndOpt() (map[string]string, map[string]string, error) {
	pluginNameToOut := make(map[string]string)
	pluginNameToOpt := make(map[string]string)
	for pluginName, pluginValue := range c.pluginNameToValue {
		if pluginValue.OptIndex >= 0 && pluginValue.OutIndex < 0 {
			return nil, nil, fmt.Errorf("cannot specify --%s_opt without --%s_out", pluginName, pluginName)
		}
		if pluginValue.OutIndex >= 0 {
			if pluginValue.OutCount > 1 {
				return nil, nil, fmt.Errorf("duplicate --%s_out", pluginName)
			}
			pluginNameToOut[pluginName] = c.pluginFake[pluginValue.OutIndex]
		}
		if pluginValue.OptIndex >= 0 {
			if pluginValue.OptCount > 1 {
				return nil, nil, fmt.Errorf("duplicate --%s_opt", pluginName)
			}
			pluginNameToOpt[pluginName] = c.pluginFake[pluginValue.OptIndex]
		}
	}
	return pluginNameToOut, pluginNameToOpt, nil
}

type pluginValue struct {
	OutIndex int
	OutCount int
	OptIndex int
	OptCount int
}

func newPluginValue() *pluginValue {
	return &pluginValue{
		OutIndex: -1,
		OptIndex: -1,
	}
}
