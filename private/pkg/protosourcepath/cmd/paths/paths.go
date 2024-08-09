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

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/protosourcepath"
	"github.com/bufbuild/protocompile"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	name = "paths"
)

// TODO(doria): this is shit and only for testing, delete or clean-up after
func main() {
	appcmd.Main(context.Background(), newCommand())
}

func newCommand() *appcmd.Command {
	builder := appext.NewBuilder(
		name,
		appext.BuilderWithTimeout(120*time.Second),
		appext.BuilderWithTracing(),
	)
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + "<descriptor>",
		Short: "",
		Long:  ``,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags:           flags.Bind,
		BindPersistentFlags: builder.BindRoot,
	}
}

type flags struct {
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {

}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	var paths []string
	for i := 0; i < container.NumArgs(); i++ {
		paths = append(paths, container.Arg(i))
	}
	compiler := protocompile.Compiler{
		Resolver:       &protocompile.SourceResolver{},
		SourceInfoMode: protocompile.SourceInfoExtraOptionLocations,
	}
	files, err := compiler.Compile(ctx, paths...)
	if err != nil {
		return err
	}
	for _, file := range files {
		sourceLocations := file.SourceLocations()
		if sourceLocations.Len() < 1 {
			fmt.Println("No source locatons found for file, skipping...", file.Path())
			continue
		}
		fmt.Println("---")
		fmt.Println("Source location info for file:", file.Path())
		for i := 1; i < sourceLocations.Len(); i++ {
			sourceLocation := sourceLocations.Get(i)
			fmt.Printf("path: %v\n", sourceLocation.Path)
			fmt.Println([]int32(sourceLocation.Path))
			fmt.Printf("len leading comments: %v\n", len(sourceLocation.LeadingComments))
			fmt.Printf("len trailing comments: %v\n", len(sourceLocation.TrailingComments))
			allAssociatedSourcePaths, err := protosourcepath.GetAssociatedSourcePaths(sourceLocation.Path, false)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("All associated source paths: %v\n", allAssociatedSourcePaths)
			fmt.Printf("All associated source paths: %v\n", rawAssociatedSourcePaths(allAssociatedSourcePaths))
			fullAssociatedSourcePaths, err := protosourcepath.GetAssociatedSourcePaths(sourceLocation.Path, true)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("Full associated source paths only: %v\n", fullAssociatedSourcePaths)
			fmt.Printf("Full associated source paths only: %v\n\n", rawAssociatedSourcePaths(fullAssociatedSourcePaths))
		}
	}
	fmt.Println("---")
	return nil
}

func rawAssociatedSourcePaths(associatedSourcePaths []protoreflect.SourcePath) [][]int32 {
	var rawAssociatedSourcePaths [][]int32
	for _, associatedSourcePath := range associatedSourcePaths {
		rawAssociatedSourcePaths = append(rawAssociatedSourcePaths, []int32(associatedSourcePath))
	}
	return rawAssociatedSourcePaths
}
