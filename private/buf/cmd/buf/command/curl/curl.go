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

package curl

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufcurl"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/netrc"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"github.com/spf13/pflag"
	"go.uber.org/multierr"
	"golang.org/x/net/http2"
)

const (
	// Input schema flags
	schemaFlagName = "schema"

	// Reflection flags
	reflectFlagName         = "reflect"
	reflectHeaderFlagName   = "reflect-header"
	reflectProtocolFlagName = "reflect-protocol"

	// Protocol/transport flags
	protocolFlagName            = "protocol"
	unixSocketFlagName          = "unix-socket"
	http2PriorKnowledgeFlagName = "http2-prior-knowledge"

	// TLS flags
	keyFlagName           = "key"
	certFlagName          = "cert"
	certFlagShortName     = "E"
	caCertFlagName        = "cacert"
	serverNameFlagName    = "servername"
	insecureFlagName      = "insecure"
	insecureFlagShortName = "k"

	// Timeout flags
	noKeepAliveFlagName    = "no-keepalive"
	keepAliveFlagName      = "keepalive-time"
	connectTimeoutFlagName = "connect-timeout"

	// Header and request body flags
	userAgentFlagName      = "user-agent"
	userAgentFlagShortName = "A"
	userFlagName           = "user"
	userFlagShortName      = "u"
	netrcFlagName          = "netrc"
	netrcFlagShortName     = "n"
	netrcFileFlagName      = "netrc-file"
	headerFlagName         = "header"
	headerFlagShortName    = "H"
	dataFlagName           = "data"
	dataFlagShortName      = "d"

	// Output flags
	outputFlagName       = "output"
	outputFlagShortName  = "o"
	emitDefaultsFlagName = "emit-defaults"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <url>",
		Short: "Invoke an RPC endpoint, a la 'cURL'",
		Long: `This command helps you invoke HTTP RPC endpoints on a server that uses gRPC or Connect.

By default, server reflection is used, unless the --reflect flag is set to false. Without server
reflection, a --schema flag must be provided to indicate the Protobuf schema for the method being
invoked.

The only positional argument is the URL of the RPC method to invoke. The name of the method to
invoke comes from the last two path components of the URL, which should be the fully-qualified
service name and method name, respectively.

The URL can use either http or https as the scheme. If http is used then HTTP 1.1 will be used
unless the --http2-prior-knowledge flag is set. If https is used then HTTP/2 will be preferred
during protocol negotiation and HTTP 1.1 used only if the server does not support HTTP/2.

The default RPC protocol used will be Connect. To use a different protocol (gRPC or gRPC-Web),
use the --protocol flag. Note that the gRPC protocol cannot be used with HTTP 1.1.

The input request is specified via the -d or --data flag. If absent, an empty request is sent. If
the flag value starts with an at-sign (@), then the rest of the flag value is interpreted as a
filename from which to read the request body. If that filename is just a dash (-), then the request
body is read from stdin. The request body is a JSON document that contains the JSON formatted
request message. If the RPC method being invoked is a client-streaming method, the request body may
consist of multiple JSON values, appended to one another. Multiple JSON documents should usually be
separated by whitespace, though this is not strictly required unless the request message type has a
custom JSON representation that is not a JSON object.

Request metadata (i.e. headers) are defined using -H or --header flags. The flag value is in
"name: value" format. But if it starts with an at-sign (@), the rest of the value is interpreted as
a filename from which headers are read, each on a separate line. If the filename is just a dash (-),
then the headers are read from stdin.

If headers and the request body are both to be read from the same file (or both read from stdin),
the file must include headers first, then a blank line, and then the request body.

Examples:

Issue a unary RPC to a plain-text (i.e. "h2c") gRPC server, where the schema for the service is
in a Buf module in the current directory, using an empty request message:

    $ buf curl --schema . --protocol grpc --http2-prior-knowledge  \
         http://localhost:20202/foo.bar.v1.FooService/DoSomething

Issue an RPC to a Connect server, where the schema comes from the Buf Schema Registry, using
a request that is defined as a command-line argument:

    $ buf curl --schema buf.build/connectrpc/eliza  \
         --data '{"name": "Bob Loblaw"}'          \
         https://demo.connectrpc.com/connectrpc.eliza.v1.ElizaService/Introduce

Issue a unary RPC to a server that supports reflection, with verbose output:

    $ buf curl --data '{"sentence": "I am not feeling well."}' -v  \
		 https://demo.connectrpc.com/connectrpc.eliza.v1.ElizaService/Say

Issue a client-streaming RPC to a gRPC-web server that supports reflection, where custom
headers and request data are both in a heredoc:

    $ buf curl --data @- --header @- --protocol grpcweb                              \
		 https://demo.connectrpc.com/connectrpc.eliza.v1.ElizaService/Converse  \
       <<EOM
    Custom-Header-1: foo-bar-baz
    Authorization: token jas8374hgnkvje9wpkerebncjqol4

    {"sentence": "Hi, doc. I feel hungry."}
    {"sentence": "What is the answer to life, the universe, and everything?"}
    {"sentence": "If you were a fish, what of fish would you be?."}
    EOM

Note that server reflection (i.e. use of the --reflect flag) does not work with HTTP 1.1 since the
protocol relies on bidirectional streaming. If server reflection is used, the assumed URL for the
reflection service is the same as the given URL, but with the last two elements removed and
replaced with the service and method name for server reflection.

If an error occurs that is due to incorrect usage or other unexpected error, this program will
return an exit code that is less than 8. If the RPC fails otherwise, this program will return an
exit code that is the gRPC code, shifted three bits to the left.
`,
		Args: appcmd.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	// Flags for defining input schema
	Schemas []string

	// Flags for server reflection
	Reflect         bool
	ReflectHeaders  []string
	ReflectProtocol string

	// Protocol details
	Protocol            string
	UnixSocket          string
	HTTP2PriorKnowledge bool

	// TLS
	Key, Cert, CACert, ServerName string
	Insecure                      bool
	// TODO: CRLFile, CertStatus

	// Timeouts
	NoKeepAlive           bool
	KeepAliveTimeSeconds  float64
	ConnectTimeoutSeconds float64

	// Handling request and response data and metadata
	UserAgent string
	User      string
	Netrc     bool
	NetrcFile string
	Headers   []string
	Data      string

	// Output options
	Output       string
	EmitDefaults bool

	// so we can inquire about which flags present on command-line
	// TODO: ideally we'd use cobra directly instead of having the appcmd wrapper,
	//  which prevents a lot of basic functionality by not exposing many cobra features
	flagSet *pflag.FlagSet
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	f.flagSet = flagSet

	flagSet.StringSliceVar(
		&f.Schemas,
		schemaFlagName,
		nil,
		fmt.Sprintf(
			`The module to use for the RPC schema. This is necessary if the server does not support
server reflection. The format of this argument is the same as for the <input> arguments to
other buf sub-commands such as build and generate. It can indicate a directory, a file, a
remote module in the Buf Schema Registry, or even standard in ("-") for feeding an image or
file descriptor set to the command in a shell pipeline.
If multiple %s flags are present, they will be consulted in order to resolve service and type
names. Setting this flags implies --%s=false unless a %s flag is explicitly present. If both
%s and %s flags are in use, reflection will be used first and the schemas will be consulted
in order thereafter if reflection fails to resolve a schema element.`,
			schemaFlagName, reflectFlagName, reflectFlagName,
			schemaFlagName, reflectFlagName,
		),
	)
	flagSet.BoolVar(
		&f.Reflect,
		reflectFlagName,
		true,
		`If true, use server reflection to determine the schema`,
	)
	flagSet.StringSliceVar(
		&f.ReflectHeaders,
		reflectHeaderFlagName,
		nil,
		fmt.Sprintf(`Request headers to include with reflection requests. This flag may only be used
when --%s is also set. This flag may be specified more than once to indicate
multiple headers. Each flag value should have the form "name: value". But a special value
of '*' may be used to indicate that all normal request headers (from --%s and -%s
flags) should also be included with reflection requests. A special value of '@<path>'
means to read headers from the file at <path>. If the path is "-" then headers are
read from stdin. It is not allowed to indicate a file with the same path as used with
the request data flag (--%s or -%s). Furthermore, it is not allowed to indicate stdin
if the schema is expected to be provided via stdin as a file descriptor set or image`,
			reflectFlagName, headerFlagName, headerFlagShortName, dataFlagName, dataFlagShortName,
		),
	)
	flagSet.StringVar(
		&f.ReflectProtocol,
		reflectProtocolFlagName,
		"",
		`The reflection protocol to use for downloading information from the server. This flag
may only be used when server reflection is used. By default, this command will try all known
reflection protocols from newest to oldest. If this results in a "Not Implemented" error,
then older protocols will be used. In practice, this means that "grpc-v1" is tried first,
and "grpc-v1alpha" is used if it doesn't work. If newer reflection protocols are introduced,
they may be preferred in the absence of this flag being explicitly set to a specific protocol.
The valid values for this flag are "grpc-v1" and "grpc-v1alpha". These correspond to services
named "grpc.reflection.v1.ServerReflection" and "grpc.reflection.v1alpha.ServerReflection"
respectively`,
	)

	flagSet.StringVar(
		&f.Protocol,
		protocolFlagName,
		connect.ProtocolConnect,
		`The RPC protocol to use. This can be one of "grpc", "grpcweb", or "connect"`,
	)
	flagSet.StringVar(
		&f.UnixSocket,
		unixSocketFlagName,
		"",
		`The path to a unix socket that will be used instead of opening a TCP socket to the host
and port indicated in the URL`,
	)
	flagSet.BoolVar(
		&f.HTTP2PriorKnowledge,
		http2PriorKnowledgeFlagName,
		false,
		`This flag can be used to indicate that HTTP/2 should be used. Without this, HTTP 1.1
will be used with URLs with an http scheme, and protocol negotiation will be used to
choose either HTTP 1.1 or HTTP/2 for URLs with an https scheme. With this flag set,
HTTP/2 is always used, even over plain-text.`,
	)

	flagSet.BoolVar(
		&f.NoKeepAlive,
		noKeepAliveFlagName,
		false,
		`By default, connections are created using TCP keepalive. If this flag is present, they
will be disabled`,
	)
	flagSet.Float64Var(
		&f.KeepAliveTimeSeconds,
		keepAliveFlagName,
		60,
		`The duration, in seconds, between TCP keepalive transmissions`,
	)
	flagSet.Float64Var(
		&f.ConnectTimeoutSeconds,
		connectTimeoutFlagName,
		0,
		`The time limit, in seconds, for a connection to be established with the server. There is
no limit if this flag is not present`,
	)

	flagSet.StringVar(
		&f.Key,
		keyFlagName,
		"",
		fmt.Sprintf(`Path to a PEM-encoded X509 private key file, for using client certificates with TLS. This
option is only valid when the URL uses the https scheme. A --%s or -%s flag must also be
present to provide the certificate and public key that corresponds to the given
private key`,
			certFlagName, certFlagShortName,
		),
	)
	flagSet.StringVarP(
		&f.Cert,
		certFlagName,
		certFlagShortName,
		"",
		fmt.Sprintf(`Path to a PEM-encoded X509 certificate file, for using client certificates with TLS. This
option is only valid when the URL uses the https scheme. A --%s flag must also be
present to provide the private key that corresponds to the given certificate`,
			keyFlagName,
		),
	)
	flagSet.StringVar(
		&f.CACert,
		caCertFlagName,
		"",
		fmt.Sprintf(`Path to a PEM-encoded X509 certificate pool file that contains the set of trusted
certificate authorities/issuers. If omitted, the system's default set of trusted
certificates are used to verify the server's certificate. This option is only valid
when the URL uses the https scheme. It is not applicable if --%s or -%s flag is used`,
			insecureFlagName, insecureFlagShortName,
		),
	)
	flagSet.BoolVarP(
		&f.Insecure,
		insecureFlagName,
		insecureFlagShortName,
		false,
		`If set, the TLS connection will be insecure and the server's certificate will NOT be
verified. This is generally discouraged. This option is only valid when the URL uses
the https scheme`,
	)
	flagSet.StringVar(
		&f.ServerName,
		serverNameFlagName,
		"",
		`The server name to use in TLS handshakes (for SNI) if the URL scheme is https. If not
specified, the default is the origin host in the URL or the value in a "Host" header if
one is provided`,
	)

	flagSet.StringVarP(
		&f.UserAgent,
		userAgentFlagName,
		userAgentFlagShortName,
		"",
		fmt.Sprintf(`The user agent string to send. This is ignored if a --%s or -%s flag is provided
that sets a header named 'User-Agent'.`,
			headerFlagName, headerFlagShortName,
		),
	)
	flagSet.StringVarP(
		&f.User,
		userFlagName,
		userFlagShortName,
		"",
		fmt.Sprintf(`The user credentials to send, via a basic authorization header. The value should be
in the format "username:password". If the value has no colon, it is assumed to just be
the username, in which case you will be prompted to enter a password. This overrides
the use of a .netrc file. This is ignored if a --%s or -%s flag is provided that sets
a header named 'Authorization'.`,
			headerFlagName, headerFlagShortName,
		),
	)
	flagSet.BoolVarP(
		&f.Netrc,
		netrcFlagName,
		netrcFlagShortName,
		false,
		fmt.Sprintf(`If true, a file named .netrc in the user's home directory will be examined to find
credentials for the request. The credentials will be sent via a basic authorization header.
The command will fail if the file does not have an entry for the hostname in the URL. This
flag is ignored if a --%s or -%s flag is present. This is ignored if a --%s or -%s flag
is provided that sets a header named 'Authorization'.`,
			userFlagName, userFlagShortName, headerFlagName, headerFlagShortName,
		),
	)
	flagSet.StringVar(
		&f.NetrcFile,
		netrcFileFlagName,
		"",
		fmt.Sprintf(`This is just like use --%s or -%s, except that the named file is used instead of
a file named .netrc in the user's home directory. This flag cannot be used with the --%s
or -%s flag. This is ignored if a --%s or -%s flag is provided that sets a header named
'Authorization'.`,
			netrcFlagName, netrcFlagShortName, netrcFlagName, netrcFlagShortName, headerFlagName, headerFlagShortName,
		),
	)
	flagSet.StringSliceVarP(
		&f.Headers,
		headerFlagName,
		headerFlagShortName,
		nil,
		fmt.Sprintf(`Request headers to include with the RPC invocation. This flag may be specified more
than once to indicate multiple headers. Each flag value should have the form "name: value".
A special value of '@<path>' means to read headers from the file at <path>. If the path
is "-" then headers are read from stdin. If the same file is indicated as used with the
request data flag (--%s or -%s), the file must contain all headers, then a blank line,
and then the request body. It is not allowed to indicate stdin if the schema is expected
to be provided via stdin as a file descriptor set or image`,
			dataFlagName, dataFlagShortName,
		),
	)
	flagSet.StringVarP(
		&f.Data,
		dataFlagName,
		dataFlagShortName,
		"",
		fmt.Sprintf(`Request data. This should be zero or more JSON documents, each indicating a request
message. For unary RPCs, there should be exactly one JSON document. A special value of
'@<path>' means to read the data from the file at <path>. If the path is "-" then the
request data is read from stdin. If the same file is indicated as used with the request
headers flags (--%s or -%s), the file must contain all headers, then a blank line, and
then the request body. It is not allowed to indicate stdin if the schema is expected to be
provided via stdin as a file descriptor set or image`,
			headerFlagName, headerFlagShortName,
		),
	)
	flagSet.StringVarP(
		&f.Output,
		outputFlagName,
		outputFlagShortName,
		"",
		`Path to output file to create with response data. If absent, response is printed to stdout`,
	)
	flagSet.BoolVar(
		&f.EmitDefaults,
		emitDefaultsFlagName,
		false,
		`Emit default values for JSON-encoded responses.`,
	)
}

