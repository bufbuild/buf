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

package bufworkspace

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"buf.build/go/standard/xslices"
	"buf.build/go/standard/xstrings"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/google/uuid"
)

const (
	// OutOfSyncDepTypeModule says that the dep is a module declared in the buf.yaml deps.
	//
	// These are updated with "buf dep update".
	OutOfSyncDepTypeModule OutOfSyncDepType = iota + 1
	// OutOfSyncDepTypePlugin says that the dep is a remote plugin declared in the buf.yaml
	// plugins.
	//
	// These are updated with "buf plugin update".
	OutOfSyncDepTypePlugin
	// OutOfSyncDepTypePolicy says that the dep is a remote policy declared in the buf.yaml
	// policies.
	//
	// These are updated with "buf policy update".
	OutOfSyncDepTypePolicy
)

// OutOfSyncDepType is the type of dep that is out of sync.
type OutOfSyncDepType int

// String returns the name of the dep type as it is referred to in user-facing messages.
func (o OutOfSyncDepType) String() string {
	switch o {
	case OutOfSyncDepTypeModule:
		return "module"
	case OutOfSyncDepTypePlugin:
		return "plugin"
	case OutOfSyncDepTypePolicy:
		return "policy"
	default:
		return strconv.Itoa(int(o))
	}
}

// OutOfSyncDep is a dep declared in a buf.yaml that the buf.lock does not satisfy.
//
// A dep declared with an explicit reference, such as
// "buf.build/bufbuild/protovalidate:v0.14.1", constrains the buf.lock to the commit that
// the reference resolves to. If the buf.lock pins a different commit, the buf.lock is out
// of sync.
type OutOfSyncDep interface {
	// Ref is the dep as declared in the buf.yaml.
	//
	// Always present.
	Ref() bufparse.Ref
	// Type is the type of dep that is out of sync.
	//
	// Always present.
	Type() OutOfSyncDepType
	// ExpectedCommitID is the commit that Ref resolves to.
	//
	// Always present.
	ExpectedCommitID() uuid.UUID
	// ExistingCommitID is the commit that the buf.lock pins the dep to.
	//
	// Always present.
	ExistingCommitID() uuid.UUID

	isOutOfSyncDep()
}

// OutOfSyncDepsForWorkspace returns the OutOfSyncDeps for the Workspace, covering the
// module, remote plugin, and remote policy deps declared in the buf.yaml.
//
// Only deps that are declared in the buf.yaml with an explicit reference and that are
// present in the buf.lock are compared.
//
// The remote plugins that a policy itself declares are not compared. For a remote policy
// these follow from the policy commit, which is compared. For a local policy these are
// declared in a local buf.policy.yaml, which is not part of the buf.yaml.
//
// This makes a call to the BSR to resolve each declared reference to a commit.
func OutOfSyncDepsForWorkspace(
	ctx context.Context,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
	pluginKeyProvider bufplugin.PluginKeyProvider,
	policyKeyProvider bufpolicy.PolicyKeyProvider,
	workspace Workspace,
) ([]OutOfSyncDep, error) {
	moduleOutOfSyncDeps, err := outOfSyncDeps(
		ctx,
		OutOfSyncDepTypeModule,
		workspace.ConfiguredDepModuleRefs(),
		bufmodule.ModuleSetRemoteModules(workspace),
		moduleKeysForRefsFunc(moduleKeyProvider, bufLockFileDigestTypeForIsV2(workspace.IsV2())),
	)
	if err != nil {
		return nil, err
	}
	pluginOutOfSyncDeps, err := outOfSyncDeps(
		ctx,
		OutOfSyncDepTypePlugin,
		refsForConfigs(workspace.PluginConfigs(), bufconfig.PluginConfig.Ref),
		workspace.RemotePluginKeys(),
		pluginKeysForRefsFunc(pluginKeyProvider),
	)
	if err != nil {
		return nil, err
	}
	policyOutOfSyncDeps, err := outOfSyncDeps(
		ctx,
		OutOfSyncDepTypePolicy,
		refsForConfigs(workspace.PolicyConfigs(), bufconfig.PolicyConfig.Ref),
		workspace.RemotePolicyKeys(),
		policyKeysForRefsFunc(policyKeyProvider),
	)
	if err != nil {
		return nil, err
	}
	return slices.Concat(moduleOutOfSyncDeps, pluginOutOfSyncDeps, policyOutOfSyncDeps), nil
}

