package internal

import (
	"sort"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/errs"
)

const (
	// FormatDir is a format.
	FormatDir Format = 1
	// FormatTar is a format.
	FormatTar Format = 2
	// FormatTarGz is a format.
	FormatTarGz Format = 3
	// FormatGit is a format.
	FormatGit Format = 4
	// FormatBin is a format.
	FormatBin Format = 5
	// FormatBinGz is a format.
	FormatBinGz Format = 6
	// FormatJSON is a format.
	FormatJSON Format = 7
	// FormatJSONGz is a format.
	FormatJSONGz Format = 8
)

var (
	formatToString = map[Format]string{
		FormatDir:    "dir",
		FormatTar:    "tar",
		FormatTarGz:  "targz",
		FormatGit:    "git",
		FormatBin:    "bin",
		FormatBinGz:  "bingz",
		FormatJSON:   "json",
		FormatJSONGz: "jsongz",
	}
	stringToFormat = map[string]Format{
		"dir":    FormatDir,
		"tar":    FormatTar,
		"targz":  FormatTarGz,
		"git":    FormatGit,
		"bin":    FormatBin,
		"bingz":  FormatBinGz,
		"json":   FormatJSON,
		"jsongz": FormatJSONGz,
	}

	formatToIsSource = map[Format]struct{}{
		FormatDir:   struct{}{},
		FormatTar:   struct{}{},
		FormatTarGz: struct{}{},
		FormatGit:   struct{}{},
	}
	formatToIsImage = map[Format]struct{}{
		FormatBin:    struct{}{},
		FormatBinGz:  struct{}{},
		FormatJSON:   struct{}{},
		FormatJSONGz: struct{}{},
	}
	formatToIsFile = map[Format]struct{}{
		FormatTar:    struct{}{},
		FormatTarGz:  struct{}{},
		FormatBin:    struct{}{},
		FormatBinGz:  struct{}{},
		FormatJSON:   struct{}{},
		FormatJSONGz: struct{}{},
	}
)

// Format is a format.
type Format int

// String returns the string value of f.
func (f Format) String() string {
	s, ok := formatToString[f]
	if !ok {
		return strconv.Itoa(int(f))
	}
	return s
}

// IsSource returns true if f represents a source type.
func (f Format) IsSource() bool {
	_, ok := formatToIsSource[f]
	return ok
}

// IsImage returns true if f represents a image type.
//
// Images are alway files
func (f Format) IsImage() bool {
	_, ok := formatToIsImage[f]
	return ok
}

// isFile returns true if f represents a file type.
func (f Format) isFile() bool {
	_, ok := formatToIsFile[f]
	return ok
}

// AllFormatsToString returns all format strings.
func AllFormatsToString() string {
	return formatsToString(allFormats())
}

// SourceFormatsToString returns source format strings.
func SourceFormatsToString() string {
	return formatsToString(sourceFormats())
}

// ImageFormatsToString returns image format strings.
func ImageFormatsToString() string {
	return formatsToString(imageFormats())
}

// allFormats returns a new slice that contains all the formats.
func allFormats() []Format {
	formats := make([]Format, 0, len(formatToString))
	for format := range formatToString {
		formats = append(formats, format)
	}
	sortFormats(formats)
	return formats
}

// sourceFormats returns a new slice that contains all the source formats.
func sourceFormats() []Format {
	formats := make([]Format, 0, len(formatToIsSource))
	for format := range formatToIsSource {
		formats = append(formats, format)
	}
	sortFormats(formats)
	return formats
}

// imageFormats returns a new slice that contains all the image formats.
func imageFormats() []Format {
	formats := make([]Format, 0, len(formatToIsImage))
	for format := range formatToIsImage {
		formats = append(formats, format)
	}
	sortFormats(formats)
	return formats
}

// fileFormats returns a new slice that contains all the file formats.
func fileFormats() []Format {
	formats := make([]Format, 0, len(formatToIsFile))
	for format := range formatToIsFile {
		formats = append(formats, format)
	}
	sortFormats(formats)
	return formats
}

// formatsToString prints the string representation of the input formats.
func formatsToString(formats []Format) string {
	if len(formats) == 0 {
		return ""
	}
	values := make([]string, len(formats))
	for i, format := range formats {
		values[i] = format.String()
	}
	return "[" + strings.Join(values, ",") + "]"
}

// parseFormatOverride parses the format.
func parseFormatOverride(valueFlagName string, formatOverride string) (Format, error) {
	value, ok := stringToFormat[strings.ToLower(strings.TrimSpace(formatOverride))]
	if !ok {
		return 0, newFormatOverrideUnknownError(valueFlagName, formatOverride)
	}
	return value, nil
}

func sortFormats(formats []Format) {
	sort.Slice(formats, func(i int, j int) bool { return formats[i].String() < formats[j].String() })
}

func newFormatOverrideUnknownError(formatOverrideFlagName string, formatOverride string) error {
	return errs.NewUserErrorf("%s: unknown format: %q", formatOverrideFlagName, formatOverride)
}