func (f *flags) validate(isSecure bool) error {
	if (f.Key != "" || f.Cert != "" || f.CACert != "" || f.ServerName != "" || f.flagSet.Changed(insecureFlagName)) &&
		!isSecure {
		return fmt.Errorf(
			"TLS flags (--%s, --%s, --%s, --%s, --%s) should not be used unless URL is secure (https)",
			keyFlagName, certFlagName, caCertFlagName, insecureFlagName, serverNameFlagName)
	}
	if (f.Key != "") != (f.Cert != "") {
		return fmt.Errorf("if one of --%s or --%s flags is used, both should be used (mutual TLS with a client certificate requires both)", keyFlagName, certFlagName)
	}
	if f.Insecure && f.CACert != "" {
		return fmt.Errorf("if --%s is set, --%s should not be set as it is unused", insecureFlagName, caCertFlagName)
	}

	if !isSecure && !f.HTTP2PriorKnowledge && f.Protocol == connect.ProtocolGRPC {
		return fmt.Errorf("grpc protocol cannot be used with plain-text URLs (http) unless --%s flag is set", http2PriorKnowledgeFlagName)
	}

	if f.Netrc && f.NetrcFile != "" {
		return fmt.Errorf("--%s and --%s flags are mutually exclusive; they may not both be specified", netrcFlagName, netrcFileFlagName)
	}

	if len(f.Schemas) > 0 && f.Reflect && !f.flagSet.Changed(reflectFlagName) {
		// Reflect just has default value; unset it since we're going to use --schema instead
		f.Reflect = false
	}
	if !f.Reflect && len(f.Schemas) == 0 {
		return fmt.Errorf("must specify --%s if --%s is false", schemaFlagName, reflectFlagName)
	}
	var schemaIsStdin bool
	for _, schema := range f.Schemas {
		isStdin := strings.HasPrefix(schema, "-")
		if isStdin && schemaIsStdin {
			// more than one schema argument wants to use stdin
			return fmt.Errorf("multiple --%s flags indicate the use of stdin which is not allowed", schemaFlagName)
		}
		if isStdin {
			schemaIsStdin = true
		}
	}
	if (len(f.ReflectHeaders) > 0 || f.flagSet.Changed(reflectProtocolFlagName)) && !f.Reflect {
		return fmt.Errorf(
			"reflection flags (--%s, --%s) should not be used if --%s is false",
			reflectHeaderFlagName, reflectProtocolFlagName, reflectFlagName)
	}
	if f.Reflect {
		if !isSecure && !f.HTTP2PriorKnowledge {
			return fmt.Errorf("--%s cannot be used with plain-text URLs (http) unless --%s flag is set", reflectFlagName, http2PriorKnowledgeFlagName)
		}
		if _, err := bufcurl.ParseReflectProtocol(f.ReflectProtocol); err != nil {
			return fmt.Errorf(
				"--%s value must be one of %s",
				reflectProtocolFlagName,
				stringutil.SliceToHumanStringOrQuoted(bufcurl.AllKnownReflectProtocolStrings),
			)
		}
	}

	switch f.Protocol {
	case connect.ProtocolConnect, connect.ProtocolGRPC, connect.ProtocolGRPCWeb:
	default:
		return fmt.Errorf(
			"--%s value must be one of %q, %q, or %q",
			protocolFlagName, connect.ProtocolConnect, connect.ProtocolGRPC, connect.ProtocolGRPCWeb)
	}

	if f.NoKeepAlive && f.flagSet.Changed(keepAliveFlagName) {
		return fmt.Errorf("--%s should not be specified if keepalive is disabled", keepAliveFlagName)
	}
	if f.KeepAliveTimeSeconds <= 0 {
		return fmt.Errorf("--%s value must be positive", keepAliveFlagName)
	}
	// these two default to zero (which means no timeout in effect)
	if f.ConnectTimeoutSeconds < 0 || (f.ConnectTimeoutSeconds == 0 && f.flagSet.Changed(connectTimeoutFlagName)) {
		return fmt.Errorf("--%s value must be positive", connectTimeoutFlagName)
	}

	var dataFile string
	if strings.HasPrefix(f.Data, "@") {
		dataFile = strings.TrimPrefix(f.Data, "@")
		if dataFile == "" {
			return fmt.Errorf("--%s value starting with '@' must indicate '-' for stdin or a filename", dataFlagName)
		}
		if dataFile == "-" && schemaIsStdin {
			return fmt.Errorf("--%s and --%s flags cannot both indicate reading from stdin", schemaFlagName, dataFlagName)
		}
	}

	headerFiles := map[string]struct{}{}
	if err := validateHeaders(f.Headers, headerFlagName, schemaIsStdin, false, headerFiles); err != nil {
		return err
	}
	reflectHeaderFiles := map[string]struct{}{}
	if err := validateHeaders(f.ReflectHeaders, reflectHeaderFlagName, schemaIsStdin, true, reflectHeaderFiles); err != nil {
		return err
	}
	for file := range reflectHeaderFiles {
		if file == dataFile {
			return fmt.Errorf("--%s and --%s flags cannot indicate the same source", dataFlagName, reflectHeaderFlagName)
		}
	}

	return nil
}

