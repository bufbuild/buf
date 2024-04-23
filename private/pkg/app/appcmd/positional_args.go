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

// Package appcmd contains helper functionality for applications using commands.
package appcmd

import (
	"github.com/spf13/cobra"
)

var (
	// NoArgs matches cobra.NoArgs.
	NoArgs = newPositionalArgs(cobra.NoArgs)
	// OnlyValidArgs matches cobra.OnlyValidArgs.
	OnlyValidArgs = newPositionalArgs(cobra.OnlyValidArgs)
	// ArbitraryArgs matches cobra.ArbitraryArgs.
	ArbitraryArgs = newPositionalArgs(cobra.ArbitraryArgs)
)

// MinimumNArgs matches cobra.MinimumNArgs.
func MinimumNArgs(n int) PositionalArgs {
	return newPositionalArgs(cobra.MinimumNArgs(n))
}

// MaximumNArgs matches cobra.MaximumNArgs.
func MaximumNArgs(n int) PositionalArgs {
	return newPositionalArgs(cobra.MaximumNArgs(n))
}

// ExactArgs matches cobra.ExactArgs.
func ExactArgs(n int) PositionalArgs {
	return newPositionalArgs(cobra.ExactArgs(n))
}

// RangeArgs matches cobra.RangeArgs.
func RangeArgs(min int, max int) PositionalArgs {
	return newPositionalArgs(cobra.RangeArgs(min, max))
}

// PostionalArgs matches cobra.PositionalArgs so that importers of appcmd do
// not need to reference cobra (and shouldn't).
type PositionalArgs interface {
	cobra() cobra.PositionalArgs
}

// *** PRIVATE ***

type positionalArgs struct {
	args cobra.PositionalArgs
}

func newPositionalArgs(args cobra.PositionalArgs) *positionalArgs {
	return &positionalArgs{
		args: args,
	}
}

func (p *positionalArgs) cobra() cobra.PositionalArgs {
	return p.args
}
