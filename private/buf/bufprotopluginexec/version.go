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

package bufprotopluginexec

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

func newVersion(major int32, minor int32, patch int32, suffix string) *pluginpb.Version {
	version := &pluginpb.Version{
		Major: proto.Int32(major),
		Minor: proto.Int32(minor),
		Patch: proto.Int32(patch),
	}
	if suffix != "" {
		version.Suffix = proto.String(suffix)
	}
	return version
}

func parseVersionForCLIVersion(value string) (_ *pluginpb.Version, retErr error) {
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("cannot parse protoc version %q: %w", value, retErr)
		}
	}()

	// protoc always starts with "libprotoc "
	value = strings.TrimPrefix(value, "libprotoc ")
	split := strings.Split(value, ".")
	if n := len(split); n != 2 && n != 3 {
		return nil, fmt.Errorf("%d components split by '.'", n)
	}
	major, err := strconv.ParseInt(split[0], 10, 32)
	if err != nil {
		return nil, err
	}
	var suffix string
	restSplit := strings.SplitN(split[len(split)-1], "-", 2)
	lastNumber, err := strconv.ParseInt(restSplit[0], 10, 32)
	if err != nil {
		return nil, err
	}
	switch len(restSplit) {
	case 1:
	case 2:
		suffix = restSplit[1]
	default:
		return nil, errors.New("more than two patch components split by '-'")
	}
	var minor int64
	var patch int64
	switch len(split) {
	case 2:
		minor = lastNumber
	case 3:
		minor, err = strconv.ParseInt(split[1], 10, 32)
		if err != nil {
			return nil, err
		}
		patch = lastNumber
	}
	return newVersion(int32(major), int32(minor), int32(patch), suffix), nil
}

func versionString(version *pluginpb.Version) string {
	var value string
	if version.GetMajor() <= 3 || version.GetPatch() != 0 {
		value = fmt.Sprintf("%d.%d.%d", version.GetMajor(), version.GetMinor(), version.GetPatch())
	} else {
		value = fmt.Sprintf("%d.%d", version.GetMajor(), version.GetMinor())
	}
	if version.Suffix != nil {
		value = value + "-" + version.GetSuffix()
	}
	return value
}

// Should I set the --experimental_allow_proto3_optional flag?
func getSetExperimentalAllowProto3OptionalFlag(version *pluginpb.Version) bool {
	if version.GetSuffix() == "buf" {
		return false
	}
	if version.GetMajor() != 3 {
		return false
	}
	return version.GetMinor() > 11 && version.GetMinor() < 15
}

// Should I notify that I am OK with the proto3 optional feature?
func getFeatureProto3OptionalSupported(version *pluginpb.Version) bool {
	if version.GetSuffix() == "buf" {
		return true
	}
	if version.GetMajor() < 3 {
		return false
	}
	if version.GetMajor() == 3 {
		return version.GetMinor() > 11
	}
	// version.GetMajor() > 3
	return true
}

// Is kotlin supported as a builtin plugin?
func getKotlinSupportedAsBuiltin(version *pluginpb.Version) bool {
	if version.GetSuffix() == "buf" {
		return true
	}
	if version.GetMajor() < 3 {
		return false
	}
	if version.GetMajor() == 3 {
		return version.GetMinor() > 16
	}
	// version.GetMajor() > 3
	return true
}

// Is rust supported as a builtin plugin?
func getRustSupportedAsBuiltin(version *pluginpb.Version) bool {
	if version.GetSuffix() == "buf" {
		return true
	}
	if version.GetMajor() < 4 {
		return false
	}
	if version.GetMajor() == 4 {
		return version.GetMinor() > 22
	}
	// version.GetMajor() 4 5
	return true
}

// Is js supported as a builtin plugin?
func getJSSupportedAsBuiltin(version *pluginpb.Version) bool {
	if version.GetSuffix() == "buf" {
		return true
	}
	if version.GetMajor() < 3 {
		return false
	}
	if version.GetMajor() == 3 {
		// v21 and above of protoc still returns "3.MAJOR.Z" for version
		return version.GetMinor() < 21
	}
	// version.GetMajor() > 3
	// This will catch if they ever change protoc's returned version
	// to the proper major version
	return false
}