func (f *flags) determineCredentials(
	ctx context.Context,
	container interface {
		app.Container
		appext.VerboseContainer
	},
	host string,
) (string, error) {
	if f.User != "" {
		// this flag overrides any netrc-related flags
		parts := strings.SplitN(f.User, ":", 2)
		username := parts[0]
		var password string
		if len(parts) < 2 {
			var err error
			password, err = promptForPassword(ctx, container, fmt.Sprintf("Enter host password for user %q:", username))
			if err != nil {
				return "", fmt.Errorf("could not prompt for password: %w", err)
			}
		} else {
			password = parts[1]
		}
		return basicAuth(username, password), nil
	}

	// process netrc-related flags
	netrcFile := f.NetrcFile
	if netrcFile == "" {
		if !f.Netrc {
			// no netrc file usage, so no creds
			return "", nil
		}
		var err error
		netrcFile, err = netrc.GetFilePath(container)
		if err != nil {
			return "", fmt.Errorf("could not determine path to .netrc file: %w", err)
		}
	}
	if _, err := os.Stat(netrcFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// This mirrors the behavior of curl when a netrc file does not exist ¯\_(ツ)_/¯
			container.VerbosePrinter().Printf("* Couldn't find host %s in the file %q; using no credentials", host, netrcFile)
			return "", nil
		}
		if !strings.Contains(err.Error(), netrcFile) {
			// make sure error message contains path to file
			return "", fmt.Errorf("could not read file: %s: %w", netrcFile, err)
		}
		return "", fmt.Errorf("could not read file: %w", err)
	}
	machine, err := netrc.GetMachineForNameAndFilePath(host, netrcFile)
	if err != nil {
		return "", fmt.Errorf("could not read file: %s: %w", netrcFile, err)
	}
	if machine == nil {
		// no creds found for this host
		container.VerbosePrinter().Printf("* Couldn't find host %s in the file %q; using no credentials", host, netrcFile)
		return "", nil
	}
	username := machine.Login()
	if strings.ContainsRune(username, ':') {
		return "", fmt.Errorf("invalid credentials found for %s in %s: username %s should not contain colon", host, netrcFile, username)
	}
	password := machine.Password()
	return basicAuth(username, password), nil
}

