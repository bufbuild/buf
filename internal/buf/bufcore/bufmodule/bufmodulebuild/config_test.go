// Copyright 2020 Buf Technologies, Inc.
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

package bufmodulebuild

import (
	"fmt"
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmoduletesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigV1Beta1Success1(t *testing.T) {
	// https://github.com/bufbuild/buf/issues/56
	t.Parallel()
	testNewConfigV1Beta1Success(
		t,
		[]string{
			"proto",
			"proto-vendor",
		},
		[]string{},
		[]string{},
	)
}

func TestNewConfigV1Beta1Error1(t *testing.T) {
	t.Parallel()
	testNewConfigV1Beta1Error(
		t,
		[]string{
			"/a/b",
		},
		[]string{},
		[]string{},
	)
}

func TestNewConfigV1Beta1Error2(t *testing.T) {
	t.Parallel()
	testNewConfigV1Beta1Error(
		t,
		[]string{},
		[]string{
			"/a/b",
		},
		[]string{},
	)
}

func TestNewConfigV1Beta1Error3(t *testing.T) {
	t.Parallel()
	testNewConfigV1Beta1Error(
		t,
		[]string{
			"a/b",
			"a/b",
		},
		[]string{},
		[]string{},
	)
}

func TestNewConfigV1Beta1Error4(t *testing.T) {
	t.Parallel()
	testNewConfigV1Beta1Error(
		t,
		[]string{
			"a/b",
			"a/b/c",
		},
		[]string{},
		[]string{},
	)
}

func TestNewConfigV1Beta1Error5(t *testing.T) {
	t.Parallel()
	testNewConfigV1Beta1Error(
		t,
		[]string{
			"a/b",
		},
		[]string{
			"a/c",
		},
		[]string{},
	)
}

func TestNewConfigV1Beta1Error6(t *testing.T) {
	t.Parallel()
	testNewConfigV1Beta1Error(
		t,
		[]string{
			".",
			"a",
		},
		[]string{},
		[]string{},
	)
}

func TestNewConfigV1Beta1Error7(t *testing.T) {
	t.Parallel()
	testNewConfigV1Beta1Error(
		t,
		[]string{
			"proto",
		},
		[]string{
			"proto/a/c",
			// error since not a directory
			"proto/d/1.proto",
		},
		[]string{},
	)
}

func TestNewConfigV1Beta1Error8(t *testing.T) {
	t.Parallel()
	testNewConfigV1Beta1Error(
		t,
		[]string{
			"proto",
		},
		[]string{},
		[]string{
			// Duplicate dependency
			bufmoduletesting.TestModuleReferenceFooBarV1String,
			bufmoduletesting.TestModuleReferenceFooBarV2String,
		},
	)
}

func TestNewConfigV1Beta1Error9(t *testing.T) {
	t.Parallel()
	testNewConfigV1Beta1Error(
		t,
		[]string{
			"proto",
		},
		[]string{},
		[]string{
			// Duplicate dependency
			bufmoduletesting.TestModuleReferenceFooBarV1String,
			bufmoduletesting.TestModuleReferenceFooBarCommitString,
		},
	)
}

func TestNewConfigV1Beta1Error10(t *testing.T) {
	t.Parallel()
	testNewConfigV1Beta1Error(
		t,
		[]string{
			"proto",
		},
		[]string{},
		[]string{
			// Duplicate dependency
			bufmoduletesting.TestModuleReferenceFooBarV1String,
			bufmoduletesting.TestModuleReferenceFooBarV1String,
		},
	)
}

func TestNewConfigV1Beta1Equal1(t *testing.T) {
	t.Parallel()
	testNewConfigV1Beta1Equal(
		t,
		[]string{
			"a",
			"b",
		},
		[]string{
			"a/foo",
		},
		[]string{
			bufmoduletesting.TestModuleReferenceFooBarV1String,
			bufmoduletesting.TestModuleReferenceFooBazCommitString,
		},
		&Config{
			RootToExcludes: map[string][]string{
				"a": {
					"foo",
				},
				"b": {},
			},
			DependencyModuleReferences: testParseDependencyModuleReferences(
				t,
				bufmoduletesting.TestModuleReferenceFooBarV1String,
				bufmoduletesting.TestModuleReferenceFooBazCommitString,
			),
		},
	)
}

func TestNewConfigV1Beta1Equal2(t *testing.T) {
	t.Parallel()
	testNewConfigV1Beta1Equal(
		t,
		[]string{
			"a",
			"b",
		},
		[]string{
			"a/foo",
			"b/foo",
			"b/bar",
		},
		[]string{},
		&Config{
			RootToExcludes: map[string][]string{
				"a": {
					"foo",
				},
				"b": {
					"bar",
					"foo",
				},
			},
		},
	)
}

func testNewConfigV1Beta1Success(t *testing.T, roots []string, excludes []string, deps []string) {
	_, err := NewConfigV1Beta1(ExternalConfigV1Beta1{Roots: roots, Excludes: excludes}, deps...)
	assert.NoError(t, err, fmt.Sprintf("%v %v %v", roots, excludes, deps))
}

func testNewConfigV1Beta1Error(t *testing.T, roots []string, excludes []string, deps []string) {
	_, err := NewConfigV1Beta1(ExternalConfigV1Beta1{Roots: roots, Excludes: excludes}, deps...)
	assert.Error(t, err, fmt.Sprintf("%v %v %v", roots, excludes, deps))
}

func testNewConfigV1Beta1Equal(
	t *testing.T,
	roots []string,
	excludes []string,
	deps []string,
	expectedConfig *Config,
) {
	config, err := NewConfigV1Beta1(ExternalConfigV1Beta1{Roots: roots, Excludes: excludes}, deps...)
	assert.NoError(t, err, fmt.Sprintf("%v %v %v", roots, excludes, deps))
	assert.Equal(t, expectedConfig, config)
}

func testParseDependencyModuleReferences(t *testing.T, deps ...string) []bufmodule.ModuleReference {
	moduleNames, err := parseDependencyModuleReferences(deps...)
	require.NoError(t, err)
	return moduleNames
}
