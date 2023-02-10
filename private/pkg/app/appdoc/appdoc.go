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

// Package appdoc provides a markdown generator for cobra commands.
// In the future this will need to be adapted to appcmd.Command when we remove usage of Cobra.
package appdoc

import (
	"bufio"
	"bytes"
	"fmt"
	"html"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var codeBlockRegex = regexp.MustCompile(`(^\s\s\s\s)|(^\t)`)

// GenerateMarkdownTree generates markdown for a whole command tree.
func GenerateMarkdownTree(cmd *cobra.Command, dir string, excludes []string) error {
	if !cmd.IsAvailableCommand() {
		return nil
	}
	for _, exclude := range excludes {
		if cmd.Name() == exclude {
			return nil
		}
	}
	for _, c := range cmd.Commands() {
		if err := GenerateMarkdownTree(c, dir, excludes); err != nil {
			return err
		}
	}
	cmdPath := commandPath(cmd)
	filename := filepath.Join(dir, cmdPath)
	if err := os.MkdirAll(path.Dir(filename), os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return GenerateMarkdownPage(cmd, f)
}

// GenerateMarkdownPage creates custom markdown output.
func GenerateMarkdownPage(cmd *cobra.Command, w io.Writer) error {
	var err error
	p := func(format string, a ...any) {
		_, err = w.Write([]byte(fmt.Sprintf(format, a...)))
	}
	p("---\n")
	p("id: %s\n", pageID(cmd))
	p("title: %s\n", cmd.CommandPath())
	p("sidebar_label: %s\n", pageName(cmd))
	p("sidebar_position: %d\n", order(cmd))
	p("slug: /%s\n", slug(cmd))
	p("---\n")
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()
	name := cmd.CommandPath()
	// Name + Version
	if cmd.Version != "" {
		p("version `%s`\n\n", cmd.Version)
	}
	p(cmd.Short)
	p("\n\n")
	// Usage
	if cmd.Runnable() {
		p("### Usage\n")
		p("```\n$ %s\n```\n\n", cmd.UseLine())
	}
	// Synopsis
	if len(cmd.Long) > 0 {
		p("### Description\n\n")
		p("%s \n\n", escapeDescription(cmd.Long))
	}
	// Examples
	if len(cmd.Example) > 0 {
		p("### Examples\n\n")
		p("```\n%s\n```\n\n", escapeDescription(cmd.Example))
	}
	// Flags
	flags := cmd.NonInheritedFlags()
	flags.SetOutput(w)
	if flags.HasAvailableFlags() {
		p("### Flags\n\n")
		p("```\n")
		flags.PrintDefaults()
		p("```\n\n")
	}
	// Parent Flags
	parentFlags := cmd.InheritedFlags()
	parentFlags.SetOutput(w)
	if parentFlags.HasAvailableFlags() {
		p("### Flags inherited from parent commands\n\n```\n")
		parentFlags.PrintDefaults()
		p("```\n\n")
	}
	// Subcommands
	if hasSubCommands(cmd) {
		p("### Subcommands\n\n")
		children := cmd.Commands()
		for _, child := range children {
			if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
				continue
			}
			commandName := name + " " + child.Name()
			childLink := child.Name()
			if hasSubCommands(child) {
				childLink = child.Name() + "/index"
			}
			p("* [%s](%s)\t - %s\n", commandName, childLink, child.Short)
		}
		p("\n")
	}
	// Parent Command
	if cmd.HasParent() {
		p("### Parent Command\n\n")
		parent := cmd.Parent()
		parentName := parent.CommandPath()
		link := "index"
		if hasSubCommands(cmd) {
			link = "../" + link
		}
		p("* [%s](%s)\t - %s\n", parentName, link, parent.Short)
		cmd.VisitParents(func(c *cobra.Command) {
			if c.DisableAutoGenTag {
				cmd.DisableAutoGenTag = c.DisableAutoGenTag
			}
		})
	}
	return err
}

// commandPath converts a cobra command to a path. It stutters the folders and paths
// in order to allow for rendering of the full command in Docusaurus: "buf/buf beta" for example.
// Spaces are used in paths because the current version of Docusaurus
// does not allow for configuring category index pages.
// This function should be removed when migration off docusaurus occurs.
func commandPath(cmd *cobra.Command) string {
	cmdPath := strings.ReplaceAll(cmd.CommandPath(), " ", "/")
	var allPath string
	var previousPath string
	for _, elem := range strings.Split(cmdPath, "/") {
		if allPath != "" {
			allPath += " " + elem
		} else {
			allPath = elem
		}
		cmdPath = path.Join(previousPath, allPath)
		previousPath = cmdPath
	}
	if cmd.HasSubCommands() {
		cmdPath = path.Join(cmdPath, "index.md")
	} else {
		cmdPath += ".md"
	}
	return cmdPath
}

func order(cmd *cobra.Command) int {
	var i int
	if !cmd.HasParent() {
		return 0
	}
	if hasSubCommands(cmd) {
		return 0
	}
	for _, sibling := range cmd.Parent().Commands() {
		if !cmd.IsAvailableCommand() || cmd.IsAdditionalHelpTopicCommand() {
			continue
		}
		i++
		if sibling.CommandPath() == cmd.CommandPath() {
			return i
		}
	}
	return -1
}

func slug(cmd *cobra.Command) string {
	return strings.ReplaceAll(cmd.CommandPath(), " ", "/")
}

func pageID(cmd *cobra.Command) string {
	if hasSubCommands(cmd) {
		return "index"
	}
	return cmd.Name()
}

func pageName(cmd *cobra.Command) string {
	return cmd.CommandPath()
}

func hasSubCommands(cmd *cobra.Command) bool {
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
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
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		text := scanner.Text()
		if codeBlockRegex.MatchString(text) {
			out.WriteString(text)
			out.WriteString("\n")
			continue
		}
		out.WriteString(html.EscapeString(text))
		out.WriteString("\n")
	}
	return out.String()
}
