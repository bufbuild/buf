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

package main

import (
	"bytes"
	"context"
	"os"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/diff"
	"github.com/bufbuild/buf/internal/pkg/licenseheader"
	"github.com/spf13/pflag"
)

const (
	use     = "license-header"
	version = "0.0.1-dev"

	copyrightHolderFlagName = "copyright-holder"
	licenseTypeFlagName     = "license-type"
	yearRangeFlagName       = "year-range"
	diffFlagName            = "diff"
)

func main() {
	appcmd.Main(context.Background(), newCommand())
}

func newCommand() *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use: use + " files...",
		Run: func(ctx context.Context, container app.Container) error {
			return run(ctx, container, flags)
		},
		BindFlags: flags.Bind,
		Version:   version,
	}
}

type flags struct {
	LicenseType     string
	CopyrightHolder string
	YearRange       string
	Diff            bool
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.LicenseType,
		licenseTypeFlagName,
		"",
		"The license type. Must be one of [none,apache,proprietary].",
	)
	flagSet.StringVar(
		&f.CopyrightHolder,
		copyrightHolderFlagName,
		"",
		"The copyright holder. Required if license type is not none.",
	)
	flagSet.StringVar(
		&f.YearRange,
		yearRangeFlagName,
		"",
		"The year range. Required if license type is not none.",
	)
	flagSet.BoolVar(
		&f.Diff,
		diffFlagName,
		false,
		"Print a diff instead of modifying the files.",
	)
}

func run(ctx context.Context, container app.Container, flags *flags) error {
	if flags.LicenseType == "" {
		return newRequiredFlagError(licenseTypeFlagName)
	}
	licenseType, err := licenseheader.ParseLicenseType(flags.LicenseType)
	if err != nil {
		return appcmd.NewInvalidArgumentErrorf("--%s: %v", licenseTypeFlagName, err)
	}
	if licenseType != licenseheader.LicenseTypeNone {
		if flags.CopyrightHolder == "" {
			return newRequiredFlagError(copyrightHolderFlagName)
		}
		if flags.YearRange == "" {
			return newRequiredFlagError(yearRangeFlagName)
		}
	}
	for i := 0; i < container.NumArgs(); i++ {
		filename := container.Arg(i)
		data, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		modifiedData, err := licenseheader.Modify(
			licenseType,
			flags.CopyrightHolder,
			flags.YearRange,
			filename,
			data,
		)
		if err != nil {
			return err
		}
		if !bytes.Equal(data, modifiedData) {
			if flags.Diff {
				diffData, err := diff.Diff(
					ctx,
					data,
					modifiedData,
					filename,
					filename,
				)
				if err != nil {
					return err
				}
				if _, err := os.Stdout.Write(diffData); err != nil {
					return err
				}
			} else {
				fileInfo, err := os.Stat(filename)
				if err != nil {
					return err
				}
				if err := os.WriteFile(filename, modifiedData, fileInfo.Mode().Perm()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func newRequiredFlagError(flagName string) error {
	return appcmd.NewInvalidArgumentErrorf("--%s is required", flagName)
}
