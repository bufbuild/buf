// Copyright 2020-2026 Buf Technologies, Inc.
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

package buflsp

import (
	"context"
	"log/slog"
	"sync"

	"buf.build/go/standard/xlog/xslog"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/google/uuid"
)

// versionCache caches "latest" lookups for BSR modules and curated plugins for
// the lifetime of the LSP session. Entries are populated lazily on first use
// by the inlay hint paths and never expire — the assumption is that module
// commits and plugin versions don't move often enough during a single editor
// session to matter.
//
// Errors (network, auth, timeouts) are intentionally NOT cached so a transient
// failure doesn't permanently disable hints; the next inlay hint request
// re-attempts the fetch. Errors are logged at debug level and never surfaced
// to the user as diagnostics — inlay hints are an enhancement, not a feature
// users should debug when the BSR is unreachable.
type versionCache struct {
	logger *slog.Logger

	// moduleCommits caches latest commit IDs keyed by module full name
	// (e.g. "buf.build/foo/bar"). Hits indicate a successful lookup.
	moduleCommits sync.Map // string -> uuid.UUID
	// pluginVersions caches latest version strings keyed by plugin full name.
	pluginVersions sync.Map // string -> string

	// fetchLocks dedups concurrent fetches for the same key by holding a
	// per-key sync.Mutex. The first caller fetches; subsequent callers wait
	// (or skip if the cache populated while they waited).
	fetchLocks sync.Map // string -> *sync.Mutex
}

func newVersionCache(logger *slog.Logger) *versionCache {
	return &versionCache{logger: logger}
}

// GetModuleCommit returns the cached latest commit for the given module full
// name, or uuid.Nil and false if the module has not been resolved yet.
func (c *versionCache) GetModuleCommit(fullName string) (uuid.UUID, bool) {
	v, ok := c.moduleCommits.Load(fullName)
	if !ok {
		return uuid.Nil, false
	}
	commit, _ := v.(uuid.UUID)
	return commit, true
}

// GetPluginVersion returns the cached latest version string for the given
// plugin full name, or "" and false if the plugin has not been resolved yet.
func (c *versionCache) GetPluginVersion(fullName string) (string, bool) {
	v, ok := c.pluginVersions.Load(fullName)
	if !ok {
		return "", false
	}
	version, _ := v.(string)
	return version, true
}

// FetchModuleCommits resolves the latest commit for each ref via the provider
// and stores successful results in the cache. Refs already cached are skipped.
// A returned bool reports whether at least one entry was newly populated, so
// callers can decide whether to send an inlay hint refresh notification.
//
// Provider errors are logged at debug level and otherwise swallowed; the cache
// is left untouched for failed entries so a future call retries them.
func (c *versionCache) FetchModuleCommits(
	ctx context.Context,
	provider bufmodule.ModuleKeyProvider,
	refs []bufparse.Ref,
	digestType bufmodule.DigestType,
) bool {
	uncached := c.uncachedRefs(refs)
	if len(uncached) == 0 {
		return false
	}
	keys, err := provider.GetModuleKeysForModuleRefs(ctx, uncached, digestType)
	if err != nil {
		c.logger.DebugContext(ctx, "version cache: fetching module commits failed",
			slog.Int("ref_count", len(uncached)),
			xslog.ErrorAttr(err),
		)
		return false
	}
	populated := false
	for _, key := range keys {
		fullName := key.FullName().String()
		if _, loaded := c.moduleCommits.LoadOrStore(fullName, key.CommitID()); !loaded {
			populated = true
		}
	}
	return populated
}

// FetchPluginVersion resolves the latest version of a single plugin via the
// provider and stores a successful result in the cache. If the entry is
// already cached or the fetch fails, the cache is left untouched. Returns
// true when a new entry was added.
func (c *versionCache) FetchPluginVersion(
	ctx context.Context,
	provider CuratedPluginVersionProvider,
	registry, owner, plugin string,
) bool {
	fullName := registry + "/" + owner + "/" + plugin
	if _, ok := c.pluginVersions.Load(fullName); ok {
		return false
	}
	mu := c.lockFor(fullName)
	mu.Lock()
	defer mu.Unlock()
	if _, ok := c.pluginVersions.Load(fullName); ok {
		return false
	}
	version, err := provider.GetLatestVersion(ctx, registry, owner, plugin)
	if err != nil {
		c.logger.DebugContext(ctx, "version cache: fetching plugin version failed",
			slog.String("plugin", fullName),
			xslog.ErrorAttr(err),
		)
		return false
	}
	if version == "" {
		// No published versions; treat like a miss. A later push to the BSR
		// would only become visible after restart — acceptable trade-off for
		// not caching empty/sentinel values.
		return false
	}
	c.pluginVersions.Store(fullName, version)
	return true
}

// uncachedRefs filters refs down to those whose full name is not yet in the
// module cache. The returned slice preserves input order.
func (c *versionCache) uncachedRefs(refs []bufparse.Ref) []bufparse.Ref {
	out := refs[:0:0]
	for _, ref := range refs {
		if _, cached := c.moduleCommits.Load(ref.FullName().String()); !cached {
			out = append(out, ref)
		}
	}
	return out
}

// lockFor returns a per-key mutex used to dedup concurrent fetches of the same
// key. A new mutex is created on first access.
func (c *versionCache) lockFor(key string) *sync.Mutex {
	if v, ok := c.fetchLocks.Load(key); ok {
		mu, _ := v.(*sync.Mutex)
		return mu
	}
	mu := new(sync.Mutex)
	actual, _ := c.fetchLocks.LoadOrStore(key, mu)
	got, _ := actual.(*sync.Mutex)
	return got
}
