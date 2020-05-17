// Copyright 2020 Buf Technologies Inc.
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

package git

import (
	"context"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

// RefName is a reference name.
type RefName interface {
	branch() string
}

// NewBranchRefName returns a new RefName for the branch.
func NewBranchRefName(branch string) RefName {
	return newRefName(branch)
}

// NewTagRefName returns a new RefName for the tag.
func NewTagRefName(tag string) RefName {
	return newRefName(tag)
}

// Cloner clones git repositories to buckets.
type Cloner interface {
	// CloneToBucket clones the repository to the bucket.
	//
	// The url must contain the scheme, including file:// if necessary.
	CloneToBucket(
		ctx context.Context,
		envContainer app.EnvContainer,
		url string,
		refName RefName,
		readWriteBucket storage.ReadWriteBucket,
		options CloneToBucketOptions,
	) error
}

// CloneToBucketOptions are options for Clone.
type CloneToBucketOptions struct {
	TransformerOptions []normalpath.TransformerOption
	RecurseSubmodules  bool
}

// NewCloner returns a new Cloner.
func NewCloner(logger *zap.Logger, options ClonerOptions) Cloner {
	return newCloner(logger, options)
}

// ClonerOptions are options for a new Cloner.
type ClonerOptions struct {
	HTTPSUsernameEnvKey      string
	HTTPSPasswordEnvKey      string
	SSHKeyFileEnvKey         string
	SSHKnownHostsFilesEnvKey string
}
