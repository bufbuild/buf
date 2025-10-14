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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"buf.build/go/app"
	"buf.build/go/standard/xos/xexec"
)

const genFileName = "usage.gen.go"

func main() {
	app.Main(context.Background(), run)
}

func run(ctx context.Context, container app.Container) error {
	reader, err := goListJSON(ctx, container)
	if err != nil {
		return err
	}
	pkgs, err := readPkgs(reader)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		if err := processPkg(pkg); err != nil {
			return err
		}
	}
	return nil
}

func goListJSON(ctx context.Context, container app.Container) (io.Reader, error) {
	packageList := app.Args(container)[1:]
	if len(packageList) == 0 {
		return nil, errors.New("must specify at least one go package")
	}
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if err := xexec.Run(
		ctx,
		"go",
		xexec.WithArgs(
			append(
				[]string{
					"list",
					"-json",
				},
				packageList...,
			)...,
		),
		xexec.WithStdout(stdout),
		xexec.WithStderr(stderr),
		xexec.WithEnv(os.Environ()),
	); err != nil {
		return nil, fmt.Errorf("error running go list -json %s:\n%s", strings.Join(packageList, " "), stderr.String())
	}
	return stdout, nil
}

func readPkgs(reader io.Reader) ([]*pkg, error) {
	decoder := json.NewDecoder(reader)
	var pkgs []*pkg
	for decoder.More() {
		pkg := &pkg{}
		if err := decoder.Decode(pkg); err != nil {
			return nil, err
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

func processPkg(pkg *pkg) error {
	if pkg.Dir == "" {
		return errors.New("empty dir from package")
	}
	if pkg.Name == "" {
		return errors.New("empty name from package")
	}
	genFilePath := filepath.Join(pkg.Dir, genFileName)
	if _, err := os.Stat(genFilePath); err == nil {
		if err := os.Remove(genFilePath); err != nil {
			return fmt.Errorf("error removing %s: %w", genFilePath, err)
		}
	}
	if err := os.WriteFile(
		genFilePath,
		[]byte(
			fmt.Sprintf(`// Generated. DO NOT EDIT.

package %s

import _ "github.com/bufbuild/buf/private/usage"`,
				pkg.Name,
			),
		),
		0644,
	); err != nil {
		return fmt.Errorf("error writing %s: %w", genFilePath, err)
	}
	return nil
}

type pkg struct {
	Dir  string
	Name string
}
