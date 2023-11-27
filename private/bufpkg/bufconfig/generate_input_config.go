// GenerateInputConfig is an input configuration.
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

package bufconfig

type InputConfigType int

const (
	InputConfigTypeModule InputConfigType = iota + 1
	InputConfigTypeDirectory
	InputConfigTypeGitRepo
	InputConfigTypeProtoFile
	InputConfigTypeTarball
	InputConfigTypeZipArchive
	InputConfigTypeBinaryImage
	InputConfigTypeJSONImage
	InputConfigTypeTextImage
)

// GenerateInputConfig is an input configuration for code generation.
type GenerateInputConfig interface {
	// Type returns the input type.
	Type() InputConfigType
	// Location returns the location for the input.
	Location() string
	// Compression returns the compression scheme, not empty only if format is
	// one of tarball, binary image, json image or text image.
	Compression() string
	// StripComponents returns the number of directories to strip for tar or zip
	// inputs, not empty only if format is tarball or zip archive.
	StripComponents() *uint32
	// Subdir returns the subdirectory to use, not empty only if format is one
	// git repo, tarball and zip archive.
	Subdir() string
	// Branch returns the git branch to checkout out, not empty only if format is git.
	Branch() string
	// Tag returns the git tag to checkout, not empty only if format is git.
	Tag() string
	// Ref returns the git ref to checkout, not empty only if format is git.
	Ref() string
	// Ref returns the depth to clone the git repo with, not empty only if format is git.
	Depth() *uint32
	// RecurseSubmodules returns whether to clone submodules recursively. Not empty
	// only if input if git.
	RecurseSubmodules() bool
	// IncludePackageFiles returns other files in the same package as the proto file,
	// not empty only if format is proto file.
	IncludePackageFiles() bool
	// IncludeTypes returns the types to generate. If GenerateConfig.GenerateTypeConfig()
	// returns a non-empty list of types.
	IncludeTypes() []string
	// ExcludePaths returns paths not to generate for.
	ExcludePaths() []string
	// IncludePaths returns paths to generate for.
	IncludePaths() []string

	isGenerateInputConfig()
}
