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

package internal

import (
	"context"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protoversion"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// Runner is a runner.
type Runner struct {
	logger       *zap.Logger
	tracer       tracing.Tracer
	ignorePrefix string
}

// NewRunner returns a new Runner.
func NewRunner(logger *zap.Logger, tracer tracing.Tracer, options ...RunnerOption) *Runner {
	runner := &Runner{
		logger: logger,
		tracer: tracer,
	}
	for _, option := range options {
		option(runner)
	}
	return runner
}

// RunnerOption is an option for a new Runner.
type RunnerOption func(*Runner)

// RunnerWithIgnorePrefix returns a new RunnerOption that sets the comment ignore prefix.
//
// This will result in failures where the location has "ignore_prefix id" in the leading
// comment being ignored.
//
// The default is to not enable comment ignores.
func RunnerWithIgnorePrefix(ignorePrefix string) RunnerOption {
	return func(runner *Runner) {
		runner.ignorePrefix = ignorePrefix
	}
}

// Check runs the Rules.
//
// An error of type bufanalysis.FileAnnotationSet will be returned on a rule failure.
func (r *Runner) Check(ctx context.Context, config *Config, previousFiles []bufprotosource.File, files []bufprotosource.File) (retErr error) {
	rules := config.Rules
	if len(rules) == 0 {
		return nil
	}
	for _, rule := range config.Rules {
		if rule.Deprecated() {
			// IgnoreIDToRootPaths relies on us only using the non-deprecated rules.
			return syserror.Newf("Rule %q was send to internal.Runner.Check even though it was deprecated", rule.ID())
		}
	}
	ctx, span := r.tracer.Start(
		ctx,
		tracing.WithErr(&retErr),
		tracing.WithAttributes(
			attribute.Key("num_files").Int(len(files)),
			attribute.Key("num_rules").Int(len(rules)),
		),
	)
	defer span.End()

	ignoreFunc := r.newIgnoreFunc(config)
	var fileAnnotations []bufanalysis.FileAnnotation
	resultC := make(chan *result, len(rules))
	for _, rule := range rules {
		rule := rule
		ruleFunc := func() ([]bufanalysis.FileAnnotation, error) {
			_, span := r.tracer.Start(ctx, tracing.WithSpanNameSuffix(rule.ID()))
			defer span.End()
			return rule.check(ignoreFunc, previousFiles, files)
		}
		go func() {
			iFileAnnotations, iErr := ruleFunc()
			resultC <- newResult(iFileAnnotations, iErr)
		}()
	}
	var err error
	for i := 0; i < len(rules); i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result := <-resultC:
			fileAnnotations = append(fileAnnotations, result.FileAnnotations...)
			err = multierr.Append(err, result.Err)
		}
	}
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		return bufanalysis.NewFileAnnotationSet(fileAnnotations...)
	}
	return nil
}

func (r *Runner) newIgnoreFunc(config *Config) IgnoreFunc {
	return func(id string, descriptors []bufprotosource.Descriptor, locations []bufprotosource.Location) bool {
		if idIsIgnored(id, descriptors, config) {
			return true
		}
		// if ignorePrefix is empty, comment ignores are not enabled for the runner
		// this is the case with breaking changes
		if r.ignorePrefix != "" && config.AllowCommentIgnores &&
			locationsAreIgnored(id, r.ignorePrefix, locations, config) {
			return true
		}
		if config.IgnoreUnstablePackages {
			for _, descriptor := range descriptors {
				if descriptorPackageIsUnstable(descriptor) {
					return true
				}
			}
		}
		return false
	}
}

func idIsIgnored(id string, descriptors []bufprotosource.Descriptor, config *Config) bool {
	for _, descriptor := range descriptors {
		// OR of descriptors
		if idIsIgnoredForDescriptor(id, descriptor, config) {
			return true
		}
	}
	return false
}

func idIsIgnoredForDescriptor(id string, descriptor bufprotosource.Descriptor, config *Config) bool {
	if descriptor == nil {
		return false
	}
	path := descriptor.File().Path()
	if normalpath.MapHasEqualOrContainingPath(config.IgnoreRootPaths, path, normalpath.Relative) {
		return true
	}
	if id == "" {
		return false
	}
	ignoreRootPaths, ok := config.IgnoreIDToRootPaths[id]
	if !ok {
		return false
	}
	return normalpath.MapHasEqualOrContainingPath(ignoreRootPaths, path, normalpath.Relative)
}

func locationsAreIgnored(id string, ignorePrefix string, locations []bufprotosource.Location, config *Config) bool {
	// we already check that ignorePrefix is non-empty, but just doing here for safety
	if id == "" || ignorePrefix == "" {
		return false
	}
	fullIgnorePrefix := ignorePrefix + " " + id
	for _, location := range locations {
		if location != nil {
			if leadingComments := location.LeadingComments(); leadingComments != "" {
				for _, line := range stringutil.SplitTrimLinesNoEmpty(leadingComments) {
					if strings.HasPrefix(line, fullIgnorePrefix) {
						return true
					}
				}
			}
		}
	}
	return false
}

func descriptorPackageIsUnstable(descriptor bufprotosource.Descriptor) bool {
	if descriptor == nil {
		return false
	}
	packageVersion, ok := protoversion.NewPackageVersionForPackage(descriptor.File().Package())
	if !ok {
		return false
	}
	return packageVersion.StabilityLevel() != protoversion.StabilityLevelStable
}

type result struct {
	FileAnnotations []bufanalysis.FileAnnotation
	Err             error
}

func newResult(fileAnnotations []bufanalysis.FileAnnotation, err error) *result {
	return &result{
		FileAnnotations: fileAnnotations,
		Err:             err,
	}
}
