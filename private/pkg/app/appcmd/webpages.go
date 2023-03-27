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
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

const (
	webpagesConfigFlag = "config"
)

var codeBlockRegex = regexp.MustCompile(`(^\s\s\s\s)|(^\t)`)

type webpagesFlags struct {
	Config string
}

// webpagesConfig configures the doc generator, example config:
// prefix: |
// ---
// title: Buf CLI
// sidebar_position: 0
// toc_max_heading_level: 2
// slug: /reference/cli/buf
// ---
// exclude_commands:
// - buf completion
// - buf ls-files
// weight_commands:
// buf beta: 1
type webpagesConfig struct {
	Prefix          string         `yaml:"prefix,omitempty"`
	ExcludeCommands []string       `yaml:"exclude_commands,omitempty"`
	WeightCommands  map[string]int `yaml:"weight_commands,omitempty"`
}

func newWebpagesFlags() *webpagesFlags {
	return &webpagesFlags{}
}

func (f *webpagesFlags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Config,
		webpagesConfigFlag,
		"",
		"Config file to use",
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
		Hidden: true,
		Run: func(ctx context.Context, container app.Container) error {
			cfg, err := readConfig(flags.Config)
			if err != nil {
				return err
			}
			excludes := make(map[string]bool)
			for _, exclude := range cfg.ExcludeCommands {
				excludes[exclude] = true
			}
			for _, cmd := range command.Commands() {
				if excludes[cmd.CommandPath()] {
					cmd.Hidden = true
				}
			}
			if _, err := os.Stdout.WriteString(cfg.Prefix); err != nil {
				return err
			}
			return generateMarkdownTree(
				command,
				"docs",
				cfg.WeightCommands,
			)
		},
		BindFlags: flags.Bind,
	}
}

// generateMarkdownTree generates markdown for a whole command tree.
func generateMarkdownTree(cmd *cobra.Command, parentDirPath string, weights map[string]int) error {
	if !cmd.IsAvailableCommand() {
		return nil
	}

	dirPath := parentDirPath
	fileName := cmd.Name() + ".md"

	if cmd.HasSubCommands() {
		dirPath = filepath.Join(parentDirPath, cmd.Name())
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return err
		}
		fileName = "index.md"
	}

	filePath := filepath.Join(dirPath, fileName)
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := generateMarkdownPage(cmd, f, dirPath, weights); err != nil {
		return err
	}

	if cmd.HasSubCommands() {
		commands := cmd.Commands()
		orderCommands(weights, commands)
		for _, command := range commands {
			if err := generateMarkdownTree(command, dirPath, weights); err != nil {
				return err
			}
		}
	}

	return nil
}

// generateMarkdownPage creates custom markdown output.
func generateMarkdownPage(cmd *cobra.Command, w io.Writer, dirPath string, weights map[string]int) error {
	var err error
	p := func(format string, a ...any) {
		_, err = w.Write([]byte(fmt.Sprintf(format, a...)))
	}
	id := websitePageID(cmd)
	p("\n")
	p("---\n")
	p("## %s {#%s}\n", cmd.CommandPath(), id)
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()
	if cmd.Version != "" {
		p("version `%s`\n\n", cmd.Version)
	}
	p(cmd.Short)
	p("\n\n")
	if cmd.Runnable() {
		p("### Usage {#%s-usage} \n", id)
		p("```terminal\n$ %s\n```\n\n", cmd.UseLine())
	}
	if len(cmd.Long) > 0 {
		p("### Description {#%s-description}\n\n", id)
		p("%s \n\n", escapeDescription(cmd.Long))
	}
	if len(cmd.Example) > 0 {
		p("### Examples {#%s-examples}\n\n", id)
		p("```\n%s\n```\n\n", escapeDescription(cmd.Example))
	}
	commandFlags := cmd.NonInheritedFlags()
	if commandFlags.HasAvailableFlags() {
		p("### Flags {#%s-flags}\n\n", id)
		if err := printFlags(cmd, commandFlags, w); err != nil {
			return err
		}
	}
	inheritedFlags := cmd.InheritedFlags()
	if inheritedFlags.HasAvailableFlags() {
		p("### Flags inherited from parent commands {#%s-persistent-flags}\n", id)
		if err := printFlags(cmd, inheritedFlags, w); err != nil {
			return err
		}
	}
	if hasSubCommands(cmd) {
		p("### Subcommands {#%s-subcommands}\n\n", id)
		children := cmd.Commands()
		orderCommands(weights, children)
		for _, child := range children {
			if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
				continue
			}
			childRelPath := child.Name() + ".md"
			if child.HasSubCommands() {
				childRelPath = filepath.Join(child.Name(), "index.md")
			}
			p("* [%s](./%s)\t - %s\n", child.CommandPath(), childRelPath, child.Short)
		}
		p("\n")
	}
	if cmd.HasParent() {
		p("### Parent Command {#%s-parent-command}\n\n", id)
		parent := cmd.Parent()
		parentName := parent.CommandPath()
		if hasSubCommands(cmd) {
			p("* [%s](../index.md)\t - %s\n", parentName, parent.Short)
		} else {
			p("* [%s](./index.md)\t - %s\n", parentName, parent.Short)
		}
		cmd.VisitParents(func(c *cobra.Command) {
			if c.DisableAutoGenTag {
				cmd.DisableAutoGenTag = c.DisableAutoGenTag
			}
		})
	}
	return err
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

func readConfig(filename string) (webpagesConfig, error) {
	if filename == "" {
		return webpagesConfig{}, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		return webpagesConfig{}, err
	}
	yamlBytes, err := io.ReadAll(file)
	if err != nil {
		return webpagesConfig{}, err
	}
	var cfg webpagesConfig
	if err := yaml.Unmarshal(yamlBytes, &cfg); err != nil {
		return webpagesConfig{}, err
	}
	return cfg, err
}

func orderCommands(weights map[string]int, commands []*cobra.Command) {
	sort.SliceStable(commands, func(i, j int) bool {
		return weights[commands[i].CommandPath()] < weights[commands[j].CommandPath()]
	})
}

func printFlags(cmd *cobra.Command, f *pflag.FlagSet, w io.Writer) error {
	var err error
	p := func(format string, a ...any) {
		_, err = w.Write([]byte(fmt.Sprintf(format, a...)))
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
		p(" {#%s-%s}", websitePageID(cmd), flag.Name)
		p("\n")
		p(usage)
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
