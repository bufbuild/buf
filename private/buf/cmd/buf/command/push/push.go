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
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/spf13/pflag"
)

const (
	labelFlagName            = "label"
	errorFormatFlagName      = "error-format"
	disableSymlinksFlagName  = "disable-symlinks"
	createFlagName           = "create"
	createVisibilityFlagName = "create-visibility"

	// All deprecated.
	tagFlagName      = "tag"
	tagFlagShortName = "t"
	draftFlagName    = "draft"
	branchFlagName   = "branch"
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
	Tags             []string
	Branch           string
	Draft            string
	Labels           []string
	ErrorFormat      string
	DisableSymlinks  bool
	Create           bool
	CreateVisibility string
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
		&f.Tags,
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
			"Create the repository if it does not exist. Must set --%s",
			createVisibilityFlagName,
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

	uploadOptions := []bufmodule.UploadOption{
		bufmodule.UploadWithLabels(combineLabelLikeFlags(flags)...),
	}
	if flags.Create {
		createModuleVisiblity, err := bufmodule.ParseModuleVisibility(flags.CreateVisibility)
		if err != nil {
			return err
		}
		uploadOptions = append(uploadOptions, bufmodule.UploadWithCreateIfNotExist(createModuleVisiblity))
	}
	commits, err := uploader.Upload(ctx, workspace, uploadOptions...)
	if err != nil {
		return err
	}

	var lines []string
	var linesErr error
	if workspace.IsV2() {
		lines = slicesext.Map(
			commits,
			func(commit bufmodule.Commit) string {
				return commit.ModuleKey().String()
			},
		)
	} else {
		if len(commits) > 1 {
			linesErr = syserror.Newf("Received multiple commits back for a v1 module. We should only ever have created a single commit for a v1 module.")
		}
		lines = slicesext.Map(
			commits,
			// Printing dashless for historical reasons.
			func(commit bufmodule.Commit) string {
				return uuidutil.ToDashless(commit.ModuleKey().CommitID())
			},
		)
	}
	if _, err := container.Stdout().Write([]byte(strings.Join(lines, "\n") + "\n")); err != nil {
		return err
	}
	return linesErr
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
	return nil
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
		if flags.CreateVisibility != "" {
			return appcmd.NewInvalidArgumentErrorf(
				"Cannot set --%s without --%s",
				createVisibilityFlagName,
				createFlagName,
			)
		}
	}
	return nil
}

func validateLabelFlags(flags *flags) error {
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

func combineLabelLikeFlags(flags *flags) []string {
	labels := append(slicesext.Copy(flags.Labels), flags.Tags...)
	if flags.Draft != "" {
		labels = append(labels, flags.Draft)
	}
	if flags.Branch != "" {
		labels = append(labels, flags.Branch)
	}
	return slicesext.ToUniqueSorted(labels)
}
