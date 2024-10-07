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

package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/slogext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/tmp"
	"go.uber.org/multierr"
)

type cloner struct {
	logger            *slog.Logger
	storageosProvider storageos.Provider
	runner            command.Runner
	options           ClonerOptions
}

func newCloner(
	logger *slog.Logger,
	storageosProvider storageos.Provider,
	runner command.Runner,
	options ClonerOptions,
) *cloner {
	return &cloner{
		logger:            logger,
		storageosProvider: storageosProvider,
		runner:            runner,
		options:           options,
	}
}

func (c *cloner) CloneToBucket(
	ctx context.Context,
	envContainer app.EnvContainer,
	url string,
	depth uint32,
	writeBucket storage.WriteBucket,
	options CloneToBucketOptions,
) (retErr error) {
	defer slogext.DebugProfile(c.logger)()

	var err error
	switch {
	case strings.HasPrefix(url, "http://"),
		strings.HasPrefix(url, "https://"),
		strings.HasPrefix(url, "ssh://"),
		strings.HasPrefix(url, "git://"),
		strings.HasPrefix(url, "file://"):
	default:
		return fmt.Errorf("invalid git url: %q", url)
	}

	if depth == 0 {
		return errors.New("depth must be > 0")
	}

	depthArg := strconv.Itoa(int(depth))

	baseDir, err := tmp.NewDir(ctx)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, baseDir.Close())
	}()

	buffer := bytes.NewBuffer(nil)
	if err := c.runner.Run(
		ctx,
		"git",
		command.RunWithArgs("init"),
		command.RunWithEnv(app.EnvironMap(envContainer)),
		command.RunWithStderr(buffer),
		command.RunWithDir(baseDir.Path()),
	); err != nil {
		return newGitCommandError(err, buffer)
	}

	buffer.Reset()
	if err := c.runner.Run(
		ctx,
		"git",
		command.RunWithArgs("remote", "add", "origin", url),
		command.RunWithEnv(app.EnvironMap(envContainer)),
		command.RunWithStderr(buffer),
		command.RunWithDir(baseDir.Path()),
	); err != nil {
		return newGitCommandError(err, buffer)
	}

	var gitConfigAuthArgs []string
	if strings.HasPrefix(url, "https://") {
		// These extraArgs MUST be first, as the -c flag potentially produced
		// is only a flag on the parent git command, not on git fetch.
		extraArgs, err := c.getArgsForHTTPSCommand(envContainer)
		if err != nil {
			return err
		}
		gitConfigAuthArgs = append(gitConfigAuthArgs, extraArgs...)
	}

	if strings.HasPrefix(url, "ssh://") {
		envContainer, err = c.getEnvContainerWithGitSSHCommand(envContainer)
		if err != nil {
			return err
		}
	}
	// First, try to fetch the fetchRef directly. If the ref is not found, we
	// will try to fetch the fallback ref with a depth to allow resolving partial
	// refs locally. If the fetch fails, we will return an error.
	var usedFallback bool
	fetchRef, fallbackRef, checkoutRef := getRefspecsForName(options.Name)
	buffer.Reset()
	if err := c.runner.Run(
		ctx,
		"git",
		command.RunWithArgs(append(
			gitConfigAuthArgs,
			"fetch",
			"--depth", depthArg,
			"--update-head-ok", // Required on branches matching the current branch of git init.
			"origin",
			fetchRef,
		)...),
		command.RunWithEnv(app.EnvironMap(envContainer)),
		command.RunWithStderr(buffer),
		command.RunWithDir(baseDir.Path()),
	); err != nil {
		// If the ref fetch failed, without a fallback, return the error.
		if fallbackRef == "" {
			return newGitCommandError(err, buffer)
		}
		// Failed to fetch the ref directly, try to fetch the fallback ref.
		usedFallback = true
		buffer.Reset()
		if err := c.runner.Run(
			ctx,
			"git",
			command.RunWithArgs(append(
				gitConfigAuthArgs,
				"fetch",
				"--depth", depthArg,
				"--update-head-ok", // Required on branches matching the current branch of git init.
				"origin",
				fallbackRef,
			)...),
			command.RunWithEnv(app.EnvironMap(envContainer)),
			command.RunWithStderr(buffer),
			command.RunWithDir(baseDir.Path()),
		); err != nil {
			return newGitCommandError(err, buffer)
		}
	}

	// Always checkout the FETCH_HEAD to populate the working directory.
	// This allows for referencing HEAD in checkouts.
	buffer.Reset()
	if err := c.runner.Run(
		ctx,
		"git",
		command.RunWithArgs("checkout", "--force", "FETCH_HEAD"),
		command.RunWithEnv(app.EnvironMap(envContainer)),
		command.RunWithStderr(buffer),
		command.RunWithDir(baseDir.Path()),
	); err != nil {
		return newGitCommandError(err, buffer)
	}
	// Should checkout if the fallback was used or if the checkout ref is different
	// from the fetch ref.
	if checkoutRef != "" && (usedFallback || checkoutRef != fetchRef) {
		buffer.Reset()
		if err := c.runner.Run(
			ctx,
			"git",
			command.RunWithArgs("checkout", "--force", checkoutRef),
			command.RunWithEnv(app.EnvironMap(envContainer)),
			command.RunWithStderr(buffer),
			command.RunWithDir(baseDir.Path()),
		); err != nil {
			return newGitCommandError(err, buffer)
		}
	}

	if options.RecurseSubmodules {
		buffer.Reset()
		if err := c.runner.Run(
			ctx,
			"git",
			command.RunWithArgs(append(
				gitConfigAuthArgs,
				"submodule",
				"update",
				"--init",
				"--recursive",
				"--force",
				"--depth",
				depthArg,
			)...),
			command.RunWithEnv(app.EnvironMap(envContainer)),
			command.RunWithStderr(buffer),
			command.RunWithDir(baseDir.Path()),
		); err != nil {
			return newGitCommandError(err, buffer)
		}
	}

	// we do NOT want to read in symlinks
	tmpReadWriteBucket, err := c.storageosProvider.NewReadWriteBucket(baseDir.Path())
	if err != nil {
		return err
	}
	var readBucket storage.ReadBucket = tmpReadWriteBucket
	if options.Matcher != nil {
		readBucket = storage.FilterReadBucket(readBucket, options.Matcher)
	}
	_, err = storage.Copy(ctx, readBucket, writeBucket)
	return err
}

