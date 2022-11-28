// Copyright 2020-2022 Buf Technologies, Inc.
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
	"sync"

	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/thread"
)

type checker struct {
	logger *zap.Logger
	runner command.Runner
}

func newChecker(logger *zap.Logger, runner command.Runner) *checker {
	return &checker{
		logger: logger,
		runner: runner,
	}
}

func (c *checker) Check(
	ctx context.Context,
	envStdioContainer app.EnvStdioContainer,
	externalConfig ExternalConfig,
) ([]Violation, error) {
	state := newState(c.logger, envStdioContainer, c.runner)
	if err := c.populateState(ctx, state, externalConfig); err != nil {
		return nil, err
	}
	for _, externalBanConfig := range externalConfig.Bans {
		if err := c.checkBan(ctx, state, externalBanConfig); err != nil {
			return nil, err
		}
	}
	return state.Violations(), nil
}

func (c *checker) checkBan(
	ctx context.Context,
	state *state,
	externalBanConfig ExternalBanConfig,
) error {
	ctx, span := trace.StartSpan(ctx, "checkBan")
	defer span.End()

	packages, err := c.getPackages(ctx, state, externalBanConfig.Packages)
	if err != nil {
		return err
	}
	banPackages, err := c.getPackages(ctx, state, externalBanConfig.Deps)
	if err != nil {
		return err
	}

	for pkg := range packages {
		deps, err := state.DepsForPackages(ctx, pkg)
		if err != nil {
			return err
		}
		for dep := range deps {
			if _, ok := banPackages[dep]; ok {
				state.AddViolation(
					newViolation(
						pkg,
						dep,
						externalBanConfig.Note,
					),
				)
			}
		}
	}

	return nil
}

func (c *checker) getPackages(
	ctx context.Context,
	state *state,
	externalPackageConfig ExternalPackageConfig,
) (map[string]struct{}, error) {
	usePackages, err := state.PackagesForPackageExpressions(ctx, externalPackageConfig.Use...)
	if err != nil {
		return nil, err
	}
	exceptPackages, err := state.PackagesForPackageExpressions(ctx, externalPackageConfig.Except...)
	if err != nil {
		return nil, err
	}
	subtractMaps(usePackages, exceptPackages)
	return usePackages, nil
}

func (c *checker) populateState(ctx context.Context, state *state, externalConfig ExternalConfig) error {
	ctx, span := trace.StartSpan(ctx, "populateState")
	defer span.End()

	var depPackageExpressions []string
	var packageExpressions []string
	for _, externalBanConfig := range externalConfig.Bans {
		depPackageExpressions = append(depPackageExpressions, externalBanConfig.Packages.Use...)
		depPackageExpressions = append(depPackageExpressions, externalBanConfig.Packages.Except...)
		packageExpressions = append(packageExpressions, externalBanConfig.Deps.Use...)
		packageExpressions = append(packageExpressions, externalBanConfig.Deps.Except...)
	}

	depPackages := make(map[string]struct{})
	var lock sync.Mutex
	var jobs []func(context.Context) error
	for _, packageExpression := range depPackageExpressions {
		packageExpression := packageExpression
		jobs = append(
			jobs,
			func(ctx context.Context) error {
				pkgs, err := state.PackagesForPackageExpressions(ctx, packageExpression)
				if err != nil {
					return err
				}
				lock.Lock()
				addMaps(depPackages, pkgs)
				lock.Unlock()
				return nil
			},
		)
	}
	for _, packageExpression := range packageExpressions {
		packageExpression := packageExpression
		jobs = append(
			jobs,
			func(ctx context.Context) error {
				_, err := state.PackagesForPackageExpressions(ctx, packageExpression)
				return err
			},
		)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := thread.Parallelize(ctx, jobs, thread.ParallelizeWithCancel(cancel)); err != nil {
		return err
	}

	jobs = make([]func(context.Context) error, 0)
	for pkg := range depPackages {
		pkg := pkg
		jobs = append(
			jobs,
			func(ctx context.Context) error {
				_, err := state.DepsForPackages(ctx, pkg)
				return err
			},
		)
	}
	return thread.Parallelize(ctx, jobs, thread.ParallelizeWithCancel(cancel))
}
