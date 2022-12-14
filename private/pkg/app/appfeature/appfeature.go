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

// Package appfeature provides support for application feature flags.
package appfeature

// FeatureFlag represents an application feature flag.
type FeatureFlag string

// Container is a simple feature flagging interface for the Buf CLI.
type Container interface {
	// FeatureEnabled returns whether the specified feature flag is enabled.
	FeatureEnabled(flag FeatureFlag) bool
	// SetFeatureDefault sets the default value for the feature flag.
	SetFeatureDefault(flag FeatureFlag, defaultValue bool)
}

// Environment provides access to environment variables.
type Environment interface {
	// Env returns the environment variable value, or an empty string if unset (or set to empty).
	Env(string) string
}

// EnvironmentFunc wraps a function (e.g. os.Getenv) as an Environment.
type EnvironmentFunc func(string) string

func (f EnvironmentFunc) Env(name string) string {
	return f(name)
}

// NewContainer creates a new Container from the Environment.
func NewContainer(env Environment) Container {
	return newContainer(env)
}
