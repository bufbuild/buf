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

package bufmodule

import (
	"sort"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"go.uber.org/multierr"
)

// protoFileTracker tracks if we found a .proto file for each Module tracked, and what the OpaqueIDs
// are for each unique .proto file path tracked are.
//
// This allows us to fulfill the documentation for ModuleReadBucket on Module where at least
// one .proto file will exist in a ModuleReadBucket, and lets us discover if there are duplicate
// paths across Modules.
type protoFileTracker struct {
	opaqueIDToProtoFileExists map[string]bool
	protoPathToOpaqueIDMap    map[string]map[string]struct{}
}

func newProtoFileTracker() *protoFileTracker {
	return &protoFileTracker{
		opaqueIDToProtoFileExists: make(map[string]bool),
		protoPathToOpaqueIDMap:    make(map[string]map[string]struct{}),
	}
}

// trackModule says to track the Module to see if it has any .proto files.
//
// If this is never called, it will simply result in no NoProtoFilesErrors being produced.
func (t *protoFileTracker) trackModule(module Module) {
	opaqueID := module.OpaqueID()
	if _, ok := t.opaqueIDToProtoFileExists[opaqueID]; !ok {
		t.opaqueIDToProtoFileExists[opaqueID] = false
	}
}

// trackFileInfo says to track the FileInfo to mark its associated Module as having .proto files
// if the FileInfo represents a .proto file, and to mark its path as having the associated Module's
// opaqueID.
func (t *protoFileTracker) trackFileInfo(fileInfo FileInfo) {
	if fileInfo.FileType() != FileTypeProto {
		return
	}
	opaqueID := fileInfo.Module().OpaqueID()
	t.opaqueIDToProtoFileExists[opaqueID] = true
	protoPathOpaqueIDMap, ok := t.protoPathToOpaqueIDMap[fileInfo.Path()]
	if !ok {
		protoPathOpaqueIDMap = make(map[string]struct{})
		t.protoPathToOpaqueIDMap[fileInfo.Path()] = protoPathOpaqueIDMap
	}
	protoPathOpaqueIDMap[opaqueID] = struct{}{}
}

// validate validates. This should be called when all tracking is complete.
func (t *protoFileTracker) validate() error {
	var noProtoFilesErrors []*NoProtoFilesError
	for opaqueID, protoFileExists := range t.opaqueIDToProtoFileExists {
		if !protoFileExists {
			noProtoFilesErrors = append(
				noProtoFilesErrors,
				&NoProtoFilesError{
					OpaqueID: opaqueID,
				},
			)
		}
	}
	var duplicateProtoPathErrors []*DuplicateProtoPathError
	for protoPath, opaqueIDMap := range t.protoPathToOpaqueIDMap {
		if len(opaqueIDMap) > 1 {
			duplicateProtoPathErrors = append(
				duplicateProtoPathErrors,
				&DuplicateProtoPathError{
					ProtoPath: protoPath,
					OpaqueIDs: slicesext.MapKeysToSortedSlice(opaqueIDMap),
				},
			)
		}
	}
	if len(noProtoFilesErrors) != 0 || len(duplicateProtoPathErrors) != 0 {
		sort.Slice(
			noProtoFilesErrors,
			func(i int, j int) bool {
				return noProtoFilesErrors[i].OpaqueID < noProtoFilesErrors[j].OpaqueID
			},
		)
		sort.Slice(
			duplicateProtoPathErrors,
			func(i int, j int) bool {
				return duplicateProtoPathErrors[i].ProtoPath < duplicateProtoPathErrors[j].ProtoPath
			},
		)
		errs := make([]error, 0, len(noProtoFilesErrors)+len(duplicateProtoPathErrors))
		for _, noProtoFilesError := range noProtoFilesErrors {
			errs = append(errs, noProtoFilesError)
		}
		for _, duplicateProtoPathError := range duplicateProtoPathErrors {
			errs = append(errs, duplicateProtoPathError)
		}
		// multierr.Combine special-cases len(errs) == 1, so no need for us to do so.
		return multierr.Combine(errs...)
	}
	return nil
}
