// Copyright 2020-2026 Buf Technologies, Inc.
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

package cas_test

import (
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/buf/buftesting"
	"github.com/bufbuild/buf/private/pkg/cas"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
)

var buftestingDirPath = filepath.Join(
	"..",
	"..",
	"buf",
	"buftesting",
)

// BenchmarkBucketToDigest exercises the per-file shake256 digest path that
// drives module digest computation (`buf push`, `buf dep update`, lockfile
// verification). The corpus is googleapis (1574 proto files), matching
// TestGoogleapis in private/bufpkg/bufimage.
func BenchmarkBucketToDigest(b *testing.B) {
	googleapisDirPath := buftesting.GetGoogleapisDirPath(b, buftestingDirPath)
	provider := storageos.NewProvider()
	bucket, err := provider.NewReadWriteBucket(googleapisDirPath)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := cas.BucketToDigest(b.Context(), bucket, cas.DigestTypeShake256); err != nil {
			b.Fatal(err)
		}
	}
}
