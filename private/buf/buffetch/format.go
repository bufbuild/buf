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

package buffetch

const (
	// FormatBinpb is the protobuf binary format.
	FormatBinpb = "binpb"
	// FormatTxtpb is the protobuf text format.
	FormatTxtpb = "txtpb"
	// FormatDir is the directory format.
	FormatDir = "dir"
	// FormatGit is the git format.
	FormatGit = "git"
	// FormatJSON is the JSON format.
	FormatJSON = "json"
	// FormatMod is the module format.
	FormatMod = "mod"
	// FormatTar is the tar format.
	FormatTar = "tar"
	// FormatZip is the zip format.
	FormatZip = "zip"
	// FormatProtoFile is the proto file format.
	FormatProtoFile = "protofile"

	// FormatBin is the binary format's old form, now deprecated.
	FormatBin = "bin"
	// FormatBingz is the binary gzipped format, now deprecated.
	FormatBingz = "bingz"
	// FormatJSONGZ is the JSON gzipped format, now deprecated.
	FormatJSONGZ = "jsongz"
	// FormatTargz is the tar gzipped format, now deprecated.
	FormatTargz = "targz"
)

var (
	// sorted
	imageFormats = []string{
		FormatBin,
		FormatBinpb,
		FormatBingz,
		FormatJSON,
		FormatJSONGZ,
		FormatTxtpb,
	}
	// sorted
	imageFormatsNotDeprecated = []string{
		FormatBinpb,
		FormatJSON,
		FormatTxtpb,
	}
	// sorted
	sourceFormats = []string{
		FormatDir,
		FormatGit,
		FormatProtoFile,
		FormatTar,
		FormatTargz,
		FormatZip,
	}
	// sorted
	sourceFormatsNotDeprecated = []string{
		FormatDir,
		FormatGit,
		FormatProtoFile,
		FormatTar,
		FormatZip,
	}
	sourceDirFormatsNotDeprecated = []string{
		FormatDir,
		FormatGit,
		FormatTar,
		FormatZip,
	}
	// sorted
	moduleFormats = []string{
		FormatMod,
	}
	// sorted
	moduleFormatsNotDeprecated = []string{
		FormatMod,
	}
	// sorted
	sourceOrModuleFormats = []string{
		FormatDir,
		FormatGit,
		FormatMod,
		FormatProtoFile,
		FormatTar,
		FormatTargz,
		FormatZip,
	}
	// sorted
	sourceOrModuleFormatsNotDeprecated = []string{
		FormatDir,
		FormatGit,
		FormatMod,
		FormatProtoFile,
		FormatTar,
		FormatZip,
	}
	// sorted
	allFormats = []string{
		FormatBin,
		FormatBinpb,
		FormatBingz,
		FormatDir,
		FormatGit,
		FormatJSON,
		FormatJSONGZ,
		FormatMod,
		FormatProtoFile,
		FormatTar,
		FormatTargz,
		FormatTxtpb,
		FormatZip,
	}
	// sorted
	allFormatsNotDeprecated = []string{
		FormatBinpb,
		FormatDir,
		FormatGit,
		FormatJSON,
		FormatMod,
		FormatProtoFile,
		FormatTar,
		FormatTxtpb,
		FormatZip,
	}

	deprecatedCompressionFormatToReplacementFormat = map[string]string{
		FormatBingz:  FormatBinpb,
		FormatJSONGZ: FormatJSON,
		FormatTargz:  FormatTar,
	}
)
