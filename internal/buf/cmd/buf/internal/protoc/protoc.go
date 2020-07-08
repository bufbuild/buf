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
	"path/filepath"
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
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

const (
	includeDirPathsFlagName       = "proto_path"
	includeImportsFlagName        = "include_imports"
	includeSourceInfoFlagName     = "include_source_info"
	printFreeFieldNumbersFlagName = "print_free_field_numbers"
	outputFlagName                = "descriptor_set_out"
	pluginPathValuesFlagName      = "plugin"
	errorFormatFlagName           = "error_format"

	pluginFakeFlagName = "protoc_plugin_fake"

	encodeFlagName          = "encode"
	decodeFlagName          = "decode"
	decodeRawFlagName       = "decode_raw"
	descriptorSetInFlagName = "descriptor_set_in"
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
		Version:       "3.12.3-buf",
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
	errorFormat           string
	pluginPathValues      []string

	pluginFake        []string
	pluginNameToValue map[string]*pluginValue

	encode          string
	decode          string
	decodeRaw       bool
	descriptorSetIn []string
}

func (c *controller) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVarP(
		&c.includeDirPaths,
		includeDirPathsFlagName,
		"I",
		[]string{"."},
		`The include directory paths. This is equivalent to roots in Buf.`,
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
	flagSet.StringVar(
		&c.errorFormat,
		errorFormatFlagName,
		"gcc",
		fmt.Sprintf(
			`The error format to use. Must be one of format %s.`,
			stringutil.SliceToString(bufanalysis.AllFormatStringsWithAliases),
		),
	)
	flagSet.StringSliceVar(
		&c.pluginPathValues,
		pluginPathValuesFlagName,
		nil,
		`The paths to the plugin executables to use, either in the form "path/to/protoc-gen-foo" or "protoc-gen-foo=path/to/binary".`,
	)

	flagSet.StringSliceVar(
		&c.pluginFake,
		pluginFakeFlagName,
		nil,
		`If you are calling this, you should not be.`,
	)
	_ = flagSet.MarkHidden(pluginFakeFlagName)

	flagSet.StringVar(
		&c.encode,
		encodeFlagName,
		"",
		`Not supported by buf.`,
	)
	_ = flagSet.MarkHidden(encodeFlagName)
	flagSet.StringVar(
		&c.decode,
		decodeFlagName,
		"",
		`Not supported by buf.`,
	)
	_ = flagSet.MarkHidden(decodeFlagName)
	flagSet.BoolVar(
		&c.decodeRaw,
		decodeRawFlagName,
		false,
		`Not supported by buf.`,
	)
	_ = flagSet.MarkHidden(decodeRawFlagName)
	flagSet.StringSliceVar(
		&c.descriptorSetIn,
		descriptorSetInFlagName,
		nil,
		`Not supported by buf.`,
	)
	_ = flagSet.MarkHidden(descriptorSetInFlagName)
}

