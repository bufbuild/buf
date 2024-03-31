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

package bufmoduleapi

import (
	"context"
	"fmt"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/gofrs/uuid/v5"
)

type universalProtoContent struct {
	// Dashless
	CommitID string
	// Dashless
	ModuleID      string
	Files         []*universalProtoFile
	V1BufYAMLFile *universalProtoFile
	V1BufLockFile *universalProtoFile
}

func newUniversalProtoContentForV1(v1ProtoContent *modulev1.DownloadResponse_Content) *universalProtoContent {
	return &universalProtoContent{
		CommitID: v1ProtoContent.Commit.Id,
		ModuleID: v1ProtoContent.Commit.ModuleId,
		Files:    slicesext.Map(v1ProtoContent.Files, newUniversalProtoFileForV1),
	}
}

func newUniversalProtoContentForV1Beta1(v1beta1ProtoContent *modulev1beta1.DownloadResponse_Content) *universalProtoContent {
	var (
		v1BufYAMLFile *universalProtoFile
		v1BufLockFile *universalProtoFile
	)
	if v1beta1ProtoContent.V1BufYamlFile != nil {
		v1BufYAMLFile = newUniversalProtoFileForV1Beta1(v1beta1ProtoContent.V1BufYamlFile)
	}
	if v1beta1ProtoContent.V1BufLockFile != nil {
		v1BufLockFile = newUniversalProtoFileForV1Beta1(v1beta1ProtoContent.V1BufLockFile)
	}
	return &universalProtoContent{
		CommitID:      v1beta1ProtoContent.Commit.Id,
		ModuleID:      v1beta1ProtoContent.Commit.ModuleId,
		Files:         slicesext.Map(v1beta1ProtoContent.Files, newUniversalProtoFileForV1Beta1),
		V1BufYAMLFile: v1BufYAMLFile,
		V1BufLockFile: v1BufLockFile,
	}
}

func getUniversalProtoContentsForRegistryAndCommitIDs(
	ctx context.Context,
	clientProvider interface {
		bufapi.V1DownloadServiceClientProvider
		bufapi.V1Beta1DownloadServiceClientProvider
	},
	registry string,
	commitIDs []uuid.UUID,
	digestType bufmodule.DigestType,
) ([]*universalProtoContent, error) {
	switch digestType {
	case bufmodule.DigestTypeB4:
		v1beta1ProtoResourceRefs := commitIDsToV1Beta1ProtoResourceRefs(commitIDs)
		v1beta1ProtoContents, err := getV1Beta1ProtoContentsForRegistryAndResourceRefs(ctx, clientProvider, registry, v1beta1ProtoResourceRefs, digestType)
		if err != nil {
			return nil, err
		}
		return slicesext.Map(v1beta1ProtoContents, newUniversalProtoContentForV1Beta1), nil
	case bufmodule.DigestTypeB5:
		v1ProtoResourceRefs := commitIDsToV1ProtoResourceRefs(commitIDs)
		v1ProtoContents, err := getV1ProtoContentsForRegistryAndResourceRefs(ctx, clientProvider, registry, v1ProtoResourceRefs)
		if err != nil {
			return nil, err
		}
		return slicesext.Map(v1ProtoContents, newUniversalProtoContentForV1), nil
	default:
		return nil, syserror.Newf("unknown DigestType: %v", digestType)
	}
}

func getV1ProtoContentsForRegistryAndResourceRefs(
	ctx context.Context,
	clientProvider bufapi.V1DownloadServiceClientProvider,
	registry string,
	v1ProtoResourceRefs []*modulev1.ResourceRef,
) ([]*modulev1.DownloadResponse_Content, error) {
	response, err := clientProvider.V1DownloadServiceClient(registry).Download(
		ctx,
		connect.NewRequest(
			&modulev1.DownloadRequest{
				// TODO FUTURE: chunking
				Values: slicesext.Map(
					v1ProtoResourceRefs,
					func(v1ProtoResourceRef *modulev1.ResourceRef) *modulev1.DownloadRequest_Value {
						return &modulev1.DownloadRequest_Value{
							ResourceRef: v1ProtoResourceRef,
						}
					},
				),
			},
		),
	)
	if err != nil {
		return nil, maybeNewNotFoundError(err)
	}
	if len(response.Msg.Contents) != len(v1ProtoResourceRefs) {
		return nil, fmt.Errorf("expected %d Contents, got %d", len(v1ProtoResourceRefs), len(response.Msg.Contents))
	}
	return response.Msg.Contents, nil
}

func getV1Beta1ProtoContentsForRegistryAndResourceRefs(
	ctx context.Context,
	clientProvider bufapi.V1Beta1DownloadServiceClientProvider,
	registry string,
	v1beta1ProtoResourceRefs []*modulev1beta1.ResourceRef,
	digestType bufmodule.DigestType,
) ([]*modulev1beta1.DownloadResponse_Content, error) {
	v1beta1ProtoDigestType, err := digestTypeToV1Beta1Proto(digestType)
	if err != nil {
		return nil, err
	}
	response, err := clientProvider.V1Beta1DownloadServiceClient(registry).Download(
		ctx,
		connect.NewRequest(
			&modulev1beta1.DownloadRequest{
				// TODO FUTURE: chunking
				Values: slicesext.Map(
					v1beta1ProtoResourceRefs,
					func(v1beta1ProtoResourceRef *modulev1beta1.ResourceRef) *modulev1beta1.DownloadRequest_Value {
						return &modulev1beta1.DownloadRequest_Value{
							ResourceRef: v1beta1ProtoResourceRef,
						}
					},
				),
				DigestType: v1beta1ProtoDigestType,
			},
		),
	)
	if err != nil {
		return nil, maybeNewNotFoundError(err)
	}
	if len(response.Msg.Contents) != len(v1beta1ProtoResourceRefs) {
		return nil, fmt.Errorf("expected %d Contents, got %d", len(v1beta1ProtoResourceRefs), len(response.Msg.Contents))
	}
	return response.Msg.Contents, nil
}
