package bufsync

import (
	"github.com/bufbuild/buf/private/pkg/git"
)

// syncableCommit holds a commit that needs to be synced.
type syncableCommit struct {
	commit git.Commit
	tags   []string
}

func newSyncableCommit(
	commit git.Commit,
	tags []string,
) *syncableCommit {
	return &syncableCommit{
		commit: commit,
		tags:   tags,
	}
}
