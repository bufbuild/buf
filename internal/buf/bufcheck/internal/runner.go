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

	"github.com/bufbuild/buf/internal/buf/ext/extfile"
	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/protodesc"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/bufbuild/buf/internal/pkg/util/utillog"
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
func (r *Runner) Check(ctx context.Context, config *Config, previousFiles []protodesc.File, files []protodesc.File) ([]*filev1beta1.FileAnnotation, error) {
	checkers := config.Checkers
	if len(checkers) == 0 {
		return nil, nil
	}
	defer utillog.Defer(r.logger, "check", zap.Int("num_files", len(files)), zap.Int("num_checkers", len(checkers)))()

	var fileAnnotations []*filev1beta1.FileAnnotation
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
		extfile.SortFileAnnotations(fileAnnotations)
		return fileAnnotations, nil
	}

	filteredFileAnnotations := make([]*filev1beta1.FileAnnotation, 0, len(fileAnnotations))
	for _, fileAnnotation := range fileAnnotations {
		if !shouldIgnoreFileAnnotation(fileAnnotation, config.IgnoreRootPaths, config.IgnoreIDToRootPaths) {
			filteredFileAnnotations = append(filteredFileAnnotations, fileAnnotation)
		}
	}
	extfile.SortFileAnnotations(filteredFileAnnotations)
	return filteredFileAnnotations, nil
}

func shouldIgnoreFileAnnotation(fileAnnotation *filev1beta1.FileAnnotation, ignoreAllRootPaths map[string]struct{}, ignoreIDToRootPaths map[string]map[string]struct{}) bool {
	if fileAnnotation.Path == "" {
		return false
	}
	if storagepath.MapContainsMatch(ignoreAllRootPaths, fileAnnotation.Path) {
		return true
	}
	if fileAnnotation.Type == "" {
		return false
	}
	ignoreRootPaths, ok := ignoreIDToRootPaths[fileAnnotation.Type]
	if !ok {
		return false
	}
	return storagepath.MapContainsMatch(ignoreRootPaths, fileAnnotation.Path)
}

type result struct {
	FileAnnotations []*filev1beta1.FileAnnotation
	Err             error
}

func newResult(fileAnnotations []*filev1beta1.FileAnnotation, err error) *result {
	return &result{
		FileAnnotations: fileAnnotations,
		Err:             err,
	}
}
