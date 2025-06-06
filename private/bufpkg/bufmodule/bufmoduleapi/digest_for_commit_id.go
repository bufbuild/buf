// Copyright 2020-2025 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapimodule"
	"github.com/google/uuid"
)

// DigestForCommitID resolves the commit ID by calling the CommitService to get
// the Digest for the Commit.
func DigestForCommitID(
	ctx context.Context,
	moduleClientProvider interface {
		bufregistryapimodule.V1CommitServiceClientProvider
		bufregistryapimodule.V1Beta1CommitServiceClientProvider
	},
	registry string,
	commitID uuid.UUID,
	digestType bufmodule.DigestType,
) (bufmodule.Digest, error) {
	universalProtoCommit, err := getUniversalProtoCommitForRegistryAndCommitID(ctx, moduleClientProvider, registry, commitID, digestType)
	if err != nil {
		return nil, err
	}
	return universalProtoCommit.Digest, nil
}
