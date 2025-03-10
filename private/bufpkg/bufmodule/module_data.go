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

package bufmodule

import (
	"context"
	"sync"

	"github.com/bufbuild/buf/private/pkg/storage"
)

// ModuleData presents raw Module data read by ModuleKey.
//
// It is not a fully-formed Module; only ModuleSetBuilders (and ModuleSets) can provide Modules.
//
// A ModuleData generally represents the data on a Module read from the BSR API or a cache.
//
// Tamper-proofing is done as part of every function.
type ModuleData interface {
	// ModuleKey contains the ModuleKey that was used to download this ModuleData.
	//
	// The Digest from this ModuleKey is used for tamper-proofing. It will be checked against
	// the actual data downloaded before Bucket() or DepModuleKeys() returns.
	ModuleKey() ModuleKey
	// Bucket returns a Bucket of the Module's files.
	//
	// This bucket is not self-contained - it requires the files from dependencies to be so.
	//
	// This bucket will only contain module files - it will be filtered via NewModuleData.
	Bucket() (storage.ReadBucket, error)
	// DepModuleKeys returns the dependencies for this specific Module.
	//
	// The dependencies are the same as that would appear in the v1 buf.lock file.
	// These include all direct and transitive dependencies. A Module constructed
	// from this ModuleData as the target will require all Modules referenced by
	// its DepModuleKeys to be present in the ModuleSet.
	//
	// This is used for digest calculations.
	DepModuleKeys() ([]ModuleKey, error)

	// V1Beta1OrV1BufYAMLObjectData gets the v1beta1 or v1 buf.yaml ObjectData.
	//
	// This may not be present. It will only be potentially present for v1 buf.yaml files.
	//
	// This is used for digest calculations. It is not used otherwise.
	V1Beta1OrV1BufYAMLObjectData() (ObjectData, error)
	// V1Beta1OrV1BufLockObjectData gets the v1beta1 or v1 buf.lock ObjectData.
	//
	// This may not be present. It will only be potentially present for v1 buf.lock files.
	//
	// This is used for digest calculations. It is not used otherwise.
	V1Beta1OrV1BufLockObjectData() (ObjectData, error)

	isModuleData()
}

// NewModuleData returns a new ModuleData.
//
// getBucket and getDepModuleKeys are meant to be lazily-loaded functions where possible.
//
// It is OK for getBucket to return a bucket that has extra files that are not part of the Module, this
// bucket will be filtered as part of this function.
func NewModuleData(
	ctx context.Context,
	moduleKey ModuleKey,
	getBucket func() (storage.ReadBucket, error),
	getDepModuleKeys func() ([]ModuleKey, error),
	getV1BufYAMLObjectData func() (ObjectData, error),
	getV1BufLockObjectData func() (ObjectData, error),
) ModuleData {
	return newModuleData(
		ctx,
		moduleKey,
		getBucket,
		getDepModuleKeys,
		getV1BufYAMLObjectData,
		getV1BufLockObjectData,
	)
}

// *** PRIVATE ***

type moduleData struct {
	moduleKey              ModuleKey
	getBucket              func() (storage.ReadBucket, error)
	getDepModuleKeys       func() ([]ModuleKey, error)
	getV1BufYAMLObjectData func() (ObjectData, error)
	getV1BufLockObjectData func() (ObjectData, error)

	checkDigest func() error
}

func newModuleData(
	ctx context.Context,
	moduleKey ModuleKey,
	getBucket func() (storage.ReadBucket, error),
	getDepModuleKeys func() ([]ModuleKey, error),
	getV1BufYAMLObjectData func() (ObjectData, error),
	getV1BufLockObjectData func() (ObjectData, error),
) *moduleData {
	moduleData := &moduleData{
		moduleKey:              moduleKey,
		getBucket:              getSyncOnceValuesGetBucketWithStorageMatcherApplied(ctx, getBucket),
		getDepModuleKeys:       sync.OnceValues(getDepModuleKeys),
		getV1BufYAMLObjectData: sync.OnceValues(getV1BufYAMLObjectData),
		getV1BufLockObjectData: sync.OnceValues(getV1BufLockObjectData),
	}
	moduleData.checkDigest = sync.OnceValue(
		func() error {
			// We have to use the get.* functions so that we don't invoke checkDigest.
			bucket, err := moduleData.getBucket()
			if err != nil {
				return err
			}
			expectedDigest, err := moduleKey.Digest()
			if err != nil {
				return err
			}
			var actualDigest Digest
			switch expectedDigest.Type() {
			case DigestTypeB4:
				// Call unexported func instead of exported method to avoid deadlocking on checking the digest again.
				v1BufYAMLObjectData, err := moduleData.getV1BufYAMLObjectData()
				if err != nil {
					return err
				}
				// Call unexported func instead of exported method to avoid deadlocking on checking the digest again.
				v1BufLockObjectData, err := moduleData.getV1BufLockObjectData()
				if err != nil {
					return err
				}
				actualDigest, err = getB4Digest(ctx, bucket, v1BufYAMLObjectData, v1BufLockObjectData)
				if err != nil {
					return err
				}
			case DigestTypeB5:
				// Call unexported func instead of exported method to avoid deadlocking on checking the digest again.
				depModuleKeys, err := moduleData.getDepModuleKeys()
				if err != nil {
					return err
				}
				// The B5 digest is calculated based on the dependencies.
				// The dependencies are not required to be resolved to a Module to calculate this Digest.
				// Each ModuleKey includes the expected Digest which is validated when loading that dependencies ModuleData, if needed.
				actualDigest, err = getB5DigestForBucketAndDepModuleKeys(ctx, bucket, depModuleKeys)
				if err != nil {
					return err
				}
			}
			if !DigestEqual(expectedDigest, actualDigest) {
				return &DigestMismatchError{
					FullName:       moduleKey.FullName(),
					CommitID:       moduleKey.CommitID(),
					ExpectedDigest: expectedDigest,
					ActualDigest:   actualDigest,
				}
			}
			return nil
		},
	)
	return moduleData
}

func (m *moduleData) ModuleKey() ModuleKey {
	return m.moduleKey
}

func (m *moduleData) Bucket() (storage.ReadBucket, error) {
	if err := m.checkDigest(); err != nil {
		return nil, err
	}
	return m.getBucket()
}

func (m *moduleData) DepModuleKeys() ([]ModuleKey, error) {
	// Do we need to tamper-proof when getting deps? Probably yes - this is data that could be tampered with.
	//
	// Note that doing so kills some of our lazy-loading, as we call DepModuleKeys
	// in ModuleSetBuilder right away. However, we still do the lazy-loading here, in the case
	// where ModuleData is loaded outside of a ModuleSetBuilder and users may defer calling this
	// function if it is not needed.
	if err := m.checkDigest(); err != nil {
		return nil, err
	}
	return m.getDepModuleKeys()
}

func (m *moduleData) V1Beta1OrV1BufYAMLObjectData() (ObjectData, error) {
	if err := m.checkDigest(); err != nil {
		return nil, err
	}
	return m.getV1BufYAMLObjectData()
}

func (m *moduleData) V1Beta1OrV1BufLockObjectData() (ObjectData, error) {
	if err := m.checkDigest(); err != nil {
		return nil, err
	}
	return m.getV1BufLockObjectData()
}

func (*moduleData) isModuleData() {}
