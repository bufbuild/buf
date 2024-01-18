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

package bufmodulecache

import (
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"go.uber.org/zap"
)

// NewModuleReader creates a new module reader using content addressable storage.
func NewModuleReader(
	logger *zap.Logger,
	verbosePrinter verbose.Printer,
	bucket storage.ReadWriteBucket,
	delegate bufmodule.ModuleReader,
) bufmodule.ModuleReader {
	return newCASModuleReader(
		bucket,
		delegate,
		logger,
		verbosePrinter,
	)
}