// NewOutOfSyncDepsError returns an error describing the given OutOfSyncDeps.
//
// Returns nil if outOfSyncDeps is empty.
func NewOutOfSyncDepsError(outOfSyncDeps []OutOfSyncDep) error {
	if len(outOfSyncDeps) == 0 {
		return nil
	}
	var builder strings.Builder
	_, _ = builder.WriteString("buf.lock is out of sync with buf.yaml:")
	for _, outOfSyncDep := range outOfSyncDeps {
		_, _ = fmt.Fprintf(
			&builder,
			"\n\t%s %s is declared in buf.yaml, but buf.lock pins commit %s instead of %s",
			outOfSyncDep.Type().String(),
			outOfSyncDep.Ref().String(),
			uuidutil.ToDashless(outOfSyncDep.ExistingCommitID()),
			uuidutil.ToDashless(outOfSyncDep.ExpectedCommitID()),
		)
	}
	updateCommands := xslices.ToUniqueSorted(
		xslices.Map(
			outOfSyncDeps,
			func(outOfSyncDep OutOfSyncDep) string {
				return updateCommandForOutOfSyncDepType(outOfSyncDep.Type())
			},
		),
	)
	_, _ = fmt.Fprintf(
		&builder,
		"\nRun %s to update buf.lock.",
		xstrings.SliceToHumanStringQuoted(updateCommands),
	)
	return errors.New(builder.String())
}

// *** PRIVATE ***

// outOfSyncDepKey is the shape shared by bufmodule.Module, bufmodule.ModuleKey,
// bufplugin.PluginKey, and bufpolicy.PolicyKey.
type outOfSyncDepKey interface {
	FullName() bufparse.FullName
	CommitID() uuid.UUID
}

func outOfSyncDeps[ExistingKey outOfSyncDepKey, ResolvedKey outOfSyncDepKey](
	ctx context.Context,
	outOfSyncDepType OutOfSyncDepType,
	configuredRefs []bufparse.Ref,
	existingKeys []ExistingKey,
	getKeysForRefs func(context.Context, []bufparse.Ref) ([]ResolvedKey, error),
) ([]OutOfSyncDep, error) {
	fullNameStringToExistingCommitID := make(map[string]uuid.UUID, len(existingKeys))
	for _, existingKey := range existingKeys {
		fullName := existingKey.FullName()
		if fullName == nil {
			continue
		}
		if commitID := existingKey.CommitID(); commitID != uuid.Nil {
			fullNameStringToExistingCommitID[fullName.String()] = commitID
		}
	}
	constrainedRefs := xslices.Filter(
		configuredRefs,
		func(ref bufparse.Ref) bool {
			if ref.Ref() == "" {
				return false
			}
			_, ok := fullNameStringToExistingCommitID[ref.FullName().String()]
			return ok
		},
	)
	if len(constrainedRefs) == 0 {
		return nil, nil
	}
	resolvedKeys, err := getKeysForRefs(ctx, constrainedRefs)
	if err != nil {
		return nil, err
	}
	if len(resolvedKeys) != len(constrainedRefs) {
		return nil, syserror.Newf(
			"got %d keys for %d %s refs",
			len(resolvedKeys),
			len(constrainedRefs),
			outOfSyncDepType.String(),
		)
	}
	var resultOutOfSyncDeps []OutOfSyncDep
	for i, resolvedKey := range resolvedKeys {
		ref := constrainedRefs[i]
		existingCommitID := fullNameStringToExistingCommitID[ref.FullName().String()]
		if resolvedKey.CommitID() == existingCommitID {
			continue
		}
		resultOutOfSyncDeps = append(
			resultOutOfSyncDeps,
			newOutOfSyncDep(ref, outOfSyncDepType, resolvedKey.CommitID(), existingCommitID),
		)
	}
	return resultOutOfSyncDeps, nil
}

