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

package licenseheader

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testCopyrightHolder = "Foo Bar, Inc."
	testYearRange       = "2020-2021"
	testApacheGoHeader  = `// Copyright 2020-2021 Foo Bar, Inc.
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
// limitations under the License.`
	testProprietaryGoHeader = `// Copyright 2020-2021 Foo Bar, Inc.
//
// All rights reserved.`
)

func TestBasic(t *testing.T) {
	t.Parallel()

	modifiedData, err := Modify(
		LicenseTypeApache,
		testCopyrightHolder,
		testYearRange,
		"foo/bar.go",
		[]byte("package foo"),
	)
	require.NoError(t, err)
	require.Equal(
		t,
		testApacheGoHeader+"\n\npackage foo",
		string(modifiedData),
	)
	modifiedData, err = Modify(
		LicenseTypeProprietary,
		testCopyrightHolder,
		testYearRange,
		"foo/bar.go",
		modifiedData,
	)
	require.NoError(t, err)
	require.Equal(
		t,
		testProprietaryGoHeader+"\n\npackage foo",
		string(modifiedData),
	)
	modifiedData, err = Modify(
		LicenseTypeNone,
		testCopyrightHolder,
		testYearRange,
		"foo/bar.go",
		modifiedData,
	)
	require.NoError(t, err)
	require.Equal(
		t,
		"package foo",
		string(modifiedData),
	)
	modifiedData, err = Modify(
		LicenseTypeApache,
		testCopyrightHolder,
		testYearRange,
		"foo/bar.go",
		[]byte(`// copyright foo

package foo`),
	)
	require.NoError(t, err)
	require.Equal(
		t,
		testApacheGoHeader+"\n\npackage foo",
		string(modifiedData),
	)
	modifiedData, err = Modify(
		LicenseTypeApache,
		testCopyrightHolder,
		testYearRange,
		"foo/bar.go",
		[]byte(`// foo
package foo`),
	)
	require.NoError(t, err)
	require.Equal(
		t,
		testApacheGoHeader+`

// foo
package foo`,
		string(modifiedData),
	)
}
