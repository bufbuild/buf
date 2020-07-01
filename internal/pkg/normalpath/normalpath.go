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

// Package normalpath provides functions similar to filepath.
//
// A normalized path is a cleaned and to-slash'ed path.
// A validated path validates that a path is relative and does not jump context.
package normalpath

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/stringutil"
)

const (
	stringOSPathSeparator = string(os.PathSeparator)
	// This has to be with "/" instead of os.PathSeparator as we use this on normalized paths
	normalizedRelPathJumpContextPrefix = "../"
)

var (
	// errNotRelative is the error returned if the path is not relative.
	errNotRelative = errors.New("expected to be relative")
	// errOutsideContextDir is the error returned if the path is outside the context directory.
	errOutsideContextDir = errors.New("is outside the context directory")
)

// Error is a path error.
type Error struct {
	Path string
	Err  error
}

// NewError returns a new Error.
func NewError(path string, err error) *Error {
	return &Error{
		Path: path,
		Err:  err,
	}
}

// Error implements error.
func (e *Error) Error() string {
	errString := ""
	if e.Err != nil {
		errString = e.Err.Error()
	}
	if errString == "" {
		errString = "error"
	}
	return e.Path + ": " + errString
}

// ErrorEquals returns true if err is an Error and err.Error == target.
func ErrorEquals(err error, target error) bool {
	if err == nil {
		return false
	}
	pathError, ok := err.(*Error)
	if !ok {
		return false
	}
	return pathError.Err == target
}

// NormalizeAndValidate normalizes and validates the given path.
//
// This calls Normalize on the path.
// Returns Error if the path is not relative or jumps context.
// This can be used to validate that paths are valid to use with Buckets.
// The error message is safe to pass to users.
func NormalizeAndValidate(path string) (string, error) {
	path = Normalize(path)
	if filepath.IsAbs(path) {
		return "", NewError(path, errNotRelative)
	}
	// https://github.com/bufbuild/buf/issues/51
	if strings.HasPrefix(path, normalizedRelPathJumpContextPrefix) {
		return "", NewError(path, errOutsideContextDir)
	}
	return path, nil
}

// Normalize normalizes the given path.
//
// This calls filepath.Clean and filepath.ToSlash on the path.
// If the path is "" or ".", this returns ".".
func Normalize(path string) string {
	return filepath.Clean(filepath.ToSlash(path))
}

// Unnormalize unnormalizes the given path.
//
// This calls filepath.FromSlash on the path.
// If the path is "", this returns "".
func Unnormalize(path string) string {
	return filepath.FromSlash(path)
}

// Base is equivalent to filepath.Base.
//
// Normalizes before returning.
func Base(path string) string {
	return Normalize(filepath.Base(Unnormalize(path)))
}

// Dir is equivalent to filepath.Dir.
//
// Normalizes before returning.
func Dir(path string) string {
	return Normalize(filepath.Dir(Unnormalize(path)))
}

// Ext is equivalent to filepath.Ext.
//
// Can return empty string.
func Ext(path string) string {
	return filepath.Ext(Unnormalize(path))
}

// Join is equivalent to filepath.Join.
//
// Empty strings are ignored,
// Can return empty string.
//
// Normalizes before returning otherwise.
func Join(paths ...string) string {
	unnormalized := make([]string, len(paths))
	for i, path := range paths {
		unnormalized[i] = Unnormalize(path)
	}
	value := filepath.Join(unnormalized...)
	if value == "" {
		return ""
	}
	return Normalize(value)
}

// Rel is equivalent to filepath.Rel.
//
// Can return empty string, especially on error.
//
// Normalizes before returning otherwise.
func Rel(basepath string, targpath string) (string, error) {
	path, err := filepath.Rel(Unnormalize(basepath), Unnormalize(targpath))
	if path == "" {
		return "", err
	}
	return Normalize(path), err
}

// ByDir maps the paths into a map from directory via Dir to the original paths.
//
// The paths for each value slice will be sorted.
//
// The path is expected to be normalized.
func ByDir(paths ...string) map[string][]string {
	m := make(map[string][]string)
	for _, path := range paths {
		path = Normalize(path)
		dir := filepath.Dir(path)
		m[dir] = append(m[dir], path)
	}
	for _, dirPaths := range m {
		sort.Strings(dirPaths)
	}
	return m
}

