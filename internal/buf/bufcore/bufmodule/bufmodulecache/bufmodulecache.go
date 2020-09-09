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

package bufmodulecache

import (
	"io"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"go.uber.org/zap"
)

// NewModuleReader returns a new ModuleReader that uses cache as a caching layer, and
// delegate as the source of truth.
func NewModuleReader(
	logger *zap.Logger,
	cache bufmodule.ModuleReadWriter,
	delegate bufmodule.ModuleReader,
	options ...ModuleReaderOption,
) bufmodule.ModuleReader {
	return newModuleReader(logger, cache, delegate, options...)
}

// ModuleReaderOption is an option for a new ModuleReader.
type ModuleReaderOption func(*moduleReader)

// WithMessageWriter adds the given Writer to print messages.
//
// This is typically stderr.
// The default is to not print messages.
func WithMessageWriter(messageWriter io.Writer) ModuleReaderOption {
	return func(moduleReader *moduleReader) {
		moduleReader.messageWriter = messageWriter
	}
}
