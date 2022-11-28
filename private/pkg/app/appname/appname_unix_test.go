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

// Matching the unix-like build tags in the Golang source i.e. https://github.com/golang/go/blob/912f0750472dd4f674b69ca1616bfaf377af1805/src/os/file_unix.go#L6

//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd js,wasm linux netbsd openbsd solaris

package appname

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bufbuild/buf/private/pkg/app"
)

func TestRoundTrip1(t *testing.T) {
	tempDir := t.TempDir()
	testRoundTrip(
		t,
		"foo-bar",
		map[string]string{
			"FOO_BAR_CONFIG_DIR": tempDir,
		},
		tempDir,
	)
}

func TestRoundTrip2(t *testing.T) {
	tempDir := t.TempDir()
	testRoundTrip(
		t,
		"foo-bar",
		map[string]string{
			"XDG_CONFIG_HOME": tempDir,
		},
		filepath.Join(tempDir, "foo-bar"),
	)
}

func TestRoundTrip3(t *testing.T) {
	tempDir := t.TempDir()
	testRoundTrip(
		t,
		"foo-bar",
		map[string]string{
			"HOME": tempDir,
		},
		filepath.Join(tempDir, ".config", "foo-bar"),
	)
}

func TestPort1(t *testing.T) {
	testPort(
		t,
		"foo-bar",
		map[string]string{
			"FOO_BAR_PORT": "4000",
		},
		4000,
	)
}

func TestPort2(t *testing.T) {
	testPort(
		t,
		"foo-bar",
		map[string]string{
			"FOO_BAR_PORT": "4000",
			"PORT":         "2000",
		},
		4000,
	)
}

func TestPort3(t *testing.T) {
	testPort(
		t,
		"foo-bar",
		map[string]string{
			"PORT": "4000",
		},
		4000,
	)
}

func TestPort4(t *testing.T) {
	testPort(
		t,
		"foo-bar",
		map[string]string{},
		0,
	)
}

func testPort(t *testing.T, appName string, env map[string]string, expected uint16) {
	container, err := NewContainer(app.NewEnvContainer(env), appName)
	require.NoError(t, err)
	port, err := container.Port()
	require.NoError(t, err)
	require.Equal(t, expected, port)
}

func testRoundTrip(t *testing.T, appName string, env map[string]string, dirPath string) {
	_, err := os.Lstat(filepath.Join(dirPath, configFileName))
	require.Error(t, err)
	container, err := NewContainer(app.NewEnvContainer(env), appName)
	require.NoError(t, err)
	inputTestConfig := &testConfig{Bar: "one", Baz: "two"}
	err = WriteConfig(container, inputTestConfig)
	require.NoError(t, err)
	_, err = os.Lstat(filepath.Join(dirPath, configFileName))
	require.NoError(t, err)
	outputTestConfig := &testConfig{}
	err = ReadConfig(container, outputTestConfig)
	require.NoError(t, err)
	require.Equal(t, inputTestConfig, outputTestConfig)
}

type testConfig struct {
	Bar string
	Baz string
}
