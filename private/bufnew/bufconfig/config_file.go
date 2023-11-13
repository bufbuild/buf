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
	"errors"
	"io"
)

const (
	// DefaultConfigFileName is the default file name you should use for buf.yaml Files.
	DefaultConfigFileName = "buf.yaml"
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
	configFile, err := readConfigFile(reader)
	if err != nil {
		return nil, err
	}
	if err := checkV2SupportedYet(configFile.FileVersion()); err != nil {
		return nil, err
	}
	return configFile, nil
}

// WriteConfigFile writes the ConfigFile to the io.Writer.
func WriteConfigFile(writer io.Writer, configFile ConfigFile) error {
	if err := checkV2SupportedYet(configFile.FileVersion()); err != nil {
		return err
	}
	return writeConfigFile(writer, configFile)
}

// *** PRIVATE ***

type configFile struct{}

func newConfigFile() *configFile {
	return &configFile{}
}

func (c *configFile) FileVersion() FileVersion {
	panic("not implemented") // TODO: Implement
}

func (c *configFile) ModuleConfigs() []ModuleConfig {
	panic("not implemented") // TODO: Implement
}

func (c *configFile) GenerateConfigs() []GenerateConfig {
	panic("not implemented") // TODO: Implement
}

func (*configFile) isConfigFile() {}

func readConfigFile(reader io.Reader) (ConfigFile, error) {
	return nil, errors.New("TODO")
}

func writeConfigFile(writer io.Writer, configFile ConfigFile) error {
	return errors.New("TODO")
}
