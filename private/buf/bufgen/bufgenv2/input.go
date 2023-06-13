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

package bufgenv2

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/buf/buffetch"
)

const (
	formatGitRepo     = "git_repo"
	formatModule      = "module"
	formatDirectory   = "directory"
	formatProtoFile   = "proto_file"
	formatBinaryImage = "binary_image"
	formatTarball     = "tarball"
	formatZipArchive  = "zip_archive"
	formatJSONImage   = "json_image"
)

const (
	optionCompression         = "compression"
	optionBranch              = "branch"
	optionTag                 = "tag"
	optionRef                 = "ref"
	optionDepth               = "depth"
	optionRecurseSubmodules   = "recurse_submodules"
	optionStripComponents     = "strip_components"
	optionSubdir              = "subdir"
	optionIncludePackageFiles = "include_package_files"
)

var allowedOptionsForFormat = map[string](map[string]bool){
	formatGitRepo: {
		optionBranch:            true,
		optionTag:               true,
		optionRef:               true,
		optionDepth:             true,
		optionRecurseSubmodules: true,
		optionSubdir:            true,
	},
	formatModule:    {},
	formatDirectory: {},
	formatProtoFile: {
		optionIncludePackageFiles: true,
	},
	formatTarball: {
		optionCompression:     true,
		optionStripComponents: true,
		optionSubdir:          true,
	},
	formatZipArchive: {
		optionStripComponents: true,
		optionSubdir:          true,
	},
	formatBinaryImage: {
		optionCompression: true,
	},
	formatJSONImage: {
		optionCompression: true,
	},
}

