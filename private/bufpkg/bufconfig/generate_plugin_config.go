// Copyright 2020-2026 Buf Technologies, Inc.
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

package bufconfig

import (
	"errors"
	"fmt"
	"math"
	"os/exec"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginref"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// GenerateStrategy is the generation strategy for a protoc plugin.
type GenerateStrategy int

const (
	// GenerateStrategyDirectory is the strategy to generate per directory.
	//
	// This is the default value for local plugins.
	GenerateStrategyDirectory GenerateStrategy = 1
	// GenerateStrategyAll is the strategy to generate with all files at once.
	//
	// This is the only strategy for remote plugins.
	GenerateStrategyAll GenerateStrategy = 2
)

// GeneratePluginConfigType is a generate plugin configuration type.
type GeneratePluginConfigType int

const (
	// GeneratePluginConfigTypeRemote is the remote plugin config type.
	GeneratePluginConfigTypeRemote GeneratePluginConfigType = iota + 1
	// GeneratePluginConfigTypeLocal is the local plugin config type.
	GeneratePluginConfigTypeLocal
	// GeneratePluginConfigTypeProtocBuiltin is the protoc built-in plugin config type.
	GeneratePluginConfigTypeProtocBuiltin
	// GeneratePluginConfigTypeLocalOrProtocBuiltin is a special plugin config type. This type indicates
	// it is to be determined whether the plugin is local or protoc built-in.
	// We defer further classification to the plugin executor. In v2 the exact
	// plugin config type is always specified and it will never be just local.
	GeneratePluginConfigTypeLocalOrProtocBuiltin
	// GeneratePluginConfigTypeImage is the image plugin config type. Nothing is
	// executed for it: the image that would be given to plugins is written to
	// the file Image() under Out().
	GeneratePluginConfigTypeImage
)

var (
	// ProtocProxyPluginNames are the names of the plugins that should be proxied through protoc
	// in the absence of a binary.
	ProtocProxyPluginNames = map[string]struct{}{
		"cpp":    {},
		"csharp": {},
		"java":   {},
		"js":     {},
		"objc":   {},
		"php":    {},
		"python": {},
		"pyi":    {},
		"ruby":   {},
		"rbs":    {},
		"kotlin": {},
		"rust":   {},
	}
)

// GeneratePluginConfig is a configuration for a plugin.
type GeneratePluginConfig interface {
	// Type returns the plugin type. This is never the zero value.
	Type() GeneratePluginConfigType
	// Name returns the plugin name. This is never empty.
	Name() string
	// Out returns the output directory for generation. This is never empty.
	Out() string
	// Opt returns the plugin options as a comma separated string.
	Opt() string
	// IncludeImports returns whether to generate code for imported files. This
	// is always false in v1.
	IncludeImports() bool
	// IncludeWKT returns whether to generate code for the well-known types.
	// This returns true only if IncludeImports returns true. This is always
	// false in v1.
	IncludeWKT() bool
	// Strategy returns the generation strategy.
	//
	// This is not empty only when the plugin is local, binary or protoc builtin.
	Strategy() GenerateStrategy
	// Path returns the path, including arguments, to invoke the binary plugin.
	//
	// This is not empty only when the plugin is local.
	Path() []string
	// ProtocPath returns a path to protoc, including any extra arguments.
	//
	// This is not empty only when the plugin is protoc-builtin.
	ProtocPath() []string
	// RemoteHost returns the remote host of the remote plugin.
	//
	// This is not empty only when the plugin is remote.
	RemoteHost() string
	// Revision returns the revision of the remote plugin.
	//
	// This is not empty only when the plugin is remote.
	Revision() int
	// IncludeTypes returns the types to include.
	IncludeTypes() []string
	// ExcludeTypes returns the types to exclude.
	ExcludeTypes() []string
	// Image returns the file name, relative to Out, to write the image to.
	//
	// This is only set when Type() is GeneratePluginConfigTypeImage.
	Image() string
	// ExcludeImports returns whether to exclude imports from the image.
	//
	// This is only set when Type() is GeneratePluginConfigTypeImage.
	ExcludeImports() bool
	// ExcludeSourceInfo returns whether to exclude source info from the image.
	//
	// This is only set when Type() is GeneratePluginConfigTypeImage.
	ExcludeSourceInfo() bool
	// AsFileDescriptorSet returns whether to write the image as a
	// FileDescriptorSet instead of a Buf image.
	//
	// This is only set when Type() is GeneratePluginConfigTypeImage.
	AsFileDescriptorSet() bool

	isGeneratePluginConfig()
}

// NewRemoteGeneratePluginConfig returns a new GeneratePluginConfig for a remote plugin.
func NewRemoteGeneratePluginConfig(
	name string,
	out string,
	opt []string,
	includeImports bool,
	includeWKT bool,
	includeTypes []string,
	excludeTypes []string,
	revision int,
) (GeneratePluginConfig, error) {
	return newRemoteGeneratePluginConfig(
		name,
		out,
		opt,
		includeImports,
		includeWKT,
		includeTypes,
		excludeTypes,
		revision,
	)
}

// NewLocalOrProtocBuiltinGeneratePluginConfig returns a new GeneratePluginConfig for a local or protoc builtin plugin.
func NewLocalOrProtocBuiltinGeneratePluginConfig(
	name string,
	out string,
	opt []string,
	includeImports bool,
	includeWKT bool,
	includeTypes []string,
	excludeTypes []string,
	strategy *GenerateStrategy,
) (GeneratePluginConfig, error) {
	return newLocalOrProtocBuiltinGeneratePluginConfig(
		name,
		out,
		opt,
		includeImports,
		includeWKT,
		includeTypes,
		excludeTypes,
		strategy,
	)
}

// NewLocalGeneratePluginConfig returns a new GeneratePluginConfig for a local plugin.
func NewLocalGeneratePluginConfig(
	name string,
	out string,
	opt []string,
	includeImports bool,
	includeWKT bool,
	types []string,
	excludeTypes []string,
	strategy *GenerateStrategy,
	path []string,
) (GeneratePluginConfig, error) {
	return newLocalGeneratePluginConfig(
		name,
		out,
		opt,
		includeImports,
		includeWKT,
		types,
		excludeTypes,
		strategy,
		path,
	)
}

// NewProtocBuiltinGeneratePluginConfig returns a new GeneratePluginConfig for a protoc
// builtin plugin.
func NewProtocBuiltinGeneratePluginConfig(
	name string,
	out string,
	opt []string,
	includeImports bool,
	includeWKT bool,
	includeTypes []string,
	excludeTypes []string,
	strategy *GenerateStrategy,
	protocPath []string,
) (GeneratePluginConfig, error) {
	return newProtocBuiltinGeneratePluginConfig(
		name,
		out,
		opt,
		includeImports,
		includeWKT,
		includeTypes,
		excludeTypes,
		strategy,
		protocPath,
	)
}

// NewImageGeneratePluginConfig returns a new GeneratePluginConfig for writing
// an image.
func NewImageGeneratePluginConfig(
	image string,
	out string,
	excludeImports bool,
	excludeSourceInfo bool,
	asFileDescriptorSet bool,
	includeTypes []string,
	excludeTypes []string,
) (GeneratePluginConfig, error) {
	return newImageGeneratePluginConfig(
		image,
		out,
		excludeImports,
		excludeSourceInfo,
		asFileDescriptorSet,
		includeTypes,
		excludeTypes,
	)
}

// NewGeneratePluginConfigWithIncludeImportsAndWKT returns a GeneratePluginConfig the
// same as the input, with include imports and include wkt overridden.
func NewGeneratePluginConfigWithIncludeImportsAndWKT(
	config GeneratePluginConfig,
	includeImports bool,
	includeWKT bool,
) (GeneratePluginConfig, error) {
	originalConfig, ok := config.(*generatePluginConfig)
	if !ok {
		return nil, syserror.Newf("unknown implementation of GeneratePluginConfig: %T", config)
	}
	generatePluginConfig := *originalConfig
	if includeImports {
		generatePluginConfig.includeImports = true
	}
	if includeWKT {
		generatePluginConfig.includeWKT = true
	}
	return &generatePluginConfig, nil
}

// *** PRIVATE ***

type generatePluginConfig struct {
	generatePluginConfigType GeneratePluginConfigType
	name                     string
	out                      string
	opts                     []string
	includeImports           bool
	includeWKT               bool
	includeTypes             []string
	excludeTypes             []string
	strategy                 *GenerateStrategy
	path                     []string
	protocPath               []string
	remoteHost               string
	revision                 int
	image                    string
	excludeImports           bool
	excludeSourceInfo        bool
	asFileDescriptorSet      bool
}

func newGeneratePluginConfigFromExternalV1Beta1(
	externalConfig externalGeneratePluginConfigV1Beta1,
) (GeneratePluginConfig, error) {
	if externalConfig.Name == "" {
		return nil, errors.New("plugin name is required")
	}
	if externalConfig.Out == "" {
		return nil, fmt.Errorf("out is required for plugin %s", externalConfig.Name)
	}
	strategy, err := parseStrategy(externalConfig.Strategy)
	if err != nil {
		return nil, err
	}
	opt, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Opt)
	if err != nil {
		return nil, err
	}
	if externalConfig.Path != "" {
		return newLocalGeneratePluginConfig(
			externalConfig.Name,
			externalConfig.Out,
			opt,
			false,
			false,
			nil,
			nil,
			strategy,
			[]string{externalConfig.Path},
		)
	}
	return newLocalOrProtocBuiltinGeneratePluginConfig(
		externalConfig.Name,
		externalConfig.Out,
		opt,
		false,
		false,
		nil,
		nil,
		strategy,
	)
}

