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

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimageutil"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleconfig"
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
	messageFullNameToFile, err := getSortedMessageFullNameToFile(protosourceFiles)
	if err != nil {
		return err
	}
	enumFullNameToFile, err := getSortedEnumFullNameToFile(protosourceFiles)
	if err != nil {
		return err
	}
	golangFileData, err := getGolangFileData(
		pathToData,
		messageFullNameToFile,
		enumFullNameToFile,
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
	module, err := bufmodulebuild.NewModuleBucketBuilder(container.Logger()).BuildForBucket(
		ctx,
		bucket,
		&bufmoduleconfig.Config{},
	)
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
	if len(fileAnnotations) > 0 {
		// stderr since we do output to stdouo
		if err := bufanalysis.PrintFileAnnotations(
			container.Stderr(),
			fileAnnotations,
			"text",
		); err != nil {
			return nil, err
		}
		return nil, app.NewError(100, "")
	}
	if err != nil {
		return nil, err
	}
	return protosource.NewFilesUnstable(ctx, bufimageutil.NewInputFiles(image.Files())...)
}

func getSortedMessageFullNameToFile(protosourceFiles []protosource.File) ([]*stringPair, error) {
	fullNameToMessage, err := protosource.FullNameToMessage(protosourceFiles...)
	if err != nil {
		return nil, err
	}
	fullNameToFile := make([]*stringPair, 0, len(fullNameToMessage))
	for fullName, message := range fullNameToMessage {
		fullNameToFile = append(
			fullNameToFile,
			newStringPair(
				fullName,
				message.File().Path(),
			),
		)
	}
	sortStringPairs(fullNameToFile)
	return fullNameToFile, nil
}

func getSortedEnumFullNameToFile(protosourceFiles []protosource.File) ([]*stringPair, error) {
	fullNameToEnum, err := protosource.FullNameToEnum(protosourceFiles...)
	if err != nil {
		return nil, err
	}
	fullNameToFile := make([]*stringPair, 0, len(fullNameToEnum))
	for fullName, enum := range fullNameToEnum {
		fullNameToFile = append(
			fullNameToFile,
			newStringPair(
				fullName,
				enum.File().Path(),
			),
		)
	}
	sortStringPairs(fullNameToFile)
	return fullNameToFile, nil
}

func getGolangFileData(
	pathToData map[string][]byte,
	messageFullNameToFile []*stringPair,
	enumFullNameToFile []*stringPair,
	packageName string,
) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	_, _ = buffer.WriteString(`// Code generated by `)
	_, _ = buffer.WriteString(programName)
	_, _ = buffer.WriteString(`. DO NOT EDIT.

package `)
	_, _ = buffer.WriteString(packageName)
	_, _ = buffer.WriteString(`

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
		_, _ = buffer.WriteString(`"`)
		_, _ = buffer.WriteString(path)
		_, _ = buffer.WriteString(`": {
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
			_, _ = buffer.WriteString(accum)
			_, _ = buffer.WriteString("\n")
			data = data[n:]
		}
		_, _ = buffer.WriteString(`},
`)
	}
	_, _ = buffer.WriteString(`}
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
`)

	return format.Source(buffer.Bytes())
}

type stringPair struct {
	key   string
	value string
}

func newStringPair(key string, value string) *stringPair {
	return &stringPair{
		key:   key,
		value: value,
	}
}

func sortStringPairs(stringPairs []*stringPair) {
	sort.Slice(
		stringPairs,
		func(i int, j int) bool {
			return stringPairs[i].key < stringPairs[j].key
		},
	)
}
