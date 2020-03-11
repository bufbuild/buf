package internal

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/cli/clios"
	"github.com/bufbuild/buf/internal/pkg/storage/storagegit/storagegitplumbing"
)

type inputRefParser struct {
	valueFlagName string
}

func newInputRefParser(valueFlagName string) *inputRefParser {
	return &inputRefParser{
		valueFlagName: valueFlagName,
	}
}

func (i *inputRefParser) ParseInputRef(value string, onlySources bool, onlyImages bool) (*InputRef, error) {
	if onlySources && onlyImages {
		return nil, errors.New("onlySources and onlyImages both set")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, newValueEmptyError(i.valueFlagName)
	}

	var path string
	var options string
	switch splitValue := strings.Split(value, "#"); len(splitValue) {
	case 1:
		path = value
	case 2:
		path = strings.TrimSpace(splitValue[0])
		options = strings.TrimSpace(splitValue[1])
		if path == "" {
			return nil, newValueStartsWithHashtagError(i.valueFlagName, value)
		}
		if options == "" {
			return nil, newValueEndsWithHashtagError(i.valueFlagName, value)
		}
	default:
		return nil, newValueMultipleHashtagsError(i.valueFlagName, value)
	}
	inputRef := &InputRef{
		Path: path,
	}
	if err := i.applyInputRefOptions(inputRef, options); err != nil {
		return nil, err
	}
	if inputRef.Format == 0 {
		format, err := i.parseFormatFromPath(path)
		if err != nil {
			return nil, err
		}
		inputRef.Format = format
	}

	if inputRef.Format == FormatGit && inputRef.GitRefName == nil {
		return nil, newMustSpecifyGitRefNameError(i.valueFlagName, value)
	}
	if inputRef.Format != FormatGit && inputRef.GitRefName != nil {
		return nil, newOptionsInvalidForFormatError(i.valueFlagName, inputRef.Format, options)
	}
	if inputRef.Format != FormatTar && inputRef.Format != FormatTarGz && inputRef.StripComponents > 0 {
		return nil, newOptionsInvalidForFormatError(i.valueFlagName, inputRef.Format, options)
	}

	if onlySources && !inputRef.Format.IsSource() {
		return nil, newFormatMustBeSourceError(inputRef.Format)
	}
	if onlyImages && !inputRef.Format.IsImage() {
		return nil, newFormatMustBeImageError(inputRef.Format)
	}
	if path == "-" && !inputRef.Format.isFile() {
		return nil, newFormatNotFileForDashPathError(i.valueFlagName, inputRef.Format)
	}

	return inputRef, nil
}

// we know that path is non-empty at this point
// we know that format override is not set at this point
func (i *inputRefParser) parseFormatFromPath(path string) (Format, error) {
	// if formatOverride is not set and path is "-", default to FormatBin
	if path == "-" || path == clios.DevNull {
		return FormatBin, nil
	}
	switch filepath.Ext(path) {
	case ".bin":
		return FormatBin, nil
	case ".json":
		return FormatJSON, nil
	case ".tar":
		return FormatTar, nil
	case ".gz":
		switch filepath.Ext(strings.TrimSuffix(path, filepath.Ext(path))) {
		case ".bin":
			return FormatBinGz, nil
		case ".json":
			return FormatJSONGz, nil
		case ".tar":
			return FormatTarGz, nil
		default:
			return 0, newPathUnknownGzError(i.valueFlagName, path)
		}
	case ".tgz":
		return FormatTarGz, nil
	case ".git":
		return FormatGit, nil
	default:
		return FormatDir, nil
	}
}

