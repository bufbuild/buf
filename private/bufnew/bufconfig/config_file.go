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
