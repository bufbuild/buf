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

package internal

import (
	"errors"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin"
)

type localPluginConfig struct {
	name           string
	strategy       Strategy
	out            string
	opt            string
	includeImports bool
	includeWKT     bool
}

func newLocalPluginConfig(
	name string,
	strategy Strategy,
	out string,
	opt string,
	includeImports bool,
	includeWKT bool,
) *localPluginConfig {
	return &localPluginConfig{
		name:           name,
		strategy:       strategy,
		out:            out,
		opt:            opt,
		includeImports: includeImports,
		includeWKT:     includeWKT,
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

func (c *localPluginConfig) Strategy() Strategy {
	return c.strategy
}

func (c *localPluginConfig) IncludeImports() bool {
	return c.includeImports
}

func (c *localPluginConfig) IncludeWKT() bool {
	return c.includeWKT
}

func (c *localPluginConfig) pluginConfig()      {}
func (c *localPluginConfig) localPluginConfig() {}

type binaryPluginConfig struct {
	name           string
	out            string
	opt            string
	strategy       Strategy
	path           []string
	includeImports bool
	includeWKT     bool
}

func newBinaryPluginConfig(
	name string,
	path []string,
	strategy Strategy,
	out string,
	opt string,
	includeImports bool,
	includeWKT bool,
) (*binaryPluginConfig, error) {
	if len(path) == 0 {
		return nil, errors.New("must specify a path to the plugin")
	}
	return &binaryPluginConfig{
		name:           name,
		path:           path,
		strategy:       strategy,
		out:            out,
		opt:            opt,
		includeImports: includeImports,
		includeWKT:     includeWKT,
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

func (c *binaryPluginConfig) Strategy() Strategy {
	return c.strategy
}

func (c *binaryPluginConfig) IncludeImports() bool {
	return c.includeImports
}

func (c *binaryPluginConfig) IncludeWKT() bool {
	return c.includeWKT
}

func (c *binaryPluginConfig) pluginConfig()       {}
func (c *binaryPluginConfig) localPluginConfig()  {}
func (c *binaryPluginConfig) binaryPluginConfig() {}

type protocBuiltinPluginConfig struct {
	name           string
	out            string
	opt            string
	strategy       Strategy
	protocPath     string
	includeImports bool
	includeWKT     bool
}

func newProtocBuiltinPluginConfig(
	name string,
	protocPath string,
	out string,
	opt string,
	includeImports bool,
	includeWKT bool,
	strategy Strategy,
) *protocBuiltinPluginConfig {
	return &protocBuiltinPluginConfig{
		name:           name,
		protocPath:     protocPath,
		out:            out,
		opt:            opt,
		strategy:       strategy,
		includeImports: includeImports,
		includeWKT:     includeWKT,
	}
}

func (c *protocBuiltinPluginConfig) PluginName() string {
	return c.name
}

func (c *protocBuiltinPluginConfig) ProtocPath() string {
	return c.protocPath
}

func (c *protocBuiltinPluginConfig) Strategy() Strategy {
	return c.strategy
}

func (c *protocBuiltinPluginConfig) IncludeImports() bool {
	return c.includeImports
}

func (c *protocBuiltinPluginConfig) IncludeWKT() bool {
	return c.includeWKT
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
	fullName       string
	remoteHost     string
	revision       int
	out            string
	opt            string
	includeImports bool
	includeWKT     bool
}

func newCuratedPluginConfig(
	fullName string,
	revision int,
	out string,
	opt string,
	includeImports bool,
	includeWKT bool,
) (*curatedPluginConfig, error) {
	remoteHost, err := parseCuratedRemoteHostName(fullName)
	if err != nil {
		return nil, err
	}
	return &curatedPluginConfig{
		fullName:       fullName,
		remoteHost:     remoteHost,
		revision:       revision,
		out:            out,
		opt:            opt,
		includeImports: includeImports,
		includeWKT:     includeWKT,
	}, nil
}

func (c *curatedPluginConfig) PluginName() string {
	return c.fullName
}

func (c *curatedPluginConfig) RemoteHost() string {
	return c.remoteHost
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

func (c *curatedPluginConfig) IncludeImports() bool {
	return c.includeImports
}

func (c *curatedPluginConfig) IncludeWKT() bool {
	return c.includeWKT
}

func (c *curatedPluginConfig) pluginConfig()        {}
func (c *curatedPluginConfig) remotePluginConfig()  {}
func (c *curatedPluginConfig) curatedPluginConfig() {}

type legacyRemotePluginConfig struct {
	fullName       string
	remoteHost     string
	out            string
	opt            string
	includeImports bool
	includeWKT     bool
}

func newLegacyRemotePluginConfig(
	fullName string,
	out string,
	opt string,
	includeImports bool,
	includeWKT bool,
) (*legacyRemotePluginConfig, error) {
	remoteHost, err := parseLegacyRemoteHostName(fullName)
	if err != nil {
		return nil, err
	}
	return &legacyRemotePluginConfig{
		fullName:       fullName,
		remoteHost:     remoteHost,
		out:            out,
		opt:            opt,
		includeImports: includeImports,
		includeWKT:     includeWKT,
	}, nil
}

func (c *legacyRemotePluginConfig) PluginName() string {
	return c.fullName
}

func (c *legacyRemotePluginConfig) RemoteHost() string {
	return c.remoteHost
}

func (c *legacyRemotePluginConfig) Out() string {
	return c.out
}

func (c *legacyRemotePluginConfig) Opt() string {
	return c.opt
}

func (c *legacyRemotePluginConfig) IncludeImports() bool {
	return c.includeImports
}

func (c *legacyRemotePluginConfig) IncludeWKT() bool {
	return c.includeWKT
}

func (c *legacyRemotePluginConfig) pluginConfig()             {}
func (c *legacyRemotePluginConfig) remotePluginConfig()       {}
func (c *legacyRemotePluginConfig) legacyRemotePluginConfig() {}

func parseCuratedRemoteHostName(fullName string) (string, error) {
	if identity, err := bufpluginref.PluginIdentityForString(fullName); err == nil {
		return identity.Remote(), nil
	}
	reference, err := bufpluginref.PluginReferenceForString(fullName, 0)
	if err == nil {
		return reference.Remote(), nil
	}
	return "", err
}

func parseLegacyRemoteHostName(fullName string) (string, error) {
	remote, _, _, _, err := bufremoteplugin.ParsePluginVersionPath(fullName)
	if err != nil {
		return "", err
	}
	return remote, nil
}
