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

package bandeps

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"go.uber.org/zap"
)

const (
	tracerName = "bufbuild/buf/bandeps"
)

// Checker is a checker.
type Checker interface {
	// Check runs bandeps in the current directory with the configuration.
	Check(
		ctx context.Context,
		envStdioContainer app.EnvStdioContainer,
		externalConfig ExternalConfig,
	) ([]Violation, error)
}

func NewChecker(logger *zap.Logger, runner command.Runner) Checker {
	return newChecker(logger, runner)
}

// Violation is a violation.
type Violation interface {
	fmt.Stringer

	Package() string
	Dep() string
	Note() string

	key() string
}

// ExternalConfig is an external configuation.
type ExternalConfig struct {
	Bans []ExternalBanConfig `json:"bans,omitempty" yaml:"bans,omitempty"`
}

// ExternalConfig is an external ban configuation.
type ExternalBanConfig struct {
	// Packages are the package expressions to get dependencies for.
	Packages ExternalPackageConfig `json:"packages,omitempty" yaml:"packages,omitempty"`
	// Deps are package expressions that cannot be depended on for Packages.
	Deps ExternalPackageConfig `json:"deps,omitempty" yaml:"deps,omitempty"`
	// Note is a note to print out regarding why this ban exists.
	Note string `json:"note,omitempty" yaml:"note,omitempty"`
}

type ExternalPackageConfig struct {
	// Use are the package expressions to list with go list.
	Use []string `json:"use,omitempty" yaml:"use,omitempty"`
	// Except are the package expressions that should be excluded from Use.
	Except []string `json:"except,omitempty" yaml:"except,omitempty"`
}
