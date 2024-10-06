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

package bufmodulestore

import (
	"context"
	"log/slog"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
)

func logDebugModuleKey(ctx context.Context, logger *slog.Logger, moduleKey bufmodule.ModuleKey, message string, fields any) {
	logger.DebugContext(
		ctx,
		message,
		append(
			[]any{
				slog.String("moduleFullName", moduleKey.ModuleFullName().String()),
				slog.String("commitID", uuidutil.ToDashless(moduleKey.CommitID())),
			},
			fields...,
		)...,
	)
}

func logDebugCommitKey(ctx context.Context, logger *slog.Logger, commitKey bufmodule.CommitKey, message string, fields ...any) {
	logger.DebugContext(
		ctx,
		message,
		append(
			[]any{
				slog.String("digestType", commitKey.DigestType().String()),
				slog.String("registry", commitKey.Registry()),
				slog.String("commitID", uuidutil.ToDashless(commitKey.CommitID())),
			},
			fields...,
		)...,
	)
}
