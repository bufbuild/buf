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

package bufformat

import (
	"strings"

	"github.com/bufbuild/protocompile/ast"
)

// fullNameMatcher determines which types should have deprecated options added.
type fullNameMatcher struct {
	prefixes []string
}

// newFullNameMatcher creates a new matcher for the given FQN prefixes.
func newFullNameMatcher(fqnPrefixes ...string) *fullNameMatcher {
	return &fullNameMatcher{prefixes: fqnPrefixes}
}

// matchesPrefix returns true if the given FQN matches using prefix matching.
func (d *fullNameMatcher) matchesPrefix(fqn string) bool {
	for _, prefix := range d.prefixes {
		if fqnMatchesPrefix(fqn, prefix) {
			return true
		}
	}
	return false
}

// matchesExact returns true if the given FQN matches exactly.
func (d *fullNameMatcher) matchesExact(fqn string) bool {
	for _, prefix := range d.prefixes {
		if fqn == prefix {
			return true
		}
	}
	return false
}

// fqnMatchesPrefix returns true if fqn starts with prefix using component-based matching.
func fqnMatchesPrefix(fqn, prefix string) bool {
	if len(prefix) > len(fqn) {
		return false
	}
	if len(prefix) == len(fqn) {
		return fqn == prefix
	}
	return prefix == "" || strings.HasPrefix(fqn, prefix+".")
}

// hasDeprecatedOption checks if a slice of declarations contains a deprecated = true option.
func hasDeprecatedOption[T any](decls []T) bool {
	for _, decl := range decls {
		if opt, ok := any(decl).(*ast.OptionNode); ok && isDeprecatedOptionNode(opt) {
			return true
		}
	}
	return false
}

// hasCompactDeprecatedOption checks if a CompactOptionsNode contains deprecated = true.
func hasCompactDeprecatedOption(opts *ast.CompactOptionsNode) bool {
	if opts == nil {
		return false
	}
	for _, opt := range opts.Options {
		if isDeprecatedOptionNode(opt) {
			return true
		}
	}
	return false
}

// isDeprecatedOptionNode checks if an option node is "deprecated = true".
func isDeprecatedOptionNode(opt *ast.OptionNode) bool {
	if opt.Name == nil || len(opt.Name.Parts) != 1 {
		return false
	}
	part := opt.Name.Parts[0]
	if part.Name == nil {
		return false
	}
	var name string
	switch n := part.Name.(type) {
	case *ast.IdentNode:
		name = n.Val
	default:
		return false
	}
	if name != "deprecated" {
		return false
	}
	if ident, ok := opt.Val.(*ast.IdentNode); ok {
		return ident.Val == "true"
	}
	return false
}

// packageNameToString extracts package name as a dot-separated string from an identifier node.
func packageNameToString(name ast.IdentValueNode) string {
	switch n := name.(type) {
	case *ast.IdentNode:
		return n.Val
	case *ast.CompoundIdentNode:
		components := make([]string, len(n.Components))
		for i, comp := range n.Components {
			components[i] = comp.Val
		}
		return strings.Join(components, ".")
	default:
		return ""
	}
}

// parentFQN returns the parent FQN by removing the last component.
// For example, "foo.bar.baz" returns "foo.bar".
func parentFQN(fqn string) string {
	if idx := strings.LastIndex(fqn, "."); idx >= 0 {
		return fqn[:idx]
	}
	return ""
}
