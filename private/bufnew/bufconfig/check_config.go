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

type checkConfig struct{}

func newCheckConfig() checkConfig {
	return checkConfig{}
}

func (c *checkConfig) UseIDs() []string {
	panic("not implemented") // TODO: Implement
}

func (c *checkConfig) ExceptIDs() string {
	panic("not implemented") // TODO: Implement
}

func (c *checkConfig) IgnorePaths() []string {
	panic("not implemented") // TODO: Implement
}

func (c *checkConfig) IgnoreIDToPaths() map[string][]string {
	panic("not implemented") // TODO: Implement
}

func (*checkConfig) isCheckConfig() {}