func promptForPassword(ctx context.Context, container app.Container, prompt string) (string, error) {
	// NB: The comments below and the mechanism of handling I/O async was
	// copied from the "registry login" command.

	// If a user sends a SIGINT to buf, the top-level application context is
	// cancelled and signal masks are reset. However, during an interactive
	// login the context is not respected; for example, it takes two SIGINTs
	// to interrupt the process.

	// Ideally we could just trigger an I/O timeout by setting the deadline on
	// stdin, but when stdin is connected to a terminal the underlying fd is in
	// blocking mode making it ineligible. As changing the mode of stdin is
	// dangerous, this change takes an alternate approach of simply returning
	// early.

	// Note that this does not gracefully handle the case where the terminal is
	// in no-echo mode, as is the case when prompting for a password
	// interactively.
	ch := make(chan struct{})
	var password string
	var err error
	go func() {
		defer close(ch)
		password, err = bufcli.PromptUserForPassword(container, prompt)
	}()
	select {
	case <-ch:
		return password, err
	case <-ctx.Done():
		ctxErr := ctx.Err()
		// Otherwise we will print "Failure: context canceled".
		if errors.Is(ctxErr, context.Canceled) {
			// Otherwise the next terminal line will be on the same line as the
			// last output from buf.
			if _, err := fmt.Fprintln(container.Stdout()); err != nil {
				return "", err
			}
			return "", errors.New("interrupted")
		}
		return "", ctxErr
	}
}

