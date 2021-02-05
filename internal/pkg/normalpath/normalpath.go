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

// Package normalpath provides functions similar to filepath.
//
// A normalized path is a cleaned and to-slash'ed path.
// A validated path validates that a path is relative and does not jump context.
package normalpath

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/stringutil"
)

const (
	// Relative is the PathType for normalized and validated paths.
	Relative PathType = 1
	// Absolute is the PathType for normalized and absolute paths.
	Absolute PathType = 2

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

// PathType is a terminate type for path comparisons.
type PathType int

// Separator gets the string value of the separator.
//
// TODO: rename to Terminator if we keep this
// TODO: we should probably refactor so we never need to use absolute paths at all
// this could be accomplished if we could for ExternalPathToRelPath on buckets
func (t PathType) Separator() string {
	switch t {
	case Relative:
		return "."
	case Absolute:
		return "/"
	default:
		return ""
	}
}

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

// Unwrap implements errors.Unwrap for Error.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is implements errors.Is for Error.
func (e *Error) Is(err error) bool {
	_, ok := err.(*Error)
	return ok
}

// NormalizeAndValidate normalizes and validates the given path.
//
// This calls Normalize on the path.
// Returns Error if the path is not relative or jumps context.
// This can be used to validate that paths are valid to use with Buckets.
// The error message is safe to pass to users.
func NormalizeAndValidate(path string) (string, error) {
	normalizedPath := Normalize(path)
	if filepath.IsAbs(normalizedPath) {
		return "", NewError(path, errNotRelative)
	}
	// https://github.com/bufbuild/buf/issues/51
	if strings.HasPrefix(normalizedPath, normalizedRelPathJumpContextPrefix) {
		return "", NewError(path, errOutsideContextDir)
	}
	return normalizedPath, nil
}

// NormalizeAndAbsolute normalizes the path and makes it absolute.
func NormalizeAndAbsolute(path string) (string, error) {
	absPath, err := filepath.Abs(Unnormalize(path))
	if err != nil {
		return "", err
	}
	return Normalize(absPath), nil
}

// NormalizeAndTransformForPathType calls NormalizeAndValidate for relative
// paths, and NormalizeAndAbsolute for absolute paths.
func NormalizeAndTransformForPathType(path string, pathType PathType) (string, error) {
	switch pathType {
	case Relative:
		return NormalizeAndValidate(path)
	case Absolute:
		return NormalizeAndAbsolute(path)
	default:
		return "", fmt.Errorf("unknown PathType: %v", pathType)
	}
}

// Normalize normalizes the given path.
//
// This calls filepath.Clean and filepath.ToSlash on the path.
// If the path is "" or ".", this returns ".".
func Normalize(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
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
// The path and value are expected to be normalized and validated if Relative is used.
// The path and value are expected to be normalized and absolute if Absolute is used.
//
// For a given dirPath:
//
//   - If path == PathType, dirPath does not contain the path.
//   - If dirPath == PathType, the dirPath contains the path.
//   - If dirPath is a directory that contains path, this returns true.
func ContainsPath(dirPath string, path string, pathType PathType) bool {
	if dirPath == path {
		return false
	}
	return EqualsOrContainsPath(dirPath, Dir(path), pathType)
}

// EqualsOrContainsPath returns true if the value is equal to or contains the path.
//
// The path and value are expected to be normalized and validated if Relative is used.
// The path and value are expected to be normalized and absolute if Absolute is used.
//
// For a given value:
//
//   - If value == PathType, the value contains the path.
//   - If value == path, the value is equal to the path.
//   - If value is a directory that contains path, this returns true.
func EqualsOrContainsPath(value string, path string, pathType PathType) bool {
	separator := pathType.Separator()
	if separator == "" {
		return false
	}
	if value == separator {
		return true
	}
	// TODO: can we optimize this with strings.HasPrefix(path, value + "/") somehow?
	for curPath := path; curPath != separator; curPath = Dir(curPath) {
		if value == curPath {
			return true
		}
	}
	return false
}

// MapHasEqualOrContainingPath returns true if the path matches any file or directory in the map.
//
// The path and value are expected to be normalized and validated if Relative is used.
// The path and value are expected to be normalized and absolute if Absolute is used.
//
// For a given key x:
//
//   - If x == PathType, the path always matches.
//   - If x == path, the path matches.
//   - If x is a directory that contains path, the path matches.
//
// If the map is empty, returns false.
func MapHasEqualOrContainingPath(m map[string]struct{}, path string, pathType PathType) bool {
	separator := pathType.Separator()
	if separator == "" {
		return false
	}
	if len(m) == 0 {
		return false
	}
	if _, ok := m[separator]; ok {
		return true
	}
	for curPath := path; curPath != separator; curPath = Dir(curPath) {
		if _, ok := m[curPath]; ok {
			return true
		}
	}
	return false
}

// MapAllEqualOrContainingPaths returns the matching paths in the map in a sorted slice.
//
// The path and all keys in m are expected to be normalized and validated.
//
// For a given key x:
//
//   - If x == PathType, the path always matches.
//   - If x == path, the path matches.
//   - If x is a directory that contains path, the path matches.
//
// If the map is empty, returns nil.
func MapAllEqualOrContainingPaths(m map[string]struct{}, path string, pathType PathType) []string {
	if len(m) == 0 {
		return nil
	}
	return stringutil.MapToSortedSlice(MapAllEqualOrContainingPathMap(m, path, pathType))
}

// MapAllEqualOrContainingPathMap returns the matching paths in the map in a new map.
//
// The path and all keys in m are expected to be normalized and validated.
//
// For a given key x:
//
//   - If x == PathType, the path always matches.
//   - If x == path, the path matches.
//   - If x is a directory that contains path, the path matches.
//
// If the map is empty, returns nil.
func MapAllEqualOrContainingPathMap(m map[string]struct{}, path string, pathType PathType) map[string]struct{} {
	separator := pathType.Separator()
	if separator == "" {
		return nil
	}
	if len(m) == 0 {
		return nil
	}
	n := make(map[string]struct{})
	if _, ok := m[separator]; ok {
		// also covers if path == separator.
		n[separator] = struct{}{}
	}
	for potentialMatch := range m {
		for curPath := path; curPath != separator; curPath = Dir(curPath) {
			if potentialMatch == curPath {
				n[potentialMatch] = struct{}{}
				break
			}
		}
	}
	return n
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

// ValidatePathComponent validates that the string is a valid
// component of a path, e.g. it can be Joined and form a valid path.
func ValidatePathComponent(component string) error {
	if component == "" {
		return errors.New("path component must not be empty")
	}
	if strings.ContainsRune(component, '/') {
		return errors.New(`path component must not contain "/" `)
	}
	if strings.Contains(component, "..") {
		return errors.New(`path component must not contain ".."`)
	}
	if url.PathEscape(component) != component {
		return fmt.Errorf(
			"path component must match its URL escaped version: %q did not match %q",
			component,
			url.PathEscape(component),
		)
	}
	return nil
}

// ValidatePathComponents validates that all the strings are valid
// components of a path, e.g. they can be Joined and form a valid path.
func ValidatePathComponents(components ...string) error {
	for _, component := range components {
		if err := ValidatePathComponent(component); err != nil {
			return err
		}
	}
	return nil
}
