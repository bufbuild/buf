// Copyright 2020-2023 Buf Technologies, Inc.
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

package appcmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	excludeCommandsFlagName = "exclude-command"
)

var codeBlockRegex = regexp.MustCompile(`(^\s\s\s\s)|(^\t)`)

type webpagesFlags struct {
	SlugPrefix      string
	ExcludeCommands []string
}

func newWebpagesFlags() *webpagesFlags {
	return &webpagesFlags{}
}

func (f *webpagesFlags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVar(
		&f.ExcludeCommands,
		excludeCommandsFlagName,
		nil,
		"Exclude these commands from doc generation",
	)
}

// newWebpagesCommand returns a "webpages" command that generates docusaurus markdown for cobra commands.
// In the future this will need to be adapted to accept a Command when cobra.Command is removed.
func newWebpagesCommand(
	command *cobra.Command,
) *Command {
	flags := newWebpagesFlags()
	return &Command{
		Use:    "webpages",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		Run: func(ctx context.Context, container app.Container) error {
			excludes := make(map[string]bool)
			for _, exclude := range flags.ExcludeCommands {
				excludes[exclude] = true
			}
			for _, cmd := range command.Commands() {
				if excludes[cmd.CommandPath()] {
					cmd.Hidden = true
				}
			}
			return generateMarkdownTree(
				command,
				os.Stdout,
			)
		},
		BindFlags: flags.Bind,
	}
}

// generateMarkdownTree generates markdown for a whole command tree.
func generateMarkdownTree(cmd *cobra.Command, w io.Writer) error {
	if !cmd.IsAvailableCommand() {
		return nil
	}
	if err := generateMarkdownPage(cmd, w); err != nil {
		return err
	}
	for _, command := range cmd.Commands() {
		if err := generateMarkdownTree(command, w); err != nil {
			return err
		}
	}
	return nil
}

// generateMarkdownPage creates custom markdown output.
func generateMarkdownPage(cmd *cobra.Command, w io.Writer) error {
	var err error
	p := func(format string, a ...any) {
		_, err = w.Write([]byte(fmt.Sprintf(format, a...)))
	}
	id := websitePageID(cmd)
	p("\n")
	p("## %s {#%s}\n", cmd.CommandPath(), id)
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()
	if cmd.Version != "" {
		p("version `%s`\n\n", cmd.Version)
	}
	p(cmd.Short)
	p("\n\n")
	if cmd.Runnable() {
		p("#### Usage {#%s-usage} \n", id)
		p("```terminal\n$ %s\n```\n\n", cmd.UseLine())
	}
	if len(cmd.Long) > 0 {
		p("#### Description {#%s-description}\n\n", id)
		p("%s \n\n", escapeDescription(cmd.Long))
	}
	if len(cmd.Example) > 0 {
		p("#### Examples {#%s-examples}\n\n", id)
		p("```\n%s\n```\n\n", escapeDescription(cmd.Example))
	}
	commandFlags := cmd.NonInheritedFlags()
	commandFlags.SetOutput(w)
	if commandFlags.HasAvailableFlags() {
		p("#### Flags {#%s-flags}\n\n", id)
		p("```\n")
		commandFlags.PrintDefaults()
		p("```\n\n")
	}
	inheritedFlags := cmd.InheritedFlags()
	inheritedFlags.SetOutput(w)
	if inheritedFlags.HasAvailableFlags() {
		p("#### Flags inherited from parent commands {#%s-persistent-flags}\n\n```\n", id)
		inheritedFlags.PrintDefaults()
		p("```\n\n")
	}
	if hasSubCommands(cmd) {
		p("#### Subcommands {#%s-subcommands}\n\n", id)
		children := cmd.Commands()
		for _, child := range children {
			if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
				continue
			}
			p("* [%s](#%s)\t - %s\n", child.CommandPath(), websitePageID(child), child.Short)
		}
		p("\n")
	}
	if cmd.HasParent() {
		p("#### Parent Command {#%s-parent-command}\n\n", id)
		parent := cmd.Parent()
		parentName := parent.CommandPath()
		p("* [%s](#%s)\t - %s\n", parentName, websitePageID(parent), parent.Short)
		cmd.VisitParents(func(c *cobra.Command) {
			if c.DisableAutoGenTag {
				cmd.DisableAutoGenTag = c.DisableAutoGenTag
			}
		})
	}
	return err
}

// commandFilePath converts a cobra command to a path. It stutters the folders and paths
// in order to allow for rendering of the full command in Docusaurus: "buf/buf beta" for example.
// Spaces are used in paths because the current version of Docusaurus
// does not allow for configuring category index pages.
// This function should be removed when migration off docusaurus occurs.
func commandFilePath(cmd *cobra.Command) string {
	commandPath := strings.Split(cmd.CommandPath(), " ")
	var parentPath, currentPath []string
	for i := range commandPath {
		currentPath = append(parentPath, strings.Join(commandPath[:i+1], " "))
		parentPath = currentPath
	}
	fullPath := path.Join(currentPath...)
	if cmd.HasSubCommands() {
		return path.Join(fullPath, "index.md")
	}
	return fullPath + ".md"
}

func websitePageID(cmd *cobra.Command) string {
	return strings.ReplaceAll(cmd.CommandPath(), " ", "-")
}

func hasSubCommands(cmd *cobra.Command) bool {
	for _, command := range cmd.Commands() {
		if !command.IsAvailableCommand() || command.IsAdditionalHelpTopicCommand() {
			continue
		}
		return true
	}
	return false
}

// escapeDescription is a bit of a hack because docusaurus markdown rendering is a bit weird.
// If the code block is indented then escaping html characters is skipped, otherwise it will
// html.Escape the string.
func escapeDescription(s string) string {
	out := &bytes.Buffer{}
	read := bufio.NewReader(strings.NewReader(s))
	var inCodeBlock bool
	for {
		line, _, err := read.ReadLine()
		if err == io.EOF {
			break
		}
		text := string(line)
		// convert indented code blocks into terminal code blocks so the
		// $ isn't copied when using the copy button
		if codeBlockRegex.MatchString(text) {
			if !inCodeBlock {
				out.WriteString("```terminal\n")
				inCodeBlock = true
			}
			// remove the indentation level from the indented code block
			text = codeBlockRegex.ReplaceAllString(text, "")
			out.WriteString(text)
			out.WriteString("\n")
			continue
		}
		// indented code blocks can have blank lines in them so
		// if the next line is a whitespace then we don't want to
		// terminate the code block
		if inCodeBlock && text == "" {
			if b, err := read.Peek(1); err == nil && unicode.IsSpace(rune(b[0])) {
				out.WriteString(text)
				out.WriteString("\n")
				continue
			}
		}
		// terminate the fenced code block with ```
		if inCodeBlock {
			out.WriteString("```\n")
			inCodeBlock = false
		}
		out.WriteString(html.EscapeString(text))
		out.WriteString("\n")
	}
	if inCodeBlock {
		out.WriteString("```\n")
	}
	return out.String()
}
