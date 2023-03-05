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

package studioagent

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufstudioagent"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/cert/certclient"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/transport/http/httpserver"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	bindFlagName              = "bind"
	portFlagName              = "port"
	originFlagName            = "origin"
	disallowedHeadersFlagName = "disallowed-header"
	forwardHeadersFlagName    = "forward-header"
	caCertFlagName            = "ca-cert"
	clientCertFlagName        = "client-cert"
	clientKeyFlagName         = "client-key"
	serverCertFlagName        = "server-cert"
	serverKeyFlagName         = "server-key"
	privateNetworkFlagName    = "private-network"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "Run an HTTP(S) server as the Studio agent",
		Args:  cobra.ExactArgs(0),
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
	BindAddress       string
	Port              string
	Origin            string
	DisallowedHeaders []string
	ForwardHeaders    map[string]string
	CACert            string
	ClientCert        string
	ClientKey         string
	ServerCert        string
	ServerKey         string
	PrivateNetwork    bool
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.BindAddress,
		bindFlagName,
		"127.0.0.1",
		"The address to be exposed to accept HTTP requests",
	)
	flagSet.StringVar(
		&f.Port,
		portFlagName,
		"8080",
		"The port to be exposed to accept HTTP requests",
	)
	flagSet.StringVar(
		&f.Origin,
		originFlagName,
		"https://studio.buf.build",
		"The allowed origin for CORS options",
	)
	flagSet.StringSliceVar(
		&f.DisallowedHeaders,
		disallowedHeadersFlagName,
		nil,
		`The header names that are disallowed by this agent. When the agent receives an enveloped request with these headers set, it will return an error rather than forward the request to the target server. Multiple headers are appended if specified multiple times`,
	)
	flagSet.StringToStringVar(
		&f.ForwardHeaders,
		forwardHeadersFlagName,
		nil,
		`The headers to be forwarded via the agent to the target server. Must be an equals sign separated key-value pair (like --forward-header=fromHeader1=toHeader1). Multiple header pairs are appended if specified multiple times`,
	)
	flagSet.StringVar(
		&f.CACert,
		caCertFlagName,
		"",
		"The CA cert to be used in the client and server TLS configuration",
	)
	flagSet.StringVar(
		&f.ClientCert,
		clientCertFlagName,
		"",
		"The cert to be used in the client TLS configuration",
	)
	flagSet.StringVar(
		&f.ClientKey,
		clientKeyFlagName,
		"",
		"The key to be used in the client TLS configuration",
	)
	flagSet.StringVar(
		&f.ServerCert,
		serverCertFlagName,
		"",
		"The cert to be used in the server TLS configuration",
	)
	flagSet.StringVar(
		&f.ServerKey,
		serverKeyFlagName,
		"",
		"The key to be used in the server TLS configuration",
	)
	flagSet.BoolVar(
		&f.PrivateNetwork,
		privateNetworkFlagName,
		false,
		`Use the agent with private network CORS`,
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	// CA cert pool is optional. If it is nil, TLS uses the host's root CA set.
	var rootCAConfig *tls.Config
	var err error
	if flags.CACert != "" {
		rootCAConfig, err = certclient.NewClientTLSConfigFromRootCertFiles(flags.CACert)
		if err != nil {
			return err
		}
	}
	// client TLS config is optional. If it is nil, it uses the default configuration from http2.Transport.
	var clientTLSConfig *tls.Config
	if flags.ClientCert != "" || flags.ClientKey != "" {
		clientTLSConfig, err = newTLSConfig(rootCAConfig, flags.ClientCert, flags.ClientKey)
		if err != nil {
			return fmt.Errorf("cannot create new client TLS config: %w", err)
		}
	}
	// server TLS config is optional. If it is nil, we serve with a h2c handler.
	var serverTLSConfig *tls.Config
	if flags.ServerCert != "" || flags.ServerKey != "" {
		serverTLSConfig, err = newTLSConfig(rootCAConfig, flags.ServerCert, flags.ServerKey)
		if err != nil {
			return fmt.Errorf("cannot create new server TLS config: %w", err)
		}
	}
	mux := bufstudioagent.NewHandler(
		container.Logger(),
		flags.Origin,
		clientTLSConfig,
		stringutil.SliceToMap(flags.DisallowedHeaders),
		flags.ForwardHeaders,
		flags.PrivateNetwork,
	)
	var httpListenConfig net.ListenConfig
	httpListener, err := httpListenConfig.Listen(ctx, "tcp", fmt.Sprintf("%s:%s", flags.BindAddress, flags.Port))
	if err != nil {
		return err
	}

	return httpserver.Run(
		ctx,
		container.Logger(),
		httpListener,
		mux,
		httpserver.RunWithTLSConfig(
			serverTLSConfig,
		),
	)
}

func newTLSConfig(baseConfig *tls.Config, certFile, keyFile string) (*tls.Config, error) {
	config := baseConfig.Clone()
	if config == nil {
		config = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("error creating x509 keypair from cert file %s and key file %s", certFile, keyFile)
	}
	config.Certificates = []tls.Certificate{cert}
	return config, nil
}
