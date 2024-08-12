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

package storage

import (
	"github.com/bufbuild/buf/private/pkg/normalpath"
)

// Matcher is a path matcher.
//
// This will cause a Bucket to operate as if it only contains matching paths.
type Matcher interface {
	// MatchPath returns true if the path matches.
	//
	// The path is expected to be normalized and validated.
	MatchPath(string) bool
	// MatchPrefix returns the matching prefixes.
	//
	// The prefix is expected to be normalized and validated.
	// If the prefix is only partially matched, the valid child prefixes are returned.
	// If the prefix is fully matched, the prefix is returned.
	// If the prefix is not matched, nil is returned.
	MatchPrefix(string) []string
	isMatcher()
}

// MatchPathExt returns a Matcher for the extension.
func MatchPathExt(ext string) Matcher {
	return pathMatcherFunc(func(path string) bool {
		return normalpath.Ext(path) == ext
	})
}

// MatchPathBase returns a Matcher for the base.
func MatchPathBase(base string) Matcher {
	return pathMatcherFunc(func(path string) bool {
		return normalpath.Base(path) == base
	})
}

// MatchPathEqual returns a Matcher for the path.
func MatchPathEqual(equalPath string) Matcher {
	return pathMatchEqual(equalPath)
}

// MatchPathEqualOrContained returns a Matcher for the path that matches
// on paths equal or contained by equalOrContainingPath.
func MatchPathEqualOrContained(equalOrContainingPath string) Matcher {
	return pathMatchEqualOrContained(equalOrContainingPath)
}

// MatchPathContained returns a Matcher for the directory that matches
// on paths by contained by containingDir.
func MatchPathContained(containingDir string) Matcher {
	return pathMatchContained(containingDir)
}

// MatchOr returns an Or of the Matchers.
func MatchOr(matchers ...Matcher) Matcher {
	return orMatcher(matchers)
}

// MatchAnd returns an And of the Matchers.
func MatchAnd(matchers ...Matcher) Matcher {
	return andMatcher(matchers)
}

// MatchNot returns an Not of the Matcher.
func MatchNot(matcher Matcher) Matcher {
	return notMatcher{matcher}
}

// ***** private *****

// We limit or/and/not to Matchers as composite logic must assume
// the input path is not modified, so that we can always return it

type pathMatcherFunc func(string) bool

func (f pathMatcherFunc) MatchPath(path string) bool {
	return f(path)
}

func (f pathMatcherFunc) MatchPrefix(prefix string) []string {
	return []string{prefix}
}

func (pathMatcherFunc) isMatcher() {}

type pathMatchEqual string

func (p pathMatchEqual) MatchPath(path string) bool {
	return string(p) == path
}

func (p pathMatchEqual) MatchPrefix(prefix string) []string {
	if normalpath.ContainsPath(prefix, string(p), normalpath.Relative) {
		return []string{string(p)}
	}
	return nil
}

func (pathMatchEqual) isMatcher() {}

type pathMatchEqualOrContained string

func (p pathMatchEqualOrContained) MatchPath(path string) bool {
	return normalpath.EqualsOrContainsPath(string(p), path, normalpath.Relative)
}

func (p pathMatchEqualOrContained) MatchPrefix(prefix string) []string {
	if normalpath.ContainsPath(string(p), prefix, normalpath.Relative) {
		return []string{prefix}
	}
	if normalpath.EqualsOrContainsPath(prefix, string(p), normalpath.Relative) {
		return []string{string(p)}
	}
	return nil
}

func (pathMatchEqualOrContained) isMatcher() {}

type pathMatchContained string

func (p pathMatchContained) MatchPath(path string) bool {
	return normalpath.ContainsPath(string(p), path, normalpath.Relative)
}

func (p pathMatchContained) MatchPrefix(prefix string) []string {
	if normalpath.ContainsPath(string(p), prefix, normalpath.Relative) {
		return []string{prefix}
	}
	if normalpath.EqualsOrContainsPath(prefix, string(p), normalpath.Relative) {
		return []string{string(p)}
	}
	return nil
}

func (pathMatchContained) isMatcher() {}

type orMatcher []Matcher

func (o orMatcher) MatchPath(path string) bool {
	for _, matcher := range o {
		if matches := matcher.MatchPath(path); matches {
			return true
		}
	}
	return false
}

func (o orMatcher) MatchPrefix(prefix string) []string {
	var matches []string
	for _, matcher := range o {
		matches = append(matches, matcher.MatchPrefix(prefix)...)
	}
	return orIncludePaths(matches)
}

func (orMatcher) isMatcher() {}

type andMatcher []Matcher

func (a andMatcher) MatchPath(path string) bool {
	for _, matcher := range a {
		if matches := matcher.MatchPath(path); !matches {
			return false
		}
	}
	return true
}

func (a andMatcher) MatchPrefix(prefix string) []string {
	var matches []string
	for _, matcher := range a {
		matches = append(matches, matcher.MatchPrefix(prefix)...)
	}
	return andIncludePaths(matches)
}

func (andMatcher) isMatcher() {}

type notMatcher struct {
	delegate Matcher
}

func (n notMatcher) MatchPath(path string) bool {
	return !n.delegate.MatchPath(path)
}

func (n notMatcher) MatchPrefix(prefix string) []string {
	matches := n.delegate.MatchPrefix(prefix)
	if len(matches) == 0 {
		return []string{prefix}
	}
	for _, match := range matches {
		if normalpath.EqualsOrContainsPath(match, prefix, normalpath.Relative) {
			return nil
		}
	}
	return []string{prefix}
}

func (notMatcher) isMatcher() {}

func orIncludePaths(paths []string) []string {
	var includes []string
	var includeSet map[string]struct{}
	for a, includeA := range paths {
		if _, isIncluded := includeSet[includeA]; isIncluded {
			continue
		}
		var hasParent bool
		for b, includeB := range paths {
			if a == b || includeA == includeB {
				continue
			}
			if normalpath.EqualsOrContainsPath(includeB, includeA, normalpath.Relative) {
				hasParent = true
				break
			}
		}
		if !hasParent {
			if includeSet == nil {
				includeSet = make(map[string]struct{})
			}
			includeSet[includeA] = struct{}{}
			includes = append(includes, includeA)
		}
	}
	return includes
}

func andIncludePaths(paths []string) []string {
	if len(paths) <= 1 {
		return paths
	}
	include := paths[0]
	for _, path := range paths[1:] {
		isParent := normalpath.EqualsOrContainsPath(path, include, normalpath.Relative)
		isChild := normalpath.EqualsOrContainsPath(include, path, normalpath.Relative)
		if !isParent && !isChild {
			return nil
		}
		if isParent {
			include = path
		}
	}
	return []string{include}
}
