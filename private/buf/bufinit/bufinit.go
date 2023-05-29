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

package bufinit

import (
	"context"

	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
)

// Initializer initializes buckets.
type Initializer interface {
	Initialize(context.Context, storage.ReadWriteBucket, ...InitializeOption) error
}

// NewInitializer returns a new Initializer
func NewInitializer(logger *zap.Logger) Initializer {
	return newInitializer(logger)
}

// InitializeOption is an option for Initialize
type InitializeOption func(*initializeOptions)

// InitializeWithExcludePaths returns a new InitializeOption that will exclude
// the given files or directories from the initialization.
//
// This is not a regex, but is recursive - excluding "a/b" will exclude "a/b/c", "a/b/b.proto", etc.
func InitializeWithExcludePaths(excludePaths ...string) InitializeOption {
	return func(initializeOptions *initializeOptions) {
		initializeOptions.excludePaths = append(initializeOptions.excludePaths, excludePaths...)
	}
}