func newGeneratePluginConfigFromExternalV1(
	externalConfig externalGeneratePluginConfigV1,
) (GeneratePluginConfig, error) {
	if externalConfig.Plugin == "" && externalConfig.Name == "" {
		return nil, fmt.Errorf("one of plugin or name is required")
	}
	if externalConfig.Plugin != "" && externalConfig.Name != "" {
		return nil, fmt.Errorf("only one of plugin or name can be set")
	}
	var pluginIdentifier string
	switch {
	case externalConfig.Plugin != "":
		pluginIdentifier = externalConfig.Plugin
	case externalConfig.Name != "":
		pluginIdentifier = externalConfig.Name
		if bufremotepluginref.IsPluginReferenceOrIdentity(pluginIdentifier) {
			// A plugin reference is not a valid local plugin name.
			return nil, fmt.Errorf("invalid local plugin name: %s", pluginIdentifier)
		}
	}
	if externalConfig.Out == "" {
		return nil, fmt.Errorf("out is required for plugin %s", pluginIdentifier)
	}
	strategy, err := parseStrategy(externalConfig.Strategy)
	if err != nil {
		return nil, err
	}
	opt, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Opt)
	if err != nil {
		return nil, err
	}
	path, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Path)
	if err != nil {
		return nil, err
	}
	protocPath, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.ProtocPath)
	if err != nil {
		return nil, err
	}
	if externalConfig.Plugin != "" && bufremotepluginref.IsPluginReferenceOrIdentity(pluginIdentifier) {
		if externalConfig.Path != nil {
			return nil, fmt.Errorf("cannot specify path for remote plugin %s", externalConfig.Plugin)
		}
		if externalConfig.Strategy != "" {
			return nil, fmt.Errorf("cannot specify strategy for remote plugin %s", externalConfig.Plugin)
		}
		if externalConfig.ProtocPath != nil {
			return nil, fmt.Errorf("cannot specify protoc_path for remote plugin %s", externalConfig.Plugin)
		}
		return newRemoteGeneratePluginConfig(
			externalConfig.Plugin,
			externalConfig.Out,
			opt,
			false,
			false,
			nil,
			nil,
			externalConfig.Revision,
		)
	}
	// At this point the plugin must be local, regardless whether it's specified
	// by key 'plugin' or 'name'.
	if len(path) > 0 {
		return newLocalGeneratePluginConfig(
			pluginIdentifier,
			externalConfig.Out,
			opt,
			false,
			false,
			nil,
			nil,
			strategy,
			path,
		)
	}
	if externalConfig.ProtocPath != nil {
		return newProtocBuiltinGeneratePluginConfig(
			pluginIdentifier,
			externalConfig.Out,
			opt,
			false,
			false,
			nil,
			nil,
			strategy,
			protocPath,
		)
	}
	// It could be either local or protoc built-in. We defer to the plugin executor
	// to decide whether the plugin is protoc-builtin or local.
	return newLocalOrProtocBuiltinGeneratePluginConfig(
		pluginIdentifier,
		externalConfig.Out,
		opt,
		false,
		false,
		nil,
		nil,
		strategy,
	)
}

