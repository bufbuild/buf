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

package push

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/spf13/pflag"
)

const (
	labelFlagName              = "label"
	errorFormatFlagName        = "error-format"
	disableSymlinksFlagName    = "disable-symlinks"
	createFlagName             = "create"
	createVisibilityFlagName   = "create-visibility"
	createDefaultLabelFlagName = "create-default-label"
	sourceControlURLFlagName   = "source-control-url"
	gitMetadataFlagName        = "git-metadata"

	// All deprecated.
	tagFlagName      = "tag"
	tagFlagShortName = "t"
	draftFlagName    = "draft"
	branchFlagName   = "branch"

	gitCommand                = "git"
	gitDefaultRemote          = "origin"
	defaultGitRemoteURLFormat = "%s/commit/%s"
	bitBucketRemoteURLFormat  = "%s/commits/%s"
	bitBucketURL              = "https://bitbucket.org"
)

var (
	useLabelInstead = fmt.Sprintf("Use --%s instead.", labelFlagName)
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <source>",
		Short: "Push to a registry",
		Long:  bufcli.GetSourceLong(`the source to push`),
		Args:  appcmd.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Tags               []string
	Branch             string
	Draft              string
	Labels             []string
	ErrorFormat        string
	DisableSymlinks    bool
	Create             bool
	CreateVisibility   string
	CreateDefaultLabel string
	SourceControlURL   string
	GitMetadata        bool
	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)
	bufcli.BindCreateVisibility(flagSet, &f.CreateVisibility, createVisibilityFlagName, createFlagName)
	flagSet.StringSliceVar(
		&f.Labels,
		labelFlagName,
		nil,
		"Associate the label with the modules pushed. Can be used multiple times.",
	)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors printed to stderr. Must be one of %s",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	flagSet.BoolVar(
		&f.Create,
		createFlagName,
		false,
		fmt.Sprintf(
			"Create the repository if it does not exist. Defaults to creating a private repository if --%s is not set.",
			createVisibilityFlagName,
		),
	)
	flagSet.StringVar(
		&f.CreateDefaultLabel,
		createDefaultLabelFlagName,
		"",
		`The repository's default label setting, if created. If this is not set, then the repository will be created with the default label "main".`,
	)
	flagSet.StringVar(
		&f.SourceControlURL,
		sourceControlURLFlagName,
		"",
		"The URL for viewing the source code of the pushed modules (e.g. the specific commit in source control).",
	)
	flagSet.BoolVar(
		&f.GitMetadata,
		gitMetadataFlagName,
		false,
		fmt.Sprintf(
			`Uses the Git source control state to set flag values. If this flag is set, we will use the following values for your flags:

	--%s to <git remote URL>/<repository name>/commit/<commit sha> (e.g. https://github.com/acme/weather/commit/ffac537e6cbbf934b08745a378932722df287a53)
	--%s for each Git tag and branch for this Git commit
	--%s to the Git default branch (e.g. main) - this is only in effect if --%s is also set

The source control URL and default branch is based on the default Git remote, %q. If you do not have this remote, we take the URL for the first remote based on alphabetical ordering.
This flag is only compatible with checkouts of Git source repositories.
This flag does not allow you to set any of the following flags yourself: --%s, --%s, --%s, --%s, --%s, --%s.`,
			sourceControlURLFlagName,
			labelFlagName,
			createDefaultLabelFlagName,
			createFlagName,
			gitDefaultRemote,
			sourceControlURLFlagName,
			labelFlagName,
			createDefaultLabelFlagName,
			tagFlagName,
			branchFlagName,
			draftFlagName,
		),
	)

	flagSet.StringSliceVarP(&f.Tags, tagFlagName, tagFlagShortName, nil, useLabelInstead)
	_ = flagSet.MarkHidden(tagFlagName)
	_ = flagSet.MarkHidden(tagFlagShortName)
	_ = flagSet.MarkDeprecated(tagFlagName, useLabelInstead)
	_ = flagSet.MarkDeprecated(tagFlagShortName, useLabelInstead)
	flagSet.StringVar(&f.Draft, draftFlagName, "", useLabelInstead)
	_ = flagSet.MarkHidden(draftFlagName)
	_ = flagSet.MarkDeprecated(draftFlagName, useLabelInstead)
	flagSet.StringVar(&f.Branch, branchFlagName, "", useLabelInstead)
	_ = flagSet.MarkHidden(branchFlagName)
	_ = flagSet.MarkDeprecated(branchFlagName, useLabelInstead)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) (retErr error) {
	if err := validateFlags(flags); err != nil {
		return err
	}

	workspace, err := getBuildableWorkspace(ctx, container, flags)
	if err != nil {
		return err
	}

	uploader, err := bufcli.NewUploader(container)
	if err != nil {
		return err
	}

	var uploadOptions []bufmodule.UploadOption
	if flags.GitMetadata {
		gitMetadataUploadOptions, err := getGitMetadataUploadOptions(ctx, container, flags)
		if err != nil {
			return err
		}
		uploadOptions = append(uploadOptions, gitMetadataUploadOptions...)
	} else {
		// Otherwise, we parse the flags set individually by the user.
		if labelUploadOption := getLabelUploadOption(flags); labelUploadOption != nil {
			uploadOptions = append(uploadOptions, labelUploadOption)
		}
		if flags.Create {
			createModuleVisiblity, err := bufmodule.ParseModuleVisibility(flags.CreateVisibility)
			if err != nil {
				return err
			}
			uploadOptions = append(
				uploadOptions,
				bufmodule.UploadWithCreateIfNotExist(createModuleVisiblity, flags.CreateDefaultLabel),
			)
		}
		if flags.SourceControlURL != "" {
			uploadOptions = append(uploadOptions, bufmodule.UploadWithSourceControlURL(flags.SourceControlURL))
		}
	}

	commits, err := uploader.Upload(ctx, workspace, uploadOptions...)
	if err != nil {
		return err
	}

	if workspace.IsV2() {
		_, err := container.Stdout().Write(
			[]byte(
				strings.Join(
					slicesext.Map(
						commits,
						func(commit bufmodule.Commit) string {
							return commit.ModuleKey().String()
						},
					),
					"\n",
				) + "\n",
			),
		)
		return err
	}
	// v1 workspace, fallback to old behavior for backwards compatibility.
	switch len(commits) {
	case 0:
		return nil
	case 1:
		_, err := container.Stdout().Write([]byte(uuidutil.ToDashless(commits[0].ModuleKey().CommitID()) + "\n"))
		return err
	default:
		return syserror.Newf("Received multiple commits back for a v1 module. We should only ever have created a single commit for a v1 module.")
	}
}