func basicAuth(username, password string) string {
	var buf bytes.Buffer
	buf.WriteString(username)
	buf.WriteByte(':')
	buf.WriteString(password)
	return "Basic " + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func validateHeaders(flags []string, flagName string, schemaIsStdin bool, allowAsterisk bool, headerFiles map[string]struct{}) error {
	var hasAsterisk bool
	for _, header := range flags {
		switch {
		case strings.HasPrefix(header, "@"):
			file := strings.TrimPrefix(header, "@")
			if _, ok := headerFiles[file]; ok {
				return fmt.Errorf("multiple --%s values refer to the same file %s", flagName, file)
			}
			if file == "" {
				return fmt.Errorf("--%s value starting with '@' must indicate '-' for stdin or a filename", flagName)
			}
			if file == "-" && schemaIsStdin {
				return fmt.Errorf("--%s and --%s flags cannot both indicate reading from stdin", schemaFlagName, flagName)
			}
			headerFiles[file] = struct{}{}
		case header == "*":
			if !allowAsterisk {
				return fmt.Errorf("--%s value '*' is not valid", flagName)
			}
			if hasAsterisk {
				return fmt.Errorf("multiple --%s values both indicate '*'", flagName)
			}
			hasAsterisk = true
		case header == "":
			return fmt.Errorf("--%s value cannot be blank", flagName)
		case strings.ContainsRune(header, '\n'):
			return fmt.Errorf("--%s value cannot contain a newline", flagName)
		default:
			parts := strings.SplitN(header, ":", 2)
			if len(parts) < 2 {
				return fmt.Errorf("--%s value is a malformed header: %q", flagName, header)
			}
		}
	}

	return nil
}

func verifyEndpointURL(urlArg string) (endpointURL *url.URL, service, method, baseURL string, err error) {
	endpointURL, err = url.Parse(urlArg)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("%q is not a valid endpoint URL: %w", urlArg, err)
	}
	if endpointURL.Scheme != "http" && endpointURL.Scheme != "https" {
		return nil, "", "", "", fmt.Errorf("invalid endpoint URL: scheme %q is not supported", endpointURL.Scheme)
	}

	if strings.HasSuffix(endpointURL.Path, "/") {
		return nil, "", "", "", fmt.Errorf("invalid endpoint URL: path %q should not end with a slash (/)", endpointURL.Path)
	}
	parts := strings.Split(endpointURL.Path, "/")
	if len(parts) < 2 || parts[len(parts)-1] == "" || parts[len(parts)-2] == "" {
		return nil, "", "", "", fmt.Errorf("invalid endpoint URL: path %q should end with two non-empty components indicating service and method", endpointURL.Path)
	}
	service, method = parts[len(parts)-2], parts[len(parts)-1]
	baseURL = strings.TrimSuffix(urlArg, service+"/"+method)
	if baseURL == urlArg {
		// should not be possible due to above checks
		return nil, "", "", "", fmt.Errorf("failed to extract base URL from %q", urlArg)
	}
	return endpointURL, service, method, baseURL, nil
}

