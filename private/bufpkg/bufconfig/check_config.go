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

package bufconfig

import (
	"github.com/bufbuild/buf/private/pkg/slicesext"
)

var (
	defaultCheckConfigV1 = newEnabledCheckConfigNoValidate(
		FileVersionV1,
		nil,
		nil,
		nil,
		nil,
		false,
	)
	defaultCheckConfigV2 = newEnabledCheckConfigNoValidate(
		FileVersionV2,
		nil,
		nil,
		nil,
		nil,
		false,
	)
)

// CheckConfig is the common interface for the configuration shared by
// LintConfig and BreakingConfig.
type CheckConfig interface {
	// FileVersion returns the file version that this configuration was derived from.
	//
	// We don't want to have to take FileVersion into account for *Configs, however
	// with lint and breaking configurations, the FileVersion changes the interpretation
	// of the IDs and categories.
	FileVersion() FileVersion

	// Disabled says whether or not the given check should be entirely disabled.
	//
	// This happens if an ignore path matches a module directory, which is valid
	// in cases such as:
	//
	//   version: v2
	//   modules:
	//     - path: proto
	//     - path: vendor
	//   lint:
	//     ignore:
	//       - vendor
	//
	// Or:
	//
	//   version: v2
	//   modules:
	//     - path: proto
	//     - path: vendor
	//       lint:
	//         ignore:
	//           - vendor
	//
	// We no longer produce an error in this case. Instead, we set Disabled(), and
	// do not run checks. This means that the following is no longer an error:
	//
	//   version: v1
	//   lint:
	//     ignore:
	//       - .
	//
	// We could make it so that ignore == moduleDirPath is only allowed for v2, however
	// this feels like overkill, so we're just going to keep this consistent for v1
	// and v2.
	Disabled() bool
	// Sorted.
	UseIDsAndCategories() []string
	// Sorted
	ExceptIDsAndCategories() []string
	// Paths are specific to the Module. Users cannot ignore paths outside of their modules for check
	// configs, which includes any imports from outside of the module.
	// Paths are relative to roots.
	// Paths are sorted.
	IgnorePaths() []string
	// Paths are specific to the Module. Users cannot ignore paths outside of their modules for
	// check configs, which includes any imports from outside of the module.
	// Paths are relative to roots.
	// Paths are sorted.
	IgnoreIDOrCategoryToPaths() map[string][]string
	// DisableBuiltin says to disable the Rules and Categories builtin to the Buf CLI and only
	// use plugins.
	//
	// This will make it as if these rules did not exist.
	DisableBuiltin() bool

	isCheckConfig()
}

// NewEnabledCheckConfig returns a new enabled CheckConfig.
func NewEnabledCheckConfig(
	fileVersion FileVersion,
	use []string,
	except []string,
	ignore []string,
	ignoreOnly map[string][]string,
	disableBuiltin bool,
) (CheckConfig, error) {
	return newEnabledCheckConfig(
		fileVersion,
		use,
		except,
		ignore,
		ignoreOnly,
		disableBuiltin,
	)
}

// NewEnabledCheckConfig returns a new enabled CheckConfig for only the use IDs and categories.
func NewEnabledCheckConfigForUseIDsAndCategories(
	fileVersion FileVersion,
	use []string,
	disableBuiltin bool,
) CheckConfig {
	return newEnabledCheckConfigNoValidate(
		fileVersion,
		slicesext.ToUniqueSorted(use),
		nil,
		nil,
		nil,
		disableBuiltin,
	)
}

// *** PRIVATE ***

type checkConfig struct {
	fileVersion    FileVersion
	disabled       bool
	use            []string
	except         []string
	ignore         []string
	ignoreOnly     map[string][]string
	disableBuiltin bool
}

func newEnabledCheckConfig(
	fileVersion FileVersion,
	use []string,
	except []string,
	ignore []string,
	ignoreOnly map[string][]string,
	disableBuiltin bool,
) (*checkConfig, error) {
	use = slicesext.ToUniqueSorted(use)
	except = slicesext.ToUniqueSorted(except)
	ignore = slicesext.ToUniqueSorted(ignore)
	ignore, err := normalizeAndCheckPaths(ignore, "ignore")
	if err != nil {
		return nil, err
	}
	newIgnoreOnly := make(map[string][]string, len(ignoreOnly))
	for k, v := range ignoreOnly {
		v = slicesext.ToUniqueSorted(v)
		v, err := normalizeAndCheckPaths(v, "ignore_only path")
		if err != nil {
			return nil, err
		}
		newIgnoreOnly[k] = v
	}
	ignoreOnly = newIgnoreOnly

	return newEnabledCheckConfigNoValidate(fileVersion, use, except, ignore, ignoreOnly, disableBuiltin), nil
}

func newEnabledCheckConfigNoValidate(
	fileVersion FileVersion,
	use []string,
	except []string,
	ignore []string,
	ignoreOnly map[string][]string,
	disableBuiltin bool,
) *checkConfig {
	return &checkConfig{
		fileVersion:    fileVersion,
		disabled:       false,
		use:            use,
		except:         except,
		ignore:         ignore,
		ignoreOnly:     ignoreOnly,
		disableBuiltin: disableBuiltin,
	}
}

func newDisabledCheckConfig(fileVersion FileVersion) *checkConfig {
	return &checkConfig{
		fileVersion: fileVersion,
		disabled:    true,
	}
}

func (c *checkConfig) Disabled() bool {
	return c.disabled
}

func (c *checkConfig) FileVersion() FileVersion {
	return c.fileVersion
}

func (c *checkConfig) UseIDsAndCategories() []string {
	return slicesext.Copy(c.use)
}

func (c *checkConfig) ExceptIDsAndCategories() []string {
	return slicesext.Copy(c.except)
}

func (c *checkConfig) IgnorePaths() []string {
	return slicesext.Copy(c.ignore)
}

func (c *checkConfig) IgnoreIDOrCategoryToPaths() map[string][]string {
	return copyStringToStringSliceMap(c.ignoreOnly)
}

func (c *checkConfig) DisableBuiltin() bool {
	return c.disableBuiltin
}

func (*checkConfig) isCheckConfig() {}
