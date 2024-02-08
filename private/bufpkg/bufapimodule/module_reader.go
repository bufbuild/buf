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

package bufapimodule

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufcas/bufcasalpha"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"go.uber.org/zap"
)

type moduleReader struct {
	logger                *zap.Logger
	downloadClientFactory DownloadServiceClientFactory
	// repositoryClientFactory may be nil
	repositoryClientFactory RepositoryServiceClientFactory
}

func newModuleReader(
	logger *zap.Logger,
	downloadClientFactory DownloadServiceClientFactory,
	opts ...ModuleReaderOption,
) *moduleReader {
	reader := &moduleReader{
		logger:                logger,
		downloadClientFactory: downloadClientFactory,
	}
	for _, opt := range opts {
		opt(reader)
	}
	return reader
}

func (m *moduleReader) GetModule(ctx context.Context, modulePin bufmoduleref.ModulePin) (bufmodule.Module, error) {
	moduleIdentity, err := bufmoduleref.NewModuleIdentity(
		modulePin.Remote(),
		modulePin.Owner(),
		modulePin.Repository(),
	)
	if err != nil {
		// malformed pin
		return nil, err
	}
	identityAndCommitOpt := bufmodule.ModuleWithModuleIdentityAndCommit(
		moduleIdentity,
		modulePin.Commit(),
	)
	resp, err := m.downloadManifestAndBlobs(ctx, modulePin)
	if err != nil {
		return nil, err
	}
	if resp.Manifest == nil {
		return nil, errors.New("expected non-nil manifest")
	}
	fileSet, err := bufcasalpha.AlphaManifestBlobAndBlobsToFileSet(resp.Manifest, resp.Blobs)
	if err != nil {
		return nil, err
	}
	if m.repositoryClientFactory != nil {
		if err := warnIfDeprecated(ctx, m.repositoryClientFactory, modulePin, m.logger); err != nil {
			return nil, err
		}
	}
	return bufmodule.NewModuleForFileSet(ctx, fileSet, identityAndCommitOpt)
}

func (m *moduleReader) downloadManifestAndBlobs(
	ctx context.Context,
	modulePin bufmoduleref.ModulePin,
) (*registryv1alpha1.DownloadManifestAndBlobsResponse, error) {
	downloadService := m.downloadClientFactory(modulePin.Remote())
	resp, err := downloadService.DownloadManifestAndBlobs(
		ctx,
		connect.NewRequest(&registryv1alpha1.DownloadManifestAndBlobsRequest{
			Owner:      modulePin.Owner(),
			Repository: modulePin.Repository(),
			Reference:  modulePin.Commit(),
		}),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			// Required by ModuleReader interface spec
			return nil, &fs.PathError{Op: "read", Path: modulePin.String(), Err: fs.ErrNotExist}
		}
		return nil, err
	}
	return resp.Msg, err
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