func getBuildableWorkspace(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) (bufworkspace.Workspace, error) {
	source, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return nil, err
	}
	controller, err := bufcli.NewController(
		container,
		bufctl.WithDisableSymlinks(flags.DisableSymlinks),
		bufctl.WithFileAnnotationErrorFormat(flags.ErrorFormat),
	)
	if err != nil {
		return nil, err
	}
	workspace, err := controller.GetWorkspace(
		ctx,
		source,
		// We actually could make it so that buf push would work with buf.work.yamls and push
		// v1 workspaces as well, but this may have unintended (and potentially breaking) consequences
		// that we don't want to deal with. If we have a v1 workspace, just outlaw pushing the whole
		// workspace, and force people into the pre-refactor behavior.
		bufctl.WithIgnoreAndDisallowV1BufWorkYAMLs(),
	)
	if err != nil {
		return nil, err
	}
	// Make sure the workspace builds.
	if _, err := controller.GetImageForWorkspace(
		ctx,
		workspace,
		bufctl.WithImageExcludeSourceInfo(true),
	); err != nil {
		return nil, err
	}
	return workspace, nil
}

func validateFlags(flags *flags) error {
	if err := validateCreateFlags(flags); err != nil {
		return err
	}
	if err := validateLabelFlags(flags); err != nil {
		return err
	}
	return validateGitMetadataFlags(flags)
}

func validateCreateFlags(flags *flags) error {
	if flags.Create {
		if flags.CreateVisibility == "" {
			return appcmd.NewInvalidArgumentErrorf(
				"--%s is required if --%s is set",
				createVisibilityFlagName,
				createFlagName,
			)
		}
		if _, err := bufmodule.ParseModuleVisibility(flags.CreateVisibility); err != nil {
			return appcmd.NewInvalidArgumentError(err.Error())
		}
	} else {
		if flags.CreateDefaultLabel != "" {
			return appcmd.NewInvalidArgumentErrorf(
				"Cannot set --%s without --%s",
				createDefaultLabelFlagName,
				createFlagName,
			)
		}
	}
	return nil
}

