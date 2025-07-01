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

package bufcobra

import (
	"bufio"
	"bytes"
	"fmt"
	"html"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// generateMarkdownTree generates markdown for a whole command tree
func generateMarkdownTree(
	command *cobra.Command,
	config *config,
	parentDirPath string,
	includeFrontMatter bool,
) error {
	if !command.IsAvailableCommand() {
		return nil
	}
	dirPath := parentDirPath
	fileName := command.Name() + markdownFileExtension
	if command.HasSubCommands() {
		// For commands with subcommands, we create a directory for the command, and create a
		// markdown index file for the command.
		dirPath = filepath.Join(parentDirPath, command.Name())
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return err
		}
		fileName = indexFileName
	}
	filePath := filepath.Join(dirPath, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := generateMarkdownPage(command, config, file, includeFrontMatter); err != nil {
		return err
	}
	if command.HasSubCommands() {
		commands := command.Commands()
		orderCommands(config.WeightCommands, commands)
		for _, command := range commands {
			if err := generateMarkdownTree(command, config, dirPath, includeFrontMatter); err != nil {
				return err
			}
		}
	}
	return nil
}

// generateMarkdownPage creates custom markdown output.
func generateMarkdownPage(
	command *cobra.Command,
	config *config,
	writer io.Writer,
	includeFrontMatter bool,
) error {
	var err error
	p := func(format string, a ...any) {
		_, err = fmt.Fprintf(writer, format, a...)
	}
	if includeFrontMatter {
		p("---\n")
		p("id: %s\n", websitePageIDForCommand(command))
		p("title: %s\n", command.CommandPath())
		p("sidebar_label: %s\n", sidebarLabelForCommand(command, config.SidebarPathThreshold))
		p("sidebar_position: %d\n", websiteSidebarPosition(command, config.WeightCommands))
		p("slug: /%s\n", path.Join(config.SlugPrefix, websiteSlugForCommand(command)))
		p("---\n")
	} else {
		p("# %s\n", command.CommandPath())
	}
	command.InitDefaultHelpCmd()
	command.InitDefaultHelpFlag()
	if command.Version != "" {
		p("version `%s`\n\n", command.Version)
	}
	p(command.Short)
	p("\n\n")
	if command.Runnable() {
		p("### Usage\n")
		p("```console\n$ %s\n```\n\n", command.UseLine())
	}
	if len(command.Long) > 0 {
		p("### Description\n\n")
		p("%s \n\n", processDescription(command.Long))
	}
	if len(command.Example) > 0 {
		p("### Examples\n\n")
		p("```console\n%s\n```\n\n", processDescription(command.Example))
	}
	commandFlags := command.NonInheritedFlags()
	if commandFlags.HasAvailableFlags() {
		p("### Flags {#flags}\n\n")
		if err := writeFlags(commandFlags, writer); err != nil {
			return err
		}
	}
	inheritedFlags := command.InheritedFlags()
	if inheritedFlags.HasAvailableFlags() {
		p("### Flags inherited from parent commands {#persistent-flags}\n")
		if err := writeFlags(inheritedFlags, writer); err != nil {
			return err
		}
	}
	if hasSubCommands(command) {
		p("### Subcommands\n\n")
		children := command.Commands()
		orderCommands(config.WeightCommands, children)
		for _, child := range children {
			if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
				continue
			}
			childRelPath := child.Name() + markdownFileExtension
			if child.HasSubCommands() {
				childRelPath = filepath.Join(child.Name(), indexFileName)
			}
			p("* [%s](./%s)\t - %s\n", child.CommandPath(), childRelPath, child.Short)
		}
		p("\n")
	}
	if command.HasParent() {
		p("### Parent Command\n\n")
		parent := command.Parent()
		parentName := parent.CommandPath()
		if hasSubCommands(command) {
			// If the current command has sub-commands, the parent command is the index file in
			// the parent directory.
			p("* [%s](../%s)\t - %s\n", parentName, indexFileName, parent.Short)
		} else {
			// If the current command is a leaf command, the parent command is the index file in
			// the current directory.
			p("* [%s](./%s)\t - %s\n", parentName, indexFileName, parent.Short)
		}
		command.VisitParents(func(c *cobra.Command) {
			if c.DisableAutoGenTag {
				command.DisableAutoGenTag = c.DisableAutoGenTag
			}
		})
	}
	return err
}

