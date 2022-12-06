// Copyright 2020-2022 Buf Technologies, Inc.
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

package bufapimodule

import (
	"context"

	registryv1alpha1 "buf.build/gen/go/bufbuild/buf/protocolbuffers/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/connect-go"
)

type moduleReader struct {
	downloadClientFactory DownloadServiceClientFactory
}

func newModuleReader(
	downloadClientFactory DownloadServiceClientFactory,
) *moduleReader {
	return &moduleReader{
		downloadClientFactory: downloadClientFactory,
	}
}

func (m *moduleReader) GetModule(ctx context.Context, modulePin bufmoduleref.ModulePin) (bufmodule.Module, error) {
	downloadService := m.downloadClientFactory(modulePin.Remote())
	resp, err := downloadService.Download(
		ctx,
		connect.NewRequest(&registryv1alpha1.DownloadRequest{
			Owner:      modulePin.Owner(),
			Repository: modulePin.Repository(),
			Reference:  modulePin.Commit(),
		}),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			// Required by ModuleReader interface spec
			return nil, storage.NewErrNotExist(modulePin.String())
		}
		return nil, err
	}
	moduleIdentity, err := bufmoduleref.NewModuleIdentity(
		modulePin.Remote(),
		modulePin.Owner(),
		modulePin.Repository(),
	)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleForProto(
		ctx, resp.Msg.Module,
		bufmodule.ModuleWithModuleIdentityAndCommit(moduleIdentity, modulePin.Commit()),
	)
}
