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

// Package bufbreakingv2 contains the VersionSpec for v2.
//
// It uses bufbreakingcheck and bufbreakingbuild.
package bufbreakingv2

import (
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
)

// VersionSpec is the version specification for v2.
//
// Changes from v1:
//
// Adds a new FIELD_SAME_DEFAULT check, which requires that the
// new schema not change default values for fields. (Defaults
// values are a feature of proto2 and editions, but not in
// proto3 syntax.)
//
// Adds new EXTENSION_NO_DELETE and PACKAGE_EXTENSION_NO_DELETE
// checks, to make sure that extensions are not deleted from a
// file or package, respectively. (In previous versions, an
// extension being deleted was simply undetected.)
//
// Removes the following deprecated checks (but retains their
// replacements):
//   - FIELD_SAME_CTYPE
//   - FIELD_SAME_LABEL
//   - FILE_SAME_JAVA_STRING_CHECK_UTF8
//   - FILE_SAME_PHP_GENERIC_SERVICES
var VersionSpec = &internal.VersionSpec{
	FileVersion:       bufconfig.FileVersionV2,
	RuleBuilders:      v2RuleBuilders,
	DefaultCategories: v2DefaultCategories,
	IDToCategories:    v2IDToCategories,
}
