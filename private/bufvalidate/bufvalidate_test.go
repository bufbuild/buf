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

package bufvalidate

import (
	"testing"

	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/stretchr/testify/require"
)

var (
	_ Validator = (*registryv1alpha1.DeprecateRepositoryByNameRequest)(nil)
	_ Validator = (*registryv1alpha1.GetRepositoryRequest)(nil)
)

func TestValidate(t *testing.T) {
	t.Run("deprecate request", func(t *testing.T) {
		message := &registryv1alpha1.DeprecateRepositoryByNameRequest{
			OwnerName: "owner",
		}
		require.Error(t, message.Validate())
		message.RepositoryName = "repository"
		require.Error(t, message.Validate())
		message.DeprecationMessage = "deprecation"
		require.NoError(t, message.Validate())
	})
	t.Run("get request", func(t *testing.T) {
		message := &registryv1alpha1.GetRepositoryRequest{}
		require.EqualError(t, message.Validate(), "GetRepositoryRequest must have a non-empty id")
		message.Id = "xyz"
		require.NoError(t, message.Validate())
	})
}
