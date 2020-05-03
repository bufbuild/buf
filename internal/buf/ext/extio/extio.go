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

package extio

import iov1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/io/v1beta1"

// InputFormatsToString returns input format strings.
func InputFormatsToString() string {
	return formatsToString(inputFormats())
}

// SourceFormatsToString returns source format strings.
func SourceFormatsToString() string {
	return formatsToString(sourceFormats())
}

// ImageFormatsToString returns image format strings.
func ImageFormatsToString() string {
	return formatsToString(imageFormats())
}

// ParseInputRef parses the input ref.
func ParseInputRef(value string) (*iov1beta1.InputRef, error) {
	return parseInputRef(value)
}

// ParseSourceRef parses the source ref.
func ParseSourceRef(value string) (*iov1beta1.SourceRef, error) {
	return parseSourceRef(value)
}

// ParseImageRef parses the image ref.
func ParseImageRef(value string) (*iov1beta1.ImageRef, error) {
	return parseImageRef(value)
}