func websitePageIDForCommand(cmd *cobra.Command) string {
	return strings.ReplaceAll(cmd.CommandPath(), " ", "-")
}

// hasSubCommands checks for whether a command has available sub-commands, not including help.
func hasSubCommands(cmd *cobra.Command) bool {
	for _, command := range cmd.Commands() {
		if !command.IsAvailableCommand() || command.IsAdditionalHelpTopicCommand() {
			continue
		}
		return true
	}
	return false
}

// processDescription is used to process description and example text. It does the following:
//
// - Converts all code blocks to console code blocks  (```console)
// - Unindents code blocks
// - Writes out the text, escaping all HTML characters.
//
// This is done because Pygments (which mkdocs-material uses for syntax highlighting)
// specifies `console` as the language for bash sessions in code blocks.
func processDescription(description string) string {
	out := &bytes.Buffer{}
	read := bufio.NewReader(strings.NewReader(description))
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
				out.WriteString("```console\n")
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

func orderCommands(weights map[string]int, commands []*cobra.Command) {
	sort.SliceStable(commands, func(i, j int) bool {
		return weights[commands[i].CommandPath()] < weights[commands[j].CommandPath()]
	})
}

func writeFlags(f *pflag.FlagSet, writer io.Writer) error {
	var err error
	p := func(format string, a ...any) {
		_, err = fmt.Fprintf(writer, format, a...)
	}
	f.VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden {
			return
		}
		if flag.Shorthand != "" && flag.ShorthandDeprecated == "" {
			p("#### -%s, --%s", flag.Shorthand, flag.Name)
		} else {
			p("#### --%s", flag.Name)
		}
		varname, usage := pflag.UnquoteUsage(flag)
		if varname != "" {
			p(" *%s*", varname)
		}
		p(" {#%s}", flag.Name)
		p("\n")
		p("%s", usage)
		if flag.NoOptDefVal != "" {
			switch flag.Value.Type() {
			case "string":
				p("[=\"%s\"]", flag.NoOptDefVal)
			case "bool":
				if flag.NoOptDefVal != "true" {
					p("[=%s]", flag.NoOptDefVal)
				}
			case "count":
				if flag.NoOptDefVal != "+1" {
					p("[=%s]", flag.NoOptDefVal)
				}
			default:
				p("[=%s]", flag.NoOptDefVal)
			}
		}
		if len(flag.Deprecated) != 0 {
			p(" (DEPRECATED: %s)", flag.Deprecated)
		}
		p("\n\n")
	})
	return err
}

// websiteSidebarPosition calculates the position of the given command in the website sidebar.
func websiteSidebarPosition(cmd *cobra.Command, weights map[string]int) int {
	// Return 0 if the command has no parent
	if !cmd.HasParent() {
		return 0
	}
	siblings := cmd.Parent().Commands()
	orderCommands(weights, siblings)
	position := 0
	for _, sibling := range siblings {
		if isCommandVisible(sibling) {
			position++
			if sibling.CommandPath() == cmd.CommandPath() {
				return position
			}
		}
	}
	return -1
}

// isCommandVisible checks if a command is visible (available, not an additional help topic, and not hidden).
func isCommandVisible(command *cobra.Command) bool {
	return command.IsAvailableCommand() && !command.IsAdditionalHelpTopicCommand() && !command.Hidden
}

func websiteSlugForCommand(command *cobra.Command) string {
	return strings.ReplaceAll(command.CommandPath(), " ", "/")
}

func sidebarLabelForCommand(command *cobra.Command, maxSidebarLen int) string {
	if len(strings.Split(command.CommandPath(), " ")) > maxSidebarLen {
		return command.Name()
	}
	return command.CommandPath()
}
