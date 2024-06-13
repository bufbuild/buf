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
	"fmt"
	"net/url"

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
// We only ever allow one of labels or tags to be set.
func UploadWithLabels(labels ...string) UploadOption {
	return func(uploadOptions *uploadOptions) {
		uploadOptions.labels = append(uploadOptions.labels, labels...)
	}
}

// UploadWithTags returns a new UploadOption that adds the given tags. This is handled
// separately from labels because we need to resolve the default label(s) when uploading.
//
// This can be called multiple times. The unique result set of tags will be used.
// We only ever allow one of labels or tags to be set.
func UploadWithTags(tags ...string) UploadOption {
	return func(uploadOptions *uploadOptions) {
		uploadOptions.tags = append(uploadOptions.tags, tags...)
	}
}

// UploadWithCreateIfNotExist returns a new UploadOption that will result in the
// Modules being created on the registry with the given visibility and default label if they do not exist.
// If the default label name is not provided, the module will be created with the default label "main".
func UploadWithCreateIfNotExist(createModuleVisibility ModuleVisibility, createDefaultLabel string) UploadOption {
	return func(uploadOptions *uploadOptions) {
		uploadOptions.createIfNotExist = true
		uploadOptions.createModuleVisibility = createModuleVisibility
		uploadOptions.createDefaultLabel = createDefaultLabel
	}
}

// UploadWithSourceControlURL returns a new UploadOption that will set the source control
// url for the module contents uploaded.
func UploadWithSourceControlURL(sourceControlURL string) UploadOption {
	return func(uploadOptions *uploadOptions) {
		uploadOptions.sourceControlURL = sourceControlURL
	}
}

// UploadWithExcludeUnnamed returns a new UploadOption that will exclude unnamed modules.
func UploadWithExcludeUnnamed() UploadOption {
	return func(uploadOptions *uploadOptions) {
		uploadOptions.excludeUnnamed = true
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
	// CreateDefaultLabel returns the default label to create Modules with. If this is an
	// emptry string, then the Modules will be created with default label "main".
	CreateDefaultLabel() string
	// Tags returns unique and sorted set of tags to be added as labels.
	// Tags are set using the `--tag` flag when calling `buf push`, and represent labels
	// that are set **in addition to** the default label when uploading module content.
	//
	// The `--tag` flag is a legacy flag that we are continuing supporting. We need to
	// handle tags differently from labels when uploading because we need to resolve the
	// default label for each module.
	//
	// We disallow the use of `--tag` when the modules we are uploading to do not all have
	// the same default label.
	Tags() []string
	// SourceControlURL returns the source control URL set by the user for the module
	// contents uploaded. We set the same source control URL for all module contents.
	SourceControlURL() string
	// ExcludeUnnamed returns whether to exclude unnamed modules.
	ExcludeUnnamed() bool

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
	createIfNotExist       bool
	createModuleVisibility ModuleVisibility
	createDefaultLabel     string
	sourceControlURL       string
	excludeUnnamed         bool
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

func (u *uploadOptions) CreateIfNotExist() bool {
	return u.createIfNotExist
}

func (u *uploadOptions) CreateModuleVisibility() ModuleVisibility {
	return u.createModuleVisibility
}

func (u *uploadOptions) CreateDefaultLabel() string {
	return u.createDefaultLabel
}

func (u *uploadOptions) SourceControlURL() string {
	return u.sourceControlURL
}

func (u *uploadOptions) ExcludeUnnamed() bool {
	return u.excludeUnnamed
}

func (u *uploadOptions) validate() error {
	if u.createIfNotExist && u.createModuleVisibility == 0 {
		return errors.New("must set a valid ModuleVisibility if CreateIfNotExist was specified")
	}
	// We validate that only one of labels or tags is set.
	// This is enforced at the flag level, so if more than one is set, we return a syserror.
	if len(u.labels) > 0 && len(u.tags) > 0 {
		return syserror.New("cannot set both labels and tags")
	}
	if u.sourceControlURL != "" {
		if _, err := url.Parse(u.sourceControlURL); err != nil {
			return fmt.Errorf("must set a valid url for the source control url: %w", err)
		}
	}
	return nil
}

func (*uploadOptions) isUploadOptions() {}