func newGeneratePluginConfigFromExternalV2(
	externalConfig externalGeneratePluginConfigV2,
) (GeneratePluginConfig, error) {
	var pluginTypeCount int
	if externalConfig.Remote != nil {
		pluginTypeCount++
	}
	if externalConfig.Local != nil {
		pluginTypeCount++
	}
	if externalConfig.ProtocBuiltin != nil {
		pluginTypeCount++
	}
	if externalConfig.Image != nil {
		pluginTypeCount++
	}
	if pluginTypeCount == 0 {
		return nil, errors.New("must specify one of remote, local, protoc_builtin or image")
	}
	if pluginTypeCount > 1 {
		return nil, errors.New("only one of remote, local, protoc_builtin or image")
	}
	if externalConfig.Out == "" {
		return nil, errors.New("must specify out")
	}
	if externalConfig.Image == nil {
		if externalConfig.ExcludeImports {
			return nil, errors.New("cannot specify exclude_imports without image")
		}
		if externalConfig.ExcludeSourceInfo {
			return nil, errors.New("cannot specify exclude_source_info without image")
		}
		if externalConfig.AsFileDescriptorSet {
			return nil, errors.New("cannot specify as_file_descriptor_set without image")
		}
	}
	var strategy string
	if externalConfig.Strategy != nil {
		strategy = *externalConfig.Strategy
	}
	parsedStrategy, err := parseStrategy(strategy)
	if err != nil {
		return nil, err
	}
	opt, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Opt)
	if err != nil {
		return nil, err
	}
	switch {
	case externalConfig.Remote != nil:
		var revision int
		if externalConfig.Revision != nil {
			revision = *externalConfig.Revision
		}
		if externalConfig.Strategy != nil {
			return nil, fmt.Errorf("cannot specify strategy for remote plugin %s", *externalConfig.Remote)
		}
		if externalConfig.ProtocPath != nil {
			return nil, fmt.Errorf("cannot specify protoc_path for remote plugin %s", *externalConfig.Remote)
		}
		return newRemoteGeneratePluginConfig(
			*externalConfig.Remote,
			externalConfig.Out,
			opt,
			externalConfig.IncludeImports,
			externalConfig.IncludeWKT,
			externalConfig.Types,
			externalConfig.ExcludeTypes,
			revision,
		)
	case externalConfig.Local != nil:
		path, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Local)
		if err != nil {
			return nil, err
		}
		localPluginName := strings.Join(path, " ")
		if externalConfig.Revision != nil {
			return nil, fmt.Errorf("cannot specify revision for local plugin %s", localPluginName)
		}
		if externalConfig.ProtocPath != nil {
			return nil, fmt.Errorf("cannot specify protoc_path for local plugin %s", localPluginName)
		}
		return newLocalGeneratePluginConfig(
			strings.Join(path, " "),
			externalConfig.Out,
			opt,
			externalConfig.IncludeImports,
			externalConfig.IncludeWKT,
			externalConfig.Types,
			externalConfig.ExcludeTypes,
			parsedStrategy,
			path,
		)
	case externalConfig.ProtocBuiltin != nil:
		protocPath, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.ProtocPath)
		if err != nil {
			return nil, err
		}
		if externalConfig.Revision != nil {
			return nil, fmt.Errorf("cannot specify revision for protoc built-in plugin %s", *externalConfig.ProtocBuiltin)
		}
		return newProtocBuiltinGeneratePluginConfig(
			*externalConfig.ProtocBuiltin,
			externalConfig.Out,
			opt,
			externalConfig.IncludeImports,
			externalConfig.IncludeWKT,
			externalConfig.Types,
			externalConfig.ExcludeTypes,
			parsedStrategy,
			protocPath,
		)
	case externalConfig.Image != nil:
		image := *externalConfig.Image
		if externalConfig.Opt != nil {
			return nil, fmt.Errorf("cannot specify opt for image %s", image)
		}
		if externalConfig.Strategy != nil {
			return nil, fmt.Errorf("cannot specify strategy for image %s", image)
		}
		if externalConfig.Revision != nil {
			return nil, fmt.Errorf("cannot specify revision for image %s", image)
		}
		if externalConfig.ProtocPath != nil {
			return nil, fmt.Errorf("cannot specify protoc_path for image %s", image)
		}
		if externalConfig.IncludeImports {
			return nil, fmt.Errorf("cannot specify include_imports for image %s", image)
		}
		if externalConfig.IncludeWKT {
			return nil, fmt.Errorf("cannot specify include_wkt for image %s", image)
		}
		return newImageGeneratePluginConfig(
			image,
			externalConfig.Out,
			externalConfig.ExcludeImports,
			externalConfig.ExcludeSourceInfo,
			externalConfig.AsFileDescriptorSet,
			externalConfig.Types,
			externalConfig.ExcludeTypes,
		)
	default:
		return nil, syserror.Newf("must specify one of remote, local, protoc_builtin or image")
	}
}