func (c *controller) Run(ctx context.Context, container applog.Container) (retErr error) {
	internal.WarnExperimental(container)
	if err := c.checkUnsupportedFlags(); err != nil {
		return err
	}

	pluginNameToPluginInfo, err := c.getPluginNameToPluginInfo()
	if err != nil {
		return err
	}
	container.Logger().Debug("generate_flags", zap.Any("info", pluginNameToPluginInfo))
	if len(pluginNameToPluginInfo) > 0 {
		return fmt.Errorf("generation is not yet supported")
	}

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
		if err := bufanalysis.PrintFileAnnotations(
			container.Stderr(),
			fileAnnotations,
			c.errorFormat,
		); err != nil {
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
	return strings.Replace(name, "-", "_", -1)
}

func (c *controller) checkUnsupportedFlags() error {
	if c.encode != "" {
		//lint:ignore ST1005 CLI error message
		return fmt.Errorf(
			`--%s is not supported by buf.

Buf only handles the binary and JSON formats for now, however we can support this flag if there is sufficient demand.
Please email us at support@buf.build if this is a need for your organization.`,
			encodeFlagName,
		)
	}
	if c.decode != "" {
		//lint:ignore ST1005 CLI error message
		return fmt.Errorf(
			`--%s is not supported by buf.

Buf only handles the binary and JSON formats for now, however we can support this flag if there is sufficient demand.
Please email us at support@buf.build if this is a need for your organization.`,
			decodeFlagName,
		)
	}
	if c.decodeRaw {
		//lint:ignore ST1005 CLI error message
		return fmt.Errorf(
			`--%s is not supported by buf.

Buf only handles the binary and JSON formats for now, however we can support this flag if there is sufficient demand.
Please email us at support@buf.build if this is a need for your organization.`,
			decodeRawFlagName,
		)
	}
	if len(c.descriptorSetIn) > 0 {
		//lint:ignore ST1005 CLI error message
		return fmt.Errorf(
			`--%s is not supported by buf.

Buf will work with cross-repository imports Buf Schema Registry, which will be based on source files, not pre-compiled images.
We think this is a much safer option that leads to less errors and more consistent results.

Please email us at support@buf.build if this is a need for your organization.`,
			descriptorSetInFlagName,
		)
	}
	return nil
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

func (c *controller) getPluginNameToPluginInfo() (map[string]*pluginInfo, error) {
	pluginNameToPluginInfo := make(map[string]*pluginInfo)
	for pluginName, pluginValue := range c.pluginNameToValue {
		if pluginValue.OptIndex >= 0 && pluginValue.OutIndex < 0 {
			return nil, fmt.Errorf("cannot specify --%s_opt without --%s_out", pluginName, pluginName)
		}
		if pluginValue.OutIndex >= 0 {
			if pluginValue.OutCount > 1 {
				return nil, fmt.Errorf("duplicate --%s_out", pluginName)
			}
			pluginInfo, ok := pluginNameToPluginInfo[pluginName]
			if !ok {
				pluginInfo = newPluginInfo(pluginName)
				pluginNameToPluginInfo[pluginName] = pluginInfo
			}
			pluginInfo.Out = c.pluginFake[pluginValue.OutIndex]
		}
		if pluginValue.OptIndex >= 0 {
			if pluginValue.OptCount > 1 {
				return nil, fmt.Errorf("duplicate --%s_opt", pluginName)
			}
			pluginInfo, ok := pluginNameToPluginInfo[pluginName]
			if !ok {
				pluginInfo = newPluginInfo(pluginName)
				pluginNameToPluginInfo[pluginName] = pluginInfo
			}
			pluginInfo.Opt = c.pluginFake[pluginValue.OptIndex]
		}
	}
	for _, pluginPathValue := range c.pluginPathValues {
		var pluginName string
		var pluginPath string
		switch split := strings.SplitN(pluginPathValue, "=", 2); len(split) {
		case 0:
			return nil, fmt.Errorf("--%s had an empty value", pluginPathValuesFlagName)
		case 1:
			pluginName = filepath.Base(split[0])
			pluginPath = split[0]
		case 2:
			pluginName = split[0]
			pluginPath = split[1]
		}
		if !strings.HasPrefix(pluginName, "protoc-gen-") {
			return nil, fmt.Errorf(`--%s had name %q which must be prefixed by "protoc-gen-"`, pluginPathValuesFlagName, pluginName)
		}
		pluginName = strings.TrimPrefix(pluginName, "protoc-gen-")
		pluginInfo, ok := pluginNameToPluginInfo[pluginName]
		if !ok {
			return nil, fmt.Errorf("cannot specify --%s=protoc-gen-%s without --%s_out", pluginPathValuesFlagName, pluginName, pluginName)
		}
		if pluginInfo.Path != "" {
			return nil, fmt.Errorf("duplicate --%s for protoc-gen-%s", pluginPathValuesFlagName, pluginName)
		}
		pluginInfo.Path = pluginPath
	}
	return pluginNameToPluginInfo, nil
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

type pluginInfo struct {
	// required
	Name string
	// Required
	Out string
	// optional
	Opt string
	// optional
	Path string
}

func newPluginInfo(name string) *pluginInfo {
	return &pluginInfo{
		Name: name,
	}
}
