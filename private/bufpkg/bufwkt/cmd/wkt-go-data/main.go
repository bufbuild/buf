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

package main

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"io"
	"math"
	"path/filepath"
	"sort"

	"golang.org/x/exp/constraints"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimageutil"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/protosource"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	programName = "wkt-go-data"
	pkgFlagName = "package"
	sliceLength = math.MaxInt64
)

var failedError = app.NewError( /* exitCode */ 100, "something went wrong")

func main() {
	appcmd.Main(context.Background(), newCommand())
}

func newCommand() *appcmd.Command {
	flags := newFlags()
	builder := appflag.NewBuilder(programName)
	return &appcmd.Command{
		Use:  fmt.Sprintf("%s path/to/google/protobuf/include", programName),
		Args: cobra.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Pkg string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Pkg,
		pkgFlagName,
		"",
		"The name of the generated package.",
	)
}

func run(ctx context.Context, container appflag.Container, flags *flags) error {
	dirPath := container.Arg(0)
	packageName := flags.Pkg
	if packageName == "" {
		packageName = filepath.Base(dirPath)
	}
	readWriteBucket, err := storageos.NewProvider(storageos.ProviderWithSymlinks()).NewReadWriteBucket(dirPath)
	if err != nil {
		return err
	}
	pathToData, err := getPathToData(ctx, readWriteBucket)
	if err != nil {
		return err
	}
	protosourceFiles, err := getProtosourceFiles(ctx, container, readWriteBucket)
	if err != nil {
		return err
	}
	fullNameToMessage, err := protosource.FullNameToMessage(protosourceFiles...)
	if err != nil {
		return err
	}
	fullNameToEnum, err := protosource.FullNameToEnum(protosourceFiles...)
	if err != nil {
		return err
	}
	golangFileData, err := getGolangFileData(
		pathToData,
		fullNameToMessage,
		fullNameToEnum,
		packageName,
	)
	if err != nil {
		return err
	}
	_, err = container.Stdout().Write(golangFileData)
	return err
}

func getPathToData(ctx context.Context, bucket storage.ReadBucket) (map[string][]byte, error) {
	pathToData := make(map[string][]byte)
	if err := storage.WalkReadObjects(
		ctx,
		bucket,
		"",
		func(readObject storage.ReadObject) error {
			data, err := io.ReadAll(readObject)
			if err != nil {
				return err
			}
			pathToData[readObject.Path()] = data
			return nil
		},
	); err != nil {
		return nil, err
	}
	return pathToData, nil
}

func getProtosourceFiles(
	ctx context.Context,
	container appflag.Container,
	bucket storage.ReadBucket,
) ([]protosource.File, error) {
	// TODO: why is this not working? this is the path that should be used, not NewModuleForBucket
	//module, err := bufmodulebuild.NewModuleBucketBuilder(container.Logger()).BuildForBucket(
	//ctx,
	//bucket,
	//&bufmoduleconfig.Config{},
	//)
	module, err := bufmodule.NewModuleForBucket(ctx, bucket)
	if err != nil {
		return nil, err
	}
	moduleFileSet, err := bufmodulebuild.NewModuleFileSetBuilder(
		container.Logger(),
		bufmodule.NewNopModuleReader(),
	).Build(
		ctx,
		module,
	)
	if err != nil {
		return nil, err
	}
	image, fileAnnotations, err := bufimagebuild.NewBuilder(container.Logger()).Build(
		ctx, moduleFileSet,
		bufimagebuild.WithExcludeSourceCodeInfo(),
	)
	if err != nil {
		return nil, err
	}
	if len(fileAnnotations) > 0 {
		// stderr since we do output to stdouot
		if err := bufanalysis.PrintFileAnnotations(
			container.Stderr(),
			fileAnnotations,
			"text",
		); err != nil {
			return nil, err
		}
		return nil, failedError
	}
	return protosource.NewFilesUnstable(ctx, bufimageutil.NewInputFiles(image.Files())...)
}

func getGolangFileData(
	pathToData map[string][]byte,
	fullNameToMessage map[string]protosource.Message,
	fullNameToEnum map[string]protosource.Enum,
	packageName string,
) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	p := func(s string) {
		_, _ = buffer.WriteString(s)
	}
	p(`// Code generated by `)
	p(programName)
	p(`. DO NOT EDIT.

package `)
	p(packageName)
	p(`

import (
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/normalpath"
)

var (
	// ReadBucket is the storage.ReadBucket with the static data generated for this package.
	ReadBucket storage.ReadBucket

	pathToData = map[string][]byte{
`)
	paths := make([]string, 0, len(pathToData))
	for path := range pathToData {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		p(`"`)
		p(path)
		p(`": {
`)
		data := pathToData[path]
		for len(data) > 0 {
			n := sliceLength
			if n > len(data) {
				n = len(data)
			}
			accum := ""
			for _, elem := range data[:n] {
				accum += fmt.Sprintf("0x%02x,", elem)
			}
			p(accum)
			p("\n")
			data = data[n:]
		}
		p(`},
`)
	}
	p(`}

	messageNameToFilePath = map[string]string{
`)
	for _, fullNameToMessagePair := range sortedPairs(fullNameToMessage) {
		p(`"`)
		p(fullNameToMessagePair.key)
		p(`": "`)
		p(fullNameToMessagePair.val.File().Path())
		p(`",`)
		p("\n")
	}
	p(`}

	enumNameToFilePath = map[string]string{
`)
	for _, fullNameToEnumPair := range sortedPairs(fullNameToEnum) {
		p(`"`)
		p(fullNameToEnumPair.key)
		p(`": "`)
		p(fullNameToEnumPair.val.File().Path())
		p(`",`)
		p("\n")
	}
	p(`}
)

func init() {
	readBucket, err := storagemem.NewReadBucket(pathToData)
	if err != nil {
		panic(err.Error())
	}
	ReadBucket = readBucket
}

// Exists returns true if the given path exists in the static data.
//
// The path is normalized within this function.
func Exists(path string) bool {
	_, ok := pathToData[normalpath.Normalize(path)]
	return ok
}

// MessageFilePath gets the file path for the given message, if the message exists.
func MessageFilePath(messageName string) (string, bool) {
	filePath, ok := messageNameToFilePath[messageName]
	return filePath, ok
}

// EnumFilePath gets the file path for the given enum, if the enum exists.
func EnumFilePath(enumName string) (string, bool) {
	filePath, ok := enumNameToFilePath[enumName]
	return filePath, ok
}
`)
	return format.Source(buffer.Bytes())
}

type keyValPair[K any, V any] struct {
	key K
	val V
}

func sortedPairs[K constraints.Ordered, V any](m map[K]V) []keyValPair[K, V] {
	ret := make([]keyValPair[K, V], 0, len(m))
	for key := range m {
		ret = append(ret, keyValPair[K, V]{key: key, val: m[key]})
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].key < ret[j].key
	})
	return ret
}
