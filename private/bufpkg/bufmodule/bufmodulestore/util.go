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
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"go.uber.org/zap"
)

func logDebugModuleKey(logger *zap.Logger, moduleKey bufmodule.ModuleKey, message string, fields ...zap.Field) {
	if checkedEntry := logger.Check(zap.DebugLevel, message); checkedEntry != nil {
		checkedEntry.Write(
			append(
				[]zap.Field{
					zap.String("moduleFullName", moduleKey.ModuleFullName().String()),
					zap.String("commitID", uuidutil.ToDashless(moduleKey.CommitID())),
				},
				fields...,
			)...,
		)
	}
}

func logDebugCommitKey(logger *zap.Logger, commitKey bufmodule.CommitKey, message string, fields ...zap.Field) {
	if checkedEntry := logger.Check(zap.DebugLevel, message); checkedEntry != nil {
		checkedEntry.Write(
			append(
				[]zap.Field{
					zap.String("digestType", commitKey.DigestType().String()),
					zap.String("registry", commitKey.Registry()),
					zap.String("commitID", uuidutil.ToDashless(commitKey.CommitID())),
				},
				fields...,
			)...,
		)
	}
}
