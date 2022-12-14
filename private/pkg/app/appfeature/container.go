// Copyright 2020-2022 Buf Technologies, Inc.
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

package appfeature

import (
	"strconv"
	"strings"
	"sync"
)

func newContainer(env Environment) Container {
	return &features{
		env:      env,
		defaults: map[FeatureFlag]bool{
			// Set default feature flag values
		},
	}
}

type features struct {
	env Environment
	// Protects defaults
	mu       sync.RWMutex
	defaults map[FeatureFlag]bool
}

var _ Container = (*features)(nil)

func (f *features) FeatureEnabled(flag FeatureFlag) bool {
	envVar := strings.TrimSpace(f.env.Env(string(flag)))
	if b, err := strconv.ParseBool(envVar); err == nil {
		return b
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.defaults[flag]
}

func (f *features) SetFeatureDefault(flag FeatureFlag, defaultValue bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.defaults[flag] = defaultValue
}
