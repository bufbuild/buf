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

// deprecationChecker determines which types should have deprecated options added.
// It is used during formatting to inject deprecation options at appropriate locations.
type deprecationChecker struct {
	prefixes [][]string // FQN prefixes split into components
}

// newDeprecationChecker creates a new deprecationChecker for the given FQN prefixes.
// Each prefix is split by "." into components for matching.
func newDeprecationChecker(fqnPrefixes []string) *deprecationChecker {
	prefixes := make([][]string, 0, len(fqnPrefixes))
	for _, prefix := range fqnPrefixes {
		if prefix != "" {
			prefixes = append(prefixes, strings.Split(prefix, "."))
		}
	}
	return &deprecationChecker{
		prefixes: prefixes,
	}
}

// isEmpty returns true if there are no deprecation prefixes configured.
func (d *deprecationChecker) isEmpty() bool {
	return len(d.prefixes) == 0
}

// shouldDeprecate returns true if the given FQN should be deprecated using prefix matching.
// This is used for packages, messages, enums, services, and RPCs.
func (d *deprecationChecker) shouldDeprecate(fqn []string) bool {
	for _, prefix := range d.prefixes {
		if fqnMatchesPrefix(fqn, prefix) {
			return true
		}
	}
	return false
}

// shouldDeprecateExact returns true if the given FQN matches exactly.
// This is used for fields and enum values which are only deprecated on exact match.
func (d *deprecationChecker) shouldDeprecateExact(fqn []string) bool {
	for _, prefix := range d.prefixes {
		if fqnMatchesExact(fqn, prefix) {
			return true
		}
	}
	return false
}

// fqnMatchesPrefix returns true if fqn starts with prefix using component-based matching.
// For example, "foo.bar" matches "foo.bar.baz" but "foo.bar.b" does NOT match "foo.bar.baz".
func fqnMatchesPrefix(fqn, prefix []string) bool {
	if len(prefix) > len(fqn) {
		return false
	}
	for i, p := range prefix {
		if fqn[i] != p {
			return false
		}
	}
	return true
}

// fqnMatchesExact returns true if fqn exactly equals prefix.
func fqnMatchesExact(fqn, prefix []string) bool {
	if len(fqn) != len(prefix) {
		return false
	}
	for i, p := range prefix {
		if fqn[i] != p {
			return false
		}
	}
	return true
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
	// Get the identifier value
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
	// Check value is "true"
	if ident, ok := opt.Val.(*ast.IdentNode); ok {
		return ident.Val == "true"
	}
	return false
}

// packageNameToComponents extracts package name components from an identifier node.
func packageNameToComponents(name ast.IdentValueNode) []string {
	switch n := name.(type) {
	case *ast.IdentNode:
		return []string{n.Val}
	case *ast.CompoundIdentNode:
		components := make([]string, len(n.Components))
		for i, comp := range n.Components {
			components[i] = comp.Val
		}
		return components
	default:
		return nil
	}
}
