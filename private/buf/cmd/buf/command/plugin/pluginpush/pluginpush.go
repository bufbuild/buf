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

package pluginpush

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	pluginv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/plugin/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiplugin"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/bufbuild/buf/private/pkg/wasm"
	"github.com/klauspost/compress/zstd"
	"github.com/spf13/pflag"
)

const (
	labelFlagName            = "label"
	binaryFlagName           = "binary"
	sourceControlURLFlagName = "source-control-url"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <remote/owner/plugin>",
		Short: "Push a plugin to a registry",
		Long:  `The first argument is the plugin full name in the format <remote/owner/plugin>.`,
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
	Labels           []string
	Binary           string
	SourceControlURL string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVar(
		&f.Labels,
		labelFlagName,
		nil,
		"Associate the label with the plugins pushed. Can be used multiple times.",
	)
	flagSet.StringVar(
		&f.Binary,
		binaryFlagName,
		"",
		"The path to the Wasm binary file to push.",
	)
	flagSet.StringVar(
		&f.SourceControlURL,
		sourceControlURLFlagName,
		"",
		"The URL for viewing the source code of the pushed plugins (e.g. the specific commit in source control).",
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
	// We parse the plugin full name from the user-provided argument.
	pluginFullName, err := bufparse.ParseFullName(container.Arg(0))
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}

	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	pluginKey, err := upload(ctx, container, flags, clientConfig, pluginFullName)
	if err != nil {
		return err
	}
	// Only one plugin key is returned.
	if _, err := fmt.Fprintf(container.Stdout(), "%s\n", pluginKey.String()); err != nil {
		return syserror.Wrap(err)
	}
	return nil
}

func upload(
	ctx context.Context,
	container appext.Container,
	flags *flags,
	clientConfig *connectclient.Config,
	pluginFullName bufparse.FullName,
) (_ bufplugin.PluginKey, retErr error) {
	switch {
	case flags.Binary != "":
		return uploadBinary(ctx, container, flags, clientConfig, pluginFullName)
	default:
		// This should never happen because the flags are validated.
		return nil, syserror.Newf("--%s must be set", binaryFlagName)
	}
}

func uploadBinary(
	ctx context.Context,
	container appext.Container,
	flags *flags,
	clientConfig *connectclient.Config,
	pluginFullName bufparse.FullName,
) (pluginKey bufplugin.PluginKey, retErr error) {
	uploadServiceClient := bufregistryapiplugin.NewClientProvider(clientConfig).
		V1Beta1UploadServiceClient(pluginFullName.Registry())

	wasmRuntimeCacheDir, err := bufcli.CreateWasmRuntimeCacheDir(container)
	if err != nil {
		return nil, err
	}
	wasmRuntime, err := wasm.NewRuntime(ctx, wasm.WithLocalCacheDir(wasmRuntimeCacheDir))
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, wasmRuntime.Close(ctx))
	}()
	// Load the binary from the `--binary` flag.
	wasmBinary, err := os.ReadFile(flags.Binary)
	if err != nil {
		return nil, fmt.Errorf("could not read binary %q: %w", flags.Binary, err)
	}
	compressionType := pluginv1beta1.CompressionType_COMPRESSION_TYPE_ZSTD
	compressedWasmBinary, err := zstdCompress(wasmBinary)
	if err != nil {
		return nil, fmt.Errorf("could not compress binary %q: %w", flags.Binary, err)
	}

	// Defer validation of the plugin binary to the server, but compile the
	// binary locally to catch any errors early.
	_, err = wasmRuntime.Compile(ctx, pluginFullName.Name(), wasmBinary)
	if err != nil {
		return nil, fmt.Errorf("could not compile binary %q: %w", flags.Binary, err)
	}
	// Upload the binary to the registry.
	content := &pluginv1beta1.UploadRequest_Content{
		PluginRef: &pluginv1beta1.PluginRef{
			Value: &pluginv1beta1.PluginRef_Name_{
				Name: &pluginv1beta1.PluginRef_Name{
					Owner:  pluginFullName.Owner(),
					Plugin: pluginFullName.Name(),
				},
			},
		},
		CompressionType: compressionType,
		Content:         compressedWasmBinary,
		ScopedLabelRefs: slicesext.Map(flags.Labels, func(label string) *pluginv1beta1.ScopedLabelRef {
			return &pluginv1beta1.ScopedLabelRef{
				Value: &pluginv1beta1.ScopedLabelRef_Name{
					Name: label,
				},
			}
		}),
		SourceControlUrl: flags.SourceControlURL,
	}
	uploadResponse, err := uploadServiceClient.Upload(ctx, connect.NewRequest(&pluginv1beta1.UploadRequest{
		Contents: []*pluginv1beta1.UploadRequest_Content{content},
	}))
	if err != nil {
		return nil, err
	}
	if len(uploadResponse.Msg.Commits) != 1 {
		return nil, syserror.Newf("unexpected number of commits returned from server: %d", len(uploadResponse.Msg.Commits))
	}
	protoCommit := uploadResponse.Msg.Commits[0]
	commitID, err := uuidutil.FromDashless(protoCommit.Id)
	if err != nil {
		return nil, err
	}
	pluginKey, err = bufplugin.NewPluginKey(
		pluginFullName,
		commitID,
		func() (bufplugin.Digest, error) {
			return v1beta1ProtoToDigest(protoCommit.Digest)
		},
	)
	if err != nil {
		return nil, err
	}
	return pluginKey, nil
}

func zstdCompress(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
	}
	defer encoder.Close()
	return encoder.EncodeAll(data, nil), nil
}

func validateFlags(flags *flags) error {
	if err := validateLabelFlags(flags); err != nil {
		return err
	}
	if err := validateTypeFlags(flags); err != nil {
		return err
	}
	return nil
}

func validateLabelFlags(flags *flags) error {
	return validateLabelFlagValues(flags)
}

func validateTypeFlags(flags *flags) error {
	var typeFlags []string
	if flags.Binary != "" {
		typeFlags = append(typeFlags, binaryFlagName)
	}
	if len(typeFlags) > 1 {
		usedFlagsErrStr := strings.Join(
			slicesext.Map(
				typeFlags,
				func(flag string) string { return fmt.Sprintf("--%s", flag) },
			),
			", ",
		)
		return appcmd.NewInvalidArgumentErrorf("These flags cannot be used in combination with one another: %s", usedFlagsErrStr)
	}
	if len(typeFlags) == 0 {
		return appcmd.NewInvalidArgumentErrorf("--%s must be set", binaryFlagName)
	}
	return nil
}

func validateLabelFlagValues(flags *flags) error {
	for _, label := range flags.Labels {
		if label == "" {
			return appcmd.NewInvalidArgumentErrorf("--%s requires a non-empty string", labelFlagName)
		}
	}
	return nil
}
