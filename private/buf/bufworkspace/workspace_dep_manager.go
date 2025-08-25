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

package bufworkspace

import (
	"context"
	"errors"
	"io/fs"
	"sort"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy/bufpolicyconfig"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// WorkspaceDepManager is a workspace that can be updated.
//
// A workspace can be updated if it was backed by a v2 buf.yaml, or a single, targeted, local
// Module from defaults or a v1beta1/v1 buf.yaml exists in the Workspace. Config overrides
// can also not be used, but this is enforced via the WorkspaceBucketOption/WorkspaceDepManagerBucketOption
// difference.
//
// buf.work.yamls are ignored when constructing an WorkspaceDepManager.
type WorkspaceDepManager interface {
	// BufLockFileDigestType returns the DigestType that the buf.lock file expects.
	BufLockFileDigestType() bufmodule.DigestType
	// ExistingBufLockFileDepModuleKeys returns the ModuleKeys from the buf.lock file.
	ExistingBufLockFileDepModuleKeys(ctx context.Context) ([]bufmodule.ModuleKey, error)
	// ExistingBufLockFileRemotePluginKeys returns the PluginKeys from the buf.lock file.
	ExistingBufLockFileRemotePluginKeys(ctx context.Context) ([]bufplugin.PluginKey, error)
	// ExistingBufLockFileRemotePolicyKeys returns the PolicyKeys from the buf.lock file.
	ExistingBufLockFileRemotePolicyKeys(ctx context.Context) ([]bufpolicy.PolicyKey, error)
	// ExistingBufLockFilePolicyNameToRemotePluginKeys returns the PluginKeys for each Policy name from the buf.lock file.
	ExistingBufLockFilePolicyNameToRemotePluginKeys(ctx context.Context) (map[string][]bufplugin.PluginKey, error)
	// UpdateBufLockFile updates the lock file that backs the Workspace to contain exactly
	// the given ModuleKeys and PluginKeys.
	//
	// If a buf.lock does not exist, one will be created.
	UpdateBufLockFile(
		ctx context.Context,
		depModuleKeys []bufmodule.ModuleKey,
		remotePluginKeys []bufplugin.PluginKey,
		remotePolicyKeys []bufpolicy.PolicyKey,
		policyNameToRemotePluginKeys map[string][]bufplugin.PluginKey,
	) error
	// ConfiguredDepModuleRefs returns the configured dependencies of the Workspace as ModuleRefs.
	//
	// These come from buf.yaml files.
	//
	// The ModuleRefs in this list will be unique by FullName. If there are two ModuleRefs
	// in the buf.yaml with the same FullName but different Refs, an error will be given
	// at workspace constructions. For example, with v1 buf.yaml, this is a union of the deps in
	// the buf.yaml files in the workspace. If different buf.yamls had different refs, an error
	// will be returned - we have no way to resolve what the user intended.
	//
	// Sorted.
	ConfiguredDepModuleRefs(ctx context.Context) ([]bufparse.Ref, error)
	// ConfiguredRemotePluginRefs returns the configured remote plugins of the Workspace as PluginRefs.
	//
	// These come from buf.yaml files.
	//
	// The PluginRefs in this list will be unique by FullName. If there are two PluginRefs
	// in the buf.yaml with the same FullName but different Refs, an error will be given
	// at workspace constructions.
	//
	// Sorted.
	ConfiguredRemotePluginRefs(ctx context.Context) ([]bufparse.Ref, error)
	// ConfiguredRemotePolicyRefs returns the configured remote plugins of the Workspace as PolicyRefs.
	//
	// These come from buf.yaml files.
	//
	// The PolicyRefs in this list will be unique by FullName. If there are two PolicyRefs
	// in the buf.yaml with the same FullName but different Refs, an error will be given
	// at workspace constructions.
	//
	// Sorted.
	ConfiguredRemotePolicyRefs(ctx context.Context) ([]bufparse.Ref, error)
	// ConfiguredLocalPolicyNameToRemotePluginRefs returns the configured remote plugins for each local policy of the Workspace.
	//
	// These come from buf.yaml files and the local buf.policy.yaml files.
	//
	// The PluginRefs for each Policy will be unique by FullName. If there are two PluginRefs
	// in the buf.yaml for a given Policy with the same FullName but different Refs, an error will be given
	// at workspace constructions.
	//
	// PluginRefs are sorted for each Policy.
	ConfiguredLocalPolicyNameToRemotePluginRefs(ctx context.Context) (map[string][]bufparse.Ref, error)

	isWorkspaceDepManager()
}

// *** PRIVATE ***

type workspaceDepManager struct {
	bucket storage.ReadWriteBucket
	// targetSubDirPath is the relative path within the bucket where a buf.yaml file should be and where a
	// buf.lock can be written.
	//
	// If isV2 is true, this will be "." - buf.yamls and buf.locks live at the root of the workspace.
	//
	// If isV2 is false, this will be the path to the single, local, targeted Module within the workspace
	// This is the only situation where we can do an update for a v1 buf.lock.
	targetSubDirPath string
	// If true, the workspace was created from v2 buf.yamls
	//
	// If false, the workspace was created from defaults, or v1beta1/v1 buf.yamls.
	//
	// This is used to determine what DigestType to use, and what version
	// of buf.lock to write.
	isV2 bool
}

func newWorkspaceDepManager(
	bucket storage.ReadWriteBucket,
	targetSubDirPath string,
	isV2 bool,
) *workspaceDepManager {
	return &workspaceDepManager{
		bucket:           bucket,
		targetSubDirPath: targetSubDirPath,
		isV2:             isV2,
	}
}

func (w *workspaceDepManager) ConfiguredDepModuleRefs(ctx context.Context) ([]bufparse.Ref, error) {
	bufYAMLFile, err := bufconfig.GetBufYAMLFileForPrefix(ctx, w.bucket, w.targetSubDirPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	if bufYAMLFile == nil {
		return nil, nil
	}
	switch fileVersion := bufYAMLFile.FileVersion(); fileVersion {
	case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
		if w.isV2 {
			return nil, syserror.Newf("buf.yaml at %q did had version %v but expected v1beta1, v1", w.targetSubDirPath, fileVersion)
		}
	case bufconfig.FileVersionV2:
		if !w.isV2 {
			return nil, syserror.Newf("buf.yaml at %q did had version %v but expected v2", w.targetSubDirPath, fileVersion)
		}
	default:
		return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
	}
	return bufYAMLFile.ConfiguredDepModuleRefs(), nil
}

func (w *workspaceDepManager) ConfiguredRemotePluginRefs(ctx context.Context) ([]bufparse.Ref, error) {
	bufYAMLFile, err := bufconfig.GetBufYAMLFileForPrefix(ctx, w.bucket, w.targetSubDirPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	if bufYAMLFile == nil {
		return nil, nil
	}
	switch fileVersion := bufYAMLFile.FileVersion(); fileVersion {
	case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
		if w.isV2 {
			return nil, syserror.Newf("buf.yaml at %q did had version %v but expected v1beta1, v1", w.targetSubDirPath, fileVersion)
		}
		// Plugins are not supported in versions less than v2.
		return nil, nil
	case bufconfig.FileVersionV2:
		if !w.isV2 {
			return nil, syserror.Newf("buf.yaml at %q did had version %v but expected v2", w.targetSubDirPath, fileVersion)
		}
	default:
		return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
	}
	pluginRefs := xslices.Filter(
		xslices.Map(
			bufYAMLFile.PluginConfigs(),
			func(value bufconfig.PluginConfig) bufparse.Ref {
				return value.Ref()
			},
		),
		func(value bufparse.Ref) bool {
			return value != nil
		},
	)
	sort.Slice(
		pluginRefs,
		func(i int, j int) bool {
			return pluginRefs[i].FullName().String() < pluginRefs[j].FullName().String()
		},
	)
	return pluginRefs, nil
}

func (w *workspaceDepManager) ConfiguredRemotePolicyRefs(ctx context.Context) ([]bufparse.Ref, error) {
	bufYAMLFile, err := bufconfig.GetBufYAMLFileForPrefix(ctx, w.bucket, w.targetSubDirPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	if bufYAMLFile == nil {
		return nil, nil
	}
	switch fileVersion := bufYAMLFile.FileVersion(); fileVersion {
	case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
		if w.isV2 {
			return nil, syserror.Newf("buf.yaml at %q did had version %v but expected v1beta1, v1", w.targetSubDirPath, fileVersion)
		}
		// Policys are not supported in versions less than v2.
		return nil, nil
	case bufconfig.FileVersionV2:
		if !w.isV2 {
			return nil, syserror.Newf("buf.yaml at %q did had version %v but expected v2", w.targetSubDirPath, fileVersion)
		}
	default:
		return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
	}
	policyRefs := xslices.Filter(
		xslices.Map(
			bufYAMLFile.PolicyConfigs(),
			func(value bufconfig.PolicyConfig) bufparse.Ref {
				return value.Ref()
			},
		),
		func(value bufparse.Ref) bool {
			return value != nil
		},
	)
	sort.Slice(
		policyRefs,
		func(i int, j int) bool {
			return policyRefs[i].FullName().String() < policyRefs[j].FullName().String()
		},
	)
	return policyRefs, nil
}

func (w *workspaceDepManager) ConfiguredLocalPolicyNameToRemotePluginRefs(ctx context.Context) (map[string][]bufparse.Ref, error) {
	bufYAMLFile, err := bufconfig.GetBufYAMLFileForPrefix(ctx, w.bucket, w.targetSubDirPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	if bufYAMLFile == nil {
		return nil, nil
	}
	switch fileVersion := bufYAMLFile.FileVersion(); fileVersion {
	case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
		if w.isV2 {
			return nil, syserror.Newf("buf.yaml at %q did had version %v but expected v1beta1, v1", w.targetSubDirPath, fileVersion)
		}
		// Policys are not supported in versions less than v2.
		return nil, nil
	case bufconfig.FileVersionV2:
		if !w.isV2 {
			return nil, syserror.Newf("buf.yaml at %q did had version %v but expected v2", w.targetSubDirPath, fileVersion)
		}
	default:
		return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
	}
	localPolicyNameToRemotePluginRefs := make(map[string][]bufparse.Ref)
	for _, policyConfig := range bufYAMLFile.PolicyConfigs() {
		if policyConfig.Ref() != nil {
			continue // Only local policies refs are considered here.
		}
		localPolicyName := policyConfig.Name()
		bufPolicyFile, err := bufpolicyconfig.GetBufPolicyYAMLFile(ctx, w.bucket, localPolicyName)
		if err != nil {
			return nil, err
		}
		pluginRefs := xslices.Filter(
			xslices.Map(
				bufPolicyFile.PluginConfigs(),
				func(value bufpolicy.PluginConfig) bufparse.Ref {
					return value.Ref()
				},
			),
			func(value bufparse.Ref) bool {
				return value != nil
			},
		)
		sort.Slice(
			pluginRefs,
			func(i int, j int) bool {
				return pluginRefs[i].FullName().String() < pluginRefs[j].FullName().String()
			},
		)
		localPolicyNameToRemotePluginRefs[localPolicyName] = pluginRefs
	}
	return localPolicyNameToRemotePluginRefs, nil
}

func (w *workspaceDepManager) BufLockFileDigestType() bufmodule.DigestType {
	if w.isV2 {
		return bufmodule.DigestTypeB5
	}
	return bufmodule.DigestTypeB4
}

func (w *workspaceDepManager) ExistingBufLockFileDepModuleKeys(ctx context.Context) ([]bufmodule.ModuleKey, error) {
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(ctx, w.bucket, w.targetSubDirPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return bufLockFile.DepModuleKeys(), nil
}

func (w *workspaceDepManager) ExistingBufLockFileRemotePluginKeys(ctx context.Context) ([]bufplugin.PluginKey, error) {
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(ctx, w.bucket, w.targetSubDirPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return bufLockFile.RemotePluginKeys(), nil
}

func (w *workspaceDepManager) ExistingBufLockFileRemotePolicyKeys(ctx context.Context) ([]bufpolicy.PolicyKey, error) {
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(ctx, w.bucket, w.targetSubDirPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return bufLockFile.RemotePolicyKeys(), nil
}

func (w *workspaceDepManager) ExistingBufLockFilePolicyNameToRemotePluginKeys(ctx context.Context) (map[string][]bufplugin.PluginKey, error) {
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(ctx, w.bucket, w.targetSubDirPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return bufLockFile.PolicyNameToRemotePluginKeys(), nil
}

func (w *workspaceDepManager) UpdateBufLockFile(ctx context.Context, depModuleKeys []bufmodule.ModuleKey, remotePluginKeys []bufplugin.PluginKey, remotePolicyKeys []bufpolicy.PolicyKey, policyNameToRemotePluginKeys map[string][]bufplugin.PluginKey) error {
	var bufLockFile bufconfig.BufLockFile
	var err error
	if w.isV2 {
		bufLockFile, err = bufconfig.NewBufLockFile(
			bufconfig.FileVersionV2,
			depModuleKeys,
			remotePluginKeys,
			remotePolicyKeys,
			policyNameToRemotePluginKeys,
		)
		if err != nil {
			return err
		}
	} else {
		fileVersion := bufconfig.FileVersionV1
		existingBufYAMLFile, err := bufconfig.GetBufYAMLFileForPrefix(ctx, w.bucket, w.targetSubDirPath)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		} else {
			fileVersion = existingBufYAMLFile.FileVersion()
		}
		if len(remotePluginKeys) > 0 {
			return syserror.Newf("remote plugins are not supported for v1 buf.yaml files")
		}
		bufLockFile, err = bufconfig.NewBufLockFile(fileVersion, depModuleKeys, nil, nil, nil)
		if err != nil {
			return err
		}
	}
	return bufconfig.PutBufLockFileForPrefix(ctx, w.bucket, w.targetSubDirPath, bufLockFile)
}

func (*workspaceDepManager) isWorkspaceDepManager() {}
