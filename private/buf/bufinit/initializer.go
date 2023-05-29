// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufinit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
)

type initializer struct {
	logger *zap.Logger
}

func newInitializer(logger *zap.Logger) *initializer {
	return &initializer{
		logger: logger,
	}
}

func (i *initializer) Initialize(
	ctx context.Context,
	readWriteBucket storage.ReadWriteBucket,
	options ...InitializeOption,
) error {
	initializeOptions := &initializeOptions{}
	for _, option := range options {
		option(initializeOptions)
	}
	return i.initialize(ctx, readWriteBucket, initializeOptions.excludePaths)
}

// Questions you might want to ask in an interactive prompt:
//
//   - Do you have a directory, or a set of directories, where you commonly include your .proto files from?
//     Based on the answer here, this might be where you start in terms of your directories you want to include
//   - Do you have files that you know you want to ignore?
//     Based on the answer here, we might be able to blanket exclude things like testdata
//
// Current "best":
//
// 1. Take the list of package-inferred include paths, these are usually right.
// 2. Delete the shortest paths that cause overlaps with other paths - can't have overlaps
// 3. Figure out what import paths are not covered by this list, add this import dir paths back.
// 4. Do the same delete of overlaps.
// 5. Report if you can compile or not.
func (i *initializer) initialize(
	ctx context.Context,
	readWriteBucket storage.ReadWriteBucket,
	excludePaths []string,
) error {
	calculator := newCalculator(i.logger)
	if len(excludePaths) > 0 {
		var notOrMatchers []storage.Matcher
		for _, excludePath := range excludePaths {
			// Doesn't have the compilicated logic in bufmodule because I don't think
			// it is necessary here
			notOrMatchers = append(
				notOrMatchers,
				storage.MatchPathEqualOrContained(excludePath),
			)
		}
		readWriteBucket = storage.MapReadWriteBucket(
			readWriteBucket,
			storage.MatchNot(
				storage.MatchOr(
					notOrMatchers...,
				),
			),
		)
	}
	calculation, err := calculator.Calculate(ctx, readWriteBucket)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(calculation, "", "  ")
	if err != nil {
		return err
	}
	//fmt.Println("*** CALCULATION ***")
	//fmt.Println()
	fmt.Println(string(data))
	//fmt.Println()

	//fmt.Println("*** MESSAGE SO FAR ***")
	//fmt.Println()
	//if len(calculation.MissingImportPathToFilePaths) > 0 {
	//for missingImportPath, filePathMap := range calculation.MissingImportPathToFilePaths {
	//fmt.Printf("%s is imported by %v but is not found in the current directory.\n", missingImportPath, stringutil.SliceToHumanString(stringutil.MapToSortedSlice(filePathMap)))
	//}
	//fmt.Println()
	//fmt.Println("Given that you have missing imports, buf will not be able to build directly.")
	//fmt.Println()
	//}
	//if importDirPaths := calculation.AllImportDirPaths(); len(importDirPaths) > 0 {
	//fmt.Println("Directories that need a buf.yaml:")
	//fmt.Println()
	//for _, importDirPath := range importDirPaths {
	//fmt.Println(importDirPath)
	//}
	//} else {
	//fmt.Println("No directories need a buf.yaml.")
	//}
	//fmt.Println()

	//fmt.Println("*** THEORETICAL PROTOC COMMAND ***")
	//fmt.Println()
	//buffer := bytes.NewBuffer(nil)
	//buffer.WriteString("protoc -o /dev/null")
	//for _, importDirPath := range calculation.AllImportDirPaths() {
	//buffer.WriteString(" \\ \n-I \"")
	//buffer.WriteString(importDirPath)
	//buffer.WriteString("\"")
	//}
	//buffer.WriteString(" \\ \n$(find . -name '*.proto')")
	//fmt.Println(buffer.String())
	return nil
}

type initializeOptions struct {
	excludePaths []string
}
