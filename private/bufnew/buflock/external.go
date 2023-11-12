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

package buflock

import (
	"time"
)

// externalFileV1 represents the v1 buf.lock file.
type externalFileV1 struct {
	Version string              `json:"version,omitempty" yaml:"version,omitempty"`
	Deps    []externalFileDepV1 `json:"deps,omitempty" yaml:"deps,omitempty"`
}

// externalFileDepV1 represents a single dep within a v1 buf.lock file.
type externalFileDepV1 struct {
	Remote     string    `json:"remote,omitempty" yaml:"remote,omitempty"`
	Owner      string    `json:"owner,omitempty" yaml:"owner,omitempty"`
	Repository string    `json:"repository,omitempty" yaml:"repository,omitempty"`
	Branch     string    `json:"branch,omitempty" yaml:"branch,omitempty"`
	Commit     string    `json:"commit,omitempty" yaml:"commit,omitempty"`
	Digest     string    `json:"digest,omitempty" yaml:"digest,omitempty"`
	CreateTime time.Time `json:"create_time,omitempty" yaml:"create_time,omitempty"`
}

// externalFileV1Beta1 represents a v1beta1 buf.lock file.
type externalFileV1Beta1 struct {
	Version string                   `json:"version,omitempty" yaml:"version,omitempty"`
	Deps    []externalFileDepV1Beta1 `json:"deps,omitempty" yaml:"deps,omitempty"`
}

// externalFileDepV1Beta1 represents a single dep within a v1beta1 buf.lock file.
type externalFileDepV1Beta1 struct {
	Remote     string    `json:"remote,omitempty" yaml:"remote,omitempty"`
	Owner      string    `json:"owner,omitempty" yaml:"owner,omitempty"`
	Repository string    `json:"repository,omitempty" yaml:"repository,omitempty"`
	Branch     string    `json:"branch,omitempty" yaml:"branch,omitempty"`
	Commit     string    `json:"commit,omitempty" yaml:"commit,omitempty"`
	Digest     string    `json:"digest,omitempty" yaml:"digest,omitempty"`
	CreateTime time.Time `json:"create_time,omitempty" yaml:"create_time,omitempty"`
}

// externalFileVersion represents just the version component of a buf.lock file.
type externalFileVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}
