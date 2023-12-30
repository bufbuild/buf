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

package bufmodulestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/zap"
)

// ModuleStore reads and writes ModulesDatas.
type CommitStore interface {
	// GetCommitsForModuleKey gets the Commits from the store for the ModuleKeys.
	//
	// Returns the found Commits, and the input ModuleKeys that were not found, each
	// ordered by the order of the input ModuleKeys.
	GetCommitsForModuleKeys(context.Context, []bufmodule.ModuleKey) (
		foundCommits []bufmodule.Commit,
		notFoundModuleKeys []bufmodule.ModuleKey,
		err error,
	)

	// Put puts the Commits to the store.
	PutCommits(ctx context.Context, commits []bufmodule.Commit) error
}

// NewCommitStore returns a new CommitStore for the given bucket.
//
// It is assumed that the CommitStore has complete control of the bucket.
//
// This is typically used to interact with a cache directory.
func NewCommitStore(
	logger *zap.Logger,
	bucket storage.ReadWriteBucket,
) CommitStore {
	return newCommitStore(logger, bucket)
}

/// *** PRIVATE ***

type commitStore struct {
	logger *zap.Logger
	bucket storage.ReadWriteBucket
}

func newCommitStore(
	logger *zap.Logger,
	bucket storage.ReadWriteBucket,
) *commitStore {
	return &commitStore{
		logger: logger,
		bucket: bucket,
	}
}

func (p *commitStore) GetCommitsForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) ([]bufmodule.Commit, []bufmodule.ModuleKey, error) {
	var foundCommits []bufmodule.Commit
	var notFoundModuleKeys []bufmodule.ModuleKey
	for _, moduleKey := range moduleKeys {
		commit, err := p.getCommitForModuleKey(ctx, moduleKey)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, nil, err
			}
			notFoundModuleKeys = append(notFoundModuleKeys, moduleKey)
		} else {
			foundCommits = append(foundCommits, commit)
		}
	}
	return foundCommits, notFoundModuleKeys, nil
}

func (p *commitStore) PutCommits(
	ctx context.Context,
	commits []bufmodule.Commit,
) error {
	for _, commit := range commits {
		if err := p.putCommit(ctx, commit); err != nil {
			return err
		}
	}
	return nil
}

func (p *commitStore) getCommitForModuleKey(
	ctx context.Context,
	moduleKey bufmodule.ModuleKey,
) (bufmodule.Commit, error) {
	bucket := p.getReadWriteBucketForDir(moduleKey)
	path := getCommitStoreFilePath(moduleKey)
	data, err := storage.ReadPath(ctx, bucket, path)
	p.logDebugModuleKey(
		moduleKey,
		"commit store get file",
		zap.Bool("found", err == nil),
		zap.Error(err),
	)
	if err != nil {
		return nil, err
	}
	var externalCommit externalCommit
	if err := json.Unmarshal(data, &externalCommit); err != nil {
		return nil, p.deleteInvalidCommitFile(ctx, moduleKey, bucket, path, "corrupted", err)
	}
	if !externalCommit.isComplete() {
		return nil, p.deleteInvalidCommitFile(ctx, moduleKey, bucket, path, "incomplete", err)
	}
	digest, err := bufmodule.ParseDigest(externalCommit.Digest)
	if err != nil {
		return nil, p.deleteInvalidCommitFile(ctx, moduleKey, bucket, path, "invalid digest", err)
	}
	return bufmodule.NewCommit(
		moduleKey,
		func() (time.Time, error) {
			return externalCommit.CreateTime, nil
		},
		bufmodule.CommitWithReceivedDigest(digest),
	), nil
}

func (p *commitStore) putCommit(
	ctx context.Context,
	commit bufmodule.Commit,
) (retErr error) {
	createTime, err := commit.CreateTime()
	if err != nil {
		return err
	}
	moduleKey := commit.ModuleKey()
	digest, err := moduleKey.Digest()
	if err != nil {
		return err
	}
	bucket := p.getReadWriteBucketForDir(moduleKey)
	path := getCommitStoreFilePath(moduleKey)
	externalCommit := externalCommit{
		CreateTime: createTime,
		Digest:     digest.String(),
	}
	if !externalCommit.isComplete() {
		return syserror.Newf("external commit is incomplete: %+v", externalCommit)
	}
	data, err := json.Marshal(externalCommit)
	if err != nil {
		return err
	}
	return storage.PutPath(ctx, bucket, path, data, storage.PutWithAtomic())
}

func (p *commitStore) getReadWriteBucketForDir(
	moduleKey bufmodule.ModuleKey,
) storage.ReadWriteBucket {
	dirPath := getCommitStoreDirPath(moduleKey)
	p.logDebugModuleKey(
		moduleKey,
		"commit store dir read write bucket",
		zap.String("dirPath", dirPath),
	)
	return storage.MapReadWriteBucket(p.bucket, storage.MapOnPrefix(dirPath))
}

func (p *commitStore) deleteInvalidCommitFile(
	ctx context.Context,
	moduleKey bufmodule.ModuleKey,
	bucket storage.WriteBucket,
	path string,
	invalidReason string,
	invalidErr error,
) error {
	p.logDebugModuleKey(
		moduleKey,
		fmt.Sprintf("commit store %s commit file", invalidReason),
		zap.Error(invalidErr),
	)
	// Attempt to delete file as it is missing information.
	if err := bucket.Delete(ctx, path); err != nil {
		// Otherwise ignore error.
		p.logDebugModuleKey(
			moduleKey,
			fmt.Sprintf("commit store could not delete %s commit file", invalidReason),
			zap.Error(err),
		)
	}
	// This will act as if the file is not found
	return &fs.PathError{Op: "read", Path: path, Err: fs.ErrNotExist}
}

func (p *commitStore) logDebugModuleKey(moduleKey bufmodule.ModuleKey, message string, fields ...zap.Field) {
	logDebugModuleKey(p.logger, moduleKey, message, fields...)
}

// Returns the directory path within the store for the module.
//
// This is "registry/owner/name", e.g. the module "buf.build/acme/weather" will return "buf.build/acme/weather".
func getCommitStoreDirPath(moduleKey bufmodule.ModuleKey) string {
	return normalpath.Join(
		moduleKey.ModuleFullName().Registry(),
		moduleKey.ModuleFullName().Owner(),
		moduleKey.ModuleFullName().Name(),
	)
}

// Returns the file path within the directory.
//
// This is "commitID.json", e.g. the commit "12345" will return "12345.json".
func getCommitStoreFilePath(moduleKey bufmodule.ModuleKey) string {
	return moduleKey.CommitID() + ".json"
}

// externalCommit is the store representation of a Commit.
//
// We could use a protobuf Message for this.
//
// Note that we do not want to use modulev1beta1.Commit. This would hard-link the API
// and persistence layers, and a bufmodule.Commit does not have all the information that
// a modulev1beta1.Commit has.
type externalCommit struct {
	CreateTime time.Time `json:"create_time,omitempty" yaml:"create_time,omitempty"`
	Digest     string    `json:"digest,omitempty" yaml:"digest,omitempty"`
}

// isComplete returns true if all the information we currently expect to be on
// an externalCommit is present.
//
// If we add to externalCommit over time, old values will be incomplete, and we
// will auto-evict them from the store.
func (e externalCommit) isComplete() bool {
	return !e.CreateTime.IsZero() && e.Digest != ""
}
