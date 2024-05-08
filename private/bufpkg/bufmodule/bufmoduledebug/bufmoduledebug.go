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

package bufmoduledebug

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/indent"
)

// ModuleSetDebugString gets a debug string with ModuleSet information.
//
// This can be printed to stderr, debug logged, etc, while debugging.
func ModuleSetDebugString(ctx context.Context, moduleSet bufmodule.ModuleSet) (string, error) {
	printer := indent.NewPrinter("  ")
	printer.Pf("module_set:")
	printer.In()
	for _, module := range moduleSet.Modules() {
		if err := printModule(ctx, printer, module); err != nil {
			return "", err
		}
	}
	printer.Out()
	return printer.String()
}

// ModuleDebugString gets a debug string with Module information.
//
// This can be printed to stderr, debug logged, etc, while debugging.
func ModuleDebugString(ctx context.Context, module bufmodule.Module) (string, error) {
	printer := indent.NewPrinter("  ")
	if err := printModule(ctx, printer, module); err != nil {
		return "", err
	}
	return printer.String()
}

// ModuleReadBucketDebugString gets a debug string with ModuleReadBucket information.
//
// This can be printed to stderr, debug logged, etc, while debugging.
func ModuleReadBucketDebugString(ctx context.Context, moduleReadBucket bufmodule.ModuleReadBucket) (string, error) {
	printer := indent.NewPrinter("  ")
	printer.Pf("module_read_bucket:")
	printer.In()
	fileInfos, err := bufmodule.GetFileInfos(ctx, moduleReadBucket)
	if err != nil {
		return "", err
	}
	printer.P("files:")
	printer.In()
	for _, fileInfo := range fileInfos {
		printer.Pf("%s:", fileInfo.Path())
		printer.In()
		printer.Pf("target: %v", fileInfo.IsTargetFile())
		printer.Pf("external_path: %s", fileInfo.ExternalPath())
		printer.Out()
	}
	printer.Out()
	printer.Out()
	return printer.String()
}

func printModule(ctx context.Context, printer indent.Printer, module bufmodule.Module) error {
	fileInfos, err := bufmodule.GetFileInfos(ctx, module)
	if err != nil {
		return err
	}
	moduleDeps, err := module.ModuleDeps()
	if err != nil {
		return err
	}
	printer.P("module:")
	printer.In()
	printer.Pf("name: %s", module.OpaqueID())
	printer.Pf("target: %v", module.IsTarget())
	printer.Pf("local: %v", module.IsLocal())
	printer.P("deps:")
	printer.In()
	for _, moduleDep := range moduleDeps {
		printer.Pf("name:", moduleDep.OpaqueID())
		printer.Pf("direct: %v", moduleDep.IsDirect())
	}
	printer.Out()
	printer.P("files:")
	printer.In()
	for _, fileInfo := range fileInfos {
		printer.Pf("%s:", fileInfo.Path())
		printer.In()
		printer.Pf("target: %v", fileInfo.IsTargetFile())
		printer.Pf("external_path: %s", fileInfo.ExternalPath())
		printer.Out()
	}
	printer.Out()
	printer.Out()
	return nil
}
