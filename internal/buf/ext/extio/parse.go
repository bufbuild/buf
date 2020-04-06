package extio

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	iov1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/io/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/cli/clios"
)

var (
	fileSchemePrefixToFileScheme = map[string]iov1beta1.FileScheme{
		"http://":  iov1beta1.FileScheme_FILE_SCHEME_HTTP,
		"https://": iov1beta1.FileScheme_FILE_SCHEME_HTTPS,
		"file://":  iov1beta1.FileScheme_FILE_SCHEME_FILE,
	}
	gitSchemePrefixToGitScheme = map[string]iov1beta1.GitScheme{
		"http://":  iov1beta1.GitScheme_GIT_SCHEME_HTTP,
		"https://": iov1beta1.GitScheme_GIT_SCHEME_HTTPS,
		"file://":  iov1beta1.GitScheme_GIT_SCHEME_FILE,
		"ssh://":   iov1beta1.GitScheme_GIT_SCHEME_SSH,
	}
)

func parseInputRef(value string) (*iov1beta1.InputRef, error) {
	rawRef, err := getRawRef(value)
	if err != nil {
		return nil, err
	}
	return getInputRef(rawRef)
}

func parseImageRef(value string) (*iov1beta1.ImageRef, error) {
	rawRef, err := getRawRef(value)
	if err != nil {
		return nil, err
	}
	return getImageRef(rawRef)
}

func parseSourceRef(value string) (*iov1beta1.SourceRef, error) {
	rawRef, err := getRawRef(value)
	if err != nil {
		return nil, err
	}
	return getSourceRef(rawRef)
}

