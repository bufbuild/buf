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

package webhooklist

import (
	"context"
	"fmt"
	"strings"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
	webhookv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/webhook/v1"
	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/cmd/buf/internal/command/registry/webhook/internal"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiwebhook"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	scopeTypeFlagName = "scope-type"
	pageSizeFlagName  = "page-size"
	pageTokenFlagName = "page-token"
	reverseFlagName   = "reverse"
	formatFlagName    = "format"

	defaultPageSize = 10
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " " + internal.ScopeArgUsage,
		Short: "List BSR webhooks",
		Long: internal.ScopeArgLong + `

Listing an organization includes the webhooks on the modules it owns. Listing
the instance includes all webhooks visible to the caller.`,
		Args: appcmd.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
		ModifyCobra: func(cmd *cobra.Command) error {
			return bufcli.RegisterFlagCompletionOutputFormat(cmd, formatFlagName)
		},
	}
}

type flags struct {
	ScopeTypes []string
	PageSize   uint32
	PageToken  string
	Reverse    bool
	Format     string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVar(
		&f.ScopeTypes,
		scopeTypeFlagName,
		nil,
		fmt.Sprintf(
			"Only list webhooks with this scope type. May be repeated. Must be one of %s",
			strings.Join(allWebhookScopeTypeStrings(), ", "),
		),
	)
	flagSet.Uint32Var(
		&f.PageSize,
		pageSizeFlagName,
		defaultPageSize,
		`The page size.`,
	)
	flagSet.StringVar(
		&f.PageToken,
		pageTokenFlagName,
		"",
		`The page token. If more results are available, a "next_page" key is present in the --format=json output`,
	)
	flagSet.BoolVar(
		&f.Reverse,
		reverseFlagName,
		false,
		`Reverse the results`,
	)
	flagSet.StringVar(
		&f.Format,
		formatFlagName,
		bufprint.FormatText.String(),
		fmt.Sprintf(`The output format to use. Must be one of %s`, bufprint.AllFormatsString),
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	registry, webhookScopeRef, err := bufcli.ParseWebhookScope(container.Arg(0))
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	webhookScopeTypes, err := parseWebhookScopeTypes(flags.ScopeTypes)
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	order := webhookv1.ListWebhooksRequest_ORDER_CREATE_TIME_DESC
	if flags.Reverse {
		order = webhookv1.ListWebhooksRequest_ORDER_CREATE_TIME_ASC
	}
	request := &webhookv1.ListWebhooksRequest{
		PageSize:      flags.PageSize,
		PageToken:     flags.PageToken,
		Order:         order,
		HasScopeTypes: webhookScopeTypes,
	}
	switch value := webhookScopeRef.GetValue().(type) {
	case *webhookv1.WebhookScopeRef_Instance_:
	case *webhookv1.WebhookScopeRef_OrganizationRef:
		request.OrganizationRefs = []*ownerv1.OrganizationRef{value.OrganizationRef}
	case *webhookv1.WebhookScopeRef_ModuleRef:
		request.ModuleRefs = []*modulev1.ModuleRef{value.ModuleRef}
	default:
		return syserror.Newf("unknown webhook scope ref: %T", value)
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	webhookServiceClient := bufregistryapiwebhook.NewClientProvider(clientConfig).V1WebhookServiceClient(registry)
	resp, err := webhookServiceClient.ListWebhooks(ctx, request)
	if err != nil {
		return err
	}
	entities, err := internal.NewWebhookEntities(ctx, clientConfig, registry, resp.GetWebhooks())
	if err != nil {
		return err
	}
	return bufprint.PrintPage(
		container.Stdout(),
		format,
		resp.GetNextPageToken(),
		nextPageCommand(container, flags, resp.GetNextPageToken()),
		entities,
	)
}

func parseWebhookScopeTypes(webhookScopeTypeStrings []string) ([]webhookv1.WebhookScopeType, error) {
	webhookScopeTypes := make([]webhookv1.WebhookScopeType, 0, len(webhookScopeTypeStrings))
	for _, webhookScopeTypeString := range webhookScopeTypeStrings {
		value, ok := webhookv1.WebhookScopeType_value[webhookScopeTypeString]
		if !ok || value == int32(webhookv1.WebhookScopeType_WEBHOOK_SCOPE_TYPE_UNSPECIFIED) {
			return nil, fmt.Errorf(
				"unknown webhook scope type %q: must be one of %s",
				webhookScopeTypeString,
				strings.Join(allWebhookScopeTypeStrings(), ", "),
			)
		}
		webhookScopeTypes = append(webhookScopeTypes, webhookv1.WebhookScopeType(value))
	}
	return webhookScopeTypes, nil
}

func allWebhookScopeTypeStrings() []string {
	enumValueDescriptors := webhookv1.WebhookScopeType(0).Descriptor().Values()
	var webhookScopeTypeStrings []string
	for i := range enumValueDescriptors.Len() {
		enumValueDescriptor := enumValueDescriptors.Get(i)
		if enumValueDescriptor.Number() == 0 {
			continue
		}
		webhookScopeTypeStrings = append(webhookScopeTypeStrings, string(enumValueDescriptor.Name()))
	}
	return webhookScopeTypeStrings
}

func nextPageCommand(container appext.Container, flags *flags, nextPageToken string) string {
	if nextPageToken == "" {
		return ""
	}
	command := fmt.Sprintf("buf registry webhook list %s", container.Arg(0))
	for _, scopeType := range flags.ScopeTypes {
		command = fmt.Sprintf("%s --%s %s", command, scopeTypeFlagName, scopeType)
	}
	if flags.PageSize != defaultPageSize {
		command = fmt.Sprintf("%s --%s %d", command, pageSizeFlagName, flags.PageSize)
	}
	if flags.Reverse {
		command = fmt.Sprintf("%s --%s", command, reverseFlagName)
	}
	if flags.Format != bufprint.FormatText.String() {
		command = fmt.Sprintf("%s --%s %s", command, formatFlagName, flags.Format)
	}
	return fmt.Sprintf("%s --%s %s", command, pageTokenFlagName, nextPageToken)
}