func validateLabelFlags(flags *flags) error {
	if err := validateLabelFlagCombinations(flags); err != nil {
		return err
	}
	return validateLabelFlagValues(flags)
}

// We do not allow overlaps between `--label`, `--tag`, `--branch`, and `--draft` flags
// when calling push. Only one type of flag is allowed to be used at a time when pushing.
func validateLabelFlagCombinations(flags *flags) error {
	var usedFlags []string
	if len(flags.Labels) > 0 {
		usedFlags = append(usedFlags, labelFlagName)
	}
	if len(flags.Tags) > 0 {
		usedFlags = append(usedFlags, tagFlagName)
	}
	if flags.Branch != "" {
		usedFlags = append(usedFlags, branchFlagName)
	}
	if flags.Draft != "" {
		usedFlags = append(usedFlags, draftFlagName)
	}
	if len(usedFlags) > 1 {
		usedFlagsErrStr := strings.Join(
			slicesext.Map(
				usedFlags,
				func(flag string) string { return fmt.Sprintf("--%s", flag) },
			),
			", ",
		)
		return appcmd.NewInvalidArgumentErrorf("These flags cannot be used in combination with one another: %s", usedFlagsErrStr)
	}
	return nil
}

func validateLabelFlagValues(flags *flags) error {
	for _, label := range flags.Labels {
		if label == "" {
			return appcmd.NewInvalidArgumentErrorf("--%s requires a non-empty string", labelFlagName)
		}
	}
	for _, tag := range flags.Tags {
		if tag == "" {
			return appcmd.NewInvalidArgumentErrorf("--%s requires a non-empty string", tagFlagName)
		}
	}
	return nil
}

// We do not allow users to set --source-control-url, --create-default-label, and --label
// flags if the --git-metadata flag is set.
func validateGitMetadataFlags(flags *flags) error {
	if flags.GitMetadata {
		var usedFlags []string
		if flags.SourceControlURL != "" {
			usedFlags = append(usedFlags, sourceControlURLFlagName)
		}
		if len(flags.Labels) > 0 {
			usedFlags = append(usedFlags, labelFlagName)
		}
		if flags.CreateDefaultLabel != "" {
			usedFlags = append(usedFlags, createDefaultLabelFlagName)
		}
		if len(flags.Tags) > 0 {
			usedFlags = append(usedFlags, tagFlagName)
		}
		if flags.Branch != "" {
			usedFlags = append(usedFlags, branchFlagName)
		}
		if flags.Draft != "" {
			usedFlags = append(usedFlags, draftFlagName)
		}
		if len(usedFlags) > 0 {
			usedFlagsErrStr := strings.Join(
				slicesext.Map(
					usedFlags,
					func(flag string) string { return fmt.Sprintf("--%s", flag) },
				),
				", ",
			)
			return appcmd.NewInvalidArgumentErrorf(
				"The following flag(s) cannot be used in combination with --%s: %s.",
				gitMetadataFlagName,
				usedFlagsErrStr,
			)
		}
	}
	return nil
}

func getGitMetadataUploadOptions(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) ([]bufmodule.UploadOption, error) {
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return nil, err
	}
	runner := command.NewRunner()
	var gitMetadataUploadOptions []bufmodule.UploadOption
	gitLabelsUploadOption, err := getGitMetadataLabelsUploadOptions(ctx, runner, input)
	if err != nil {
		return nil, err
	}
	if gitLabelsUploadOption != nil {
		gitMetadataUploadOptions = append(gitMetadataUploadOptions, gitLabelsUploadOption)
	}
	remotes, err := getGitRemotes(ctx, runner, input)
	if err != nil {
		return nil, err
	}
	gitSourceControlURLUploadOption, err := getGitMetadataSourceControlURLUploadOption(ctx, runner, remotes, input)
	if err != nil {
		return nil, err
	}
	if gitSourceControlURLUploadOption != nil {
		gitMetadataUploadOptions = append(gitMetadataUploadOptions, gitSourceControlURLUploadOption)
	}
	if flags.Create {
		gitDefaultBranch, err := getGitDefaultBranch(ctx, runner, container, remotes, input)
		if err != nil {
			return nil, err
		}
		createModuleVisibility, err := bufmodule.ParseModuleVisibility(flags.CreateVisibility)
		if err != nil {
			return nil, err
		}
		gitMetadataUploadOptions = append(
			gitMetadataUploadOptions,
			bufmodule.UploadWithCreateIfNotExist(createModuleVisibility, gitDefaultBranch),
		)
	}
	return gitMetadataUploadOptions, nil
}

