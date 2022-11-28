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

// Package main implements the ddiff command that diffs two directories.
package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
)

const (
	use = "ddiff"
)

func main() {
	appcmd.Main(context.Background(), newCommand())
}

func newCommand() *appcmd.Command {
	return &appcmd.Command{
		Use:  use + " dir1 dir2",
		Args: cobra.ExactArgs(2),
		Run:  run,
	}
}

func run(ctx context.Context, container app.Container) error {
	oneDirPath := filepath.Clean(container.Arg(0))
	twoDirPath := filepath.Clean(container.Arg(1))
	oneReadWriteBucket, err := storageos.NewProvider(storageos.ProviderWithSymlinks()).NewReadWriteBucket(oneDirPath)
	if err != nil {
		return err
	}
	twoReadWriteBucket, err := storageos.NewProvider(storageos.ProviderWithSymlinks()).NewReadWriteBucket(twoDirPath)
	if err != nil {
		return err
	}
	var oneExternalPathPrefix string
	if oneDirPath != "." {
		oneExternalPathPrefix = oneDirPath + string(os.PathSeparator)
	}
	var twoExternalPathPrefix string
	if twoDirPath != "." {
		twoExternalPathPrefix = twoDirPath + string(os.PathSeparator)
	}
	return storage.Diff(
		ctx,
		command.NewRunner(),
		container.Stdout(),
		oneReadWriteBucket,
		twoReadWriteBucket,
		storage.DiffWithExternalPaths(),
		storage.DiffWithExternalPathPrefixes(
			oneExternalPathPrefix,
			twoExternalPathPrefix,
		),
	)
}
