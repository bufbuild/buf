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

package depupdate

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/cmd/buf/internal/command/dep/internal"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const (
	onlyFlagName = "only"
)

// NewCommand returns a new update Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
	deprecated string,
	hidden bool,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <directory>",
		Short: "Update pinned module dependencies in a buf.lock",
		Long: `Fetch the latest digests for the specified module references in buf.yaml,
and write them and their transitive dependencies to buf.lock.

The first argument is the directory of the local module to update.
Defaults to "." if no argument is specified.`,
		Args:       appcmd.MaximumNArgs(1),
		Deprecated: deprecated,
		Hidden:     hidden,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Only []string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVar(
		&f.Only,
		onlyFlagName,
		nil,
		"The name of the dependency to update. When set, only this dependency and its transitive dependencies are updated. May be passed multiple times",
	)
	// TODO FUTURE: implement
	_ = flagSet.MarkHidden(onlyFlagName)
}

// run update the buf.lock file for a specific module.
func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	dirPath := "."
	if container.NumArgs() > 0 {
		dirPath = container.Arg(0)
	}
	if len(flags.Only) > 0 {
		// TODO FUTURE: implement
		return syserror.Newf("--%s is not implemented", onlyFlagName)
	}

	logger := container.Logger()
	controller, err := bufcli.NewController(container)
	if err != nil {
		return err
	}
	workspaceDepManager, err := controller.GetWorkspaceDepManager(ctx, dirPath)
	if err != nil {
		return err
	}
	configuredDepModuleRefs, err := workspaceDepManager.ConfiguredDepModuleRefs(ctx)
	if err != nil {
		return err
	}
	// Apply git branch auto-label overrides for matching dependencies.
	// originalRefs maps the full name string of each overridden ref to its original ref,
	// so that fallback can use the original hard-coded label instead of the default label.
	configuredDepModuleRefs, originalRefs, err := applyGitBranchLabelOverrides(ctx, container, dirPath, configuredDepModuleRefs)
	if err != nil {
		return err
	}
	configuredDepModuleKeys, err := resolveModuleRefsWithFallback(
		ctx,
		container,
		logger,
		configuredDepModuleRefs,
		originalRefs,
		workspaceDepManager.BufLockFileDigestType(),
	)
	if err != nil {
		return err
	}
	logger.DebugContext(
		ctx,
		"all deps",
		slog.Any("deps", xslices.Map(configuredDepModuleKeys, bufmodule.ModuleKey.String)),
	)

	existingDepModuleKeys, err := workspaceDepManager.ExistingBufLockFileDepModuleKeys(ctx)
	if err != nil {
		return err
	}
	if configuredDepModuleKeys == nil && existingDepModuleKeys == nil {
		// No new configured deps were found, and no existing buf.lock deps were found, so there
		// is nothing to update, we can return here.
		// This ensures we do not create an empty buf.lock when one did not exist in the first
		// place and we do not need to go through the entire operation of updating non-existent
		// deps and building the image for tamper-proofing.
		logger.Warn(fmt.Sprintf("No configured dependencies were found to update in %q.", dirPath))
		return nil
	}
	existingRemotePluginKeys, err := workspaceDepManager.ExistingBufLockFileRemotePluginKeys(ctx)
	if err != nil {
		return err
	}
	existingRemotePolicyKeys, err := workspaceDepManager.ExistingBufLockFileRemotePolicyKeys(ctx)
	if err != nil {
		return err
	}
	existingPolicyNameToRemotePluginKeys, err := workspaceDepManager.ExistingBufLockFilePolicyNameToRemotePluginKeys(ctx)
	if err != nil {
		return err
	}

	// Write the updated buf.lock to an in-memory bucket and overlay it on top of
	// the workspace bucket for validation. Only persist to disk after the workspace
	// builds successfully.
	overlayBucket := storagemem.NewReadWriteBucket()
	bufLockFile, err := newBufLockFile(
		workspaceDepManager.BufLockFileDigestType(),
		configuredDepModuleKeys,
		existingRemotePluginKeys,
		existingRemotePolicyKeys,
		existingPolicyNameToRemotePluginKeys,
	)
	if err != nil {
		return err
	}
	if err := bufconfig.PutBufLockFileForPrefix(ctx, overlayBucket, ".", bufLockFile); err != nil {
		return err
	}
	workspace, err := controller.GetWorkspace(
		ctx,
		dirPath,
		bufctl.WithIgnoreAndDisallowV1BufWorkYAMLs(),
		bufctl.WithBucketOverlay(overlayBucket),
	)
	if err != nil {
		return err
	}
	// Validate that the workspace builds.
	// Building also has the side effect of doing tamper-proofing.
	if _, err := controller.GetImageForWorkspace(
		ctx,
		workspace,
		// This is a performance optimization - we don't need source code info.
		bufctl.WithImageExcludeSourceInfo(true),
	); err != nil {
		return err
	}
	// Build succeeded, persist the buf.lock to disk.
	if err := workspaceDepManager.UpdateBufLockFile(
		ctx,
		configuredDepModuleKeys,
		existingRemotePluginKeys,
		existingRemotePolicyKeys,
		existingPolicyNameToRemotePluginKeys,
	); err != nil {
		return err
	}
	// Log warnings for users on unused configured deps.
	return internal.LogUnusedConfiguredDepsForWorkspace(workspace, logger)
}

