package bufsync

import "github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"

type moduleTags struct {
	targetModuleIdentity bufmoduleref.ModuleIdentity
	taggedCommitsToSync  []TaggedCommit
}

func newModuleTags(
	targetModuleIdentity bufmoduleref.ModuleIdentity,
	taggedCommitsToSync []TaggedCommit,
) *moduleTags {
	return &moduleTags{
		targetModuleIdentity: targetModuleIdentity,
		taggedCommitsToSync:  taggedCommitsToSync,
	}
}

func (b *moduleTags) TargetModuleIdentity() bufmoduleref.ModuleIdentity {
	return b.targetModuleIdentity
}

func (b *moduleTags) TaggedCommitsToSync() []TaggedCommit {
	return b.taggedCommitsToSync
}

var _ ModuleTags = (*moduleTags)(nil)
