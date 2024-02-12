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
	defaultCheckConfigV1 = newCheckConfigNoValidate(
		FileVersionV1,
		nil,
		nil,
		nil,
		nil,
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

	// Sorted.
	UseIDsAndCategories() []string
	// Sorted
	ExceptIDsAndCategories() []string
	// Paths are specific to the Module.
	// Paths are relative to roots.
	// Sorted
	IgnorePaths() []string
	// Paths are specific to the Module.
	// Paths are relative to roots.
	// Paths sorted.
	IgnoreIDOrCategoryToPaths() map[string][]string

	isCheckConfig()
}

// NewCheckConfig returns a new CheckConfig.
func NewCheckConfig(
	fileVersion FileVersion,
	use []string,
	except []string,
	ignore []string,
	ignoreOnly map[string][]string,
) (CheckConfig, error) {
	return newCheckConfig(
		fileVersion,
		use,
		except,
		ignore,
		ignoreOnly,
	)
}

// NewCheckConfig returns a new CheckConfig for only the use IDs and categories.
func NewCheckConfigForUseIDsAndCategories(
	fileVersion FileVersion,
	use []string,
) CheckConfig {
	return newCheckConfigNoValidate(
		fileVersion,
		slicesext.ToUniqueSorted(use),
		nil,
		nil,
		nil,
	)
}

// *** PRIVATE ***

type checkConfig struct {
	fileVersion FileVersion
	use         []string
	except      []string
	ignore      []string
	ignoreOnly  map[string][]string
}

// TODO: validation of paths

func newCheckConfig(
	fileVersion FileVersion,
	use []string,
	except []string,
	ignore []string,
	ignoreOnly map[string][]string,
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

	return newCheckConfigNoValidate(fileVersion, use, except, ignore, ignoreOnly), nil
}

func newCheckConfigNoValidate(
	fileVersion FileVersion,
	use []string,
	except []string,
	ignore []string,
	ignoreOnly map[string][]string,
) *checkConfig {
	return &checkConfig{
		fileVersion: fileVersion,
		use:         use,
		except:      except,
		ignore:      ignore,
		ignoreOnly:  ignoreOnly,
	}
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

func (*checkConfig) isCheckConfig() {}
