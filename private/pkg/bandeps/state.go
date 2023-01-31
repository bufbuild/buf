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
	"sync"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type state struct {
	logger            *zap.Logger
	envStdioContainer app.EnvStdioContainer
	runner            command.Runner
	violationMap      map[string]Violation
	// map from ./foo/bar/... to actual packages
	packageExpressionToPackages     map[string]*packagesResult
	packageExpressionToPackagesLock *keyRWLock
	// map from packages to dependencies
	packageToDeps     map[string]*depsResult
	packageToDepsLock *keyRWLock
	lock              sync.RWMutex
	calls             int
	cacheHits         int
	tracer            trace.Tracer
}

func newState(
	logger *zap.Logger,
	envStdioContainer app.EnvStdioContainer,
	runner command.Runner,
) *state {
	return &state{
		logger:                          logger,
		envStdioContainer:               envStdioContainer,
		runner:                          runner,
		violationMap:                    make(map[string]Violation),
		packageExpressionToPackages:     make(map[string]*packagesResult),
		packageExpressionToPackagesLock: newKeyRWLock(),
		packageToDeps:                   make(map[string]*depsResult),
		packageToDepsLock:               newKeyRWLock(),
		tracer:                          otel.GetTracerProvider().Tracer(tracerName),
	}
}

func (s *state) PackagesForPackageExpressions(
	ctx context.Context,
	packageExpressions ...string,
) (map[string]struct{}, error) {
	packages := make(map[string]struct{})
	for _, packageExpression := range packageExpressions {
		iPackages, err := s.packagesForPackageExpression(ctx, packageExpression)
		if err != nil {
			return nil, err
		}
		addMaps(packages, iPackages)
	}
	return packages, nil
}

func (s *state) DepsForPackages(
	ctx context.Context,
	pkgs ...string,
) (map[string]struct{}, error) {
	deps := make(map[string]struct{})
	for _, pkg := range pkgs {
		iDeps, err := s.depsForPackage(ctx, pkg)
		if err != nil {
			return nil, err
		}
		addMaps(deps, iDeps)
	}
	return deps, nil
}

func (s *state) AddViolation(violation Violation) {
	violationKey := violation.key()
	s.lock.Lock()
	if _, ok := s.violationMap[violationKey]; !ok {
		s.violationMap[violationKey] = violation
	}
	s.lock.Unlock()
}

func (s *state) Violations() []Violation {
	s.lock.RLock()
	violations := make([]Violation, 0, len(s.violationMap))
	for _, violation := range s.violationMap {
		violations = append(violations, violation)
	}
	s.lock.RUnlock()
	sortViolations(violations)
	return violations
}

func (s *state) packagesForPackageExpression(
	ctx context.Context,
	packageExpression string,
) (map[string]struct{}, error) {
	defer func() {
		// not worrying about locks
		s.logger.Debug("cache", zap.Int("calls", s.calls), zap.Int("hits", s.cacheHits))
	}()

	s.packageExpressionToPackagesLock.RLock(packageExpression)
	s.lock.RLock()
	packageResult, ok := s.packageExpressionToPackages[packageExpression]
	s.lock.RUnlock()
	s.packageExpressionToPackagesLock.RUnlock(packageExpression)
	if ok {
		s.lock.Lock()
		s.calls++
		s.cacheHits++
		s.lock.Unlock()
		return packageResult.Packages, packageResult.Err
	}

	s.packageExpressionToPackagesLock.Lock(packageExpression)
	defer s.packageExpressionToPackagesLock.Unlock(packageExpression)

	s.lock.RLock()
	packageResult, ok = s.packageExpressionToPackages[packageExpression]
	s.lock.RUnlock()
	if ok {
		s.lock.Lock()
		s.calls++
		s.cacheHits++
		s.lock.Unlock()
		return packageResult.Packages, packageResult.Err
	}
	packages, err := s.packagesForPackageExpressionUncached(ctx, packageExpression)
	// we always hold key lock and then this lock so lock ordering is ok
	s.lock.Lock()
	s.packageExpressionToPackages[packageExpression] = newPackagesResult(packages, err)
	s.calls++
	s.lock.Unlock()
	return packages, err
}

func (s *state) packagesForPackageExpressionUncached(
	ctx context.Context,
	packageExpression string,
) (map[string]struct{}, error) {
	ctx, span := s.tracer.Start(ctx, "packagesForPackageExpressionUncached", trace.WithAttributes(
		attribute.Key("packageExpression").String(packageExpression),
	))
	defer span.End()

	data, err := command.RunStdout(ctx, s.envStdioContainer, s.runner, `go`, `list`, packageExpression)
	if err != nil {
		return nil, err
	}
	return stringutil.SliceToMap(getNonEmptyLines(string(data))), nil
}

func (s *state) depsForPackage(
	ctx context.Context,
	pkg string,
) (map[string]struct{}, error) {
	defer func() {
		// not worrying about locks
		s.logger.Debug("cache", zap.Int("calls", s.calls), zap.Int("hits", s.cacheHits))
	}()

	s.packageToDepsLock.RLock(pkg)
	s.lock.RLock()
	depResult, ok := s.packageToDeps[pkg]
	s.lock.RUnlock()
	s.packageToDepsLock.RUnlock(pkg)
	if ok {
		s.lock.Lock()
		s.calls++
		s.cacheHits++
		s.lock.Unlock()
		return depResult.Deps, depResult.Err
	}

	s.packageToDepsLock.Lock(pkg)
	defer s.packageToDepsLock.Unlock(pkg)

	s.lock.RLock()
	depResult, ok = s.packageToDeps[pkg]
	s.lock.RUnlock()
	if ok {
		s.lock.Lock()
		s.calls++
		s.cacheHits++
		s.lock.Unlock()
		return depResult.Deps, depResult.Err
	}
	deps, err := s.depsForPackageUncached(ctx, pkg)
	// we always hold key lock and then this lock so lock ordering is ok
	s.lock.Lock()
	s.packageToDeps[pkg] = newDepsResult(deps, err)
	s.calls++
	s.lock.Unlock()
	return deps, err
}

func (s *state) depsForPackageUncached(
	ctx context.Context,
	pkg string,
) (map[string]struct{}, error) {
	ctx, span := s.tracer.Start(ctx, "depsForPackageUncached", trace.WithAttributes(
		attribute.Key("package").String(pkg),
	))
	defer span.End()

	data, err := command.RunStdout(ctx, s.envStdioContainer, s.runner, `go`, `list`, `-f`, `{{join .Deps "\n"}}`, pkg)
	if err != nil {
		return nil, err
	}
	return stringutil.SliceToMap(getNonEmptyLines(string(data))), nil
}

type packagesResult struct {
	Packages map[string]struct{}
	Err      error
}

func newPackagesResult(
	packages map[string]struct{},
	err error,
) *packagesResult {
	return &packagesResult{
		Packages: packages,
		Err:      err,
	}
}

type depsResult struct {
	Deps map[string]struct{}
	Err  error
}

func newDepsResult(
	deps map[string]struct{},
	err error,
) *depsResult {
	return &depsResult{
		Deps: deps,
		Err:  err,
	}
}