func newRemoteGeneratePluginConfig(
	name string,
	out string,
	opt []string,
	includeImports bool,
	includeWKT bool,
	includeTypes []string,
	excludeTypes []string,
	revision int,
) (*generatePluginConfig, error) {
	if includeWKT && !includeImports {
		return nil, errors.New("cannot include well-known types without including imports")
	}
	remoteHost, err := parseRemoteHostName(name)
	if err != nil {
		return nil, err
	}
	if revision < 0 || revision > math.MaxInt32 {
		return nil, fmt.Errorf("revision %d is out of accepted range %d-%d", revision, 0, math.MaxInt32)
	}
	return &generatePluginConfig{
		generatePluginConfigType: GeneratePluginConfigTypeRemote,
		name:                     name,
		remoteHost:               remoteHost,
		revision:                 revision,
		out:                      out,
		opts:                     opt,
		includeImports:           includeImports,
		includeWKT:               includeWKT,
		includeTypes:             includeTypes,
		excludeTypes:             excludeTypes,
	}, nil
}

func newLocalOrProtocBuiltinGeneratePluginConfig(
	name string,
	out string,
	opt []string,
	includeImports bool,
	includeWKT bool,
	includeTypes []string,
	excludeTypes []string,
	strategy *GenerateStrategy,
) (*generatePluginConfig, error) {
	if includeWKT && !includeImports {
		return nil, errors.New("cannot include well-known types without including imports")
	}
	return &generatePluginConfig{
		generatePluginConfigType: GeneratePluginConfigTypeLocalOrProtocBuiltin,
		name:                     name,
		strategy:                 strategy,
		out:                      out,
		opts:                     opt,
		includeImports:           includeImports,
		includeWKT:               includeWKT,
		includeTypes:             includeTypes,
		excludeTypes:             excludeTypes,
	}, nil
}

