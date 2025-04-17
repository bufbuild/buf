// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufpolicy

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/bufbuild/buf/private/pkg/slicesext"
)

var (
	// NopUploader is a no-op Uploader.
	NopUploader Uploader = nopUploader{}
)

// Uploader uploads Policies.
type Uploader interface {
	// Upload uploads the given Policies.
	Upload(ctx context.Context, policies []Policy, options ...UploadOption) ([]Commit, error)
}

// UploadOption is an option for an Upload.
type UploadOption func(*uploadOptions)

// UploadWithLabels returns a new UploadOption that adds the given labels.
//
// This can be called multiple times. The unique result set of labels will be used.
func UploadWithLabels(labels ...string) UploadOption {
	return func(uploadOptions *uploadOptions) {
		uploadOptions.labels = append(uploadOptions.labels, labels...)
	}
}

// UploadWithCreateIfNotExist returns a new UploadOption that will result in the
// Policies being created on the registry with the given visibility if they do not exist.
func UploadWithCreateIfNotExist(createPolicyVisibility PolicyVisibility) UploadOption {
	return func(uploadOptions *uploadOptions) {
		uploadOptions.createIfNotExist = true
		uploadOptions.createPolicyVisibility = createPolicyVisibility
	}
}

// UploadWithSourceControlURL returns a new UploadOption that will set the source control
// url for the policy contents uploaded.
func UploadWithSourceControlURL(sourceControlURL string) UploadOption {
	return func(uploadOptions *uploadOptions) {
		uploadOptions.sourceControlURL = sourceControlURL
	}
}

// UploadOptions are the possible options for upload.
//
// This is used by Uploader implementations.
type UploadOptions interface {
	// Labels returns the unique and sorted set of labels to add. Labels
	// are set using the `--label` flag when calling `buf policy upload`
	// and represent the labels that are set when uploading policy data.
	Labels() []string
	// CreateIfNotExist says to create Policies if they do not exist on the registry.
	CreateIfNotExist() bool
	// CreatePolicyVisibility returns the visibility to create Policies with.
	//
	// Will always be present if CreateIfNotExist() is true.
	CreatePolicyVisibility() PolicyVisibility
	// SourceControlURL returns the source control URL set by the user for the policy
	// contents uploaded. We set the same source control URL for all policy contents.
	SourceControlURL() string

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

func (nopUploader) Upload(context.Context, []Policy, ...UploadOption) ([]Commit, error) {
	return nil, errors.New("unimplemented: no-op Uploader called")
}

type uploadOptions struct {
	labels                 []string
	createIfNotExist       bool
	createPolicyVisibility PolicyVisibility
	sourceControlURL       string
}

func newUploadOptions() *uploadOptions {
	return &uploadOptions{}
}

func (u *uploadOptions) Labels() []string {
	return slicesext.ToUniqueSorted(u.labels)
}

func (u *uploadOptions) CreateIfNotExist() bool {
	return u.createIfNotExist
}

func (u *uploadOptions) CreatePolicyVisibility() PolicyVisibility {
	return u.createPolicyVisibility
}

func (u *uploadOptions) SourceControlURL() string {
	return u.sourceControlURL
}

func (u *uploadOptions) isUploadOptions() {}

func (u *uploadOptions) validate() error {
	if u.createIfNotExist && u.createPolicyVisibility == 0 {
		return errors.New("must set a valid PolicyVisibility if CreateIfNotExist was specified")
	}
	if u.sourceControlURL != "" {
		if _, err := url.Parse(u.sourceControlURL); err != nil {
			return fmt.Errorf("must set a valid url for the source control url: %w", err)
		}
	}
	return nil
}
