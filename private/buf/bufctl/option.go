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

import "github.com/bufbuild/buf/private/buf/buffetch"

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

// TODO: split up to per-function.
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

func WithProtoFileInfosIncludeImports(protoFileInfosIncludeImports bool) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.protoFileInfosIncludeImports = protoFileInfosIncludeImports
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

// *** PRIVATE ***

type functionOptions struct {
	targetPaths                  []string
	targetExcludePaths           []string
	imageExcludeSourceInfo       bool
	imageExcludeImports          bool
	imageTypes                   []string
	imageAsFileDescriptorSet     bool
	protoFileInfosIncludeImports bool
	configOverride               string
}

func newFunctionOptions() *functionOptions {
	return &functionOptions{}
}

func (f *functionOptions) withPathsForBucketExtender(
	bucketExtender buffetch.BucketExtender,
) (*functionOptions, error) {
	deref := *f
	c := &deref
	for i, inputTargetPath := range c.targetPaths {
		targetPath, err := bucketExtender.PathForExternalPath(inputTargetPath)
		if err != nil {
			return nil, err
		}
		c.targetPaths[i] = targetPath
	}
	for i, inputTargetExcludePath := range c.targetExcludePaths {
		targetExcludePath, err := bucketExtender.PathForExternalPath(inputTargetExcludePath)
		if err != nil {
			return nil, err
		}
		c.targetExcludePaths[i] = targetExcludePath
	}
	return c, nil
}

func (f *functionOptions) getGetBucketOptions() []buffetch.GetBucketOption {
	if f.configOverride != "" {
		// If we have a config override, we do not search for buf.yamls or buf.work.yamls,
		// instead acting as if the config override was the only configuration file available.
		//
		// Note that this is slightly different behavior than the pre-refactor CLI had, but this
		// was always the intended behavior. The pre-refactor CLI would error if you had a buf.work.yaml,
		// and did the same search behavior for buf.yamls, which didn't really make sense. In the new
		// world where buf.yamls also represent the behavior of buf.work.yamls, you should be able
		// to specify whatever want here.
		return []buffetch.GetBucketOption{
			buffetch.GetBucketWithNoSearch(),
		}
	}
	return nil
}