func getInputRef(rawRef *rawRef) (*iov1beta1.InputRef, error) {
	switch rawRef.Format {
	case formatBin, formatBinGz, formatJSON, formatJSONGz:
		imageRef, err := getImageRef(rawRef)
		if err != nil {
			return nil, err
		}
		return &iov1beta1.InputRef{
			Value: &iov1beta1.InputRef_ImageRef{
				ImageRef: imageRef,
			},
		}, nil
	case formatTar, formatTarGz, formatGit, formatDir:
		sourceRef, err := getSourceRef(rawRef)
		if err != nil {
			return nil, err
		}
		return &iov1beta1.InputRef{
			Value: &iov1beta1.InputRef_SourceRef{
				SourceRef: sourceRef,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown format: %v", rawRef.Format)
	}
}

func getImageRef(rawRef *rawRef) (*iov1beta1.ImageRef, error) {
	var imageFormat iov1beta1.ImageFormat
	switch rawRef.Format {
	case formatBin:
		imageFormat = iov1beta1.ImageFormat_IMAGE_FORMAT_BIN
	case formatBinGz:
		imageFormat = iov1beta1.ImageFormat_IMAGE_FORMAT_BINGZ
	case formatJSON:
		imageFormat = iov1beta1.ImageFormat_IMAGE_FORMAT_JSON
	case formatJSONGz:
		imageFormat = iov1beta1.ImageFormat_IMAGE_FORMAT_JSONGZ
	case formatTar, formatTarGz, formatGit, formatDir:
		return nil, newFormatMustBeImageError(rawRef.Format)
	default:
		return nil, fmt.Errorf("unexpected format: %v", rawRef.Format)
	}
	fileScheme, path, err := getFileSchemeAndPath(rawRef.RawPath)
	if err != nil {
		return nil, err
	}
	return &iov1beta1.ImageRef{
		FileScheme:  fileScheme,
		ImageFormat: imageFormat,
		Path:        path,
	}, nil
}

func getSourceRef(rawRef *rawRef) (*iov1beta1.SourceRef, error) {
	switch rawRef.Format {
	case formatTar, formatTarGz:
		archiveRef, err := getArchiveRef(rawRef)
		if err != nil {
			return nil, err
		}
		return &iov1beta1.SourceRef{
			Value: &iov1beta1.SourceRef_ArchiveRef{
				ArchiveRef: archiveRef,
			},
		}, nil
	case formatGit:
		gitRef, err := getGitRef(rawRef)
		if err != nil {
			return nil, err
		}
		return &iov1beta1.SourceRef{
			Value: &iov1beta1.SourceRef_GitRef{
				GitRef: gitRef,
			},
		}, nil
	case formatDir:
		bucketRef, err := getBucketRef(rawRef)
		if err != nil {
			return nil, err
		}
		return &iov1beta1.SourceRef{
			Value: &iov1beta1.SourceRef_BucketRef{
				BucketRef: bucketRef,
			},
		}, nil
	case formatBin, formatBinGz, formatJSON, formatJSONGz:
		return nil, newFormatMustBeSourceError(rawRef.Format)
	default:
		return nil, fmt.Errorf("unexpected format: %v", rawRef.Format)
	}
}

func getArchiveRef(rawRef *rawRef) (*iov1beta1.ArchiveRef, error) {
	var archiveFormat iov1beta1.ArchiveFormat
	switch rawRef.Format {
	case formatTar:
		archiveFormat = iov1beta1.ArchiveFormat_ARCHIVE_FORMAT_TAR
	case formatTarGz:
		archiveFormat = iov1beta1.ArchiveFormat_ARCHIVE_FORMAT_TARGZ
	default:
		return nil, fmt.Errorf("unexpected format: %v", rawRef.Format)
	}
	fileScheme, path, err := getFileSchemeAndPath(rawRef.RawPath)
	if err != nil {
		return nil, err
	}
	return &iov1beta1.ArchiveRef{
		FileScheme:      fileScheme,
		ArchiveFormat:   archiveFormat,
		Path:            path,
		StripComponents: rawRef.ArchiveStripComponents,
	}, nil
}

func getGitRef(rawRef *rawRef) (*iov1beta1.GitRef, error) {
	gitScheme, path, err := getGitSchemeAndPath(rawRef.RawPath)
	if err != nil {
		return nil, err
	}
	gitRef := &iov1beta1.GitRef{
		GitScheme: gitScheme,
		Path:      path,
	}

	if rawRef.GitBranch == "" && rawRef.GitTag == "" {
		// already did this in getRawRef but just in case:
		return nil, newMustSpecifyGitRefNameError(path)
	}
	if rawRef.GitBranch != "" && rawRef.GitTag != "" {
		return nil, newCannotSpecifyMultipleGitRefNamesError()
	}
	if rawRef.GitBranch != "" {
		gitRef.Reference = &iov1beta1.GitRef_Branch{
			Branch: rawRef.GitBranch,
		}
	} else {
		gitRef.Reference = &iov1beta1.GitRef_Tag{
			Tag: rawRef.GitTag,
		}
	}

	if rawRef.GitRecurseSubmodules {
		gitRef.GitSubmoduleBehavior = iov1beta1.GitSubmoduleBehavior_GIT_SUBMODULE_BEHAVIOR_RECURSIVE
	} else {
		gitRef.GitSubmoduleBehavior = iov1beta1.GitSubmoduleBehavior_GIT_SUBMODULE_BEHAVIOR_NONE
	}

	return gitRef, nil
}

func getBucketRef(rawRef *rawRef) (*iov1beta1.BucketRef, error) {
	bucketScheme, path, err := getBucketSchemeAndPath(rawRef.RawPath)
	if err != nil {
		return nil, err
	}
	return &iov1beta1.BucketRef{
		BucketScheme: bucketScheme,
		Path:         path,
	}, nil
}

func getFileSchemeAndPath(rawPath string) (iov1beta1.FileScheme, string, error) {
	// TODO: do we want to normalize to absolute path at all?
	// this is tricky

	if rawPath == "-" {
		return iov1beta1.FileScheme_FILE_SCHEME_STDIO, "", nil
	}
	if rawPath == clios.DevNull {
		return iov1beta1.FileScheme_FILE_SCHEME_NULL, "", nil
	}
	for prefix, fileScheme := range fileSchemePrefixToFileScheme {
		if strings.HasPrefix(rawPath, prefix) {
			return fileScheme, filepath.Clean(strings.TrimPrefix(rawPath, prefix)), nil
		}
	}
	return iov1beta1.FileScheme_FILE_SCHEME_FILE, filepath.Clean(rawPath), nil
}

func getGitSchemeAndPath(rawPath string) (iov1beta1.GitScheme, string, error) {
	// TODO: do we want to normalize to absolute path at all, and ssh user?
	// this is tricky

	if rawPath == "-" {
		return 0, "", newInvalidGitPathError(rawPath)
	}
	if rawPath == clios.DevNull {
		return 0, "", newInvalidGitPathError(rawPath)
	}
	for prefix, gitScheme := range gitSchemePrefixToGitScheme {
		if strings.HasPrefix(rawPath, prefix) {
			return gitScheme, filepath.Clean(strings.TrimPrefix(rawPath, prefix)), nil
		}
	}
	return iov1beta1.GitScheme_GIT_SCHEME_FILE, filepath.Clean(rawPath), nil
}

func getBucketSchemeAndPath(rawPath string) (iov1beta1.BucketScheme, string, error) {
	// TODO: do we want to normalize to absolute path at all?
	// this is tricky

	if rawPath == "-" {
		return 0, "", newInvalidDirPathError(rawPath)
	}
	if rawPath == clios.DevNull {
		return 0, "", newInvalidDirPathError(rawPath)
	}
	return iov1beta1.BucketScheme_BUCKET_SCHEME_DIR, filepath.Clean(rawPath), nil
}

type rawRef struct {
	// RawPath is not normalized yet
	RawPath                string
	Format                 format
	GitBranch              string
	GitTag                 string
	GitRecurseSubmodules   bool
	ArchiveStripComponents uint32
}

func getRawRef(value string) (*rawRef, error) {
	rawPath, options, err := getRawPathAndOptions(value)
	if err != nil {
		return nil, err
	}
	impliedFormat, err := getImpliedFormat(rawPath)
	if err != nil {
		return nil, err
	}
	rawRef := &rawRef{
		RawPath: rawPath,
		Format:  impliedFormat,
	}
	for key, value := range options {
		switch key {
		case "format":
			if rawPath == clios.DevNull {
				return nil, newFormatOverrideNotAllowedForDevNullError(clios.DevNull)
			}
			format, err := parseFormat(value)
			if err != nil {
				return nil, err
			}
			rawRef.Format = format
		case "branch":
			if rawRef.GitBranch != "" || rawRef.GitTag != "" {
				return nil, newCannotSpecifyMultipleGitRefNamesError()
			}
			rawRef.GitBranch = value
		case "tag":
			if rawRef.GitBranch != "" || rawRef.GitTag != "" {
				return nil, newCannotSpecifyMultipleGitRefNamesError()
			}
			rawRef.GitTag = value
		case "recurse_submodules":
			// TODO: need to refactor to make sure this is not set for any non-git input
			// ie right now recurse_submodules=false will not error
			switch value {
			case "true":
				rawRef.GitRecurseSubmodules = true
			case "false":
			default:
				return nil, newOptionsCouldNotParseRecurseSubmodulesError(value)
			}
		case "strip_components":
			// TODO: need to refactor to make sure this is not set for any non-tarball
			// ie right now strip_components=0 will not error
			stripComponents, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil, newOptionsCouldNotParseStripComponentsError(value)
			}
			rawRef.ArchiveStripComponents = uint32(stripComponents)
		default:
			return nil, newOptionsInvalidKeyError(key)
		}
	}

	switch rawRef.Format {
	case formatGit:
	default:
		if rawRef.GitBranch != "" || rawRef.GitTag != "" || rawRef.GitRecurseSubmodules {
			return nil, newOptionsInvalidForFormatError(rawRef.Format, value)
		}
	}
	switch rawRef.Format {
	case formatTar, formatTarGz:
	default:
		if rawRef.ArchiveStripComponents > 0 {
			return nil, newOptionsInvalidForFormatError(rawRef.Format, value)
		}
	}
	return rawRef, nil
}

// rawPath will be non-empty
func getRawPathAndOptions(value string) (string, map[string]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil, newValueEmptyError()
	}

	switch splitValue := strings.Split(value, "#"); len(splitValue) {
	case 1:
		return value, nil, nil
	case 2:
		rawPath := strings.TrimSpace(splitValue[0])
		optionsString := strings.TrimSpace(splitValue[1])
		if rawPath == "" {
			return "", nil, newValueStartsWithHashtagError(value)
		}
		if optionsString == "" {
			return "", nil, newValueEndsWithHashtagError(value)
		}
		options := make(map[string]string)
		for _, pair := range strings.Split(optionsString, ",") {
			split := strings.Split(pair, "=")
			if len(split) != 2 {
				return "", nil, newOptionsInvalidError(optionsString)
			}
			key := strings.TrimSpace(split[0])
			value := strings.TrimSpace(split[1])
			if key == "" || value == "" {
				return "", nil, newOptionsInvalidError(optionsString)
			}
			if _, ok := options[key]; ok {
				return "", nil, newOptionsDuplicateKeyError(key)
			}
			options[key] = value
		}
		return rawPath, options, nil
	default:
		return "", nil, newValueMultipleHashtagsError(value)
	}
}

