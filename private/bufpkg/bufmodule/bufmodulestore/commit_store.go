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
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"go.uber.org/zap"
)

var externalCommitVersion = "v1"

// CommitStore reads and writes Commits.
type CommitStore interface {
	// GetCommitsForModuleKeys gets the Commits from the store for the ModuleKeys.
	//
	// Returns the found Commits, and the input IDs that were not found, each
	// ordered by the order of the input IDs.
	GetCommitsForModuleKeys(ctx context.Context, moduleKeys []bufmodule.ModuleKey) (
		foundCommits []bufmodule.Commit,
		notFoundModuleKeys []bufmodule.ModuleKey,
		err error,
	)
	// GetCommitsForCommitKeys gets the Commits from the store for the CommitKeys.
	//
	// Returns the found Commits, and the input IDs that were not found, each
	// ordered by the order of the input IDs.
	GetCommitsForCommitKeys(ctx context.Context, commitKeys []bufmodule.CommitKey) (
		foundCommits []bufmodule.Commit,
		notFoundCommitKeys []bufmodule.CommitKey,
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
		expectedDigest, err := moduleKey.Digest()
		if err != nil {
			return nil, nil, err
		}
		commitKey, err := bufmodule.ModuleKeyToCommitKey(moduleKey)
		if err != nil {
			return nil, nil, err
		}
		commit, err := p.getCommitForCommitKey(ctx, commitKey, expectedDigest)
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

func (p *commitStore) GetCommitsForCommitKeys(
	ctx context.Context,
	commitKeys []bufmodule.CommitKey,
) ([]bufmodule.Commit, []bufmodule.CommitKey, error) {
	var foundCommits []bufmodule.Commit
	var notFoundCommitKeys []bufmodule.CommitKey
	for _, commitKey := range commitKeys {
		commit, err := p.getCommitForCommitKey(ctx, commitKey, nil)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, nil, err
			}
			notFoundCommitKeys = append(notFoundCommitKeys, commitKey)
		} else {
			foundCommits = append(foundCommits, commit)
		}
	}
	return foundCommits, notFoundCommitKeys, nil
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

func (p *commitStore) getCommitForCommitKey(
	ctx context.Context,
	commitKey bufmodule.CommitKey,
	// may be nil
	expectedDigest bufmodule.Digest,
) (_ bufmodule.Commit, retErr error) {
	bucket := p.getReadWriteBucketForDir(commitKey)
	path := getCommitStoreFilePath(commitKey)
	data, err := storage.ReadPath(ctx, bucket, path)
	p.logDebugCommitKey(
		commitKey,
		"commit store get file",
		zap.Bool("found", err == nil),
		zap.Error(err),
	)
	if err != nil {
		return nil, err
	}
	var invalidReason string
	defer func() {
		if retErr != nil {
			retErr = p.deleteInvalidCommitFile(ctx, commitKey, bucket, path, invalidReason, retErr)
		}
	}()
	var externalCommit externalCommit
	if err := json.Unmarshal(data, &externalCommit); err != nil {
		invalidReason = "corrupted"
		return nil, err
	}
	if !externalCommit.isValid() {
		invalidReason = "invalid"
		return nil, err
	}
	digest, err := bufmodule.ParseDigest(externalCommit.Digest)
	if err != nil {
		invalidReason = "invalid digest"
		return nil, err
	}
	if commitKey.DigestType() != digest.Type() {
		invalidReason = "mismatched digest type"
		return nil, err
	}
	moduleFullName, err := bufmodule.NewModuleFullName(
		commitKey.Registry(),
		externalCommit.Owner,
		externalCommit.Module,
	)
	if err != nil {
		invalidReason = "invalid module name"
		return nil, err
	}
	moduleKey, err := bufmodule.NewModuleKey(
		moduleFullName,
		commitKey.CommitID(),
		func() (bufmodule.Digest, error) {
			return digest, nil
		},
	)
	if err != nil {
		invalidReason = "invalid module key"
		return nil, err
	}
	return bufmodule.NewCommit(
		moduleKey,
		func() (time.Time, error) {
			return externalCommit.CreateTime, nil
		},
		bufmodule.CommitWithExpectedDigest(expectedDigest),
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
	commitKey, err := bufmodule.ModuleKeyToCommitKey(moduleKey)
	if err != nil {
		return err
	}
	bucket := p.getReadWriteBucketForDir(commitKey)
	path := getCommitStoreFilePath(commitKey)
	externalCommit := externalCommit{
		Version:    externalCommitVersion,
		Owner:      moduleKey.ModuleFullName().Owner(),
		Module:     moduleKey.ModuleFullName().Name(),
		CreateTime: createTime,
		Digest:     digest.String(),
	}
	if !externalCommit.isValid() {
		return syserror.Newf("external commit is invalid: %+v", externalCommit)
	}
	data, err := json.Marshal(externalCommit)
	if err != nil {
		return err
	}
	return storage.PutPath(ctx, bucket, path, data, storage.PutWithAtomic())
}

func (p *commitStore) getReadWriteBucketForDir(commitKey bufmodule.CommitKey) storage.ReadWriteBucket {
	dirPath := getCommitStoreDirPath(commitKey)
	p.logDebugCommitKey(
		commitKey,
		"commit store dir read write bucket",
		zap.String("dirPath", dirPath),
	)
	return storage.MapReadWriteBucket(p.bucket, storage.MapOnPrefix(dirPath))
}

func (p *commitStore) deleteInvalidCommitFile(
	ctx context.Context,
	commitKey bufmodule.CommitKey,
	bucket storage.WriteBucket,
	path string,
	invalidReason string,
	invalidErr error,
) error {
	p.logDebugCommitKey(
		commitKey,
		fmt.Sprintf("commit store %s commit file", invalidReason),
		zap.Error(invalidErr),
	)
	// Attempt to delete file as it is missing information.
	if err := bucket.Delete(ctx, path); err != nil {
		// Otherwise ignore error.
		p.logDebugCommitKey(
			commitKey,
			fmt.Sprintf("commit store could not delete %s commit file", invalidReason),
			zap.Error(err),
		)
	}
	// This will act as if the file is not found
	return &fs.PathError{Op: "read", Path: path, Err: fs.ErrNotExist}
}

func (p *commitStore) logDebugCommitKey(commitKey bufmodule.CommitKey, message string, fields ...zap.Field) {
	logDebugCommitKey(p.logger, commitKey, message, fields...)
}

// Returns the directory path within the store for the Commit.
//
// This is "digestType/registry, i.e. "b5/buf.build".
func getCommitStoreDirPath(
	commitKey bufmodule.CommitKey,
) string {
	return normalpath.Join(
		commitKey.DigestType().String(),
		commitKey.Registry(),
	)
}

// Returns the file path within the directory.
//
// This is "dashlessCommitID.json", e.g. the commit "12345-abcde" will return "12345abcde.json".
func getCommitStoreFilePath(commitKey bufmodule.CommitKey) string {
	return uuidutil.ToDashless(commitKey.CommitID()) + ".json"
}

// externalCommit is the store representation of a Commit.
//
// We could use a protobuf Message for this.
//
// Note that we do not want to use registry-proto Commits. This would hard-link the API
// and persistence layers, and a bufmodule.Commit does not have all the information that
// a registry-proto Commit has.
type externalCommit struct {
	Version    string    `json:"version,omitempty" yaml:"version,omitempty"`
	Owner      string    `json:"owner,omitempty" yaml:"owner,omitempty"`
	Module     string    `json:"module,omitempty" yaml:"module,omitempty"`
	CreateTime time.Time `json:"create_time,omitempty" yaml:"create_time,omitempty"`
	Digest     string    `json:"digest,omitempty" yaml:"digest,omitempty"`
}

// isValid returns true if all the information we currently expect to be on
// an externalCommit is present, and the version matches.
//
// If we add to externalCommit over time or change the version, old values will be
// incomplete, and we will auto-evict them from the store.
func (e externalCommit) isValid() bool {
	return e.Version == externalCommitVersion &&
		e.Owner != "" &&
		e.Module != "" &&
		!e.CreateTime.IsZero() &&
		e.Digest != ""
}