func getGitRemotes(
	ctx context.Context,
	runner command.Runner,
	input string,
) ([]string, error) {
	buffer := bytes.NewBuffer(nil)
	if err := runner.Run(
		ctx,
		gitCommand,
		command.RunWithArgs("remote"),
		command.RunWithStdout(buffer),
		command.RunWithStderr(buffer),
		command.RunWithDir(input),
	); err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(buffer.String()), "\n"), nil
}

// This returns an upload option for all Git metadata labels. We set labels for the following
// Git matadata:
//   - tags on current commit
//   - current branch
func getGitMetadataLabelsUploadOptions(
	ctx context.Context,
	runner command.Runner,
	input string,
) (bufmodule.UploadOption, error) {
	tags, err := getGitTagsOnCurrentCommit(ctx, runner, input)
	if err != nil {
		return nil, err
	}
	branch, err := getCurrentGitBranch(ctx, runner, input)
	if err != nil {
		return nil, err
	}
	labels := append(tags, branch)
	if len(labels) > 0 {
		return bufmodule.UploadWithLabels(labels...), nil
	}
	return nil, nil
}

func getGitTagsOnCurrentCommit(
	ctx context.Context,
	runner command.Runner,
	input string,
) ([]string, error) {
	buffer := bytes.NewBuffer(nil)
	if err := runner.Run(
		ctx,
		gitCommand,
		command.RunWithArgs("tag", "--points-at", "HEAD"),
		command.RunWithStdout(buffer),
		command.RunWithStderr(buffer),
		command.RunWithDir(input),
	); err != nil {
		return nil, err
	}
	if len(buffer.Bytes()) > 0 {
		return strings.Split(strings.TrimSpace(buffer.String()), "\n"), nil
	}
	return nil, nil
}

func getCurrentGitBranch(
	ctx context.Context,
	runner command.Runner,
	input string,
) (string, error) {
	buffer := bytes.NewBuffer(nil)
	if err := runner.Run(
		ctx,
		gitCommand,
		command.RunWithArgs("branch", "--show-current"),
		command.RunWithStdout(buffer),
		command.RunWithStderr(buffer),
		command.RunWithDir(input),
	); err != nil {
		return "", err
	}
	return strings.TrimSpace(buffer.String()), nil
}

func getGitMetadataSourceControlURLUploadOption(
	ctx context.Context,
	runner command.Runner,
	remotes []string,
	input string,
) (bufmodule.UploadOption, error) {
	remoteToURL := map[string]string{}
	for _, remote := range remotes {
		buffer := bytes.NewBuffer(nil)
		if err := runner.Run(
			ctx,
			gitCommand,
			command.RunWithArgs("config", "--get", fmt.Sprintf("remote.%s.url", remote)),
			command.RunWithStdout(buffer),
			command.RunWithStderr(buffer),
			command.RunWithDir(input),
		); err != nil {
			return nil, err
		}
		remoteToURL[remote] = strings.TrimSpace(buffer.String())
	}
	// We prioritize the Git default remote, "origin", URL
	if gitDefaultRemoteURL, ok := remoteToURL[gitDefaultRemote]; ok {
		// First we need to trim the `.git` suffix if there is one.
		gitDefaultRemoteURL = strings.TrimSuffix(gitDefaultRemoteURL, ".git")
		// Then we get the current HEAD commit.
		currentHEADCommit, err := getCurrentHEADGitCommit(ctx, runner, input)
		if err != nil {
			return nil, err
		}
		// Bitbucket is the only URL that uses the "/commits" route, both Github and Gitlab
		// use "/commit", so we default to that if the remote URL does not point at Bitbucket.
		if strings.HasPrefix(gitDefaultRemoteURL, bitBucketURL) {
			return bufmodule.UploadWithSourceControlURL(fmt.Sprintf(
				bitBucketRemoteURLFormat,
				gitDefaultRemoteURL,
				currentHEADCommit,
			)), nil
		}
		return bufmodule.UploadWithSourceControlURL(fmt.Sprintf(
			defaultGitRemoteURLFormat,
			gitDefaultRemoteURL,
			currentHEADCommit,
		)), nil
	}
	// Otherwise we sort by alphabetical order and return the first URL
	sortedRemotes := slicesext.MapKeysToSortedSlice(remoteToURL)
	if len(sortedRemotes) > 0 {
		return bufmodule.UploadWithSourceControlURL(remoteToURL[sortedRemotes[0]]), nil
	}
	return nil, nil
}

