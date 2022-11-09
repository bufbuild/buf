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

package push

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/connect-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	tagFlagName             = "tag"
	tagFlagShortName        = "t"
	draftFlagName           = "draft"
	errorFormatFlagName     = "error-format"
	disableSymlinksFlagName = "disable-symlinks"
	// deprecated
	trackFlagName = "track"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <source>",
		Short: "Push a module to a registry.",
		Long:  bufcli.GetSourceLong(`the source to push`),
		Args:  cobra.MaximumNArgs(1),
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
	Tags            []string
	Draft           string
	ErrorFormat     string
	DisableSymlinks bool
	// Deprecated
	Tracks []string
	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)
	flagSet.StringSliceVarP(
		&f.Tags,
		tagFlagName,
		tagFlagShortName,
		nil,
		fmt.Sprintf(
			"Create a tag for the pushed commit. Multiple tags are created if specified multiple times. Cannot be used together with --%s.",
			draftFlagName,
		),
	)
	flagSet.StringVar(
		&f.Draft,
		draftFlagName,
		"",
		fmt.Sprintf(
			"Make the pushed commit a draft with the specified name. Cannot be used together with --%s (-%s).",
			tagFlagName,
			tagFlagShortName,
		),
	)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors printed to stderr. Must be one of %s.",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	flagSet.StringSliceVar(
		&f.Tracks,
		trackFlagName,
		nil,
		"Do not use. This flag never had any effect",
	)
	_ = flagSet.MarkHidden(trackFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) (retErr error) {
	if len(flags.Tracks) > 0 {
		return appcmd.NewInvalidArgumentErrorf("--%s has never had any effect, do not use.", trackFlagName)
	}
	if err := bufcli.ValidateErrorFormatFlag(flags.ErrorFormat, errorFormatFlagName); err != nil {
		return err
	}
	if len(flags.Tags) > 0 && flags.Draft != "" {
		return appcmd.NewInvalidArgumentErrorf("--%s (-%s) and --%s cannot be used together.", tagFlagName, tagFlagShortName, draftFlagName)
	}
	source, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	storageosProvider := bufcli.NewStorageosProvider(flags.DisableSymlinks)
	runner := command.NewRunner()
	// We are pushing to the BSR, this module has to be independently buildable
	// given the configuration it has without any enclosing workspace.
	sourceBucket, sourceConfig, err := bufcli.BucketAndConfigForSource(
		ctx,
		container.Logger(),
		container,
		storageosProvider,
		runner,
		source,
	)
	if err != nil {
		return err
	}
	moduleIdentity := sourceConfig.ModuleIdentity
	module, err := bufcli.ReadModule(
		ctx,
		container.Logger(),
		sourceBucket,
		sourceConfig,
	)
	if err != nil {
		return err
	}
	protoModule, err := bufmodule.ModuleToProtoModule(ctx, module)
	if err != nil {
		return err
	}
	manifest, err := manifestBlob(ctx, sourceBucket)
	if err != nil {
		return err
	}
	blobs, err := bucketBlobs(ctx, sourceBucket)
	if err != nil {
		return err
	}
	apiProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
	service, err := apiProvider.NewPushService(ctx, moduleIdentity.Remote())
	if err != nil {
		return err
	}
	localModulePin, err := service.Push(
		ctx,
		moduleIdentity.Owner(),
		moduleIdentity.Repository(),
		"",
		protoModule,
		flags.Tags,
		nil,
		flags.Draft,
		manifest,
		blobs,
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			if _, err := container.Stderr().Write(
				[]byte("The latest commit has the same content; not creating a new commit.\n"),
			); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if localModulePin == nil {
		return errors.New("Missing local module pin in the registry's response.")
	}
	if _, err := container.Stdout().Write([]byte(localModulePin.Commit + "\n")); err != nil {
		return err
	}
	return nil
}

// manifestBlob returns a canonical manifest in blob form from a bucket.
func manifestBlob(
	ctx context.Context,
	bucket storage.ReadBucket,
) (*modulev1alpha1.Blob, error) {
	moduleManifest, err := manifest.NewFromBucket(ctx, bucket)
	if err != nil {
		return nil, err
	}
	manifestText, err := moduleManifest.MarshalText()
	if err != nil {
		return nil, err
	}
	return manifest.NewBlobFromReader(bytes.NewReader(manifestText))
}

// bucketBlobs returns a deduplicated set of blobs for the files in bucket.
func bucketBlobs(
	ctx context.Context,
	bucket storage.ReadBucket,
) ([]*modulev1alpha1.Blob, error) {
	var blobs []*modulev1alpha1.Blob
	haveBlob := make(map[string]struct{})
	if walkErr := bucket.Walk(ctx, "", func(info storage.ObjectInfo) error {
		path := info.Path()
		file, err := bucket.Get(ctx, path)
		if err != nil {
			return err
		}
		defer file.Close()
		blob, err := manifest.NewBlobFromReader(file)
		if err != nil {
			return err
		}
		hexDigest := hex.EncodeToString(blob.Hash.Digest)
		if _, ok := haveBlob[hexDigest]; !ok {
			blobs = append(blobs, blob)
			haveBlob[hexDigest] = struct{}{}
		}
		return nil
	}); walkErr != nil {
		return nil, walkErr
	}
	return blobs, nil
}
