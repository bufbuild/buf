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

package bufconfig

import (
	"io"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
)

// TODO: need to handle bufmigrate, that likely moves into this package and the buflock package.
// TODO: need to handle buf mod init --doc

const (
	// DefaultConfigFileName is the default file name you should use for buf.yaml Files.
	DefaultConfigFileName = "buf.yaml"
	// DefaultGenOnlyFileName is the default file name you should use for buf.gen.yaml Files.
	//
	// This is not included in AllFileNames.
	//
	// For v2, generation configuration is merged into buf.yaml.
	DefaultGenOnlyFileName = "buf.gen.yaml"
)

var (
	// AllConfigFileNames are all file names we have ever used for configuration files.
	//
	// Originally we thought we were going to move to buf.mod, and had this around for
	// a while, but then reverted back to buf.yaml. We still need to support buf.mod as
	// we released with it, however.
	AllConfigFileNames = []string{
		DefaultConfigFileName,
		"buf.mod",
	}
)

// ConfigFile represents a buf.yaml file.
type ConfigFile interface {
	// FileVersion returns the version of the buf.yaml file this was read from.
	FileVersion() FileVersion

	// ModuleConfigs returns the ModuleConfigs for the File.
	//
	// For v1 buf.yaml, this will only have a single ModuleConfig.
	// For buf.gen.yaml, this will be empty.
	ModuleConfigs() []ModuleConfig
	// GenerateConfigs returns the GenerateConfigs for the File.
	//
	// For v1 buf.yaml, this will be empty.
	GenerateConfigs() []GenerateConfig

	isConfigFile()
}

// ReadConfigFile reads the ConfigFile from the io.Reader.
func ReadConfigFile(reader io.Reader) (ConfigFile, error) {
	return readConfigFile(reader)
}

// WriteConfigFile writes the ConfigFile to the io.Writer.
func WriteConfigFile(writer io.Writer, configFile ConfigFile) error {
	return writeConfigFile(writer, configFile)
}

// GenOnlyFile represents a buf.gen.yaml file.
//
// For v2, generation configuration has been merged into Files.
type GenOnlyFile interface {
	GenerateConfig

	// FileVersion returns the version of the buf.gen.yaml file this was read from.
	FileVersion() FileVersion

	isGenOnlyFile()
}

// ReadGenOnlyFile reads the GenOnlyFile from the io.Reader.
func ReadGenOnlyFile(reader io.Reader) (GenOnlyFile, error) {
	return readGenOnlyFile(reader)
}

// WriteGenOnlyFile writes the GenOnlyFile to the io.Writer.
func WriteGenOnlyFile(writer io.Writer, genOnlyFile GenOnlyFile) error {
	return writeGenOnlyFile(writer, genOnlyFile)
}

// ModuleConfig is configuration for a specific Module.
//
// ModuleConfigs do not expose BucketID or OpaqueID, however RootPath is effectively BucketID,
// and ModuleFullName -> fallback to RootPath effectively is OpaqueID. Given that it is up to
// the user of this package to decide what to do with these fields, we do not name RootPath as
// BucketID, and we do not expose OpaqueID.
type ModuleConfig interface {
	// RootPath returns the root path of the Module, if set.
	//
	// For v1 buf.yamls, this is always empty.
	//
	// If not empty, this will be used as the BucketID within Workspaces. For v1, it is up
	// to the Workspace constructor to come up with a BucketID (likely the directory name
	// within buf.work.yaml).
	RootPath() string
	// ModuleFullName returns the ModuleFullName for the Module, if available.
	//
	// This may be nil.
	ModuleFullName() bufmodule.ModuleFullName
	// LintConfig returns the lint configuration.
	//
	// If this was not set, this will be set to the default lint configuration.
	LintConfig() LintConfig
	// BreakingConfig returns the breaking configuration.
	//
	// If this was not set, this will be set to the default breaking configuration.
	BreakingConfig() BreakingConfig

	// TODO: RootToExcludes
	// TODO: DependencyModuleReferences: how do these fit in? We likely add them here,
	// and do not have ModuleConfigs at the bufworkspace level.

	isModuleConfig()
}

// CheckConfig is the common interface for the configuration shared by
// LintConfig and BreakingConfig.
type CheckConfig interface {
	UseIDs() []string
	ExceptIDs() string
	// Paths are specific to the Module.
	IgnorePaths() []string
	// Paths are specific to the Module.
	IgnoreIDToPaths() map[string][]string

	isCheckConfig()
}

// LintConfig is lint configuration for a specific Module.
type LintConfig interface {
	CheckConfig

	EnumZeroValueSuffix() string
	RPCAllowSameRequestResponse() bool
	RPCAllowGoogleProtobufEmptyRequests() bool
	RPCAllowGoogleProtobufEmptyResponses() bool
	ServiceSuffix() string
	AllowCommentIgnores() bool

	isLintConfig()
}

// BreakingConfig is breaking configuration for a specific Module.
type BreakingConfig interface {
	CheckConfig

	IgnoreUnstablePackages() bool

	isBreakingConfig()
}

// GenerateConfig is a generation configuration.
//
// TODO
type GenerateConfig interface {
	isGenerateConfig()
}
