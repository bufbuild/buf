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

package bufprint

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

type runtimeLibraryPrinter struct {
	writer io.Writer
}

func newRuntimeLibraryPrinter(writer io.Writer) *runtimeLibraryPrinter {
	return &runtimeLibraryPrinter{
		writer: writer,
	}
}

func (t *runtimeLibraryPrinter) PrintRuntimeLibraries(ctx context.Context, format Format, runtimeLibraries ...*registryv1alpha1.RuntimeLibrary) error {
	switch format {
	case FormatText:
		return t.printRuntimeLibrariesText(ctx, runtimeLibraries...)
	case FormatJSON:
		outputRuntimeLibraries := make([]outputRuntimeLibrary, 0, len(runtimeLibraries))
		for _, runtimeLibrary := range runtimeLibraries {
			outputRuntimeLibraries = append(outputRuntimeLibraries, registryRuntimeLibraryToOutputRuntimeLibrary(runtimeLibrary))
		}
		return json.NewEncoder(t.writer).Encode(outputRuntimeLibraries)
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func (t *runtimeLibraryPrinter) printRuntimeLibrariesText(ctx context.Context, runtimeLibraries ...*registryv1alpha1.RuntimeLibrary) error {
	if len(runtimeLibraries) == 0 {
		return nil
	}
	return WithTabWriter(
		t.writer,
		[]string{
			"Name",
			"Version",
		},
		func(tabWriter TabWriter) error {
			for _, runtimeLibrary := range runtimeLibraries {
				if err := tabWriter.Write(
					runtimeLibrary.Name,
					runtimeLibrary.Version,
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

type outputRuntimeLibrary struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

func registryRuntimeLibraryToOutputRuntimeLibrary(runtimeLibrary *registryv1alpha1.RuntimeLibrary) outputRuntimeLibrary {
	return outputRuntimeLibrary{
		Name:    runtimeLibrary.Name,
		Version: runtimeLibrary.Version,
	}
}