func moduleKeysForRefsFunc(
	moduleKeyProvider bufmodule.ModuleKeyProvider,
	digestType bufmodule.DigestType,
) func(context.Context, []bufparse.Ref) ([]bufmodule.ModuleKey, error) {
	return func(ctx context.Context, refs []bufparse.Ref) ([]bufmodule.ModuleKey, error) {
		return moduleKeyProvider.GetModuleKeysForModuleRefs(ctx, refs, digestType)
	}
}

func pluginKeysForRefsFunc(
	pluginKeyProvider bufplugin.PluginKeyProvider,
) func(context.Context, []bufparse.Ref) ([]bufplugin.PluginKey, error) {
	return func(ctx context.Context, refs []bufparse.Ref) ([]bufplugin.PluginKey, error) {
		return pluginKeyProvider.GetPluginKeysForPluginRefs(ctx, refs, bufplugin.DigestTypeP1)
	}
}

func policyKeysForRefsFunc(
	policyKeyProvider bufpolicy.PolicyKeyProvider,
) func(context.Context, []bufparse.Ref) ([]bufpolicy.PolicyKey, error) {
	return func(ctx context.Context, refs []bufparse.Ref) ([]bufpolicy.PolicyKey, error) {
		return policyKeyProvider.GetPolicyKeysForPolicyRefs(ctx, refs, bufpolicy.DigestTypeO1)
	}
}

// refsForConfigs returns the Refs of the configs that are remote.
func refsForConfigs[Config any](configs []Config, getRef func(Config) bufparse.Ref) []bufparse.Ref {
	return xslices.Filter(
		xslices.Map(configs, getRef),
		func(ref bufparse.Ref) bool {
			return ref != nil
		},
	)
}

func updateCommandForOutOfSyncDepType(outOfSyncDepType OutOfSyncDepType) string {
	switch outOfSyncDepType {
	case OutOfSyncDepTypePlugin:
		return "buf plugin update"
	case OutOfSyncDepTypePolicy:
		return "buf policy update"
	default:
		return "buf dep update"
	}
}

func bufLockFileDigestTypeForIsV2(isV2 bool) bufmodule.DigestType {
	if isV2 {
		return bufmodule.DigestTypeB5
	}
	return bufmodule.DigestTypeB4
}

type outOfSyncDep struct {
	ref              bufparse.Ref
	outOfSyncDepType OutOfSyncDepType
	expectedCommitID uuid.UUID
	existingCommitID uuid.UUID
}

func newOutOfSyncDep(
	ref bufparse.Ref,
	outOfSyncDepType OutOfSyncDepType,
	expectedCommitID uuid.UUID,
	existingCommitID uuid.UUID,
) *outOfSyncDep {
	return &outOfSyncDep{
		ref:              ref,
		outOfSyncDepType: outOfSyncDepType,
		expectedCommitID: expectedCommitID,
		existingCommitID: existingCommitID,
	}
}

func (o *outOfSyncDep) Ref() bufparse.Ref {
	return o.ref
}

func (o *outOfSyncDep) Type() OutOfSyncDepType {
	return o.outOfSyncDepType
}

func (o *outOfSyncDep) ExpectedCommitID() uuid.UUID {
	return o.expectedCommitID
}

func (o *outOfSyncDep) ExistingCommitID() uuid.UUID {
	return o.existingCommitID
}

func (*outOfSyncDep) isOutOfSyncDep() {}