func newLocalGeneratePluginConfig(
	name string,
	out string,
	opt []string,
	includeImports bool,
	includeWKT bool,
	includeTypes []string,
	excludeTypes []string,
	strategy *GenerateStrategy,
	path []string,
) (*generatePluginConfig, error) {
	if len(path) == 0 {
		return nil, errors.New("must specify a path to the plugin")
	}
	if includeWKT && !includeImports {
		return nil, errors.New("cannot include well-known types without including imports")
	}
	return &generatePluginConfig{
		generatePluginConfigType: GeneratePluginConfigTypeLocal,
		name:                     name,
		path:                     path,
		strategy:                 strategy,
		out:                      out,
		opts:                     opt,
		includeImports:           includeImports,
		includeWKT:               includeWKT,
		includeTypes:             includeTypes,
		excludeTypes:             excludeTypes,
	}, nil
}

func newProtocBuiltinGeneratePluginConfig(
	name string,
	out string,
	opt []string,
	includeImports bool,
	includeWKT bool,
	includeTypes []string,
	excludeTypes []string,
	strategy *GenerateStrategy,
	protocPath []string,
) (*generatePluginConfig, error) {
	if includeWKT && !includeImports {
		return nil, errors.New("cannot include well-known types without including imports")
	}
	return &generatePluginConfig{
		generatePluginConfigType: GeneratePluginConfigTypeProtocBuiltin,
		name:                     name,
		protocPath:               protocPath,
		out:                      out,
		opts:                     opt,
		strategy:                 strategy,
		includeImports:           includeImports,
		includeWKT:               includeWKT,
		includeTypes:             includeTypes,
		excludeTypes:             excludeTypes,
	}, nil
}

