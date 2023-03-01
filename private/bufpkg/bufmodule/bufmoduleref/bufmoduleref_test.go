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

package bufmoduleref

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/buflock"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPutDependencyModulePinsToBucket(t *testing.T) {
	digester, err := manifest.NewDigester(manifest.DigestTypeShake256)
	require.NoError(t, err)
	nullDigest, err := digester.Digest(&bytes.Buffer{})
	require.NoError(t, err)
	const lockV1Header = buflock.Header + "version: v1\n"
	testPutDependencyModulePinsToBucket(
		t,
		"no pins",
		[]ModulePin{},
		lockV1Header,
	)
	testPutDependencyModulePinsToBucket(
		t,
		"one pin",
		[]ModulePin{
			pin(t, "repository"),
		},
		lockV1Header+deps(
			t,
			buflock.ExternalConfigDependencyV1{
				Remote:     "remote",
				Owner:      "owner",
				Repository: "repository",
				Commit:     "commit",
				Digest:     nullDigest.String(),
			},
		),
	)
	testPutDependencyModulePinsToBucket(
		t,
		"two pins",
		[]ModulePin{
			pin(t, "repo-a"),
			pin(t, "repo-b"),
		},
		lockV1Header+deps(
			t,
			buflock.ExternalConfigDependencyV1{
				Remote:     "remote",
				Owner:      "owner",
				Repository: "repo-a",
				Commit:     "commit",
				Digest:     nullDigest.String(),
			},
			buflock.ExternalConfigDependencyV1{
				Remote:     "remote",
				Owner:      "owner",
				Repository: "repo-b",
				Commit:     "commit",
				Digest:     nullDigest.String(),
			},
		),
	)
}

func TestDependencyModulePinsForBucket(t *testing.T) {
	testDependencyModulePinsForBucket(
		t,
		"no pins",
		[]ModulePin{},
	)
	testDependencyModulePinsForBucket(
		t,
		"one pin",
		[]ModulePin{
			pin(t, "repo"),
		},
	)
	testDependencyModulePinsForBucket(
		t,
		"two pins",
		[]ModulePin{
			pin(t, "repo-a"),
			pin(t, "repo-b"),
		},
	)
}

func TestValidateModulePinsConsistentDigests(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	modulePin := pin(t, "repo")
	bucket := bucketWithBufLock(t, modulePin)
	// Pin matches all fields
	require.NoError(t, ValidateModulePinsConsistentDigests(ctx, bucket, []ModulePin{modulePin}))
	// Change digest and nothing else
	modulePinChangedDigest, err := NewModulePin(
		modulePin.Remote(),
		modulePin.Owner(),
		modulePin.Repository(),
		modulePin.Branch(),
		modulePin.Commit(),
		createDigest(t, []byte("abc")),
		modulePin.CreateTime(),
	)
	require.NoError(t, err)
	err = ValidateModulePinsConsistentDigests(ctx, bucket, []ModulePin{modulePinChangedDigest})
	var digestChangedErr *DigestChangedError
	assert.ErrorAs(t, err, &digestChangedErr)
	assert.Equal(t, modulePin.Digest(), digestChangedErr.CurrentDigest)
	assert.Equal(t, modulePinChangedDigest.Digest(), digestChangedErr.UpdatedPin.Digest())
	// Change commit and digest - this is ok
	modulePinChangedCommitAndDigest, err := NewModulePin(
		modulePin.Remote(),
		modulePin.Owner(),
		modulePin.Repository(),
		modulePin.Branch(),
		"updatedcommit",
		createDigest(t, []byte("abc")),
		modulePin.CreateTime(),
	)
	require.NoError(t, err)
	require.NoError(t, ValidateModulePinsConsistentDigests(ctx, bucket, []ModulePin{modulePinChangedCommitAndDigest}))
}

func bucketWithBufLock(t *testing.T, pin ModulePin) storage.ReadWriteBucket {
	t.Helper()
	bufLock := &buflock.Config{
		Dependencies: []buflock.Dependency{
			{
				Remote:     pin.Remote(),
				Owner:      pin.Owner(),
				Repository: pin.Repository(),
				Commit:     pin.Commit(),
				Digest:     pin.Digest(),
			},
		},
	}
	bucket, err := storagemem.NewReadWriteBucketWithOptions()
	require.NoError(t, err)
	err = buflock.WriteConfig(context.Background(), bucket, bufLock)
	require.NoError(t, err)
	return bucket
}

func pin(t *testing.T, repository string) ModulePin {
	t.Helper()
	pin, err := NewModulePin(
		"remote",
		"owner",
		repository,
		"branch",
		"commit",
		createDigest(t, []byte{}),
		time.Now(),
	)
	require.NoError(t, err)
	return pin
}

func createDigest(t *testing.T, b []byte) string {
	t.Helper()
	digester, err := manifest.NewDigester(manifest.DigestTypeShake256)
	require.NoError(t, err)
	digest, err := digester.Digest(bytes.NewReader(b))
	require.NoError(t, err)
	return digest.String()
}

func deps(
	t *testing.T,
	dependencies ...buflock.ExternalConfigDependencyV1,
) string {
	deps, err := encoding.MarshalYAML(
		&buflock.ExternalConfigV1{Deps: dependencies},
	)
	require.NoError(t, err)
	return string(deps)
}

func testPutDependencyModulePinsToBucket(
	t *testing.T,
	desc string,
	modulePins []ModulePin,
	buflock string,
) {
	t.Helper()
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		writeBucket := storagemem.NewReadWriteBucket()
		err := PutDependencyModulePinsToBucket(
			ctx,
			writeBucket,
			modulePins,
		)
		require.NoError(t, err)
		file, err := writeBucket.Get(ctx, "buf.lock")
		require.NoError(t, err)
		defer file.Close()
		actual, err := io.ReadAll(file)
		require.NoError(t, err)
		assert.Equal(t, buflock, string(actual))
	})
}

func testDependencyModulePinsForBucket(
	t *testing.T,
	desc string,
	modulePins []ModulePin,
) {
	t.Helper()
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		writeBucket := storagemem.NewReadWriteBucket()
		// we can assume put works given we've tested put in isolation
		err := PutDependencyModulePinsToBucket(
			ctx,
			writeBucket,
			modulePins,
		)
		require.NoError(t, err)
		retPins, err := DependencyModulePinsForBucket(ctx, writeBucket)
		require.NoError(t, err)
		assert.Equal(t, len(modulePins), len(retPins))
		for i, actual := range retPins {
			assert.Equal(t, modulePins[i].Remote(), actual.Remote())
			assert.Equal(t, modulePins[i].Owner(), actual.Owner())
			assert.Equal(t, modulePins[i].Repository(), actual.Repository())
			assert.Equal(t, modulePins[i].Commit(), actual.Commit())
			assert.Equal(t, modulePins[i].Digest(), actual.Digest())
			assert.Equal(t, "", actual.Branch()) // branch is never consumed
		}
	})
}
