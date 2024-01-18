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

package modopen

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

// NewCommand returns a new open Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	return &appcmd.Command{
		Use:   name + " <directory>",
		Short: "Open the module's homepage in a web browser",
		Long:  `The first argument is the directory of the local module to open. Defaults to "." if no argument is specified.`,
		Args:  cobra.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container)
			},
			bufcli.NewErrorInterceptor(),
		),
	}
}

// run tidy to trim the buf.lock file for a specific module.
func run(
	ctx context.Context,
	container appflag.Container,
) error {
	directoryInput, err := bufcli.GetInputValue(container, "", ".")
	if err != nil {
		return err
	}
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		directoryInput,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	existingConfigFilePath, err := bufconfig.ExistingConfigFilePath(ctx, readWriteBucket)
	if err != nil {
		return err
	}
	if existingConfigFilePath == "" {
		return bufcli.ErrNoConfigFile
	}
	config, err := bufconfig.GetConfigForBucket(ctx, readWriteBucket)
	if err != nil {
		return err
	}
	var moduleIdentityString string
	if config.ModuleIdentity != nil {
		moduleIdentityString = config.ModuleIdentity.IdentityString()
	}
	if moduleIdentityString == "" {
		return fmt.Errorf("%s has no module name", existingConfigFilePath)
	}
	return browser.OpenURL("https://" + moduleIdentityString)
}
