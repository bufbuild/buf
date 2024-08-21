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
	return pathMatcherFunc(func(path string) bool {
		return path == equalPath
	})
}

// MatchPathEqualOrContained returns a Matcher for the path that matches
// on paths equal or contained by equalOrContainingPath.
func MatchPathEqualOrContained(equalOrContainingPath string) Matcher {
	return pathMatcherFunc(func(path string) bool {
		return normalpath.EqualsOrContainsPath(equalOrContainingPath, path, normalpath.Relative)
	})
}

// MatchPathContained returns a Matcher for the directory that matches
// on paths by contained by containingDir.
func MatchPathContained(containingDir string) Matcher {
	return pathMatcherFunc(func(path string) bool {
		return normalpath.ContainsPath(containingDir, path, normalpath.Relative)
	})
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

func (pathMatcherFunc) isMatcher() {}

type orMatcher []Matcher

func (o orMatcher) MatchPath(path string) bool {
	for _, matcher := range o {
		if matches := matcher.MatchPath(path); matches {
			return true
		}
	}
	return false
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

func (andMatcher) isMatcher() {}

type notMatcher struct {
	delegate Matcher
}

func (n notMatcher) MatchPath(path string) bool {
	return !n.delegate.MatchPath(path)
}

func (notMatcher) isMatcher() {}
