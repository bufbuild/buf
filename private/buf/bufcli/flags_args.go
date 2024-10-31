// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufcli

import (
	"errors"
	"fmt"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	pluginv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/plugin/v1beta1"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/pflag"
)

const (
	DefaultArchiveStatus = unarchivedArchiveStatus

	inputHashtagFlagName      = "__hashtag__"
	inputHashtagFlagShortName = "#"

	publicVisibility  = "public"
	privateVisibility = "private"

	archivedArchiveStatus   = "archived"
	unarchivedArchiveStatus = "unarchived"
	allArchiveStatus        = "all"
)

var (
	// allVisibiltyStrings are the possible options that a user can set the visibility flag with.
	allVisibiltyStrings = []string{
		publicVisibility,
		privateVisibility,
	}
	allArchiveStatusStrings = []string{
		archivedArchiveStatus,
		unarchivedArchiveStatus,
		allArchiveStatus,
	}
)

// BindAsFileDescriptorSet binds the exclude-imports flag.
func BindAsFileDescriptorSet(flagSet *pflag.FlagSet, addr *bool, flagName string) {
	flagSet.BoolVar(
		addr,
		flagName,
		false,
		`Output as a google.protobuf.FileDescriptorSet instead of an image
Note that images are wire compatible with FileDescriptorSets, but this flag strips
the additional metadata added for Buf usage`,
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
		"Exclude source info",
	)
}

// BindPaths binds the paths flag.
func BindPaths(
	flagSet *pflag.FlagSet,
	pathsAddr *[]string,
	pathsFlagName string,
) {
	flagSet.StringSliceVar(
		pathsAddr,
		pathsFlagName,
		nil,
		`Limit to specific files or directories, e.g. "proto/a/a.proto", "proto/a"
If specified multiple times, the union is taken`,
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

// BindExcludePaths binds the exclude-path flag.
func BindExcludePaths(
	flagSet *pflag.FlagSet,
	excludePathsAddr *[]string,
	excludePathsFlagName string,
) {
	flagSet.StringSliceVar(
		excludePathsAddr,
		excludePathsFlagName,
		nil,
		`Exclude specific files or directories, e.g. "proto/a/a.proto", "proto/a"
If specified multiple times, the union is taken`,
	)
}

// BindDisableSymlinks binds the disable-symlinks flag.
func BindDisableSymlinks(flagSet *pflag.FlagSet, addr *bool, flagName string) {
	flagSet.BoolVar(
		addr,
		flagName,
		false,
		`Do not follow symlinks when reading sources or configuration from the local filesystem
By default, symlinks are followed in this CLI, but never followed on the Buf Schema Registry`,
	)
}

// BindVisibility binds the visibility flag.
func BindVisibility(flagSet *pflag.FlagSet, addr *string, flagName string, emptyDefault bool) {
	defaultVisibility := privateVisibility
	if emptyDefault {
		defaultVisibility = ""
	}
	flagSet.StringVar(
		addr,
		flagName,
		defaultVisibility,
		fmt.Sprintf(`The module's visibility setting. Must be one of %s`, stringutil.SliceToString(allVisibiltyStrings)),
	)
}

// BindCreateVisibility binds the create-visibility flag. Kept in this package
// so we can keep allVisibilityStrings private.
func BindCreateVisibility(flagSet *pflag.FlagSet, addr *string, flagName string, createFlagName string) {
	flagSet.StringVar(
		addr,
		flagName,
		privateVisibility,
		fmt.Sprintf(`The module's visibility setting, if created. Can only be set with --%s. Must be one of %s`, createFlagName, stringutil.SliceToString(allVisibiltyStrings)),
	)
}

// BindArchiveStatus binds the archive-status flag. Kept in this package so we can
// keep allArchiveStatusStrings private.
func BindArchiveStatus(flagSet *pflag.FlagSet, addr *string, flagName string) {
	flagSet.StringVar(
		addr,
		flagName,
		DefaultArchiveStatus,
		fmt.Sprintf(`The archive status of the labels listed. Must be one of %s`, stringutil.SliceToString(allArchiveStatusStrings)),
	)
}

// Binds a string pointer flag, which indicates flag presence, i.e. `--flag ""` is not the same as not passing the flag.
//
// This is useful for buf registry organization/module update, where we only modify the fields specified.
//
// Value must not be nil.
func BindStringPointer(flagSet *pflag.FlagSet, name string, value **string, usage string) {
	flagSet.Var(
		&stringPointerValue{
			valuePointer: value,
		},
		name,
		usage,
	)
}

// Implements pflag.Value.
type stringPointerValue struct {
	// This must not be nil at construction time.
	valuePointer **string
}

func (b *stringPointerValue) Type() string {
	return "string"
}

func (b *stringPointerValue) String() string {
	if *b.valuePointer == nil {
		return ""
	}
	return **b.valuePointer
}

func (b *stringPointerValue) Set(value string) error {
	*b.valuePointer = &value
	return nil
}

// GetInputLong gets the long command description for an input-based command.
func GetInputLong(inputArgDescription string) string {
	return fmt.Sprintf(
		`The first argument is %s, which must be one of format %s.
This defaults to "." if no argument is specified.`,
		inputArgDescription,
		buffetch.AllFormatsString,
	)
}

// GetSourceLong gets the long command description for an input-based command.
func GetSourceLong(inputArgDescription string) string {
	return fmt.Sprintf(
		`The first argument is %s, which must be one of format %s.
This defaults to "." if no argument is specified.`,
		inputArgDescription,
		buffetch.SourceFormatsString,
	)
}

// GetSourceDirLong gets the long command description for a directory-based command.
func GetSourceDirLong(inputArgDescription string) string {
	return fmt.Sprintf(
		`The first argument is %s, which must be a directory.
This defaults to "." if no argument is specified.`,
		inputArgDescription,
	)
}

// GetSourceOrModuleLong gets the long command description for an input-based command.
func GetSourceOrModuleLong(inputArgDescription string) string {
	return fmt.Sprintf(
		`The first argument is %s, which must be one of format %s.
This defaults to "." if no argument is specified.`,
		inputArgDescription,
		buffetch.SourceOrModuleFormatsString,
	)
}

// GetInputValue gets the first arg.
//
// Also parses the special input hashtag flag that deals with the situation "buf build -#format=json".
// The existence of 0 or 1 args should be handled by the Args field on Command.
func GetInputValue(
	container app.ArgContainer,
	inputHashtag string,
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
			return "", errors.New("first argument is present but empty")
		}
		// if arg is non-empty and inputHashtag is non-empty, this means two arguments were specified
		if inputHashtag != "" {
			return "", errors.New("only 1 argument allowed but 2 arguments specified")
		}
	default:
		return "", fmt.Errorf("only 1 argument allowed but %d arguments specified", numArgs)
	}
	if arg != "" {
		return arg, nil
	}
	return defaultValue, nil
}