func newInputConfig(ctx context.Context, externalConfig ExternalInputConfigV2) (*InputConfig, error) {
	formatsSpecified, optionsSpecified := getFormatsAndOptionsSpecified(externalConfig)
	if len(formatsSpecified) == 0 {
		return nil, errors.New("must specify input type")
	}
	if len(formatsSpecified) > 1 {
		return nil, fmt.Errorf("each input can only have one format, already have format %s", formatsSpecified[0])
	}
	format := formatsSpecified[0]
	allowedOptions, ok := allowedOptionsForFormat[format]
	if !ok {
		// this should not happen
		return nil, fmt.Errorf("unable to find allowed options for format %s", format)
	}
	for _, optionSet := range optionsSpecified {
		if !allowedOptions[optionSet] {
			return nil, fmt.Errorf("option %s is not allowed for format %s", optionSet, format)
		}
	}
	inputConfig := InputConfig{
		Types:        externalConfig.Types,
		IncludePaths: externalConfig.IncludePaths,
		ExcludePaths: externalConfig.ExcludePaths,
	}
	refBuilder := buffetch.NewRefBuilder()
	var err error
	switch format {
	case formatGitRepo:
		var options []buffetch.GetGitRefOption
		if branch := externalConfig.Branch; branch != nil {
			options = append(options, buffetch.WithGetGitRefBranch(*branch))
		}
		if tag := externalConfig.Tag; tag != nil {
			options = append(options, buffetch.WithGetGitRefTag(*tag))
		}
		if ref := externalConfig.Ref; ref != nil {
			options = append(options, buffetch.WithGetGitRefRef(*ref))
		}
		if depth := externalConfig.Depth; depth != nil {
			options = append(options, buffetch.WithGetGitRefDepth(*depth))
		}
		if recurseSubmodules := externalConfig.RecurseSubmodules; recurseSubmodules != nil && *recurseSubmodules {
			options = append(options, buffetch.WithGetGitRefRecurseSubmodules())
		}
		if subDir := externalConfig.Subdir; subDir != nil {
			options = append(options, buffetch.WithGetGitRefSubDir(*subDir))
		}
		inputConfig.InputRef, err = refBuilder.GetGitRef(
			ctx,
			formatGitRepo,
			*externalConfig.GitRepo,
			options...,
		)
	case formatModule:
		inputConfig.InputRef, err = refBuilder.GetModuleRef(
			ctx,
			formatModule,
			*externalConfig.Module,
		)
	case formatDirectory:
		inputConfig.InputRef, err = refBuilder.GetDirRef(
			ctx,
			formatDirectory,
			*externalConfig.Directory,
		)
	case formatProtoFile:
		var options []buffetch.GetProtoFileRefOption
		if externalConfig.IncludePackageFiles != nil && *externalConfig.IncludePackageFiles {
			options = append(options, buffetch.WithGetProtoFileRefIncludePackageFiles())
		}
		inputConfig.InputRef, err = refBuilder.GetProtoFileRef(
			ctx,
			formatProtoFile,
			*externalConfig.ProtoFile,
			options...,
		)
	case formatTarball:
		var options []buffetch.GetTarballRefOption
		if compression := externalConfig.Compression; compression != nil {
			options = append(options, buffetch.WithGetTarballRefCompression(*compression))
		}
		if stripComponents := externalConfig.StripComponents; stripComponents != nil {
			options = append(options, buffetch.WithGetTarballRefStripComponents(*stripComponents))
		}
		if subDir := externalConfig.Subdir; subDir != nil {
			options = append(options, buffetch.WithGetTarballRefSubDir(*subDir))
		}
		inputConfig.InputRef, err = refBuilder.GetTarballRef(
			ctx,
			formatTarball,
			*externalConfig.Tarball,
			options...,
		)
	case formatZipArchive:
		var options []buffetch.GetZipArchiveRefOption
		if stripComponents := externalConfig.StripComponents; stripComponents != nil {
			options = append(options, buffetch.WithGetZipArchiveRefStripComponents(*stripComponents))
		}
		if subDir := externalConfig.Subdir; subDir != nil {
			options = append(options, buffetch.WithGetZipArchiveRefSubDir(*subDir))
		}
		inputConfig.InputRef, err = refBuilder.GetZipArchiveRef(
			ctx,
			formatZipArchive,
			*externalConfig.ZipArchive,
			options...,
		)
	case formatBinaryImage:
		var options []buffetch.GetImageRefOption
		if compression := externalConfig.Compression; compression != nil {
			options = append(options, buffetch.WithGetImageRefOption(*compression))
		}
		inputConfig.InputRef, err = refBuilder.GetBinaryImageRef(
			ctx,
			formatBinaryImage,
			*externalConfig.BinaryImage,
			options...,
		)
	case formatJSONImage:
		var options []buffetch.GetImageRefOption
		if compression := externalConfig.Compression; compression != nil {
			options = append(options, buffetch.WithGetImageRefOption(*compression))
		}
		inputConfig.InputRef, err = refBuilder.GetJSONImageRef(
			ctx,
			formatJSONImage,
			*externalConfig.JSONImage,
			options...,
		)
	default:
		// this should not happen
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
	if err != nil {
		return nil, err
	}
	return &inputConfig, nil
}

func getFormatsAndOptionsSpecified(externalConfig ExternalInputConfigV2) ([]string, []string) {
	var formats []string
	var options []string
	if externalConfig.Module != nil {
		formats = append(formats, formatModule)
	}
	if externalConfig.Directory != nil {
		formats = append(formats, formatDirectory)
	}
	if externalConfig.ProtoFile != nil {
		formats = append(formats, formatProtoFile)
	}
	if externalConfig.BinaryImage != nil {
		formats = append(formats, formatBinaryImage)
	}
	if externalConfig.Tarball != nil {
		formats = append(formats, formatTarball)
	}
	if externalConfig.ZipArchive != nil {
		formats = append(formats, formatZipArchive)
	}
	if externalConfig.JSONImage != nil {
		formats = append(formats, formatJSONImage)
	}
	if externalConfig.GitRepo != nil {
		formats = append(formats, formatGitRepo)
	}

	if externalConfig.Compression != nil {
		options = append(options, optionCompression)
	}
	if externalConfig.StripComponents != nil {
		options = append(options, optionStripComponents)
	}
	if externalConfig.Subdir != nil {
		options = append(options, optionSubdir)
	}
	if externalConfig.Branch != nil {
		options = append(options, optionBranch)
	}
	if externalConfig.Tag != nil {
		options = append(options, optionTag)
	}
	if externalConfig.Ref != nil {
		options = append(options, optionRef)
	}
	if externalConfig.Depth != nil {
		options = append(options, optionDepth)
	}
	if externalConfig.RecurseSubmodules != nil {
		options = append(options, optionRecurseSubmodules)
	}
	if externalConfig.IncludePackageFiles != nil {
		options = append(options, optionIncludePackageFiles)
	}
	return formats, options
}
