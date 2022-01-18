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

// Package buflock manages the buf.lock lock file.
package buflock

import (
	"context"
	"time"

	"github.com/bufbuild/buf/private/pkg/storage"
)

const (
	// ExternalConfigFilePath defines the path to the lock file, relative to the root of the module.
	ExternalConfigFilePath = "buf.lock"
	// V1Version is the string used to identify the v1 version of the lock file.
	V1Version = "v1"
	// V1Beta1Version is the string used to identify the v1beta1 version of the lock file.
	V1Beta1Version = "v1beta1"
	// Header is the header prepended to any lock files.
	Header = "# Generated by buf. DO NOT EDIT.\n"
)

// Config holds the parsed lock file information.
type Config struct {
	Dependencies []Dependency
}

// Dependency describes a single pinned dependency.
type Dependency struct {
	Remote     string
	Owner      string
	Repository string
	Branch     string
	Commit     string
}

// ReadConfig reads the lock file at ExternalConfigFilePath relative
// to the root of the bucket.
func ReadConfig(ctx context.Context, readBucket storage.ReadBucket) (*Config, error) {
	return readConfig(ctx, readBucket)
}

// WriteConfig writes the lock file to the WriteBucket at ExternalConfigFilePath.
func WriteConfig(ctx context.Context, writeBucket storage.WriteBucket, config *Config) error {
	return writeConfig(ctx, writeBucket, config)
}

// ExternalConfigV1 represents the v1 lock file.
type ExternalConfigV1 struct {
	Version string                       `json:"version,omitempty" yaml:"version,omitempty"`
	Deps    []ExternalConfigDependencyV1 `json:"deps,omitempty" yaml:"deps,omitempty"`
}

// ExternalConfigV1Beta1 represents the v1beta1 lock file.
type ExternalConfigV1Beta1 struct {
	Version string                            `json:"version,omitempty" yaml:"version,omitempty"`
	Deps    []ExternalConfigDependencyV1Beta1 `json:"deps,omitempty" yaml:"deps,omitempty"`
}

// ExternalConfigDependencyV1 represents a single dependency within
// the v1 lock file.
type ExternalConfigDependencyV1 struct {
	Remote     string    `json:"remote,omitempty" yaml:"remote,omitempty"`
	Owner      string    `json:"owner,omitempty" yaml:"owner,omitempty"`
	Repository string    `json:"repository,omitempty" yaml:"repository,omitempty"`
	Branch     string    `json:"branch,omitempty" yaml:"branch,omitempty"`
	Commit     string    `json:"commit,omitempty" yaml:"commit,omitempty"`
	Digest     string    `json:"digest,omitempty" yaml:"digest,omitempty"`
	CreateTime time.Time `json:"create_time,omitempty" yaml:"create_time,omitempty"`
}

// DependencyForExternalConfigDependencyV1 returns the Dependency representation of a ExternalConfigDependencyV1.
func DependencyForExternalConfigDependencyV1(dep ExternalConfigDependencyV1) Dependency {
	return Dependency{
		Remote:     dep.Remote,
		Owner:      dep.Owner,
		Repository: dep.Repository,
		Branch:     dep.Branch,
		Commit:     dep.Commit,
	}
}

// ExternalConfigDependencyV1ForDependency returns the ExternalConfigDependencyV1 of a Dependency.
//
// Note, some fields will be their empty value since not all values are available on the Dependency.
func ExternalConfigDependencyV1ForDependency(dep Dependency) ExternalConfigDependencyV1 {
	return ExternalConfigDependencyV1{
		Remote:     dep.Remote,
		Owner:      dep.Owner,
		Repository: dep.Repository,
		Branch:     dep.Branch,
		Commit:     dep.Commit,
	}
}

// ExternalConfigDependencyV1Beta1 represents a single dependency within
// the v1beta1 lock file.
type ExternalConfigDependencyV1Beta1 struct {
	Remote     string    `json:"remote,omitempty" yaml:"remote,omitempty"`
	Owner      string    `json:"owner,omitempty" yaml:"owner,omitempty"`
	Repository string    `json:"repository,omitempty" yaml:"repository,omitempty"`
	Branch     string    `json:"branch,omitempty" yaml:"branch,omitempty"`
	Commit     string    `json:"commit,omitempty" yaml:"commit,omitempty"`
	Digest     string    `json:"digest,omitempty" yaml:"digest,omitempty"`
	CreateTime time.Time `json:"create_time,omitempty" yaml:"create_time,omitempty"`
}

// DepedencyForExternalConfigDependencyV1Beta1 returns the Dependency representation of a ExternalConfigDependencyV1Beta1.
func DependencyForExternalConfigDependencyV1Beta1(dep ExternalConfigDependencyV1Beta1) Dependency {
	return Dependency{
		Remote:     dep.Remote,
		Owner:      dep.Owner,
		Repository: dep.Repository,
		Branch:     dep.Branch,
		Commit:     dep.Commit,
	}
}

// ExternalConfigDependencyV1Beta1ForDependency returns the ExternalConfigDependencyV1Beta1 of a Dependency.
//
// Note, some fields will be their empty value since not all values are available on the Dependency.
func ExternalConfigDependencyV1Beta1ForDependency(dep Dependency) ExternalConfigDependencyV1Beta1 {
	return ExternalConfigDependencyV1Beta1{
		Remote:     dep.Remote,
		Owner:      dep.Owner,
		Repository: dep.Repository,
		Branch:     dep.Branch,
		Commit:     dep.Commit,
	}
}

// ExternalConfigVersion defines the subset of all lock
// file versions that is used to determine the version.
type ExternalConfigVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}
