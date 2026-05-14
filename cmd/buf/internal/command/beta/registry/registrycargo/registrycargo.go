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

// Package registrycargo implements the "buf beta registry cargo" command,
// which acts as a Cargo "cargo:token-from-stdout" credential provider for
// BSR-hosted Cargo registries.
package registrycargo

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/private/bufpkg/bufconnect"
	"github.com/bufbuild/buf/private/pkg/netrc"
	"github.com/spf13/pflag"
)

// errSilent is returned to exit with a non-zero status without printing a
// "Failure: ..." line to stderr. The top-level wrapError in cmd/buf/buf.go
// returns errors whose Error() is "" unchanged, bypassing the failure
// wrapping. This is the same mechanism used by other commands that need a
// silent non-zero exit (see private/pkg/bandeps/cmd/bandeps/main.go).
var errSilent = errors.New("")

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " [host...]",
		Short: "Cargo credential provider for BSR-hosted Cargo registries",
		Long: `This command implements Cargo's "cargo:token-from-stdout" credential-provider protocol for BSR-hosted Cargo registries.

Add the following to your ~/.cargo/config.toml to make Cargo use buf as its credential provider for the public BSR at buf.build:

    [registry]
    global-credential-providers = ["cargo:token-from-stdout buf beta registry cargo"]

To opt in to a different set of hosts (for example, an Enterprise BSR instance), list them as positional arguments. The positional arguments replace the default; buf.build is not implicitly included. Ports in positional arguments are stripped before matching, so the allow-list operates at the hostname level only:

    [registry]
    global-credential-providers = ["cargo:token-from-stdout buf beta registry cargo bsr.example.com"]

Multiple hosts are supported:

    [registry]
    global-credential-providers = ["cargo:token-from-stdout buf beta registry cargo buf.build bsr.example.com"]

Tokens are looked up using the existing buf authentication chain: the BUF_TOKEN environment variable, then ~/.netrc. Manage tokens with "buf registry login".

Failure behavior:

  - If the host extracted from CARGO_REGISTRY_INDEX_URL is not in the allow-list (or no host can be extracted, or CARGO_REGISTRY_INDEX_URL is unset), the command exits non-zero with no output so Cargo can fall through to its next configured credential provider.
  - If the host is in the allow-list but no token resolves, the command writes a "Failure:" message to stderr pointing at "buf registry login" and exits non-zero.

Pass --debug to log host-resolution and token-lookup steps.`,
		Args: appcmd.ArbitraryArgs,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct{}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	logger := container.Logger()

	rawURL := container.Env("CARGO_REGISTRY_INDEX_URL")
	if rawURL == "" {
		logger.Debug("CARGO_REGISTRY_INDEX_URL not set; nothing to do")
		return errSilent
	}
	host := hostFromCargoRegistryURL(rawURL)
	if host == "" {
		logger.Debug(
			"no host could be extracted from CARGO_REGISTRY_INDEX_URL",
			"url", rawURL,
		)
		return errSilent
	}

	allowedHosts := effectiveAllowedHosts(container)
	if !slices.Contains(allowedHosts, host) {
		logger.Debug(
			"host not in allow-list; falling through to next cargo credential provider",
			"host", host,
			"allowed_hosts", allowedHosts,
		)
		return errSilent
	}

	envTokenProvider, err := bufconnect.NewTokenProviderFromContainer(container)
	if err != nil {
		// Deliberately do not include err in the user-visible message:
		// bufconnect.NewTokenProviderFromContainer formats the raw BUF_TOKEN
		// value into its errors (see newMultipleTokenProvider in
		// static_token_provider.go), which would echo the user's secret to
		// stderr. The full error is available at --debug.
		logger.Debug("BUF_TOKEN failed to parse", "error", err.Error())
		return fmt.Errorf(
			`the %[1]s environment variable could not be parsed. Either unset %[1]s, or run "buf registry login %[2]s" to populate ~/.netrc instead. Run with --debug for parser details`,
			bufconnect.TokenEnvKey, host,
		)
	}
	netrcTokenProvider := bufconnect.NewNetrcTokenProvider(container, netrc.GetMachineForName)

	for _, provider := range []bufconnect.TokenProvider{envTokenProvider, netrcTokenProvider} {
		if token := provider.RemoteToken(host); token != "" {
			if _, err := fmt.Fprintf(container.Stdout(), "Bearer %s\n", token); err != nil {
				return err
			}
			return nil
		}
	}

	logger.Debug("no token found for host", "host", host)
	return fmt.Errorf(
		`no token found for %[1]s. Run "buf registry login %[1]s", using a Buf API token as the password. For details, visit https://buf.build/docs/bsr/authentication`,
		host,
	)
}

// hostFromCargoRegistryURL returns the host extracted from a Cargo registry
// index URL, or "" if no host can be determined.
//
// It strips a leading "sparse+" prefix and parses the remainder as a URL.
// The returned host has any port stripped and is lowercased so the allow-
// list comparison in run() is case-insensitive (DNS hosts are
// case-insensitive but net/url preserves the original case).
func hostFromCargoRegistryURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	trimmed := strings.TrimPrefix(rawURL, "sparse+")
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}

// effectiveAllowedHosts returns the positional host allow-list, normalized
// (lowercased, port stripped) for case-insensitive comparison against the
// URL host. If no positional arguments were supplied, returns the default
// [buf.build].
func effectiveAllowedHosts(container appext.Container) []string {
	numArgs := container.NumArgs()
	if numArgs == 0 {
		return []string{bufconnect.DefaultRemote}
	}
	hosts := make([]string, numArgs)
	for i := range numArgs {
		hosts[i] = normalizeHost(container.Arg(i))
	}
	return hosts
}

// normalizeHost lowercases s and strips an optional port. It accepts
// "host", "host:port", and "[host]:port" forms; if splitting fails (e.g.
// the input is a bare hostname with no port), the lowercased input is
// returned unchanged. This makes positional-arg allow-list entries
// comparable to URL-extracted hosts, which url.URL.Hostname() also
// returns without a port.
//
// Bare IPv6 literals (e.g. "::1") trip net.SplitHostPort's "too many
// colons" check and fall through to the unchanged-input branch, matching
// what url.URL.Hostname() returns for "[::1]" hosts.
func normalizeHost(s string) string {
	lower := strings.ToLower(s)
	if host, _, err := net.SplitHostPort(lower); err == nil {
		return host
	}
	return lower
}
