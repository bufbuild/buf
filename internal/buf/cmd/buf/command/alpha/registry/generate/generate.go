// Copyright 2020-2021 Buf Technologies, Inc.
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

package generate

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/multierr"
)

const (
	baseOutDirPathFlagName      = "output"
	baseOutDirPathFlagShortName = "o"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <buf.build/owner/repository:reference> <buf.build/owner/" + bufplugin.TemplatesPathName + "/plugin:version>",
		Short: "Generate files for a module using a template.",
		Args:  cobra.ExactArgs(2),
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
	BaseOutDirPath string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(
		&f.BaseOutDirPath,
		baseOutDirPathFlagName,
		baseOutDirPathFlagShortName,
		".",
		`The base directory to generate to. This is prepended to the out directories in the generation template.`,
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	moduleReference, err := bufmodule.ModuleReferenceForString(
		container.Arg(0),
	)
	if err != nil {
		return fmt.Errorf("failed to parse module reference: %w", err)
	}
	templateVersionPath := container.Arg(1)
	remote, templateOwner, templateName, templateVersion, err := bufplugin.ParseTemplateVersionPath(templateVersionPath)
	if err != nil {
		return fmt.Errorf("failed to parse template version path: %w", err)
	}
	registryProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return fmt.Errorf("failed to create API provider: %w", err)
	}
	generateService, err := registryProvider.NewGenerateService(ctx, remote)
	if err != nil {
		return fmt.Errorf("failed to dial generate service: %w", err)
	}
	imageService, err := registryProvider.NewImageService(ctx, remote)
	if err != nil {
		return fmt.Errorf("failed to dial image service: %w", err)
	}
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(flags.BaseOutDirPath)
	if err != nil {
		return fmt.Errorf("failed to create output bucket: %w", err)
	}
	image, err := imageService.GetImage(
		ctx,
		moduleReference.Owner(),
		moduleReference.Repository(),
		moduleReference.Reference(),
	)
	if err != nil {
		return fmt.Errorf("failed to get image for reference %q: %w", moduleReference, err)
	}
	files, runtimeLibraries, err := generateService.Generate(
		ctx,
		image,
		templateOwner,
		templateName,
		templateVersion,
	)
	if err != nil {
		return fmt.Errorf("failed to generate files: %w", err)
	}
	for _, file := range files {
		if err := writeFile(ctx, readWriteBucket, file); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}
	if len(runtimeLibraries) > 0 {
		if _, err := container.Stderr().Write(
			[]byte("The associated plugins declared the following runtime library dependencies:\n"),
		); err != nil {
			return err
		}
		// TODO: make into a printer and add to bufprint
		if err := bufprint.WithTabWriter(
			container.Stderr(),
			[]string{
				"Name",
				"Version",
			},
			func(writer bufprint.TabWriter) error {
				for _, library := range runtimeLibraries {
					if err := writer.Write(
						library.Name,
						library.Version,
					); err != nil {
						return err
					}
				}
				return nil
			},
		); err != nil {
			return err
		}
	}
	return nil
}

func writeFile(ctx context.Context, writeBucket storage.WriteBucket, file *registryv1alpha1.File) (retErr error) {
	writeObjectCloser, err := writeBucket.Put(ctx, file.Path)
	if err != nil {
		return fmt.Errorf("failed to put file to bucket: %w", err)
	}
	defer func() {
		retErr = multierr.Append(retErr, writeObjectCloser.Close())
	}()
	if _, err := writeObjectCloser.Write(file.Content); err != nil {
		return fmt.Errorf("failed to write out file content: %w", err)
	}
	return nil
}
