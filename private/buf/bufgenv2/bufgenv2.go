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

package bufgenv2

import "github.com/bufbuild/buf/private/bufpkg/bufimage"

// TODO this would be part of a runner or likewise
// this is just for demonstration of bringing the management stuff into one function
func ApplyManagement(image bufimage.Image, managedConfig *ManagedConfig) error {
	markSweeper := NewMarkSweeper(image)
	for _, imageFile := range image.Files() {
		if err := applyManagementForFile(markSweeper, imageFile, managedConfig); err != nil {
			return err
		}
	}
	return markSweeper.Sweep()
}

type Marker interface {
	Mark(bufimage.ImageFile, []int32)
}

type Sweeper interface {
	Sweep() error
}

type MarkSweeper interface {
	Marker
	Sweeper
}

func NewMarkSweeper(image bufimage.Image) MarkSweeper {
	return nil
}

type DisabledFunc func(FileOption, bufimage.ImageFile) bool

// TODO: likely want something like *string or otherwise, see https://github.com/bufbuild/buf/issues/1949
type OverrideFunc func(bufimage.ImageFile) (string, error)

type Config struct {
	Managed *ManagedConfig
	Plugins []*PluginConfig
	Inputs  []*InputConfig
}

// TODO: We use nil or not to denote enabled or not, but that deems dangerous
type ManagedConfig struct {
	DisabledFunc             DisabledFunc
	FileOptionToOverrideFunc map[FileOption]OverrideFunc
}

type PluginConfig struct {
	Remote        string
	Binary        []string
	ProtocBuiltin string
	Wasm          string
	// Only valid with Remote
	Revision int
	// Only valid with ProtocBuiltin
	ProtocPath string
	// Required
	Out            string
	Opt            string
	IncludeImports bool
	IncludeWKT     bool
	// Required
	// TODO: Strategy enum in bufgen
	Strategy string
}

type InputConfig struct {
	Path  string
	Types []string
}

type ExternalConfigV2 struct {
	// Must be V2 in this current code setup, but we'd want this to be alongside V1
	Version string                   `json:"version,omitempty" yaml:"version,omitempty"`
	Managed ExternalManagedConfigV2  `json:"managed,omitempty" yaml:"managed,omitempty"`
	Plugins []ExternalPluginConfigV2 `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Inputs  []ExternalInputConfigV2  `json:"inputs,omitempty" yaml:"inputs,omitempty"`
}

type ExternalManagedConfigV2 struct {
	Enable   bool                              `json:"enable,omitempty" yaml:"enable,omitempty"`
	Disable  []ExternalManagedDisableConfigV2  `json:"disable,omitempty" yaml:"disable,omitempty"`
	Override []ExternalManagedOverrideConfigV2 `json:"override,omitempty" yaml:"override,omitempty"`
}

func (m ExternalManagedConfigV2) IsEmpty() bool {
	return !m.Enable && len(m.Disable) == 0 && len(m.Override) == 0
}

type ExternalManagedDisableConfigV2 struct {
	// Must be validated to be a valid FileOption
	FileOption string `json:"file_option,omitempty" yaml:"file_option,omitempty"`
	// Must be validated to be a valid module path
	Module string `json:"module,omitempty" yaml:"module,omitempty"`
	// Must be normalized and validated
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

type ExternalManagedOverrideConfigV2 struct {
	// Must be validated to be a valid FileOption
	// Required
	FileOption string `json:"file_option,omitempty" yaml:"file_option,omitempty"`
	// Must be validated to be a valid module path
	Module string `json:"module,omitempty" yaml:"module,omitempty"`
	// Must be normalized and validated
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
	// Only one of Value and Prefix can be set
	// TODO: may be interface{}, what to do about boo, optimize_mode, etc
	Value  string `json:"value,omitempty" yaml:"value,omitempty"`
	Prefix string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
}

type ExternalPluginConfigV2 struct {
	// Only one of Remote, Binary, Wasm, ProtocBuiltin can be set
	Remote string `json:"remote,omitempty" yaml:"remote,omitempty"`
	// Can be multiple arguments
	// All arguments must be strings
	Binary        interface{} `json:"binary,omitempty" yaml:"binary,omitempty"`
	Wasm          string      `json:"wasm,omitempty" yaml:"wasm,omitempty"`
	ProtocBuiltin string      `json:"protoc_builtin,omitempty" yaml:"protoc_builtin,omitempty"`
	// Only valid with Remote
	Revision int `json:"revision,omitempty" yaml:"revision,omitempty"`
	// Only valid with ProtocBuiltin
	ProtocPath string `json:"protoc_path,omitempty" yaml:"protoc_path,omitempty"`
	// Required
	Out string `json:"out,omitempty" yaml:"out,omitempty"`
	// Can be one string or multiple strings
	Opt            interface{} `json:"opt,omitempty" yaml:"opt,omitempty"`
	IncludeImports bool        `json:"include_imports,omitempty" yaml:"include_imports,omitempty"`
	IncludeWKT     bool        `json:"include_wkt,omitempty" yaml:"include_wkt,omitempty"`
	// Must be a valid Strategy
	Strategy string `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

type ExternalInputConfigV2 struct {
	// TODO: split up into Git, Module, etc
	Path  string
	Types []string
}
