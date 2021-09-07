package bufmodulecache

import (
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
)

// newCacheKey returns the key associated with the given module pin.
// The cache key is of the form: remote/owner/repository/commit.
func newCacheKey(modulePin bufmodule.ModulePin) string {
	return normalpath.Join(modulePin.Remote(), modulePin.Owner(), modulePin.Repository(), modulePin.Commit())
}
