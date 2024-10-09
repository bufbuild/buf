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

package webpages

import (
	"fmt"
	"regexp"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	webpagesConfigFlagName     = "config"
	includeFrontMatterFlagName = "include-front-matter"

	indexFileName         = "index.md"
	markdownFileExtension = ".md"
)

var codeBlockRegex = regexp.MustCompile(`(^\s\s\s\s)|(^\t)`)

// AddWebpagesCommand takes a cobra command and adds a webpages subcommand use to generate
// markdown documentation for the given command.
func AddWebpagesCommand(
	rootCobraCommand *cobra.Command,
) error {
	rootCobraCommand.AddCommand(newCommand(rootCobraCommand))
	return nil
}

func newCommand(
	rootCobraCommand *cobra.Command,
) *cobra.Command {
	command := &cobra.Command{
		Use:    "webpages",
		Hidden: true,
		Short:  "Generate markdown files for CLI reference documentation.",
		Long: fmt.Sprintf(`Generate markdown files for CLI reference documentation.

By default, this generates markdown pages with the command name as a H1 title. For markdown
files with Docusaurus compatible front matter, use --%s flag.`,
			includeFrontMatterFlagName,
		),
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return run(command.Flags(), rootCobraCommand)
		},
	}
	command.Flags().String(
		webpagesConfigFlagName,
		"",
		"Path to config file to use",
	)
	command.Flags().Bool(
		includeFrontMatterFlagName,
		false,
		"Include Docusaurus compatible front matter in generated markdown.",
	)
	return command
}

func run(
	flags *pflag.FlagSet,
	rootCobraCommand *cobra.Command,
) error {
	configPath, err := flags.GetString(webpagesConfigFlagName)
	if err != nil {
		return err
	}
	// TODO: rework this to be flags, no more config files
	config, err := readConfigFromFile(configPath)
	if err != nil {
		return err
	}

	excludes := slicesext.ToStructMap(config.ExcludeCommands)
	for _, command := range rootCobraCommand.Commands() {
		if _, ok := excludes[command.CommandPath()]; ok {
			command.Hidden = true
		}
	}
	includeFrontMatter, err := flags.GetBool(includeFrontMatterFlagName)
	if err != nil {
		return err
	}
	return generateMarkdownTree(
		rootCobraCommand,
		config,
		config.OutputDir,
		includeFrontMatter,
	)
}