func (c *cloner) getArgsForHTTPSCommand(envContainer app.EnvContainer) ([]string, error) {
	if c.options.HTTPSUsernameEnvKey == "" || c.options.HTTPSPasswordEnvKey == "" {
		return nil, nil
	}
	httpsUsernameSet := envContainer.Env(c.options.HTTPSUsernameEnvKey) != ""
	httpsPasswordSet := envContainer.Env(c.options.HTTPSPasswordEnvKey) != ""
	if !httpsUsernameSet {
		if httpsPasswordSet {
			return nil, fmt.Errorf("%s set but %s not set", c.options.HTTPSPasswordEnvKey, c.options.HTTPSUsernameEnvKey)
		}
		return nil, nil
	}
	c.logger.Debug("git_credential_helper_override")
	return []string{
		"-c",
		fmt.Sprintf(
			// TODO: is this OK for windows/other platforms?
			// we might need an alternate flow where the binary has a sub-command to do this, and calls itself
			//
			// putting the variable name in this script, NOT the actual variable value
			// we do not want to store the variable on disk, ever
			// this is especially important if the program dies
			// note that this means i.e. HTTPS_PASSWORD=foo invoke_program does not work as
			// this variable needs to be in the actual global environment
			// TODO this is a mess
			"credential.helper=!f(){ echo username=${%s}; echo password=${%s}; };f",
			c.options.HTTPSUsernameEnvKey,
			c.options.HTTPSPasswordEnvKey,
		),
	}, nil
}

