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

//go:build windows
// +build windows

package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtoFileRef(t *testing.T) {
	tempDirPath := t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "protofileref", "buf.gen.yaml"),
		filepath.Join("testdata", "protofileref", "a", "v1", "a.proto"),
	)
	_, err := os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "A.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "B.java"))
	require.Contains(t, err.Error(), "The system cannot find the file specified.")
}
