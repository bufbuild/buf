// Copyright 2020 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/pkg/instrument"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/protosource"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// Runner is a runner.
type Runner struct {
	logger       *zap.Logger
	ignorePrefix string
}

// NewRunner returns a new Runner.
//
// ignorePrefix should be empty if comment ignores are not allowed
func NewRunner(logger *zap.Logger, ignorePrefix string) *Runner {
	return &Runner{
		logger:       logger,
		ignorePrefix: ignorePrefix,
	}
}

// Check runs the Checkers.
func (r *Runner) Check(ctx context.Context, config *Config, previousFiles []protosource.File, files []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	checkers := config.Checkers
	if len(checkers) == 0 {
		return nil, nil
	}
	defer instrument.Start(r.logger, "check", zap.Int("num_files", len(files)), zap.Int("num_checkers", len(checkers))).End()

	ignoreFunc := r.newIgnoreFunc(config)
	var fileAnnotations []bufanalysis.FileAnnotation
	resultC := make(chan *result, len(checkers))
	for _, checker := range checkers {
		checker := checker
		go func() {
			iFileAnnotations, iErr := checker.check(ignoreFunc, previousFiles, files)
			resultC <- newResult(iFileAnnotations, iErr)
		}()
	}
	var err error
	for i := 0; i < len(checkers); i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-resultC:
			fileAnnotations = append(fileAnnotations, result.FileAnnotations...)
			err = multierr.Append(err, result.Err)
		}
	}
	if err != nil {
		return nil, err
	}
	bufanalysis.SortFileAnnotations(fileAnnotations)
	return fileAnnotations, nil

}

func (r *Runner) newIgnoreFunc(config *Config) IgnoreFunc {
	if r.ignorePrefix == "" || !config.AllowCommentIgnores {
		return func(id string, descriptor protosource.Descriptor, location protosource.Location) bool {
			return idIsIgnored(id, descriptor, config)
		}
	}
	return func(id string, descriptor protosource.Descriptor, location protosource.Location) bool {
		return locationIsIgnored(id, r.ignorePrefix, location, config) || idIsIgnored(id, descriptor, config)
	}
}

func idIsIgnored(id string, descriptor protosource.Descriptor, config *Config) bool {
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

func locationIsIgnored(id string, ignorePrefix string, location protosource.Location, config *Config) bool {
	if id == "" || ignorePrefix == "" {
		return false
	}
	if location == nil {
		return false
	}
	leadingComments := location.LeadingComments()
	if leadingComments == "" {
		return false
	}
	fullIgnorePrefix := ignorePrefix + " " + id
	for _, line := range stringutil.SplitTrimLinesNoEmpty(leadingComments) {
		if strings.HasPrefix(line, fullIgnorePrefix) {
			return true
		}
	}
	return false
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
