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

package bufimage_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/bufbuild/buf/private/buf/buftesting"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
)

// BenchmarkBuildImage builds an image from the googleapis corpus (1574 proto
// files), matching TestGoogleapis. Exercises the proto-source read path that
// allocates an io.Copy fallback buffer per file.
func BenchmarkBuildImage(b *testing.B) {
	googleapisDirPath := buftesting.GetGoogleapisDirPath(b, buftestingDirPath)
	moduleSet, err := bufmoduletesting.NewModuleSetForDirPath(googleapisDirPath)
	if err != nil {
		b.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := bufimage.BuildImage(
			b.Context(),
			logger,
			bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
			bufimage.WithExcludeSourceCodeInfo(),
			bufimage.WithNoParallelism(),
		); err != nil {
			b.Fatal(err)
		}
	}
}
