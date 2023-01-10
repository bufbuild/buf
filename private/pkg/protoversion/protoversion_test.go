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

package protoversion

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageHasPackageVersion(t *testing.T) {
	// note we allow this in the linter as we check this in PACKAGE_DEFINED
	// however, for the purposes of packageHasPackageVersion, this does not
	testNewPackageVersionForPackage(t, nil, false, "")
	testNewPackageVersionForPackage(t, nil, false, "foo")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar")
	testNewPackageVersionForPackage(t, nil, false, "foo.v1.bar")
	testNewPackageVersionForPackage(t, nil, false, "v1")
	testNewPackageVersionForPackage(t, nil, false, "v1beta1")
	testNewPackageVersionForPackage(t, newPackageVersion(1, StabilityLevelStable, 0, 0, ""), true, "foo.v1")
	testNewPackageVersionForPackage(t, newPackageVersion(2, StabilityLevelStable, 0, 0, ""), true, "foo.v2")
	testNewPackageVersionForPackage(t, newPackageVersion(1, StabilityLevelStable, 0, 0, ""), true, "foo.bar.v1")
	testNewPackageVersionForPackage(t, newPackageVersion(1, StabilityLevelAlpha, 1, 0, ""), true, "foo.bar.v1alpha1")
	testNewPackageVersionForPackage(t, newPackageVersion(1, StabilityLevelAlpha, 2, 0, ""), true, "foo.bar.v1alpha2")
	testNewPackageVersionForPackage(t, newPackageVersion(1, StabilityLevelBeta, 1, 0, ""), true, "foo.bar.v1beta1")
	testNewPackageVersionForPackage(t, newPackageVersion(1, StabilityLevelBeta, 2, 0, ""), true, "foo.bar.v1beta2")
	testNewPackageVersionForPackage(t, newPackageVersion(1, StabilityLevelAlpha, 1, 1, ""), true, "foo.bar.v1p1alpha1")
	testNewPackageVersionForPackage(t, newPackageVersion(1, StabilityLevelAlpha, 2, 1, ""), true, "foo.bar.v1p1alpha2")
	testNewPackageVersionForPackage(t, newPackageVersion(1, StabilityLevelBeta, 1, 1, ""), true, "foo.bar.v1p1beta1")
	testNewPackageVersionForPackage(t, newPackageVersion(1, StabilityLevelBeta, 2, 1, ""), true, "foo.bar.v1p1beta2")
	testNewPackageVersionForPackage(t, newPackageVersion(1, StabilityLevelTest, 0, 0, ""), true, "foo.bar.v1test")
	testNewPackageVersionForPackage(t, newPackageVersion(1, StabilityLevelTest, 0, 0, "foo"), true, "foo.bar.v1testfoo")
	testNewPackageVersionForPackage(t, newPackageVersion(4, StabilityLevelTest, 0, 0, ""), true, "foo.bar.v4test")
	testNewPackageVersionForPackage(t, newPackageVersion(4, StabilityLevelTest, 0, 0, "foo"), true, "foo.bar.v4testfoo")
	testNewPackageVersionForPackage(t, newPackageVersion(1, StabilityLevelAlpha, 0, 0, ""), true, "foo.bar.v1alpha")
	testNewPackageVersionForPackage(t, newPackageVersion(1, StabilityLevelAlpha, 0, 1, ""), true, "foo.bar.v1p1alpha")
	testNewPackageVersionForPackage(t, newPackageVersion(1, StabilityLevelBeta, 0, 0, ""), true, "foo.bar.v1beta")
	testNewPackageVersionForPackage(t, newPackageVersion(1, StabilityLevelBeta, 0, 1, ""), true, "foo.bar.v1p1beta")
	testNewPackageVersionForPackage(t, nil, false, "foo.v0")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v0")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v0alpha1")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v0alpha2")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v0beta1")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v0beta2")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v0p1alpha1")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v0p1alpha2")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v0p1beta1")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v0p1beta2")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v0test")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v0testfoo")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v1alpha0")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v1beta0")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v1p1alpha0")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v1p1beta0")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v1p0alpha1")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v1p0beta1")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.vv1")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.vv1alpha1")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.vv1alpha2")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.vv1beta1")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.vv1beta2")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.vv1p1alpha1")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.vv1p1alpha2")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.vv1p1beta1")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.vv1p1beta2")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.vv1test")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.vv1testfoo")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v1aalpha1")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v1p1test")
	testNewPackageVersionForPackage(t, nil, false, "foo.bar.v1p1testfoo")
}

func testNewPackageVersionForPackage(t *testing.T, expectedPackageVersion PackageVersion, expectedOK bool, pkg string) {
	packageVersion, ok := NewPackageVersionForPackage(pkg)
	assert.Equal(t, expectedOK, ok, pkg)
	if expectedOK {
		assert.Equal(t, expectedPackageVersion, packageVersion, pkg)
		split := strings.Split(pkg, ".")
		assert.Equal(t, split[len(split)-1], packageVersion.String(), pkg)
	} else {
		assert.Nil(t, packageVersion)
	}
}
