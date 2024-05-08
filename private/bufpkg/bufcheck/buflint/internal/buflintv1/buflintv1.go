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

// Package buflintv1 contains the VersionSpec for v1.
//
// It uses buflintcheck and buflintbuild.
//
// Changes are specified on VersionSpec.
package buflintv1

import (
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
)

// VersionSpec is the version specification for v1.
//
// The SYNTAX_SPECIFIED rule was added to BASIC, DEFAULT.
// The IMPORT_USED rule was added to BASIC, DEFAULT.
// ENUM_FIRST_VALUE_ZERO was added to BASIC, DEFAULT.
// PACKAGE_NO_IMPORT_CYCLE was added as an uncategorized lint rule.
// The FIELD_NO_DESCRIPTOR rule was removed altogether.
//
// A number of categories were removed between v1beta1 and v1. The difference
// is shown below:
//
// Removed:
//   - FILE_LAYOUT
//   - PACKAGE_AFFINITY
//   - SENSIBLE
//   - STYLE_BASIC
//   - STYLE_DEFAULT
//   - OTHER
//
// Thus, the only remaining categories are:
//
// Categories:
//   - MINIMAL
//   - BASIC
//   - DEFAULT
//   - COMMENTS
//   - UNARY_RPC
//
// The rules included in the MINIMAL lint category have also been adjusted.
// The difference is shown below:
//
// Removed:
//   - ENUM_NO_ALLOW_ALIAS
//   - FIELD_NO_DESCRIPTOR
//   - IMPORT_NO_PUBLIC
//   - IMPORT_NO_WEAK
//   - PACKAGE_SAME_CSHARP_NAMESPACE
//   - PACKAGE_SAME_GO_PACKAGE
//   - PACKAGE_SAME_JAVA_MULTIPLE_FILES
//   - PACKAGE_SAME_JAVA_PACKAGE
//   - PACKAGE_SAME_PHP_NAMESPACE
//   - PACKAGE_SAME_RUBY_PACKAGE
//   - PACKAGE_SAME_SWIFT_PREFIX
//
// With these changes applied, the final result of MINIMAL is:
//
// MINIMAL:
//   - DIRECTORY_SAME_PACKAGE
//   - PACKAGE_DEFINED
//   - PACKAGE_DIRECTORY_MATCH
//   - PACKAGE_SAME_DIRECTORY
var VersionSpec = &internal.VersionSpec{
	FileVersion:       bufconfig.FileVersionV1,
	RuleBuilders:      v1RuleBuilders,
	DefaultCategories: v1DefaultCategories,
	IDToCategories:    v1IDToCategories,
}
