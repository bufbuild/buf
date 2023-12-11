// Copyright 2020-2023 Buf Technologies, Inc.
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
	"sort"

	"github.com/bufbuild/buf/private/pkg/slicesext"
)

var (
	defaultCheckConfigV1Beta1 = NewCheckConfig(
		FileVersionV1Beta1,
		nil,
		nil,
		nil,
		nil,
	)
	defaultCheckConfigV1 = NewCheckConfig(
		FileVersionV1,
		nil,
		nil,
		nil,
		nil,
	)
	defaultCheckConfigV2 = NewCheckConfig(
		FileVersionV2,
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
	// TODO: should we make these now relative to root of workspace in v2?
	IgnorePaths() []string
	// Paths are specific to the Module.
	// Paths are relative to roots.
	// TODO: should we make these now relative to root of workspace in v2?
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
) CheckConfig {
	return newCheckConfig(
		fileVersion,
		use,
		except,
		ignore,
		ignoreOnly,
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
) *checkConfig {
	sort.Strings(use)
	sort.Strings(except)
	newIgnoreOnly := make(map[string][]string, len(ignoreOnly))
	for k, v := range ignoreOnly {
		v = slicesext.ToUniqueSorted(v)
		newIgnoreOnly[k] = v
	}
	return &checkConfig{
		fileVersion: fileVersion,
		use:         slicesext.ToUniqueSorted(use),
		except:      slicesext.ToUniqueSorted(except),
		ignore:      slicesext.ToUniqueSorted(ignore),
		ignoreOnly:  newIgnoreOnly,
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