func getImpliedFormat(rawPath string) (format, error) {
	// if format option is not set and path is "-", default to formatBin
	if rawPath == "-" || rawPath == clios.DevNull {
		return formatBin, nil
	}
	switch filepath.Ext(rawPath) {
	case ".bin":
		return formatBin, nil
	case ".json":
		return formatJSON, nil
	case ".tar":
		return formatTar, nil
	case ".gz":
		switch filepath.Ext(strings.TrimSuffix(rawPath, filepath.Ext(rawPath))) {
		case ".bin":
			return formatBinGz, nil
		case ".json":
			return formatJSONGz, nil
		case ".tar":
			return formatTarGz, nil
		default:
			return 0, newPathUnknownGzError(rawPath)
		}
	case ".tgz":
		return formatTarGz, nil
	case ".git":
		return formatGit, nil
	default:
		return formatDir, nil
	}
}

func newValueEmptyError() error {
	return errors.New("required")
}

func newValueMultipleHashtagsError(value string) error {
	return fmt.Errorf("%q has multiple #s which is invalid", value)
}

func newValueStartsWithHashtagError(value string) error {
	return fmt.Errorf("%q starts with # which is invalid", value)
}

func newValueEndsWithHashtagError(value string) error {
	return fmt.Errorf("%q ends with # which is invalid", value)
}

