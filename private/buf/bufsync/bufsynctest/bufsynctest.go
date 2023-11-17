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

package bufsynctest

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestHandler is a bufsync.Handler with a few helpful utilities for tests to set up
// and assert some state.
type TestHandler interface {
	bufsync.Handler
	SetSyncPoint(
		ctx context.Context,
		t *testing.T,
		targetModuleIdentity bufmoduleref.ModuleIdentity,
		branchName string,
		gitHash git.Hash,
	)
}

func RunTestSuite(t *testing.T, handlerProvider func() TestHandler) {
	t.Run("no_previous_sync_points", func(t *testing.T) {
		t.Parallel()
		handler := handlerProvider()
		testNoPreviousSyncPoints(t, handler, makeRunFunc(handler))
	})
	t.Run("put_tags", func(t *testing.T) {
		t.Parallel()
		handler := handlerProvider()
		testPutTags(t, handler, makeRunFunc(handler))
	})
	t.Run("duplicate_identities", func(t *testing.T) {
		t.Parallel()
		handler := handlerProvider()
		testDuplicateIdentities(t, handler, makeRunFunc(handler))
	})
}

// runFunc runs Plan and Sync on the provided Repository with the provided options, returning any error that occured along the way.
// If Plan errors, Sync is not invoked.
type runFunc func(t *testing.T, repo git.Repository, options ...bufsync.SyncerOption) (bufsync.ExecutionPlan, error)

func makeRunFunc(handler bufsync.Handler) runFunc {
	return func(t *testing.T, repo git.Repository, options ...bufsync.SyncerOption) (bufsync.ExecutionPlan, error) {
		syncer, err := bufsync.NewSyncer(
			zaptest.NewLogger(t),
			repo,
			storagegit.NewProvider(repo.Objects()),
			handler,
			options...,
		)
		require.NoError(t, err)
		plan, err := syncer.Plan(context.Background())
		if err != nil {
			return plan, err
		}
		return plan, syncer.Sync(context.Background())
	}
}
