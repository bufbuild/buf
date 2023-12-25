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

package bufmoduleapi

import (
	"context"
	"fmt"
	"io/fs"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
)

// DigestForCommitID resolves the commit ID by calling the CommitService to get
// the Digest for the Commit.
func DigestForCommitID(
	ctx context.Context,
	clientProvider bufapi.ClientProvider,
	remote string,
	commitID string,
) (bufcas.Digest, error) {
	response, err := clientProvider.CommitServiceClient(remote).GetCommits(
		ctx,
		connect.NewRequest(
			&modulev1beta1.GetCommitsRequest{
				CommitRefs: []*modulev1beta1.CommitRef{
					{
						Id: commitID,
					},
				},
				DigestType: modulev1beta1.DigestType_DIGEST_TYPE_B5,
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return nil, &fs.PathError{Op: "read", Path: commitID, Err: fs.ErrNotExist}
		}
		return nil, err
	}
	if len(response.Msg.Commits) != 1 {
		return nil, fmt.Errorf("expected 1 Commit, got %d", len(response.Msg.Commits))
	}
	commit := response.Msg.Commits[0]
	return bufcas
}
