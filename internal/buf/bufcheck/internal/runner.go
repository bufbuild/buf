// Copyright 2020 Buf Technologies Inc.
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

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufsrc"
	"github.com/bufbuild/buf/internal/pkg/instrument"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// Runner is a runner.
type Runner struct {
	logger *zap.Logger
}

// NewRunner returns a new Runner.
func NewRunner(logger *zap.Logger) *Runner {
	return &Runner{
		logger: logger,
	}
}

// Check runs the Checkers.
func (r *Runner) Check(ctx context.Context, config *Config, previousFiles []bufsrc.File, files []bufsrc.File) ([]bufanalysis.FileAnnotation, error) {
	checkers := config.Checkers
	if len(checkers) == 0 {
		return nil, nil
	}
	defer instrument.Start(r.logger, "check", zap.Int("num_files", len(files)), zap.Int("num_checkers", len(checkers))).End()

	var fileAnnotations []bufanalysis.FileAnnotation
	resultC := make(chan *result, len(checkers))
	for _, checker := range checkers {
		checker := checker
		go func() {
			iFileAnnotations, iErr := checker.check(previousFiles, files)
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

	if len(config.IgnoreRootPaths) == 0 && len(config.IgnoreIDToRootPaths) == 0 {
		bufanalysis.SortFileAnnotations(fileAnnotations)
		return fileAnnotations, nil
	}

	filteredFileAnnotations := make([]bufanalysis.FileAnnotation, 0, len(fileAnnotations))
	for _, fileAnnotation := range fileAnnotations {
		if !shouldIgnoreFileAnnotation(fileAnnotation, config.IgnoreRootPaths, config.IgnoreIDToRootPaths) {
			filteredFileAnnotations = append(filteredFileAnnotations, fileAnnotation)
		}
	}
	bufanalysis.SortFileAnnotations(filteredFileAnnotations)
	return filteredFileAnnotations, nil
}

func shouldIgnoreFileAnnotation(fileAnnotation bufanalysis.FileAnnotation, ignoreAllRootPaths map[string]struct{}, ignoreIDToRootPaths map[string]map[string]struct{}) bool {
	fileRef := fileAnnotation.FileRef()
	if fileRef == nil {
		return false
	}
	path := fileRef.RootRelFilePath()
	if normalpath.MapContainsMatch(ignoreAllRootPaths, path) {
		return true
	}
	if fileAnnotation.Type() == "" {
		return false
	}
	ignoreRootPaths, ok := ignoreIDToRootPaths[fileAnnotation.Type()]
	if !ok {
		return false
	}
	return normalpath.MapContainsMatch(ignoreRootPaths, path)
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
