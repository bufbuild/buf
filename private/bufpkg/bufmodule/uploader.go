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

package bufmodule

import (
	"context"
	"errors"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

var (
	// NopUploader is a no-op Uploader.
	NopUploader Uploader = nopUploader{}
)

// Uploader uploads ModuleSets.
type Uploader interface {
	// Upload uploads the given ModuleSet.
	Upload(ctx context.Context, moduleSet ModuleSet, options ...UploadOption) ([]Commit, error)
}

// UploadOption is an option for an Upload.
type UploadOption func(*uploadOptions)

// UploadWithLabels returns a new UploadOption that adds the given labels.
//
// This can be called multiple times. The unique result set of labels will be used.
// We only ever allow one of labels, tags, or branchOrDraft set.
func UploadWithLabels(labels ...string) UploadOption {
	return func(uploadOptions *uploadOptions) {
		uploadOptions.labels = append(uploadOptions.labels, labels...)
	}
}

// UploadWithTags returns a new UploadOption that adds the given tags. This is handled
// separately from labels because we disallow the use of the `--tag` flag when uploading a
// workspace (e.g. a ModuleSet with 1+ target Modules).
//
// This can be called multiple times. The unique result set of tags will be used.
// We only ever allow one of labels, tags, or branchOrDraft set.
func UploadWithTags(tags ...string) UploadOption {
	return func(uploadOptions *uploadOptions) {
		uploadOptions.tags = append(uploadOptions.tags, tags...)
	}
}

// UploadWithBranchOrDraft returns a new UploadOption that adds a branch/draft. This is
// handled separately from labels because we disallow the use of `--branch`/`--draft` when
// uploading a workspace (e.g. a ModuleSet with 1+ target Modules).
//
// If this is called multiple times, the last value is used.
// We only ever allow one of labels, tags, or branchOrDraft set.
func UploadWithBranchOrDraft(branchOrDraft string) UploadOption {
	return func(uploadOptions *uploadOptions) {
		uploadOptions.branchOrDraft = branchOrDraft
	}
}

// UploadWithCreateIfNotExist returns a new UploadOption that will result in the
// Modules being created on the registry with the given visibility if they do not exist.
func UploadWithCreateIfNotExist(createModuleVisibility ModuleVisibility) UploadOption {
	return func(uploadOptions *uploadOptions) {
		uploadOptions.createIfNotExist = true
		uploadOptions.createModuleVisibility = createModuleVisibility
	}
}

// UploadOptions are the possible options for upload.
//
// This is used by Uploader implementations.
type UploadOptions interface {
	// Labels returns the unique and sorted set of labels to add.
	// Labels are set using the `--label` flag when calling `buf push` and represent the
	// labels that are set when uploading module content.
	Labels() []string
	// CreateIfNotExist says to create Modules if they do not exist on the registry.
	CreateIfNotExist() bool
	// CreateModuleVisibility returns the visibility to create Modules with.
	//
	// Will always be present if CreateIfNotExist() is true.
	CreateModuleVisibility() ModuleVisibility
	// Tags returns unique and sorted set of tags to be added as labels.
	// Tags are set using the `--tag` flag when calling `buf push`, and represent labels
	// that are set **in addition to** the default label when uploading module content.
	//
	// The `--tag` flag is a legacy flag that we are continuing supporting. We need to
	// handle tags differently from labels when uploading because we need to resolve the
	// default label and we disallow the setting for `--tag` when uploading >1 module content.
	Tags() []string
	// BranchOrDraft returns a branch/draft to be set as a label.
	// A branch or draft is a single label that we set when uploading module content.
	// A branch is set using the `--branch` flag and a draft is set using the `--draft` flag.
	// Drafts are the successor to branches, and both are now superceded by labels. The
	// `--branch` flag and `--draft` flag only take a single value each. They cannot be used
	// in combination with one another or with `--label` or `--tag`.
	//
	// The `--branch` and `--draft` flags are legacy flags that we are continuing to support.
	// We need to handle branch/draft differently from labels when uploading because we
	// disallow the setting of `--branch` and `--draft` when uploading >1 module content.
	BranchOrDraft() string

	isUploadOptions()
}

// NewUploadOptions returns a new UploadOptions.
func NewUploadOptions(options []UploadOption) (UploadOptions, error) {
	uploadOptions := newUploadOptions()
	for _, option := range options {
		option(uploadOptions)
	}
	if err := uploadOptions.validate(); err != nil {
		return nil, err
	}
	return uploadOptions, nil
}

// *** PRIVATE ***

type nopUploader struct{}

func (nopUploader) Upload(context.Context, ModuleSet, ...UploadOption) ([]Commit, error) {
	return nil, errors.New("unimplemented: no-op Uploader called")
}

type uploadOptions struct {
	labels                 []string
	tags                   []string
	branchOrDraft          string
	createIfNotExist       bool
	createModuleVisibility ModuleVisibility
}

func newUploadOptions() *uploadOptions {
	return &uploadOptions{}
}

func (u *uploadOptions) Labels() []string {
	return slicesext.ToUniqueSorted(u.labels)
}

func (u *uploadOptions) Tags() []string {
	return slicesext.ToUniqueSorted(u.tags)
}

func (u *uploadOptions) BranchOrDraft() string {
	return u.branchOrDraft
}

func (u *uploadOptions) CreateIfNotExist() bool {
	return u.createIfNotExist
}

func (u *uploadOptions) CreateModuleVisibility() ModuleVisibility {
	return u.createModuleVisibility
}

func (u *uploadOptions) validate() error {
	if u.createIfNotExist && u.createModuleVisibility == 0 {
		return errors.New("must set a valid ModuleVisibility if CreateIfNotExist was specified")
	}
	// We validate that only one of labels, tags, and branchOrDraft is set.
	// This is enforced at the flag level, so if more than one is set, we return a syserror.
	if len(u.labels) > 0 && len(u.tags) > 0 ||
		len(u.labels) > 0 && u.branchOrDraft != "" ||
		len(u.tags) > 0 && u.branchOrDraft != "" {
		return syserror.New("more than one of labels, tags, or branch/draft has been set")
	}
	return nil
}

func (*uploadOptions) isUploadOptions() {}
