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

package app

import (
	"strconv"
	"strings"
	"sync"
)

func newFeatureContainer(env EnvContainer) FeatureContainer {
	return &featureContainer{
		env:      env,
		defaults: map[FeatureFlag]bool{
			// Set default feature flag values
		},
	}
}

type featureContainer struct {
	env EnvContainer
	// Protects defaults
	mu       sync.RWMutex
	defaults map[FeatureFlag]bool
}

var _ FeatureContainer = (*featureContainer)(nil)

func (f *featureContainer) FeatureEnabled(flag FeatureFlag) bool {
	envVar := strings.TrimSpace(f.env.Env(string(flag)))
	if b, err := strconv.ParseBool(envVar); err == nil {
		return b
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.defaults[flag]
}

func (f *featureContainer) SetFeatureDefault(flag FeatureFlag, defaultValue bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.defaults[flag] = defaultValue
}
