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

package bufmodulecache

import (
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// NewModuleDataProvider returns a new ModuleDataProvider that caches the results of the delegate.
//
// The given Bucket is used as a cache. This package can choose to use the bucket however it wishes.
//
// TODO: Actually implement this. Right now it is just a passthrough.
func NewModuleDataProvider(
	delegate bufmodule.ModuleDataProvider,
	bucket storage.ReadWriteBucket,
) bufmodule.ModuleDataProvider {
	return delegate
}