// Components splits the path into it's components.
//
// This calls filepath.Split repeatedly.
//
// The path is expected to be normalized.
func Components(path string) []string {
	var components []string
	dir := Unnormalize(path)
	for {
		var file string
		dir, file = filepath.Split(dir)
		// puts in reverse
		components = append(components, file)
		if dir == stringOSPathSeparator {
			components = append(components, dir)
			break
		}
		dir = strings.TrimSuffix(dir, stringOSPathSeparator)
		if dir == "" {
			break
		}
	}
	// https://github.com/golang/go/wiki/SliceTricks#reversing
	for i := len(components)/2 - 1; i >= 0; i-- {
		opp := len(components) - 1 - i
		components[i], components[opp] = components[opp], components[i]
	}
	for i, component := range components {
		components[i] = Normalize(component)
	}
	return components
}

// ContainsPath returns true if the dirPath contains the path.
//
// The path and dirPath are expected to be normalized and validated.
//
// For a given dirPath:
//
//   - If path == ".", dirPathPath does not contain the path.
//   - If dirPath == ".", the dirPath contains the path.
//   - If dirPath is a directory that contains path, this returns true.
func ContainsPath(dirPath string, path string) bool {
	if path == "." {
		return false
	}
	return EqualsOrContainsPath(dirPath, Dir(path))
}

// EqualsOrContainsPath returns true if the value is equal to or contains the path.
//
// The path and value are expected to be normalized and validated.
//
// For a given value:
//
//   - If value == ".", the value contains the path.
//   - If value == path, the value is equal to the path.
//   - If value is a directory that contains path, this returns true.
func EqualsOrContainsPath(value string, path string) bool {
	if value == "." {
		return true
	}
	// TODO: can we optimize this with strings.HasPrefix(path, value + "/") somehow?
	for curPath := path; curPath != "."; curPath = Dir(curPath) {
		if value == curPath {
			return true
		}
	}
	return false
}

// MapHasEqualOrContainingPath returns true if the path matches any file or directory in the map.
//
// The path and all keys in m are expected to be normalized and validated.
//
// For a given key x:
//
//   - If x == ".", the path always matches.
//   - If x == path, the path matches.
//   - If x is a directory that contains path, the path matches.
//
// If the map is empty, returns false.
//
// All files and directories in the map are expected to be normalized.
func MapHasEqualOrContainingPath(m map[string]struct{}, path string) bool {
	if len(m) == 0 {
		return false
	}
	if _, ok := m["."]; ok {
		return true
	}
	for curPath := path; curPath != "."; curPath = Dir(curPath) {
		if _, ok := m[curPath]; ok {
			return true
		}
	}
	return false
}

// MapAllEqualOrContainingPaths returns the matching paths in the map.
//
// The path and all keys in m are expected to be normalized and validated.
//
// For a given key x:
//
//   - If x == ".", the path always matches.
//   - If x == path, the path matches.
//   - If x is a directory that contains path, the path matches.
//
// If the map is empty, returns nil.
//
// All files and directories in the map are expected to be normalized.
// The path is normalized within this call.
func MapAllEqualOrContainingPaths(m map[string]struct{}, path string) []string {
	if len(m) == 0 {
		return nil
	}
	n := make(map[string]struct{})
	if _, ok := m["."]; ok {
		// also covers if path == ".".
		n["."] = struct{}{}
	}
	for potentialMatch := range m {
		for curPath := path; curPath != "."; curPath = Dir(curPath) {
			if potentialMatch == curPath {
				n[potentialMatch] = struct{}{}
				break
			}
		}
	}
	return stringutil.MapToSortedSlice(n)
}

// StripComponents strips the specified number of components.
//
// Path expected to be normalized.
// Returns false if the path does not have more than the specified number of components.
func StripComponents(path string, countUint32 uint32) (string, bool) {
	count := int(countUint32)
	if count == 0 {
		return path, true
	}
	components := Components(path)
	if len(components) <= count {
		return "", false
	}
	return Join(components[count:]...), true
}