// options can be empty but if not, will already be trimmed
func (i *inputRefParser) applyInputRefOptions(inputRef *InputRef, options string) error {
	if options == "" {
		return nil
	}
	for _, pair := range strings.Split(options, ",") {
		split := strings.Split(pair, "=")
		if len(split) != 2 {
			return newOptionsInvalidError(i.valueFlagName, options)
		}
		key := strings.TrimSpace(split[0])
		value := strings.TrimSpace(split[1])
		if key == "" || value == "" {
			return newOptionsInvalidError(i.valueFlagName, options)
		}
		switch key {
		case "format":
			if inputRef.Path == clios.DevNull {
				return newFormatOverrideNotAllowedForDevNullError(i.valueFlagName, clios.DevNull)
			}
			format, err := parseFormatOverride(i.valueFlagName, value)
			if err != nil {
				return err
			}
			inputRef.Format = format
		case "branch":
			if inputRef.GitRefName != nil {
				return newCannotSpecifyMultipleGitRefNamesError(i.valueFlagName)
			}
			inputRef.GitRefName = storagegitplumbing.NewBranchRefName(value)
		case "tag":
			if inputRef.GitRefName != nil {
				return newCannotSpecifyMultipleGitRefNamesError(i.valueFlagName)
			}
			inputRef.GitRefName = storagegitplumbing.NewTagRefName(value)
		case "strip_components":
			stripComponents, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return newOptionsCouldNotParseStripComponentsError(i.valueFlagName, value)
			}
			inputRef.StripComponents = uint32(stripComponents)
		default:
			return newOptionsInvalidKeyError(i.valueFlagName, key)
		}
	}
	return nil
}

func newValueEmptyError(valueFlagName string) error {
	return fmt.Errorf("%s is required", valueFlagName)
}

func newValueMultipleHashtagsError(valueFlagName string, value string) error {
	return fmt.Errorf("%s: %q has multiple #s which is invalid", valueFlagName, value)
}

func newValueStartsWithHashtagError(valueFlagName string, value string) error {
	return fmt.Errorf("%s: %q starts with # which is invalid", valueFlagName, value)
}

func newValueEndsWithHashtagError(valueFlagName string, value string) error {
	return fmt.Errorf("%s: %q ends with # which is invalid", valueFlagName, value)
}

func newFormatNotFileForDashPathError(valueFlagName string, format Format) error {
	return fmt.Errorf(`%s: path was "-" but format was %q which is not a file format (allowed formats are %s)`, valueFlagName, format.String(), formatsToString(fileFormats()))
}

func newFormatMustBeSourceError(format Format) error {
	return fmt.Errorf("format was %q but must be a source format (allowed formats are %s)", format.String(), formatsToString(sourceFormats()))
}

func newFormatMustBeImageError(format Format) error {
	return fmt.Errorf("format was %q but must be a image format (allowed formats are %s)", format.String(), formatsToString(imageFormats()))
}

func newMustSpecifyGitRefNameError(valueFlagName string, path string) error {
	return fmt.Errorf(`%s: must specify git reference (example: "%s#branch=master" or "%s#tag=v1.0.0")`, valueFlagName, path, path)
}

func newCannotSpecifyMultipleGitRefNamesError(valueFlagName string) error {
	return fmt.Errorf(`%s: must specify only one of "branch", "tag"`, valueFlagName)
}

func newPathUnknownGzError(valueFlagName string, path string) error {
	return fmt.Errorf("%s: path %q had .gz extension with unknown format", valueFlagName, path)
}

func newOptionsInvalidError(valueFlagName string, s string) error {
	return fmt.Errorf("%s: invalid options: %q", valueFlagName, s)
}

func newOptionsInvalidKeyError(valueFlagName string, key string) error {
	return fmt.Errorf("%s: invalid options key: %q", valueFlagName, key)
}

func newOptionsInvalidForFormatError(valueFlagName string, format Format, s string) error {
	return fmt.Errorf("%s: invalid options for format %q: %q", valueFlagName, format.String(), s)
}

func newOptionsCouldNotParseStripComponentsError(valueFlagName string, s string) error {
	return fmt.Errorf("%s: could not parse strip_components value %q", valueFlagName, s)
}

func newFormatOverrideNotAllowedForDevNullError(valueFlagName string, devNull string) error {
	return fmt.Errorf("%s: not allowed if path is %s", valueFlagName, devNull)
}
