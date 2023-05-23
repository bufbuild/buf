package storagegit

import (
	"testing"

	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"github.com/bufbuild/buf/private/pkg/storage/storagetesting"
	"github.com/stretchr/testify/require"
)

func TestNewBucketAtTreeHash(t *testing.T) {
	repo := gittest.ScaffoldGitRepository(t)
	provider := NewProvider(repo.Reader)
	// get last commit
	var commit git.Commit
	require.NoError(t, repo.CommitIterator.ForEachCommit(repo.CommitIterator.BaseBranch(), func(c git.Commit) error {
		commit = c
		return nil
	}))
	require.NotNil(t, commit)
	bucket, err := provider.NewReadBucketForTreeHash(commit.Tree())
	require.NoError(t, err)

	storagetesting.AssertPaths(
		t,
		bucket,
		"",
		"proto/acme/grocerystore/v1/c.proto",
		"proto/acme/grocerystore/v1/d.proto",
		"proto/acme/grocerystore/v1/g.proto",
		"proto/acme/grocerystore/v1/h.proto",
		"proto/acme/petstore/v1/a.proto",
		"proto/acme/petstore/v1/b.proto",
		"proto/acme/petstore/v1/e.proto",
		"proto/acme/petstore/v1/f.proto",
		"proto/buf.yaml",
		"randomBinary",
	)
	storagetesting.AssertObjectInfo(
		t,
		bucket,
		"proto/acme/grocerystore/v1/c.proto",
		"proto/acme/grocerystore/v1/c.proto",
	)
	storagetesting.AssertNotExist(t, bucket, "random-path")
	storagetesting.AssertPathToContent(
		t,
		bucket,
		"",
		map[string]string{
			"proto/acme/grocerystore/v1/c.proto": "toysrus",
			"proto/acme/grocerystore/v1/d.proto": "petsrus",
			"proto/acme/grocerystore/v1/g.proto": "hamlet",
			"proto/acme/grocerystore/v1/h.proto": "bethoven",
			"proto/acme/petstore/v1/a.proto":     "cats",
			"proto/acme/petstore/v1/b.proto":     "animals",
			"proto/acme/petstore/v1/e.proto":     "loblaws",
			"proto/acme/petstore/v1/f.proto":     "merchant of venice",
			"proto/buf.yaml":                     "some buf.yaml",
			"randomBinary":                       "some executable",
		},
	)
}