// newBufLockFile creates a BufLockFile for the given digest type and keys.
func newBufLockFile(
	digestType bufmodule.DigestType,
	depModuleKeys []bufmodule.ModuleKey,
	remotePluginKeys []bufplugin.PluginKey,
	remotePolicyKeys []bufpolicy.PolicyKey,
	policyNameToRemotePluginKeys map[string][]bufplugin.PluginKey,
) (bufconfig.BufLockFile, error) {
	switch digestType {
	case bufmodule.DigestTypeB5:
		return bufconfig.NewBufLockFile(
			bufconfig.FileVersionV2,
			depModuleKeys,
			remotePluginKeys,
			remotePolicyKeys,
			policyNameToRemotePluginKeys,
		)
	default:
		// For v1beta1/v1 workspaces, plugins and policies are not supported.
		return bufconfig.NewBufLockFile(
			bufconfig.FileVersionV1,
			depModuleKeys,
			nil,
			nil,
			nil,
		)
	}
}

// applyGitBranchLabelOverrides reads the buf.yaml from dirPath and overrides the
// label on any dep whose full name appears in use_git_branch_as_label.
//
// Returns the modified refs and a map from full name string to the original ref
// for each overridden dependency, so that fallback resolution can use the original
// hard-coded label instead of the default label.
func applyGitBranchLabelOverrides(
	ctx context.Context,
	container appext.Container,
	dirPath string,
	refs []bufparse.Ref,
) ([]bufparse.Ref, map[string]bufparse.Ref, error) {
	useGitBranchAsLabel, disableLabelForBranch, ok := readGitBranchLabelConfig(dirPath)
	if !ok || len(useGitBranchAsLabel) == 0 {
		return refs, nil, nil
	}
	// We only need to determine the branch once, using any matching module name.
	// Use the first matching module name to check.
	branchName, enabled, err := bufcli.GetGitBranchLabelForModule(
		ctx,
		container.Logger(),
		container,
		dirPath,
		useGitBranchAsLabel[0],
		useGitBranchAsLabel,
		disableLabelForBranch,
	)
	if err != nil {
		return nil, nil, err
	}
	if !enabled {
		return refs, nil, nil
	}
	result := make([]bufparse.Ref, len(refs))
	originalRefs := make(map[string]bufparse.Ref)
	for i, moduleRef := range refs {
		if slices.Contains(useGitBranchAsLabel, moduleRef.FullName().String()) {
			overriddenRef, err := bufparse.NewRef(
				moduleRef.FullName().Registry(),
				moduleRef.FullName().Owner(),
				moduleRef.FullName().Name(),
				branchName,
			)
			if err != nil {
				return nil, nil, err
			}
			originalRefs[moduleRef.FullName().String()] = moduleRef
			result[i] = overriddenRef
		} else {
			result[i] = moduleRef
		}
	}
	return result, originalRefs, nil
}

