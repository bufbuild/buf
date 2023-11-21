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

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/diff"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/licenseheader"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	use = "license-header"

	copyrightHolderFlagName = "copyright-holder"
	licenseTypeFlagName     = "license-type"
	yearRangeFlagName       = "year-range"
	diffFlagName            = "diff"
	exitCodeFlagName        = "exit-code"
	ignoreFlagName          = "ignore"
	ignoreFlagShortName     = "e"
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
	}
}

type flags struct {
	LicenseType     string
	CopyrightHolder string
	YearRange       string
	Diff            bool
	ExitCode        bool
	Ignore          []string
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
	_ = cobra.MarkFlagRequired(flagSet, licenseTypeFlagName)
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
	flagSet.BoolVar(
		&f.ExitCode,
		exitCodeFlagName,
		false,
		fmt.Sprintf("Exit with a non-zero exit code if a diff is present. Only valid with %s.", diffFlagName),
	)
	flagSet.StringSliceVarP(
		&f.Ignore,
		ignoreFlagName,
		ignoreFlagShortName,
		nil,
		`File paths to ignore.
These are extended regexes in the style of egrep.
If a file matches any of these values, it will be ignored.
Only works if there are no arguments and license-header does its own search for files.`,
	)
}

func run(ctx context.Context, container app.Container, flags *flags) error {
	if flags.ExitCode && !flags.Diff {
		return appcmd.NewInvalidArgumentErrorf("cannot specify %s without %s", exitCodeFlagName, diffFlagName)
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
	runner := command.NewRunner()
	filenames, err := getFilenames(ctx, container, runner, flags.Ignore)
	if err != nil {
		return err
	}
	for _, filename := range filenames {
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
					runner,
					data,
					modifiedData,
					filename,
					filename,
				)
				if err != nil {
					return err
				}
				if len(diffData) > 0 {
					if _, err := os.Stdout.Write(diffData); err != nil {
						return err
					}
					if flags.ExitCode {
						return app.NewError(100, "")
					}
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

func getFilenames(
	ctx context.Context,
	container app.Container,
	runner command.Runner,
	ignores []string,
) ([]string, error) {
	if container.NumArgs() > 0 {
		if len(ignores) > 0 {
			return nil, appcmd.NewInvalidArgumentErrorf("cannot use flag %q with any arguments", ignoreFlagName)
		}
		return app.Args(container), nil
	}
	ignoreRegexps := make([]*regexp.Regexp, len(ignores))
	for i, ignore := range ignores {
		ignoreRegexp, err := regexp.CompilePOSIX(ignore)
		if err != nil {
			return nil, err
		}
		ignoreRegexps[i] = ignoreRegexp
	}
	return git.NewLister(runner).ListFilesAndUnstagedFiles(
		ctx,
		container,
		git.ListFilesAndUnstagedFilesOptions{
			IgnorePathRegexps: ignoreRegexps,
		},
	)
}

func newRequiredFlagError(flagName string) error {
	return appcmd.NewInvalidArgumentErrorf("required flag %q not set", flagName)
}
