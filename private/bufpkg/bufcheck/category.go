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

package bufcheck

import "buf.build/go/bufplugin/check"

var _ check.Category = &category{}
var _ Category = &category{}
var _ RuleOrCategory = &category{}

type category struct {
	check.Category

	pluginName string
}

func newCategory(checkCategory check.Category, pluginName string) *category {
	return &category{
		Category:   checkCategory,
		pluginName: pluginName,
	}
}

func (r *category) PluginName() string {
	return r.pluginName
}

func (*category) isCategory()       {}
func (*category) isRuleOrCategory() {}