// moduleRefResolver is a function that resolves module refs to module keys.
// Extracted as a type to allow testing with a mock resolver.
type moduleRefResolver func(
	ctx context.Context,
	container appext.Container,
	moduleRefs []bufparse.Ref,
	digestType bufmodule.DigestType,
) ([]bufmodule.ModuleKey, error)

// resolveModuleRefsWithFallback resolves module refs to module keys. If the
// batch resolution fails (e.g., because a branch label doesn't exist on the BSR),
// it falls back to resolving each ref individually, retrying failed branch-labeled
// refs with their original ref (which may have a hard-coded label from buf.yaml).
//
// originalRefs maps full name strings to the original ref before branch label override.
// If nil, no overrides were applied and fallback is not attempted.
func resolveModuleRefsWithFallback(
	ctx context.Context,
	container appext.Container,
	logger *slog.Logger,
	refs []bufparse.Ref,
	originalRefs map[string]bufparse.Ref,
	digestType bufmodule.DigestType,
) ([]bufmodule.ModuleKey, error) {
	return doResolveModuleRefsWithFallback(
		ctx,
		container,
		logger,
		refs,
		originalRefs,
		digestType,
		internal.ModuleKeysAndTransitiveDepModuleKeysForModuleRefs,
	)
}

// doResolveModuleRefsWithFallback contains the core logic, accepting a resolver
// function to allow testing.
func doResolveModuleRefsWithFallback(
	ctx context.Context,
	container appext.Container,
	logger *slog.Logger,
	refs []bufparse.Ref,
	originalRefs map[string]bufparse.Ref,
	digestType bufmodule.DigestType,
	resolve moduleRefResolver,
) ([]bufmodule.ModuleKey, error) {
	// First, try resolving all refs in a single batch.
	moduleKeys, err := resolve(ctx, container, refs, digestType)
	if err == nil {
		return moduleKeys, nil
	}
	// If no overrides were applied, the error is genuine.
	if len(originalRefs) == 0 {
		return nil, err
	}
	logger.DebugContext(ctx, "batch resolution failed, falling back to per-ref resolution", slog.String("error", err.Error()))
	// Fall back to per-ref resolution.
	var allModuleKeys []bufmodule.ModuleKey
	for _, moduleRef := range refs {
		keys, resolveErr := resolve(ctx, container, []bufparse.Ref{moduleRef}, digestType)
		if resolveErr == nil {
			allModuleKeys = append(allModuleKeys, keys...)
			continue
		}
		// Check if this ref was overridden with a branch label.
		fallbackRef, wasOverridden := originalRefs[moduleRef.FullName().String()]
		if !wasOverridden {
			// Not an overridden ref, the error is genuine.
			return nil, resolveErr
		}
		// Branch-labeled ref failed, retry with the original ref from buf.yaml.
		logger.DebugContext(
			ctx,
			"branch label not found, falling back to original label",
			slog.String("module", moduleRef.FullName().String()),
			slog.String("branch_label", moduleRef.Ref()),
			slog.String("fallback_label", fallbackRef.Ref()),
		)
		keys, resolveErr = resolve(ctx, container, []bufparse.Ref{fallbackRef}, digestType)
		if resolveErr != nil {
			return nil, resolveErr
		}
		allModuleKeys = append(allModuleKeys, keys...)
	}
	return allModuleKeys, nil
}

// readGitBranchLabelConfig reads the buf.yaml from dirPath and returns the
// auto-label configuration. Returns ok=false if the buf.yaml cannot be read.
func readGitBranchLabelConfig(dirPath string) (useGitBranchAsLabel []string, disableLabelForBranch []string, ok bool) {
	data, err := os.ReadFile(dirPath + "/buf.yaml")
	if err != nil {
		return nil, nil, false
	}
	bufYAMLFile, err := bufconfig.ReadBufYAMLFile(bytes.NewReader(data), "buf.yaml")
	if err != nil {
		return nil, nil, false
	}
	return bufYAMLFile.UseGitBranchAsLabel(), bufYAMLFile.DisableLabelForBranch(), true
}