func run(ctx context.Context, container appext.Container, f *flags) (err error) {
	endpointURL, service, method, baseURL, err := verifyEndpointURL(container.Arg(0))
	if err != nil {
		return err
	}
	isSecure := endpointURL.Scheme == "https"
	if err := f.validate(isSecure); err != nil {
		return err
	}

	var clientOptions []connect.ClientOption
	switch f.Protocol {
	case connect.ProtocolGRPC:
		clientOptions = []connect.ClientOption{connect.WithGRPC()}
	case connect.ProtocolGRPCWeb:
		clientOptions = []connect.ClientOption{connect.WithGRPCWeb()}
	}
	if f.Protocol != connect.ProtocolGRPC {
		// The transport will log trailers to the verbose printer. But if
		// we're not using standard grpc protocol, trailers are actually encoded
		// in an end-of-stream message for streaming calls. So this interceptor
		// will print the trailers for streaming calls when the response stream
		// is drained.
		clientOptions = append(clientOptions, connect.WithInterceptors(bufcurl.TraceTrailersInterceptor(container.VerbosePrinter())))
	}

	dataSource := "(argument)"
	var dataFileReference string
	if strings.HasPrefix(f.Data, "@") {
		dataFileReference = strings.TrimPrefix(f.Data, "@")
		if dataFileReference == "-" {
			dataSource = "(stdin)"
		} else {
			dataSource = dataFileReference
			if absFile, err := filepath.Abs(dataFileReference); err == nil {
				dataFileReference = absFile
			}
		}
	}
	requestHeaders, dataReader, err := bufcurl.LoadHeaders(f.Headers, dataFileReference, nil)
	if err != nil {
		return err
	}
	userAgent := f.UserAgent
	if userAgent == "" {
		userAgent = bufcurl.DefaultUserAgent(f.Protocol, bufcli.Version)
	}
	if len(requestHeaders.Values("user-agent")) == 0 {
		requestHeaders.Set("user-agent", userAgent)
	}
	var basicCreds *string
	if len(requestHeaders.Values("authorization")) == 0 {
		creds, err := f.determineCredentials(ctx, container, endpointURL.Host)
		if err != nil {
			return err
		}
		if creds != "" {
			requestHeaders.Set("authorization", creds)
		}
		// set this to non-nil so we know we've already determined credentials
		basicCreds = &creds
	}
	if dataReader == nil {
		if dataFileReference == "-" {
			dataReader = os.Stdin
		} else if dataFileReference != "" {
			f, err := os.Open(dataFileReference)
			if err != nil {
				return bufcurl.ErrorHasFilename(err, dataFileReference)
			}
			dataReader = f
		} else if f.Data != "" {
			dataReader = io.NopCloser(strings.NewReader(f.Data))
		}
		// dataReader is left nil when nothing specified on command-line
	}
	defer func() {
		if dataReader != nil {
			err = multierr.Append(err, dataReader.Close())
		}
	}()

	transport, err := makeHTTPClient(f, isSecure, bufcurl.GetAuthority(endpointURL, requestHeaders), container.VerbosePrinter())
	if err != nil {
		return err
	}

	output := container.Stdout()
	if f.Output != "" {
		output, err = os.Create(f.Output)
		if err != nil {
			return bufcurl.ErrorHasFilename(err, f.Output)
		}
	}

	resolvers := make([]protoencoding.Resolver, 0, len(f.Schemas)+1)
	if f.Reflect {
		reflectHeaders, _, err := bufcurl.LoadHeaders(f.ReflectHeaders, "", requestHeaders)
		if err != nil {
			return err
		}
		if len(reflectHeaders.Values("authorization")) == 0 {
			var creds string
			if basicCreds != nil {
				creds = *basicCreds
			} else {
				if creds, err = f.determineCredentials(ctx, container, endpointURL.Host); err != nil {
					return err
				}
			}
			if creds != "" {
				reflectHeaders.Set("authorization", creds)
			}
		}
		if len(reflectHeaders.Values("user-agent")) == 0 {
			reflectHeaders.Set("user-agent", userAgent)
		}
		reflectProtocol, err := bufcurl.ParseReflectProtocol(f.ReflectProtocol)
		if err != nil {
			return err
		}
		res, closeRes := bufcurl.NewServerReflectionResolver(ctx, transport, clientOptions, baseURL, reflectProtocol, reflectHeaders, container.VerbosePrinter())
		defer closeRes()
		resolvers = append(resolvers, res)
	}
	controller, err := bufcli.NewController(container)
	if err != nil {
		return err
	}
	for _, schema := range f.Schemas {
		image, err := controller.GetImage(ctx, schema)
		if err != nil {
			return err
		}
		resolvers = append(resolvers, image.Resolver())
	}
	res := protoencoding.CombineResolvers(resolvers...)

	methodDescriptor, err := bufcurl.ResolveMethodDescriptor(res, service, method)
	if err != nil {
		return err
	}

	// Now we can finally issue the RPC
	invoker := bufcurl.NewInvoker(container, methodDescriptor, res, f.EmitDefaults, transport, clientOptions, container.Arg(0), output)
	return invoker.Invoke(ctx, dataSource, dataReader, requestHeaders)
}

