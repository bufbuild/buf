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

package policypush

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"slices"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy/bufpolicyconfig"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/google/uuid"
	"github.com/spf13/pflag"
)

const (
	labelFlagName            = "label"
	createFlagName           = "create"
	createVisibilityFlagName = "create-visibility"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <policy>",
		Short: "Push a policy to a registry",
		Long:  `The first argument is the path to the local policy config.`,
		Args:  appcmd.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Labels           []string
	Create           bool
	CreateVisibility string
	SourceControlURL string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindCreateVisibility(flagSet, &f.CreateVisibility, createVisibilityFlagName, createFlagName)
	flagSet.StringSliceVar(
		&f.Labels,
		labelFlagName,
		nil,
		"Associate the label with the policies pushed. Can be used multiple times.",
	)
	flagSet.BoolVar(
		&f.Create,
		createFlagName,
		false,
		fmt.Sprintf(
			"Create the policy if it does not exist. Defaults to creating a private policy on the BSR if --%s is not set.",
			createVisibilityFlagName,
		),
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) (retErr error) {
	if err := validateFlags(flags); err != nil {
		return err
	}
	commit, err := upload(ctx, container, flags)
	if err != nil {
		return err
	}
	// Only one commit is returned.
	if _, err := fmt.Fprintf(container.Stdout(), "%s\n", commit.PolicyKey().String()); err != nil {
		return syserror.Wrap(err)
	}
	return nil
}

func upload(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) (_ bufpolicy.Commit, retErr error) {
	var policy bufpolicy.Policy
	if container.NumArgs() != 1 {
		// This should never happen because the args length is validated.
		return nil, syserror.Newf("policy arg must be provided")
	}
	policyFilePath := container.Arg(0)
	// We read the policy YAML file.
	data, err := os.ReadFile(policyFilePath)
	if err != nil {
		return nil, appcmd.NewInvalidArgumentErrorf("could not read policy file %q: %w", policyFilePath, err)
	}
	// Parse the policy YAML file to validate it upfront.
	policyYamlFile, err := bufpolicyconfig.ReadBufPolicyYAMLFile(bytes.NewReader(data), policyFilePath)
	if err != nil {
		return nil, appcmd.NewInvalidArgumentErrorf("unable to validate policy file %q: %w", policyFilePath, err)
	}
	// We parse the policy full name from the user-provided argument.
	if policyYamlFile.Name() == "" {
		return nil, appcmd.NewInvalidArgumentErrorf("policy file %q must have a name", policyFilePath)
	}
	policyFullName, err := bufparse.ParseFullName(policyYamlFile.Name())
	if err != nil {
		return nil, appcmd.NewInvalidArgumentErrorf("unable to parse policy full name %q: %w", policyYamlFile.Name(), err)
	}
	policy, err = bufpolicy.NewPolicy(policyFilePath, policyFullName, policyFullName.Name(), uuid.Nil, policyYamlFile.PolicyConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create policy from file %q: %w", policyFilePath, err)
	}
	uploader, err := bufcli.NewPolicyUploader(container)
	if err != nil {
		return nil, err
	}
	var options []bufpolicy.UploadOption
	if flags.Create {
		createPolicyVisibility, err := bufpolicy.ParsePolicyVisibility(flags.CreateVisibility)
		if err != nil {
			return nil, err
		}
		options = append(options, bufpolicy.UploadWithCreateIfNotExist(
			createPolicyVisibility,
		))
	}
	if len(flags.Labels) > 0 {
		options = append(options, bufpolicy.UploadWithLabels(flags.Labels...))
	}
	if flags.SourceControlURL != "" {
		options = append(options, bufpolicy.UploadWithSourceControlURL(flags.SourceControlURL))
	}
	commits, err := uploader.Upload(ctx, []bufpolicy.Policy{policy}, options...)
	if err != nil {
		return nil, err
	}
	if len(commits) != 1 {
		return nil, syserror.Newf("unexpected number of commits returned from server: %d", len(commits))
	}
	return commits[0], nil
}

func validateFlags(flags *flags) error {
	if err := validateLabelFlags(flags); err != nil {
		return err
	}
	if err := validateCreateFlags(flags); err != nil {
		return err
	}
	return nil
}

func validateLabelFlags(flags *flags) error {
	return validateLabelFlagValues(flags)
}

func validateLabelFlagValues(flags *flags) error {
	if slices.Contains(flags.Labels, "") {
		return appcmd.NewInvalidArgumentErrorf("--%s requires a non-empty string", labelFlagName)
	}
	return nil
}

func validateCreateFlags(flags *flags) error {
	if flags.Create {
		if flags.CreateVisibility == "" {
			return appcmd.NewInvalidArgumentErrorf("--%s must be set if --%s is set", createVisibilityFlagName, createFlagName)
		}
		if _, err := bufpolicy.ParsePolicyVisibility(flags.CreateVisibility); err != nil {
			return appcmd.WrapInvalidArgumentError(err)
		}
	}
	return nil
}