func newFormatMustBeSourceError(format format) error {
	return fmt.Errorf("format was %q but must be a source format (allowed formats are %s)", format.String(), formatsToString(sourceFormats()))
}

func newFormatMustBeImageError(format format) error {
	return fmt.Errorf("format was %q but must be a image format (allowed formats are %s)", format.String(), formatsToString(imageFormats()))
}

func newMustSpecifyGitRefNameError(path string) error {
	return fmt.Errorf(`must specify git reference (example: "%s#branch=master" or "%s#tag=v1.0.0")`, path, path)
}

func newCannotSpecifyMultipleGitRefNamesError() error {
	return fmt.Errorf(`must specify only one of "branch", "tag"`)
}

func newPathUnknownGzError(path string) error {
	return fmt.Errorf("path %q had .gz extension with unknown format", path)
}

func newOptionsInvalidError(s string) error {
	return fmt.Errorf("invalid options: %q", s)
}

func newOptionsInvalidKeyError(key string) error {
	return fmt.Errorf("invalid options key: %q", key)
}

func newOptionsDuplicateKeyError(key string) error {
	return fmt.Errorf("duplicate options key: %q", key)
}

func newOptionsInvalidForFormatError(format format, s string) error {
	return fmt.Errorf("invalid options for format %q: %q", format.String(), s)
}

func newOptionsCouldNotParseStripComponentsError(s string) error {
	return fmt.Errorf("could not parse strip_components value %q", s)
}

func newOptionsCouldNotParseRecurseSubmodulesError(s string) error {
	return fmt.Errorf("could not parse recurse_submodules value %q", s)
}

func newFormatOverrideNotAllowedForDevNullError(devNull string) error {
	return fmt.Errorf("not allowed if path is %s", devNull)
}

func newInvalidGitPathError(path string) error {
	return fmt.Errorf("invalid git path: %q", path)
}

func newInvalidDirPathError(path string) error {
	return fmt.Errorf("invalid dir path: %q", path)
}
