package internal

import (
	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
)

// InputRef is a parsed input reference.
type InputRef struct {
	// Format is the format of the input.
	// Required.
	Format Format
	// Path is the path of the input.
	// The special value "-" indicates stdin or stdout.
	// If this is "-", Format == FormatTar, FormatTarGz, FormatBin, FormatBinGz, FormatJSON, FormatJSONGz.
	// Required.
	Path string

	// GitBranch is the branch of the git repository.
	// This will only be set if Format == FormatGit.
	// Optional regardless.
	GitBranch string
	// StripComponents is the number of components to strip from a tarball.
	// This will only be set if Format == FormatTar, FormatTarGz
	StripComponents uint32
}

// InputRefParser parses InputRefs.
type InputRefParser interface {
	// ParseInputRef parses the InputRef from the value.
	//
	// Value should always be non-empty - if you want this to be ".", specify it.
	// If onlySources is true, the Format will only be FormatDir, FormatTar, FormatTarGz, FormatGit.
	// If onlyImages is true, the Format will only be FormatBin, FormatBinGz, FormatJSON, FormatJSONGz.
	// If onlySources and onlyImages is true, this returns system error.
	// Format will be valid and only one of these eight types.
	ParseInputRef(value string, onlySources bool, onlyImages bool) (*InputRef, error)
}

// NewInputRefParser returns a new InputRefParser.
func NewInputRefParser(valueFlagName string) InputRefParser {
	return newInputRefParser(valueFlagName)
}

// ConfigOverrideParser parses config overrides.
type ConfigOverrideParser interface {
	// ParseConfigOverride parses the config override.
	//
	// If the trimmed input is empty, this returns system error.
	ParseConfigOverride(value string) (*bufconfig.Config, error)
}

// NewConfigOverrideParser returns a new ConfigOverrideParser.
func NewConfigOverrideParser(
	configProvider bufconfig.Provider,
	configOverrideFlagName string,
) ConfigOverrideParser {
	return newConfigOverrideParser(
		configProvider,
		configOverrideFlagName,
	)
}

// NewRelProtoFilePathResolver returns a new ProtoFilePathResolver that will:
//
// - Apply the chained resolver, if it is not nil.
// - Add the dirPath as a prefix.
// - Make the path relative to pwd if the path is relative, or return the path if it is absolute.
func NewRelProtoFilePathResolver(
	dirPath string,
	chainedResolver bufbuild.ProtoRealFilePathResolver,
) (bufbuild.ProtoRealFilePathResolver, error) {
	return newRelProtoFilePathResolver(
		dirPath,
		chainedResolver,
	)
}
