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

package internal

import (
	"errors"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/spf13/pflag"
)

const (
	// FlagDeprecationMessageSuffix is the suffix for flag deprecation messages.
	FlagDeprecationMessageSuffix = `
We recommend migrating, however this flag continues to work.
See https://docs.buf.build/faq for more details.`

	inputHashtagFlagName      = "__hashtag__"
	inputHashtagFlagShortName = "#"
)

// BindAsFileDescriptorSet binds the exclude-imports flag.
func BindAsFileDescriptorSet(flagSet *pflag.FlagSet, addr *bool, flagName string) {
	flagSet.BoolVar(
		addr,
		flagName,
		false,
		`Output as a google.protobuf.FileDescriptorSet instead of an image.

Note that images are wire-compatible with FileDescriptorSets, however this flag will strip
the additional metadata added for Buf usage.`,
	)
}

// BindExcludeImports binds the exclude-imports flag.
func BindExcludeImports(flagSet *pflag.FlagSet, addr *bool, flagName string) {
	flagSet.BoolVar(
		addr,
		flagName,
		false,
		"Exclude imports.",
	)
}

// BindExcludeSourceInfo binds the exclude-source-info flag.
func BindExcludeSourceInfo(flagSet *pflag.FlagSet, addr *bool, flagName string) {
	flagSet.BoolVar(
		addr,
		flagName,
		false,
		"Exclude source info.",
	)
}

// BindFiles binds the files flag.
func BindFiles(flagSet *pflag.FlagSet, addr *[]string, flagName string) {
	flagSet.StringSliceVar(
		addr,
		flagName,
		nil,
		`Limit to specific files. This is an advanced feature and is not recommended.`,
	)
}

// BindInputHashtag binds the input hashtag flag.
//
// This needs to be added to any command that has the input as the first argument.
// This deals with the situation "buf build -#format=json" which results in
// a parse error from pflag.
func BindInputHashtag(flagSet *pflag.FlagSet, addr *string) {
	flagSet.StringVarP(
		addr,
		inputHashtagFlagName,
		inputHashtagFlagShortName,
		"",
		"",
	)
	_ = flagSet.MarkHidden(inputHashtagFlagName)
}

// GetInputLong gets the long command description for an input-based command.
func GetInputLong(inputArgDescription string) string {
	return fmt.Sprintf(
		`The first argument is %s.
The first argument must be one of format %s.
If no argument is specified, defaults to ".".`,
		inputArgDescription,
		buffetch.AllFormatsString,
	)
}

// GetInputValue gets either the first arg or the deprecated flag, but not both.
//
// Also parses the special input hashtag flag that deals with the situation "buf build -#format=json".
// The existence of 0 or 1 args should be handled by the Args field on Command.
func GetInputValue(
	container appflag.Container,
	inputHashtag string,
	deprecatedFlag string,
	deprecatedFlagName string,
	defaultValue string,
) (string, error) {
	var arg string
	switch numArgs := container.NumArgs(); numArgs {
	case 0:
		if inputHashtag != "" {
			arg = "-#" + inputHashtag
		}
	case 1:
		arg = container.Arg(0)
		if arg == "" {
			return "", errors.New("First argument is present but empty.")
		}
		// if arg is non-empty and inputHashtag is non-empty, this means two arguments were specified
		if inputHashtag != "" {
			return "", errors.New("Only 1 argument allowed but 2 arguments specified.")
		}
	default:
		return "", fmt.Errorf("Only 1 argument allowed but %d arguments specified.", numArgs)
	}
	if arg != "" && deprecatedFlag != "" {
		return "", fmt.Errorf("Cannot specify both first argument and deprecated flag --%s.", deprecatedFlagName)
	}
	if arg != "" {
		return arg, nil
	}
	if deprecatedFlag != "" {
		return deprecatedFlag, nil
	}
	return defaultValue, nil
}

// GetFlagOrDeprecatedFlag gets the flag, or the deprecated flag.
func GetFlagOrDeprecatedFlag(
	flag string,
	flagName string,
	deprecatedFlag string,
	deprecatedFlagName string,
) (string, error) {
	if flag != "" && deprecatedFlag != "" {
		return "", fmt.Errorf("Cannot specify both --%s and --%s.", flagName, deprecatedFlagName)
	}
	if flag != "" {
		return flag, nil
	}
	return deprecatedFlag, nil
}
