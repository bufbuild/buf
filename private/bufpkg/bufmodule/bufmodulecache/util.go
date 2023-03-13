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
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/connect-go"
	"go.uber.org/zap"
)

// newCacheKey returns the key associated with the given module pin.
// The cache key is of the form: remote/owner/repository/commit.
func newCacheKey(modulePin bufmoduleref.ModulePin) string {
	return normalpath.Join(modulePin.Remote(), modulePin.Owner(), modulePin.Repository(), modulePin.Commit())
}

// warnIfDeprecated emits a warning message to logger if the repository
// is deprecated on the BSR.
func warnIfDeprecated(
	ctx context.Context,
	clientFactory RepositoryServiceClientFactory,
	modulePin bufmoduleref.ModulePin,
	logger *zap.Logger,
) error {
	repositoryService := clientFactory(modulePin.Remote())
	resp, err := repositoryService.GetRepositoryByFullName(
		ctx,
		connect.NewRequest(&registryv1alpha1.GetRepositoryByFullNameRequest{
			FullName: fmt.Sprintf("%s/%s", modulePin.Owner(), modulePin.Repository()),
		}),
	)
	if err != nil {
		return err
	}
	repository := resp.Msg.Repository
	if repository.Deprecated {
		warnMsg := fmt.Sprintf(`Repository "%s" is deprecated`, modulePin.IdentityString())
		if repository.DeprecationMessage != "" {
			warnMsg = fmt.Sprintf("%s: %s", warnMsg, repository.DeprecationMessage)
		}
		logger.Sugar().Warn(warnMsg)
	}
	return nil
}
