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

package bufctl

import (
	"github.com/bufbuild/buf/private/buf/buffetch"
)

type ControllerOption func(*controller)

func WithDisableSymlinks(disableSymlinks bool) ControllerOption {
	return func(controller *controller) {
		controller.disableSymlinks = disableSymlinks
	}
}

func WithFileAnnotationErrorFormat(fileAnnotationErrorFormat string) ControllerOption {
	return func(controller *controller) {
		controller.fileAnnotationErrorFormat = fileAnnotationErrorFormat
	}
}

func WithFileAnnotationsToStdout() ControllerOption {
	return func(controller *controller) {
		controller.fileAnnotationsToStdout = true
	}
}

func WithCopyToInMemory() ControllerOption {
	return func(controller *controller) {
		controller.copyToInMemory = true
	}
}

// TODO FUTURE: split up to per-function.
type FunctionOption func(*functionOptions)

func WithTargetPaths(targetPaths []string, targetExcludePaths []string) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.targetPaths = targetPaths
		functionOptions.targetExcludePaths = targetExcludePaths
	}
}

func WithImageExcludeSourceInfo(imageExcludeSourceInfo bool) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.imageExcludeSourceInfo = imageExcludeSourceInfo
	}
}

func WithImageExcludeImports(imageExcludeImports bool) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.imageExcludeImports = imageExcludeImports
	}
}

func WithImageTypes(imageTypes []string) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.imageTypes = imageTypes
	}
}

func WithImageAsFileDescriptorSet(imageAsFileDescriptorSet bool) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.imageAsFileDescriptorSet = imageAsFileDescriptorSet
	}
}

// WithConfigOverride applies the config override.
//
// This flag will only work if no buf.work.yaml is detected, and the buf.yaml is a
// v1beta1 buf.yaml, v1 buf.yaml, or no buf.yaml. This flag will not work if a buf.work.yaml
// is detected, or a v2 buf.yaml is detected.
//
// If used with an image or module ref, this has no effect on the build, i.e. excludes are
// not respected, and the module name is ignored. This matches old behavior.
//
// This implements the soon-to-be-deprected --config flag.
//
// See bufconfig.GetBufYAMLFileForPrefixOrOverride for more details.
//
// *** DO NOT USE THIS OUTSIDE OF THE CLI AND/OR IF YOU DON'T UNDERSTAND IT. ***
// *** DO NOT ADD THIS TO ANY NEW COMMANDS. ***
//
// Current commands that use this: build, breaking, lint, generate, format,
// export, ls-breaking-rules, ls-lint-rules.
func WithConfigOverride(configOverride string) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.configOverride = configOverride
	}
}

// WithIgnoreAndDisallowV1BufWorkYAMLs returns a new FunctionOption that says
// to ignore dependencies from buf.work.yamls at the root of the bucket, and to also
// disallow directories with buf.work.yamls to be directly targeted.
//
// See bufworkspace.WithIgnoreAndDisallowV1BufWorkYAMLs for more details.
func WithIgnoreAndDisallowV1BufWorkYAMLs() FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.ignoreAndDisallowV1BufWorkYAMLs = true
	}
}

// WithMessageValidation returns a new FunctionOption that says to validate the
// message as it is being read.
//
// We want to do this as part of the read/unmarshal, as protoyaml has specific logic
// on unmarshal that will pretty print validations.
func WithMessageValidation() FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.messageValidation = true
	}
}

// *** PRIVATE ***

type functionOptions struct {
	copyToInMemory bool

	targetPaths                     []string
	targetExcludePaths              []string
	imageExcludeSourceInfo          bool
	imageExcludeImports             bool
	imageTypes                      []string
	imageAsFileDescriptorSet        bool
	configOverride                  string
	ignoreAndDisallowV1BufWorkYAMLs bool
	messageValidation               bool
}

func newFunctionOptions(controller *controller) *functionOptions {
	return &functionOptions{
		copyToInMemory: controller.copyToInMemory,
	}
}

func (f *functionOptions) getGetReadBucketCloserOptions() []buffetch.GetReadBucketCloserOption {
	var getReadBucketCloserOptions []buffetch.GetReadBucketCloserOption
	if f.copyToInMemory {
		getReadBucketCloserOptions = append(
			getReadBucketCloserOptions,
			buffetch.GetReadBucketCloserWithCopyToInMemory(),
		)
	}
	if len(f.targetPaths) > 0 {
		getReadBucketCloserOptions = append(
			getReadBucketCloserOptions,
			buffetch.GetReadBucketCloserWithTargetPaths(f.targetPaths),
		)
	}
	if len(f.targetExcludePaths) > 0 {
		getReadBucketCloserOptions = append(
			getReadBucketCloserOptions,
			buffetch.GetReadBucketCloserWithTargetExcludePaths(f.targetExcludePaths),
		)
	}
	if f.configOverride != "" {
		// If we have a config override, we do not search for buf.yamls or buf.work.yamls,
		// instead acting as if the config override was the only configuration file available.
		//
		// Note that this is slightly different behavior than the pre-refactor CLI had, but this
		// was always the intended behavior. The pre-refactor CLI would error if you had a buf.work.yaml,
		// and did the same search behavior for buf.yamls, which didn't really make sense. In the new
		// world where buf.yamls also represent the behavior of buf.work.yamls, you should be able
		// to specify whatever want here.
		getReadBucketCloserOptions = append(
			getReadBucketCloserOptions,
			buffetch.GetReadBucketCloserWithNoSearch(),
		)
	}
	return getReadBucketCloserOptions
}

func (f *functionOptions) getGetReadWriteBucketOptions() []buffetch.GetReadWriteBucketOption {
	if f.configOverride != "" {
		// If we have a config override, we do not search for buf.yamls or buf.work.yamls,
		// instead acting as if the config override was the only configuration file available.
		//
		// Note that this is slightly different behavior than the pre-refactor CLI had, but this
		// was always the intended behavior. The pre-refactor CLI would error if you had a buf.work.yaml,
		// and did the same search behavior for buf.yamls, which didn't really make sense. In the new
		// world where buf.yamls also represent the behavior of buf.work.yamls, you should be able
		// to specify whatever want here.
		return []buffetch.GetReadWriteBucketOption{
			buffetch.GetReadWriteBucketWithNoSearch(),
		}
	}
	return nil
}