func newImageGeneratePluginConfig(
	image string,
	out string,
	excludeImports bool,
	excludeSourceInfo bool,
	asFileDescriptorSet bool,
	includeTypes []string,
	excludeTypes []string,
) (*generatePluginConfig, error) {
	if err := validateImageFileName(image); err != nil {
		return nil, err
	}
	return &generatePluginConfig{
		generatePluginConfigType: GeneratePluginConfigTypeImage,
		name:                     image,
		image:                    image,
		out:                      out,
		includeTypes:             includeTypes,
		excludeTypes:             excludeTypes,
		excludeImports:           excludeImports,
		excludeSourceInfo:        excludeSourceInfo,
		asFileDescriptorSet:      asFileDescriptorSet,
	}, nil
}

func (p *generatePluginConfig) Type() GeneratePluginConfigType {
	return p.generatePluginConfigType
}

func (p *generatePluginConfig) Name() string {
	return p.name
}

func (p *generatePluginConfig) Out() string {
	return p.out
}

func (p *generatePluginConfig) Opt() string {
	return strings.Join(p.opts, ",")
}

func (p *generatePluginConfig) IncludeImports() bool {
	return p.includeImports
}

func (p *generatePluginConfig) IncludeWKT() bool {
	return p.includeWKT
}

func (p *generatePluginConfig) IncludeTypes() []string {
	return p.includeTypes
}

func (p *generatePluginConfig) ExcludeTypes() []string {
	return p.excludeTypes
}

func (p *generatePluginConfig) Strategy() GenerateStrategy {
	if p.generatePluginConfigType == GeneratePluginConfigTypeImage {
		return 0
	}
	if p.strategy == nil {
		return GenerateStrategyDirectory
	}
	return *p.strategy
}

func (p *generatePluginConfig) Path() []string {
	return p.path
}

func (p *generatePluginConfig) ProtocPath() []string {
	return p.protocPath
}

func (p *generatePluginConfig) RemoteHost() string {
	return p.remoteHost
}

func (p *generatePluginConfig) Revision() int {
	return p.revision
}

func (p *generatePluginConfig) Image() string {
	return p.image
}

func (p *generatePluginConfig) ExcludeImports() bool {
	return p.excludeImports
}

func (p *generatePluginConfig) ExcludeSourceInfo() bool {
	return p.excludeSourceInfo
}

func (p *generatePluginConfig) AsFileDescriptorSet() bool {
	return p.asFileDescriptorSet
}

func (p *generatePluginConfig) isGeneratePluginConfig() {}

