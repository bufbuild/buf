// Copyright 2020 Buf Technologies, Inc.
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

package bufpath

import "github.com/bufbuild/buf/internal/pkg/normalpath"

type nopPathResolver struct{}

func newNopPathResolver() *nopPathResolver {
	return &nopPathResolver{}
}

func (*nopPathResolver) ExternalPathToRelPath(externalPath string) (string, error) {
	return normalpath.NormalizeAndValidate(externalPath)
}

func (*nopPathResolver) RelPathToExternalPath(relPath string) (string, error) {
	relPath, err := normalpath.NormalizeAndValidate(relPath)
	if err != nil {
		return "", err
	}
	return normalpath.Unnormalize(relPath), nil
}
