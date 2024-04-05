// Copyright 2020-2024 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/private/pkg/slicesext"
)

// LintPluginConfig is configuration for a lint plugin.
type LintPluginConfig interface {
	Path() string
	Args() []string

	isLintPluginConfig()
}

// NewLintPluginConfig returns a new LintPluginConfig.
func NewLintPluginConfig(
	path string,
	args []string,
) (LintPluginConfig, error) {
	return newLintPluginConfig(
		path,
		args,
	)
}

// *** PRIVATE ***

type lintPluginConfig struct {
	path string
	args []string
}

func newLintPluginConfig(
	path string,
	args []string,
) (*lintPluginConfig, error) {
	if path == "" {
		return nil, errors.New("empty path for LintPluginConfig")
	}
	return &lintPluginConfig{
		path: path,
		args: args,
	}, nil
}

func (l *lintPluginConfig) Path() string {
	return l.path
}

func (l *lintPluginConfig) Args() []string {
	return slicesext.Copy(l.args)
}

func (*lintPluginConfig) isLintPluginConfig() {}
