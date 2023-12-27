// Copyright 2020-2023 Buf Technologies, Inc.
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

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	ownerv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleapi"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
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
	if err := validateCreateFlags(flags); err != nil {
		return err
	}

	moduleSet, err := getBuildableModuleSet(ctx, container, flags)
	if err != nil {
		return err
	}
	singleRegistryHostname, err := getSingleRegistryHostname(moduleSet)
	if err != nil {
		return err
	}

	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	clientProvider := bufapi.NewClientProvider(clientConfig)

	// We just do this for the future world in where we might want to allow
	// more than one registry, even though we don't allow this with the below upload request.
	registryToTargetModules, err := getRegistryToTargetModuleWithModuleFullName(moduleSet)
	if err != nil {
		return err
	}
	moduleVisiblity, err := bufmoduleapi.ParseModuleVisibility(flags.CreateVisibility)
	if err != nil {
		return err
	}
	if flags.Create {
		if err := createTargetModulesIfNotExist(
			ctx,
			clientProvider,
			registryToTargetModules,
			moduleVisiblity,
		); err != nil {
			return err
		}
	} else {
		if err := validateTargetModulesExist(
			ctx,
			clientProvider,
			registryToTargetModules,
		); err != nil {
			return err
		}
	}

	protoUploadRequest, err := bufmoduleapi.NewUploadRequest(
		ctx,
		moduleSet,
		bufmoduleapi.UploadRequestWithLabels(combineLabelLikeFlags(flags)...),
	)
	response, err := clientProvider.UploadServiceClient(singleRegistryHostname).Upload(
		ctx,
		connect.NewRequest(protoUploadRequest),
	)
	if err != nil {
		return err
	}
	if _, err := container.Stdout().Write(
		[]byte(
			strings.Join(
				slicesext.Map(
					response.Msg.Commits,
					func(protoCommit *modulev1beta1.Commit) string { return protoCommit.Id },
				),
				"\n",
			) + "\n",
		),
	); err != nil {
		return err
	}
	return nil
}

func getBuildableModuleSet(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) (bufmodule.ModuleSet, error) {
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
	workspace, err := controller.GetWorkspace(ctx, source)
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

func getRegistryToTargetModuleWithModuleFullName(moduleSet bufmodule.ModuleSet) (map[string][]bufmodule.Module, error) {
	registryToTargetModules := make(map[string][]bufmodule.Module)
	for _, module := range bufmodule.ModuleSetTargetModules(moduleSet) {
		moduleFullName := module.ModuleFullName()
		if moduleFullName == nil {
			return nil, newRequireModuleFullNameOnUploadError(module)
		}
		registryToTargetModules[moduleFullName.Registry()] = append(
			registryToTargetModules[moduleFullName.Registry()],
			module,
		)
	}
	return registryToTargetModules, nil
}

func validateTargetModulesExist(
	ctx context.Context,
	clientProvider bufapi.ClientProvider,
	registryToTargetModules map[string][]bufmodule.Module,
) error {
	for registry, targetModules := range registryToTargetModules {
		if _, err := clientProvider.ModuleServiceClient(registry).GetModules(
			ctx,
			connect.NewRequest(
				&modulev1beta1.GetModulesRequest{
					ModuleRefs: slicesext.Map(
						targetModules,
						func(module bufmodule.Module) *modulev1beta1.ModuleRef {
							return &modulev1beta1.ModuleRef{
								Value: &modulev1beta1.ModuleRef_Name_{
									Name: &modulev1beta1.ModuleRef_Name{
										Owner:  module.ModuleFullName().Owner(),
										Module: module.ModuleFullName().Name(),
									},
								},
							}
						},
					),
				},
			),
		); err != nil {
			return err
		}
	}
	return nil
}

func createTargetModulesIfNotExist(
	ctx context.Context,
	clientProvider bufapi.ClientProvider,
	registryToTargetModules map[string][]bufmodule.Module,
	moduleVisibility modulev1beta1.ModuleVisibility,
) error {
	for registry, targetModules := range registryToTargetModules {
		if _, err := clientProvider.ModuleServiceClient(registry).CreateModules(
			ctx,
			connect.NewRequest(
				&modulev1beta1.CreateModulesRequest{
					Values: slicesext.Map(
						targetModules,
						func(module bufmodule.Module) *modulev1beta1.CreateModulesRequest_Value {
							return &modulev1beta1.CreateModulesRequest_Value{
								OwnerRef: &ownerv1beta1.OwnerRef{
									Value: &ownerv1beta1.OwnerRef_Name{
										Name: module.ModuleFullName().Owner(),
									},
								},
								Name:       module.ModuleFullName().Name(),
								Visibility: moduleVisibility,
							}
						},
					),
				},
			),
		); err != nil && connect.CodeOf(err) != connect.CodeAlreadyExists {
			return err
		}
	}
	return nil
}

// getSingleRegistryHostname validates that all Modules have ModuleFullNames, and that
// all Modules have the same registry, and returns that registry
//
// We do the same validation in bufmoduleapi.NewUploadRequest, but we want to do it upfront
// here so we don't do any RPC calls otherwise, including create calls. We might want to just
// move NewUploadRequest into this file.
func getSingleRegistryHostname(moduleSet bufmodule.ModuleSet) (string, error) {
	// We check upfront if all modules have names, before contining onwards.
	for _, module := range moduleSet.Modules() {
		if module.ModuleFullName() == nil {
			return "", newRequireModuleFullNameOnUploadError(module)
		}
	}
	// Validate we're all within one registry for now.
	registryHostnames := slicesext.ToUniqueSorted(
		slicesext.Map(
			moduleSet.Modules(),
			func(module bufmodule.Module) string { return module.ModuleFullName().Registry() },
		),
	)
	if len(registryHostnames) > 1 {
		// TODO: This messes up legacy federation.
		return "", fmt.Errorf("multiple registries detected: %s", strings.Join(registryHostnames, ", "))
	}
	return registryHostnames[0], nil
}

func validateCreateFlags(flags *flags) error {
	if flags.Create {
		if flags.CreateVisibility == "" {
			return appcmd.NewInvalidArgumentErrorf(
				"--%s is required if --%s is set.",
				createVisibilityFlagName,
				createFlagName,
			)
		}
		if _, err := bufmoduleapi.ParseModuleVisibility(flags.CreateVisibility); err != nil {
			return appcmd.NewInvalidArgumentError(err.Error())
		}
	} else {
		if flags.CreateVisibility != "" {
			return appcmd.NewInvalidArgumentErrorf(
				"Cannot set --%s without --%s.",
				createVisibilityFlagName,
				createFlagName,
			)
		}
	}
	return nil
}

func combineLabelLikeFlags(flags *flags) []string {
	return slicesext.ToUniqueSorted(
		append(
			flags.Labels,
			append(
				flags.Tags,
				flags.Draft,
				flags.Branch,
			)...,
		),
	)
}

func newRequireModuleFullNameOnUploadError(module bufmodule.Module) error {
	// This error will likely actually go back to users.
	// TODO: We copied this from NewUploadRequest, we may want to make this a system error over there.
	return fmt.Errorf("A name must be specified in buf.yaml for module %s for push.", module.OpaqueID())
}
