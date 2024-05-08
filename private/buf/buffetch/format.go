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

package buffetch

const (
	// formatBinpb is the protobuf binary format.
	formatBinpb = "binpb"
	// formatTxtpb is the protobuf text format.
	formatTxtpb = "txtpb"
	// formatDir is the directory format.
	formatDir = "dir"
	// formatGit is the git format.
	formatGit = "git"
	// formatJSON is the JSON format.
	formatJSON = "json"
	// formatYAML is the YAML format.
	formatYAML = "yaml"
	// formatMod is the module format.
	formatMod = "mod"
	// formatTar is the tar format.
	formatTar = "tar"
	// formatZip is the zip format.
	formatZip = "zip"
	// formatProtoFile is the proto file format.
	formatProtoFile = "protofile"

	// formatBin is the binary format's old form, now deprecated.
	formatBin = "bin"
	// formatBingz is the binary gzipped format, now deprecated.
	formatBingz = "bingz"
	// formatJSONGZ is the JSON gzipped format, now deprecated.
	formatJSONGZ = "jsongz"
	// formatTargz is the tar gzipped format, now deprecated.
	formatTargz = "targz"
)

var (
	// sorted
	messageFormats = []string{
		formatBin,
		formatBinpb,
		formatBingz,
		formatJSON,
		formatJSONGZ,
		formatTxtpb,
		formatYAML,
	}
	// sorted
	messageFormatsNotDeprecated = []string{
		formatBinpb,
		formatJSON,
		formatTxtpb,
		formatYAML,
	}
	// sorted
	sourceFormats = []string{
		formatDir,
		formatGit,
		formatProtoFile,
		formatTar,
		formatTargz,
		formatZip,
	}
	// sorted
	sourceFormatsNotDeprecated = []string{
		formatDir,
		formatGit,
		formatProtoFile,
		formatTar,
		formatZip,
	}
	sourceDirFormatsNotDeprecated = []string{
		formatDir,
		formatGit,
		formatTar,
		formatZip,
	}
	// sorted
	dirFormats = []string{
		formatDir,
	}
	// sorted
	dirOrProtoFileFormats = []string{
		formatDir,
		formatProtoFile,
	}
	// sorted
	moduleFormats = []string{
		formatMod,
	}
	// sorted
	moduleFormatsNotDeprecated = []string{
		formatMod,
	}
	// sorted
	sourceOrModuleFormats = []string{
		formatDir,
		formatGit,
		formatMod,
		formatProtoFile,
		formatTar,
		formatTargz,
		formatZip,
	}
	// sorted
	sourceOrModuleFormatsNotDeprecated = []string{
		formatDir,
		formatGit,
		formatMod,
		formatProtoFile,
		formatTar,
		formatZip,
	}
	// sorted
	allFormats = []string{
		formatBin,
		formatBinpb,
		formatBingz,
		formatDir,
		formatGit,
		formatJSON,
		formatJSONGZ,
		formatMod,
		formatProtoFile,
		formatTar,
		formatTargz,
		formatTxtpb,
		formatYAML,
		formatZip,
	}
	// sorted
	allFormatsNotDeprecated = []string{
		formatBinpb,
		formatDir,
		formatGit,
		formatJSON,
		formatMod,
		formatProtoFile,
		formatTar,
		formatTxtpb,
		formatYAML,
		formatZip,
	}

	deprecatedCompressionFormatToReplacementFormat = map[string]string{
		formatBingz:  formatBinpb,
		formatJSONGZ: formatJSON,
		formatTargz:  formatTar,
	}

	messageEncodingToFormat = map[MessageEncoding]string{
		MessageEncodingBinpb: formatBinpb,
		MessageEncodingJSON:  formatJSON,
		MessageEncodingTxtpb: formatTxtpb,
		MessageEncodingYAML:  formatYAML,
	}
)