func newExternalGeneratePluginConfigV2FromPluginConfig(
	pluginConfig GeneratePluginConfig,
) (externalGeneratePluginConfigV2, error) {
	generatePluginConfig, ok := pluginConfig.(*generatePluginConfig)
	if !ok {
		return externalGeneratePluginConfigV2{}, syserror.Newf("unknown implementation of GeneratePluginConfig: %T", generatePluginConfig)
	}
	externalPluginConfigV2 := externalGeneratePluginConfigV2{
		Out:            generatePluginConfig.Out(),
		IncludeImports: generatePluginConfig.IncludeImports(),
		IncludeWKT:     generatePluginConfig.IncludeWKT(),
	}
	opts := generatePluginConfig.opts
	switch {
	case len(opts) == 1:
		externalPluginConfigV2.Opt = opts[0]
	case len(opts) > 1:
		externalPluginConfigV2.Opt = opts
	}
	strategy := generatePluginConfig.strategy
	switch {
	case strategy != nil && *strategy == GenerateStrategyDirectory:
		externalPluginConfigV2.Strategy = new("directory")
	case strategy != nil && *strategy == GenerateStrategyAll:
		externalPluginConfigV2.Strategy = new("all")
	}
	switch generatePluginConfig.Type() {
	case GeneratePluginConfigTypeRemote:
		externalPluginConfigV2.Remote = new(generatePluginConfig.Name())
		if revision := generatePluginConfig.Revision(); revision != 0 {
			externalPluginConfigV2.Revision = &revision
		}
	case GeneratePluginConfigTypeLocal:
		path := generatePluginConfig.Path()
		switch {
		case len(path) == 1:
			externalPluginConfigV2.Local = path[0]
		case len(path) > 1:
			externalPluginConfigV2.Local = path
		}
	case GeneratePluginConfigTypeProtocBuiltin:
		externalPluginConfigV2.ProtocBuiltin = new(generatePluginConfig.Name())
		if protocPath := generatePluginConfig.ProtocPath(); len(protocPath) > 0 {
			if len(protocPath) == 1 {
				externalPluginConfigV2.ProtocPath = protocPath[0]
			} else {
				externalPluginConfigV2.ProtocPath = protocPath
			}
		}
	case GeneratePluginConfigTypeLocalOrProtocBuiltin:
		binaryName := "protoc-gen-" + generatePluginConfig.Name()
		// First, check if this is a binary.
		_, err := exec.LookPath(binaryName)
		if err == nil || errors.Is(err, exec.ErrDot) {
			externalPluginConfigV2.Local = binaryName
			break
		}
		// If not, check if it is a protoc plugin.
		if _, isProtocBuiltin := ProtocProxyPluginNames[generatePluginConfig.Name()]; isProtocBuiltin {
			externalPluginConfigV2.ProtocBuiltin = new(generatePluginConfig.Name())
			break
		}
		// Otherwise, assume this is a binary.
		externalPluginConfigV2.Local = binaryName
	case GeneratePluginConfigTypeImage:
		externalPluginConfigV2.Image = new(generatePluginConfig.Image())
		externalPluginConfigV2.ExcludeImports = generatePluginConfig.ExcludeImports()
		externalPluginConfigV2.ExcludeSourceInfo = generatePluginConfig.ExcludeSourceInfo()
		externalPluginConfigV2.AsFileDescriptorSet = generatePluginConfig.AsFileDescriptorSet()
		externalPluginConfigV2.Types = generatePluginConfig.IncludeTypes()
		externalPluginConfigV2.ExcludeTypes = generatePluginConfig.ExcludeTypes()
	}
	return externalPluginConfigV2, nil
}

func parseStrategy(s string) (*GenerateStrategy, error) {
	var strategy GenerateStrategy
	switch s {
	case "":
		return nil, nil
	case "directory":
		strategy = GenerateStrategyDirectory
	case "all":
		strategy = GenerateStrategyAll
	default:
		return nil, fmt.Errorf("unknown strategy: %s", s)
	}
	return &strategy, nil
}

// validateImageFileName validates that the image file name is a normalized
// relative path with a supported extension.
func validateImageFileName(image string) error {
	if image == "" {
		return errors.New("image must not be empty")
	}
	if image == "-" {
		return errors.New(`image must not be "-"`)
	}
	if strings.ContainsRune(image, '#') {
		return fmt.Errorf("image %s must not contain '#'", image)
	}
	normalizedImage, err := normalpath.NormalizeAndValidate(image)
	if err != nil {
		return fmt.Errorf("invalid image %s: %w", image, err)
	}
	if normalizedImage != image {
		return fmt.Errorf("image %s must be a normalized path, use %s", image, normalizedImage)
	}
	uncompressedImage := image
	for _, compressionExt := range []string{".gz", ".zst"} {
		if trimmedImage, ok := strings.CutSuffix(image, compressionExt); ok {
			uncompressedImage = trimmedImage
			break
		}
	}
	switch normalpath.Ext(uncompressedImage) {
	case ".binpb", ".json", ".txtpb", ".yaml":
		return nil
	default:
		return fmt.Errorf(
			"image %s must have one of the extensions .binpb, .json, .txtpb or .yaml, optionally followed by .gz or .zst",
			image,
		)
	}
}

func parseRemoteHostName(fullName string) (string, error) {
	if identity, err := bufremotepluginref.PluginIdentityForString(fullName); err == nil {
		return identity.Remote(), nil
	}
	reference, err := bufremotepluginref.PluginReferenceForString(fullName, 0)
	if err == nil {
		return reference.Remote(), nil
	}
	return "", err
}