func (c *cloner) getEnvContainerWithGitSSHCommand(envContainer app.EnvContainer) (app.EnvContainer, error) {
	gitSSHCommand, err := c.getGitSSHCommand(envContainer)
	if err != nil {
		return nil, err
	}
	if gitSSHCommand != "" {
		c.logger.Debug("git_ssh_command_override")
		return app.NewEnvContainerWithOverrides(
			envContainer,
			map[string]string{
				"GIT_SSH_COMMAND": gitSSHCommand,
			},
		), nil
	}
	return envContainer, nil
}

func (c *cloner) getGitSSHCommand(envContainer app.EnvContainer) (string, error) {
	sshKeyFilePath := envContainer.Env(c.options.SSHKeyFileEnvKey)
	sshKnownHostsFiles := envContainer.Env(c.options.SSHKnownHostsFilesEnvKey)
	if sshKeyFilePath == "" {
		if sshKnownHostsFiles != "" {
			return "", fmt.Errorf("%s set but %s not set", c.options.SSHKnownHostsFilesEnvKey, c.options.SSHKeyFileEnvKey)
		}
		return "", nil
	}
	if sshKnownHostsFilePaths := getSSHKnownHostsFilePaths(sshKnownHostsFiles); len(sshKnownHostsFilePaths) > 0 {
		return fmt.Sprintf(
			`ssh -q -i "%s" -o "IdentitiesOnly=yes" -o "UserKnownHostsFile=%s"`,
			sshKeyFilePath,
			strings.Join(sshKnownHostsFilePaths, " "),
		), nil
	}
	// we want to set StrictHostKeyChecking=no because the SSH key file variable was set, so
	// there is an ask to override the default ssh settings here
	return fmt.Sprintf(
		`ssh -q -i "%s" -o "IdentitiesOnly=yes" -o "UserKnownHostsFile=%s" -o "StrictHostKeyChecking=no"`,
		sshKeyFilePath,
		app.DevNullFilePath,
	), nil
}

func getSSHKnownHostsFilePaths(sshKnownHostsFiles string) []string {
	if sshKnownHostsFiles == "" {
		return nil
	}
	var filePaths []string
	for _, filePath := range strings.Split(sshKnownHostsFiles, ":") {
		filePath = strings.TrimSpace(filePath)
		if filePath != "" {
			filePaths = append(filePaths, filePath)
		}
	}
	return filePaths
}

// getRefspecsForName returns the refs to fetch and checkout. A fallback ref is
// used for partial refs. If the first fetch fails, the fallback ref is fetched
// to allow resolving partial refs locally. The checkout ref is the ref to
// checkout after the fetch.
func getRefspecsForName(gitName Name) (fetchRef string, fallbackRef string, checkoutRef string) {
	// Default to fetching HEAD and checking out FETCH_HEAD.
	if gitName == nil {
		return "HEAD", "", ""
	}
	checkout, cloneBranch := gitName.checkout(), gitName.cloneBranch()
	if checkout != "" && cloneBranch != "" {
		// If a branch, tag, or commit is specified, we fetch the ref directly.
		return createFetchRefSpec(cloneBranch), "", checkout
	} else if cloneBranch != "" {
		// If a branch is specified, we fetch the branch directly.
		return cloneBranch, "", ""
	} else if checkout != "" && checkout != "HEAD" {
		// If a checkout ref is specified, we fetch the ref directly.
		// We fallback to fetching the HEAD to resolve partial refs.
		// We checkout the ref after the fetch if the fallback was used.
		return checkout, "HEAD", checkout
	}
	return "HEAD", "", ""
}

// createFetchRefSpec create a refspec to ensure a local reference is created
// when fetching a branch or tag. This allows to checkout the ref with
// `git checkout` even if the ref is remote tracking. For example:
//
//	+origin/main:origin/main
func createFetchRefSpec(fetchRef string) string {
	return "+" + fetchRef + ":" + fetchRef
}

func newGitCommandError(err error, buffer *bytes.Buffer) error {
	return fmt.Errorf("%v\n%v", err, buffer.String())
}