func getCurrentHEADGitCommit(
	ctx context.Context,
	runner command.Runner,
	input string,
) (string, error) {
	buffer := bytes.NewBuffer(nil)
	if err := runner.Run(
		ctx,
		gitCommand,
		command.RunWithArgs("rev-parse", "HEAD"),
		command.RunWithStdout(buffer),
		command.RunWithStderr(buffer),
		command.RunWithDir(input),
	); err != nil {
		return "", err
	}
	return strings.TrimSpace(buffer.String()), nil
}

func getGitDefaultBranch(
	ctx context.Context,
	runner command.Runner,
	envContainer app.EnvContainer,
	remotes []string,
	input string,
) (string, error) {
	remoteToHEADBranch := map[string]string{}
	for _, remote := range remotes {
		buffer := bytes.NewBuffer(nil)
		if err := runner.Run(
			ctx,
			gitCommand,
			command.RunWithArgs("remote", "show", remote),
			command.RunWithStdout(buffer),
			command.RunWithStderr(buffer),
			command.RunWithDir(input),
			command.RunWithEnv(app.EnvironMap(envContainer)),
		); err != nil {
			return "", err
		}
		branch, err := getHEADBranchFromGitRemoteOutput(buffer.Bytes())
		if err != nil {
			return "", err
		}
		remoteToHEADBranch[remote] = branch
	}
	// We prioritze the Git default remote, "origin", HEAD branch as the default branch
	if gitDefaultRemoteHEADBranch, ok := remoteToHEADBranch[gitDefaultRemote]; ok {
		return gitDefaultRemoteHEADBranch, nil
	}
	// Otherwise we sort by alphabetical order and return the first URL
	sortedRemotes := slicesext.MapKeysToSortedSlice(remoteToHEADBranch)
	if len(sortedRemotes) > 0 {
		return remoteToHEADBranch[sortedRemotes[0]], nil
	}
	return "", nil
}

// This checks for the HEAD branch from the output of `git remote show <remote>`.
// sed -n '/HEAD branch/s/.*: //p'
func getHEADBranchFromGitRemoteOutput(output []byte) (string, error) {
	// First find where "HEAD branch"
	index := bytes.Index(output, []byte("HEAD branch: "))
	if index < 0 {
		return "", errors.New("no HEAD branch in git remote output")
	}
	// Then find the next new line
	firstNewLine := bytes.Index(output[index:], []byte("\n"))
	if index < 0 {
		return "", errors.New("no HEAD branch information found")
	}
	// Return this line with the prefix "HEAD branch: " trimmed off.
	// The index of the new line is not included here.
	return strings.TrimPrefix(string(output[index:index+firstNewLine]), "HEAD branch: "), nil
}

func getLabelUploadOption(flags *flags) bufmodule.UploadOption {
	// We do not allow the mixing of flags, so post-validation, we only expect one of the
	// flags to be set. And so we return the corresponding bufmodule.UploadOption if any
	// flags are set.
	if len(flags.Labels) > 0 {
		return bufmodule.UploadWithLabels(slicesext.ToUniqueSorted(flags.Labels)...)
	}
	if len(flags.Tags) > 0 {
		return bufmodule.UploadWithTags(slicesext.ToUniqueSorted(flags.Tags)...)
	}
	if flags.Branch != "" {
		// We upload to a single label, the branch name.
		return bufmodule.UploadWithLabels(flags.Branch)
	}
	if flags.Draft != "" {
		// We upload to a single label, the draft name.
		return bufmodule.UploadWithLabels(flags.Draft)
	}
	return nil
}
