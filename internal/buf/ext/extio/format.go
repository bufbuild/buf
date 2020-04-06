package extio

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	// FormatDir is a format.
	formatDir format = 1
	// formatTar is a format.
	formatTar format = 2
	// formatTarGz is a format.
	formatTarGz format = 3
	// formatGit is a format.
	formatGit format = 4
	// formatBin is a format.
	formatBin format = 5
	// formatBinGz is a format.
	formatBinGz format = 6
	// formatJSON is a format.
	formatJSON format = 7
	// formatJSONGz is a format.
	formatJSONGz format = 8
)

var (
	formatToString = map[format]string{
		formatDir:    "dir",
		formatTar:    "tar",
		formatTarGz:  "targz",
		formatGit:    "git",
		formatBin:    "bin",
		formatBinGz:  "bingz",
		formatJSON:   "json",
		formatJSONGz: "jsongz",
	}
	stringToFormat = map[string]format{
		"dir":    formatDir,
		"tar":    formatTar,
		"targz":  formatTarGz,
		"git":    formatGit,
		"bin":    formatBin,
		"bingz":  formatBinGz,
		"json":   formatJSON,
		"jsongz": formatJSONGz,
	}

	formatToIsSource = map[format]struct{}{
		formatDir:   {},
		formatTar:   {},
		formatTarGz: {},
		formatGit:   {},
	}
	formatToIsImage = map[format]struct{}{
		formatBin:    {},
		formatBinGz:  {},
		formatJSON:   {},
		formatJSONGz: {},
	}
)

// format is a format.
type format int

// String returns the string value of f.
func (f format) String() string {
	s, ok := formatToString[f]
	if !ok {
		return strconv.Itoa(int(f))
	}
	return s
}

// inputFormats returns a new slice that contains all the input formats.
func inputFormats() []format {
	formats := make([]format, 0, len(formatToString))
	for format := range formatToString {
		formats = append(formats, format)
	}
	sortFormats(formats)
	return formats
}

// sourceFormats returns a new slice that contains all the source formats.
func sourceFormats() []format {
	formats := make([]format, 0, len(formatToIsSource))
	for format := range formatToIsSource {
		formats = append(formats, format)
	}
	sortFormats(formats)
	return formats
}

// imageFormats returns a new slice that contains all the image formats.
func imageFormats() []format {
	formats := make([]format, 0, len(formatToIsImage))
	for format := range formatToIsImage {
		formats = append(formats, format)
	}
	sortFormats(formats)
	return formats
}

// formatsToString prints the string representation of the input formats.
func formatsToString(formats []format) string {
	if len(formats) == 0 {
		return ""
	}
	values := make([]string, len(formats))
	for i, format := range formats {
		values[i] = format.String()
	}
	return "[" + strings.Join(values, ",") + "]"
}

// parseFormat parses the format.
func parseFormat(formatString string) (format, error) {
	value, ok := stringToFormat[strings.ToLower(strings.TrimSpace(formatString))]
	if !ok {
		return 0, newFormatUnknownError(formatString)
	}
	return value, nil
}

func sortFormats(formats []format) {
	sort.Slice(formats, func(i int, j int) bool { return formats[i].String() < formats[j].String() })
}

func newFormatUnknownError(formatString string) error {
	return fmt.Errorf("unknown format: %q", formatString)
}