func makeHTTPClient(f *flags, isSecure bool, authority string, printer verbose.Printer) (connect.HTTPClient, error) {
	var dialer net.Dialer
	if f.ConnectTimeoutSeconds != 0 {
		dialer.Timeout = secondsToDuration(f.ConnectTimeoutSeconds)
	}
	if f.NoKeepAlive {
		dialer.KeepAlive = -1
	} else {
		dialer.KeepAlive = secondsToDuration(f.KeepAliveTimeSeconds)
	}
	var dialFunc func(ctx context.Context, network, address string) (net.Conn, error)
	if f.UnixSocket != "" {
		dialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
			printer.Printf("* Dialing unix socket %s...", f.UnixSocket)
			return dialer.DialContext(ctx, "unix", f.UnixSocket)
		}
	} else {
		dialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
			printer.Printf("* Dialing (%s) %s...", network, address)
			conn, err := dialer.DialContext(ctx, network, address)
			if err != nil {
				return nil, err
			}
			printer.Printf("* Connected to %s", conn.RemoteAddr().String())
			return conn, err
		}
	}

	var dialTLSFunc func(ctx context.Context, network, address string) (net.Conn, error)
	if isSecure {
		tlsConfig, err := bufcurl.MakeVerboseTLSConfig(&bufcurl.TLSSettings{
			KeyFile:             f.Key,
			CertFile:            f.Cert,
			CACertFile:          f.CACert,
			ServerName:          f.ServerName,
			Insecure:            f.Insecure,
			HTTP2PriorKnowledge: f.HTTP2PriorKnowledge,
		}, authority, printer)
		if err != nil {
			return nil, err
		}
		dialTLSFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := dialFunc(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			printer.Printf("* ALPN: offering %s", strings.Join(tlsConfig.NextProtos, ","))
			tlsConn := tls.Client(conn, tlsConfig)
			if err := tlsConn.HandshakeContext(ctx); err != nil {
				return nil, err
			}
			return tlsConn, nil
		}
	}

	var transport http.RoundTripper
	switch {
	case f.HTTP2PriorKnowledge && isSecure:
		transport = &http2.Transport{
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return dialTLSFunc(ctx, network, addr)
			},
		}
	case f.HTTP2PriorKnowledge:
		transport = &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return dialFunc(ctx, network, addr)
			},
		}
	default:
		transport = &http.Transport{
			Proxy:             http.ProxyFromEnvironment,
			DialContext:       dialFunc,
			DialTLSContext:    dialTLSFunc,
			ForceAttemptHTTP2: true,
			MaxIdleConns:      1,
		}
	}
	return bufcurl.NewVerboseHTTPClient(transport, printer), nil
}

func secondsToDuration(secs float64) time.Duration {
	return time.Duration(float64(time.Second) * secs)
}
