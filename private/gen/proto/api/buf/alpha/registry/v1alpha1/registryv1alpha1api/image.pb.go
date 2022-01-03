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

// Code generated by protoc-gen-go-api. DO NOT EDIT.

package registryv1alpha1api

import (
	context "context"
	v1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
)

// ImageService serves compiled images.
type ImageService interface {
	// GetImage serves a compiled image for the local module. It automatically
	// downloads dependencies if necessary.
	GetImage(
		ctx context.Context,
		owner string,
		repository string,
		reference string,
	) (image *v1.Image, err error)
}
