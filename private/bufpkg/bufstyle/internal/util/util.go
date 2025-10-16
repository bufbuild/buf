// Copyright 2020-2025 Buf Technologies, Inc.
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

package util

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// ForEachComment iterates over every Comment and calls f.
func ForEachComment(pass *analysis.Pass, f func(*ast.Comment) error) error {
	for _, file := range pass.Files {
		for _, commentGroup := range file.Comments {
			for _, comment := range commentGroup.List {
				if err := f(comment); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ForEachObject iterates over every Object and calls f.
func ForEachObject(pass *analysis.Pass, f func(types.Object) error) error {
	if typesInfo := pass.TypesInfo; typesInfo != nil {
		for _, object := range pass.TypesInfo.Defs {
			if object != nil {
				if err := f(object); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
