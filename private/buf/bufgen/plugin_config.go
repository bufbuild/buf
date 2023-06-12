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

package bufgen

import (
	"errors"

	"github.com/bufbuild/buf/private/buf/bufgen/internal"
)

type localPluginConfig struct {
	name     string
	strategy internal.Strategy
	out      string
	opt      string
}

func newLocalPluginConfig(
	name string,
	strategy internal.Strategy,
	out string,
	opt string,
) *localPluginConfig {
	return &localPluginConfig{
		name:     name,
		strategy: strategy,
		out:      out,
		opt:      opt,
	}
}

func (c *localPluginConfig) PluginName() string {
	return c.name
}

func (c *localPluginConfig) Out() string {
	return c.out
}

func (c *localPluginConfig) Opt() string {
	return c.opt
}

func (c *localPluginConfig) Strategy() internal.Strategy {
	return c.strategy
}

func (c *localPluginConfig) pluginConfig()      {}
func (c *localPluginConfig) localPluginConfig() {}

type binaryPluginConfig struct {
	name     string
	out      string
	opt      string
	strategy internal.Strategy
	path     []string // pluginName is the first element
}

func newBinaryPluginConfig(
	name string,
	path []string,
	strategy internal.Strategy,
	out string,
	opt string,
) (*binaryPluginConfig, error) {
	if len(path) == 0 {
		return nil, errors.New("must specify a path to the plugin")
	}
	return &binaryPluginConfig{
		name:     name,
		path:     path,
		strategy: strategy,
		out:      out,
		opt:      opt,
	}, nil
}

func (c *binaryPluginConfig) Path() []string {
	return c.path
}

func (c *binaryPluginConfig) PluginName() string {
	if c.name != "" {
		return c.name
	}
	return c.path[0]
}

func (c *binaryPluginConfig) Out() string {
	return c.out
}

func (c *binaryPluginConfig) Opt() string {
	return c.opt
}

func (c *binaryPluginConfig) Strategy() internal.Strategy {
	return c.strategy
}

func (c *binaryPluginConfig) pluginConfig()       {}
func (c *binaryPluginConfig) localPluginConfig()  {}
func (c *binaryPluginConfig) binaryPluginConfig() {}

type protocBuiltinPluginConfig struct {
	out        string
	opt        string
	strategy   internal.Strategy
	name       string // pluginName is this
	protocPath string
}

func newProtocBuiltinPluginConfig(
	name string,
	protocPath string,
	out string,
	opt string,
	strategy internal.Strategy,
) *protocBuiltinPluginConfig {
	return &protocBuiltinPluginConfig{
		name:       name,
		protocPath: protocPath,
		out:        out,
		opt:        opt,
		strategy:   strategy,
	}
}

func (c *protocBuiltinPluginConfig) PluginName() string {
	return c.name
}

func (c *protocBuiltinPluginConfig) ProtocPath() string {
	return c.protocPath
}

func (c *protocBuiltinPluginConfig) Strategy() internal.Strategy {
	return c.strategy
}

func (c *protocBuiltinPluginConfig) Out() string {
	return c.out
}

func (c *protocBuiltinPluginConfig) Opt() string {
	return c.opt
}

func (c *protocBuiltinPluginConfig) pluginConfig()              {}
func (c *protocBuiltinPluginConfig) localPluginConfig()         {}
func (c *protocBuiltinPluginConfig) protocBuiltinPluginConfig() {}

type curatedPluginConfig struct {
	plugin   string // pluginName is this
	revision int
	out      string
	opt      string
}

func newCuratedPluginConfig(
	plugin string,
	revision int, // TODO: maybe pointer to indicate absence
	out string,
	opt string,
) *curatedPluginConfig {
	return &curatedPluginConfig{
		plugin:   plugin,
		revision: revision,
		out:      out,
		opt:      opt,
	}
}

func (c *curatedPluginConfig) PluginName() string {
	return c.plugin
}

func (c *curatedPluginConfig) Remote() string {
	return c.plugin
}
func (c *curatedPluginConfig) Revision() int {
	return c.revision
}

func (c *curatedPluginConfig) Out() string {
	return c.out
}

func (c *curatedPluginConfig) Opt() string {
	return c.opt
}

func (c *curatedPluginConfig) pluginConfig()        {}
func (c *curatedPluginConfig) remotePluginConfig()  {}
func (c *curatedPluginConfig) curatedPluginConfig() {}

type legacyRemotePluginConfig struct {
	out    string
	opt    string
	remote string // pluginName is this
}

func newLegacyRemotePluginConfig(
	remote string,
	out string,
	opt string,
) *legacyRemotePluginConfig {
	return &legacyRemotePluginConfig{
		remote: remote,
		out:    out,
		opt:    opt,
	}
}

func (c *legacyRemotePluginConfig) PluginName() string {
	return c.remote
}

func (c *legacyRemotePluginConfig) Remote() string {
	return c.remote
}

func (c *legacyRemotePluginConfig) Out() string {
	return c.out
}

func (c *legacyRemotePluginConfig) Opt() string {
	return c.opt
}

func (c *legacyRemotePluginConfig) pluginConfig()             {}
func (c *legacyRemotePluginConfig) remotePluginConfig()       {}
func (c *legacyRemotePluginConfig) legacyRemotePluginConfig() {}