// VisibilityFlagToVisibilityAllowUnspecified parses the given string as a modulev1.ModuleVisibility
// where an empty string will be parsed as unspecified
func VisibilityFlagToVisibilityAllowUnspecified(visibility string) (modulev1.ModuleVisibility, error) {
	switch visibility {
	case publicVisibility:
		return modulev1.ModuleVisibility_MODULE_VISIBILITY_PUBLIC, nil
	case privateVisibility:
		return modulev1.ModuleVisibility_MODULE_VISIBILITY_PRIVATE, nil
	case "":
		return modulev1.ModuleVisibility_MODULE_VISIBILITY_UNSPECIFIED, nil
	default:
		return 0, fmt.Errorf("invalid visibility: %s", visibility)
	}
}

// VisibilityFlagToPluginVisibilityAllowUnspecified parses the given string as a pluginv1.PluginVisibility
// where an empty string will be parsed as unspecified.
func VisibilityFlagToPluginVisibilityAllowUnspecified(visibility string) (pluginv1beta1.PluginVisibility, error) {
	switch visibility {
	case publicVisibility:
		return pluginv1beta1.PluginVisibility_PLUGIN_VISIBILITY_PUBLIC, nil
	case privateVisibility:
		return pluginv1beta1.PluginVisibility_PLUGIN_VISIBILITY_PRIVATE, nil
	case "":
		return pluginv1beta1.PluginVisibility_PLUGIN_VISIBILITY_UNSPECIFIED, nil
	default:
		return 0, fmt.Errorf("invalid visibility: %s", visibility)
	}
}

// ArchiveStatusFlagToArchiveStatusFilter parses the given string as a modulev1.ListLabelsRequest_ArchiveFilter.
func ArchiveStatusFlagToArchiveStatusFilter(archiveStatus string) (modulev1.ListLabelsRequest_ArchiveFilter, error) {
	switch archiveStatus {
	case archivedArchiveStatus:
		return modulev1.ListLabelsRequest_ARCHIVE_FILTER_ARCHIVED_ONLY, nil
	case unarchivedArchiveStatus:
		return modulev1.ListLabelsRequest_ARCHIVE_FILTER_UNARCHIVED_ONLY, nil
	case allArchiveStatus:
		return modulev1.ListLabelsRequest_ARCHIVE_FILTER_ALL, nil
	default:
		return 0, fmt.Errorf("invalid archive status: %s", archiveStatus)
	}
}

// ValidateRequiredFlag validates that the required flag is set.
func ValidateRequiredFlag[T comparable](flagName string, value T) error {
	var zero T
	if value == zero {
		return appcmd.NewInvalidArgumentErrorf("--%s is required", flagName)
	}
	return nil
}

// ValidateErrorFormatFlagLint validates the error format flag for lint.
func ValidateErrorFormatFlagLint(errorFormatString string, errorFormatFlagName string) error {
	return validateErrorFormatFlag(AllLintFormatStrings, errorFormatString, errorFormatFlagName)
}

func validateErrorFormatFlag(validFormatStrings []string, errorFormatString string, errorFormatFlagName string) error {
	for _, formatString := range validFormatStrings {
		if errorFormatString == formatString {
			return nil
		}
	}
	return appcmd.NewInvalidArgumentErrorf("--%s: invalid format: %q", errorFormatFlagName, errorFormatString)
}
