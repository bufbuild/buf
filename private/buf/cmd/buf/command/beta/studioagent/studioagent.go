// Copyright 2020-2022 Buf Technologies, Inc.
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

package studioagent

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufstudioagent"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/interrupt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	portFlagName              = "port"
	disallowedHeadersFlagName = "disallowed-headers"
	forwardHeadersFlagName    = "forward-headers"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <origin>",
		Short: "Run a HTTP server as the studio agent with the origin be set as allowed origin for CORS options.",
		Args:  cobra.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
			bufcli.NewErrorInterceptor(),
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Port              string
	DisallowedHeaders []string
	ForwardHeaders    []string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Port,
		portFlagName,
		"8080",
		"The port to be exposed to accept HTTP request.",
	)
	flagSet.StringSliceVar(
		&f.DisallowedHeaders,
		disallowedHeadersFlagName,
		nil,
		`The headers to be disallowed via the agent to the target server. Must be a comma-separated string (like --disallowed-headers=header1,header2).`,
	)
	flagSet.StringSliceVar(
		&f.ForwardHeaders,
		forwardHeadersFlagName,
		nil,
		`The headers to be forwarded via the agent to the target server. Must be a comma-separated string of colon-separated key-value pair (like --forward-headers=fromHeader1:toHeader1,fromHeader2:toHeader2).`,
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	disallowedHeaders := make(map[string]struct{}, len(flags.DisallowedHeaders))
	for _, header := range flags.DisallowedHeaders {
		disallowedHeaders[header] = struct{}{}
	}
	forwardHeaders := make(map[string]string, len(flags.ForwardHeaders))
	for _, pair := range flags.ForwardHeaders {
		s := strings.Split(pair, ":")
		if len(s) != 2 {
			return fmt.Errorf("unknown key-pair value of forward-headers: %s", pair)
		}
		forwardHeaders[s[0]] = s[1]
	}
	config, err := bufcli.NewConfig(container)
	if err != nil {
		return err
	}
	mux := bufstudioagent.NewHandler(container.Logger(), container.Arg(0), config.TLS, disallowedHeaders, forwardHeaders)
	server := http.Server{
		Addr:    ":" + flags.Port,
		Handler: mux,
	}
	signalC, closer := interrupt.NewSignalChannel()
	go func() {
		if err = server.ListenAndServe(); err != nil {
			closer()
			return
		}
	}()
	<-signalC
	if err != nil {
		return err
	}
	return server.Shutdown(ctx)
}
