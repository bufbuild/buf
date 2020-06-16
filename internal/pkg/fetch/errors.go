// Copyright 2020 Buf Technologies, Inc.
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

package fetch

import (
	"errors"
	"fmt"
	"strings"
)

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

func newFormatNotAllowedError(format string, allowedFormats map[string]struct{}) error {
	return fmt.Errorf("format was %q but must be one of %s", format, formatsToString(allowedFormats))
}

func newFormatCannotBeDeterminedError(value string) error {
	return fmt.Errorf("format cannot be determined from %q", value)
}

func newMustSpecifyGitRepositoryRefNameError(path string) error {
	return fmt.Errorf(`must specify git reference (example: "%s#branch=master" or "%s#tag=v1.0.0")`, path, path)
}

func newCannotSpecifyMultipleGitRepositoryRefNamesError() error {
	return fmt.Errorf(`must specify only one of "branch", "tag"`)
}

func newPathUnknownGzError(path string) error {
	return fmt.Errorf("path %q had .gz extension with unknown format", path)
}

func newCompressionUnknownError(compression string, valid ...string) error {
	return fmt.Errorf("unknown compression: %q (valid values are %q)", compression, strings.Join(valid, ","))
}

func newCannotSpecifyCompressionForZipError() error {
	return errors.New("cannot specify compression type for zip files")
}

func newNoPathError() error {
	return errors.New("value has no path once processed")
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

func newOptionsInvalidForFormatError(format string, s string) error {
	return fmt.Errorf("invalid options for format %q: %q", format, s)
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

func newInvalidFilePathError(path string) error {
	return fmt.Errorf("invalid file path: %q", path)
}

func newFormatUnknownError(formatString string) error {
	return fmt.Errorf("unknown format: %q", formatString)
}

func newReadDisabledError(scheme string) error {
	return fmt.Errorf("reading assets from %s disabled", scheme)
}

func newReadHTTPDisabledError() error {
	return newReadDisabledError("http")
}

func newReadGitDisabledError() error {
	return newReadDisabledError("git")
}

func newReadLocalDisabledError() error {
	return newReadDisabledError("local")
}

func newReadStdioDisabledError() error {
	return newReadDisabledError("stdin")
}

func newWriteDisabledError(scheme string) error {
	return fmt.Errorf("writing assets to %s disabled", scheme)
}

func newWriteHTTPDisabledError() error {
	return newWriteDisabledError("http")
}

func newWriteLocalDisabledError() error {
	return newWriteDisabledError("local")
}

func newWriteStdioDisabledError() error {
	return newWriteDisabledError("stdout")
}
