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

package bufpolicy

import (
	"context"
	"io/fs"
)

var (
	// NopPolicyDataProvider is a no-op PolicyDataProvider.
	NopPolicyDataProvider PolicyDataProvider = nopPolicyDataProvider{}
)

// PolicyDataProvider provides PolicyDatas.
type PolicyDataProvider interface {
	// GetPolicyDatasForPolicyKeys gets the PolicyDatas for the PolicyKeys.
	//
	// Returned PolicyDatas will be in the same order as the input PolicyKeys.
	//
	// The input PolicyKeys are expected to be unique by FullName. The implementation
	// may error if this is not the case.
	//
	// The input PolicyKeys are expected to have the same DigestType. The implementation
	// may error if this is not the case.
	//
	// If there is no error, the length of the PolicyDatas returned will match the length of the PolicyKeys.
	// If there is an error, no PolicyDatas will be returned.
	// If any PolicyKey is not found, an error with fs.ErrNotExist will be returned.
	GetPolicyDatasForPolicyKeys(context.Context, []PolicyKey) ([]PolicyData, error)
}

// *** PRIVATE ***

type nopPolicyDataProvider struct{}

func (nopPolicyDataProvider) GetPolicyDatasForPolicyKeys(
	context.Context,
	[]PolicyKey,
) ([]PolicyData, error) {
	return nil, fs.ErrNotExist
}
