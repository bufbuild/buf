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
func UploadWithLabels(labels ...string) UploadOption {
	return func(uploadOptions *uploadOptions) {
		uploadOptions.labels = append(uploadOptions.labels, labels...)
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
	Labels() []string
	// CreateIfNotExist says to create Modules if they do not exist on the registry.
	CreateIfNotExist() bool
	// CreateModuleVisibility returns the visibility to create Modules with.
	//
	// Will always be present if CreateIfNotExist() is true.
	CreateModuleVisibility() ModuleVisibility

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
	createIfNotExist       bool
	createModuleVisibility ModuleVisibility
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

func (u *uploadOptions) CreateModuleVisibility() ModuleVisibility {
	return u.createModuleVisibility
}

func (u *uploadOptions) validate() error {
	if u.createIfNotExist && u.createModuleVisibility == 0 {
		return errors.New("must set a valid ModuleVisibility if CreateIfNotExist was specified")
	}
	return nil
}

func (*uploadOptions) isUploadOptions() {}
