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
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/tmp"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const (
	// bufCloneOrigin is the name for the remote. It helps distinguish the origin of
	// the repo we're cloning from the "origin" of our clone (which is the repo
	// being cloned).
	// We can fetch directly from an origin URL, but without any remote set git LFS
	// will fail to fetch so we need to pick something.
	bufCloneOrigin = "bufCloneOrigin"
)

type cloner struct {
	logger            *zap.Logger
	tracer            tracing.Tracer
	storageosProvider storageos.Provider
	runner            command.Runner
	options           ClonerOptions
}

func newCloner(
	logger *zap.Logger,
	tracer tracing.Tracer,
	storageosProvider storageos.Provider,
	runner command.Runner,
	options ClonerOptions,
) *cloner {
	return &cloner{
		logger:            logger,
		tracer:            tracer,
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
	ctx, span := c.tracer.Start(ctx, tracing.WithErr(&retErr))
	defer span.End()

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
		err := errors.New("depth must be > 0")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	depthArg := strconv.Itoa(int(depth))

	bareDir, err := tmp.NewDir()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, bareDir.Close())
	}()
	worktreeDir, err := tmp.NewDir()
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, worktreeDir.Close())
	}()

	buffer := bytes.NewBuffer(nil)
	if err := c.runner.Run(
		ctx,
		"git",
		command.RunWithArgs("init", "--bare"),
		command.RunWithEnv(app.EnvironMap(envContainer)),
		command.RunWithStderr(buffer),
		command.RunWithDir(bareDir.AbsPath()),
	); err != nil {
		return newGitCommandError(err, buffer, bareDir)
	}

	buffer.Reset()
	remoteArgs := []string{
		"--git-dir=" + bareDir.AbsPath(),
		"remote",
		"add",
		bufCloneOrigin,
		url,
	}
	if err := c.runner.Run(
		ctx,
		"git",
		command.RunWithArgs(remoteArgs...),
		command.RunWithEnv(app.EnvironMap(envContainer)),
		command.RunWithStderr(buffer),
	); err != nil {
		return newGitCommandError(err, buffer, bareDir)
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
	fetchRef, worktreeRef, checkoutRef := getRefspecsForName(options.Name)
	fetchArgs := append(
		gitConfigAuthArgs,
		"--git-dir="+bareDir.AbsPath(),
		"fetch",
		"--depth", depthArg,
		bufCloneOrigin,
		fetchRef,
	)

	if strings.HasPrefix(url, "ssh://") {
		envContainer, err = c.getEnvContainerWithGitSSHCommand(envContainer)
		if err != nil {
			return err
		}
	}
	buffer.Reset()
	if err := c.runner.Run(
		ctx,
		"git",
		command.RunWithArgs(fetchArgs...),
		command.RunWithEnv(app.EnvironMap(envContainer)),
		command.RunWithStderr(buffer),
	); err != nil {
		return newGitCommandError(err, buffer, bareDir)
	}

	buffer.Reset()
	args := append(
		gitConfigAuthArgs,
		"--git-dir="+bareDir.AbsPath(),
		"worktree",
		"add",
		worktreeDir.AbsPath(),
		worktreeRef,
	)
	if err := c.runner.Run(
		ctx,
		"git",
		command.RunWithArgs(args...),
		command.RunWithEnv(app.EnvironMap(envContainer)),
		command.RunWithStderr(buffer),
	); err != nil {
		return newGitCommandError(err, buffer, worktreeDir)
	}

	if checkoutRef != "" {
		buffer.Reset()
		args := append(
			gitConfigAuthArgs,
			"checkout",
			checkoutRef,
		)
		if err := c.runner.Run(
			ctx,
			"git",
			command.RunWithArgs(args...),
			command.RunWithEnv(app.EnvironMap(envContainer)),
			command.RunWithStderr(buffer),
			command.RunWithDir(worktreeDir.AbsPath()),
		); err != nil {
			return newGitCommandError(err, buffer, worktreeDir)
		}
	}

	if options.RecurseSubmodules {
		submoduleArgs := append(
			gitConfigAuthArgs,
			"submodule",
			"update",
			"--init",
			"--recursive",
			"--depth",
			depthArg,
		)
		buffer.Reset()
		if err := c.runner.Run(
			ctx,
			"git",
			command.RunWithArgs(submoduleArgs...),
			command.RunWithEnv(app.EnvironMap(envContainer)),
			command.RunWithStderr(buffer),
			command.RunWithDir(worktreeDir.AbsPath()),
		); err != nil {
			// Suppress printing of temp path
			return fmt.Errorf("%v\n%v", err, strings.Replace(buffer.String(), worktreeDir.AbsPath(), "", -1))
		}
	}

	// we do NOT want to read in symlinks
	tmpReadWriteBucket, err := c.storageosProvider.NewReadWriteBucket(worktreeDir.AbsPath())
	if err != nil {
		return err
	}
	var readBucket storage.ReadBucket = tmpReadWriteBucket
	if options.Mapper != nil {
		readBucket = storage.MapReadBucket(readBucket, options.Mapper)
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

// getRefspecsForName decides the refspecs to use in the subsequent git fetch,
// git worktree add and git checkout. When checkoutRefspec is empty, Name
// explicitly refer to a named ref and the checkout isn't a necessary step.
func getRefspecsForName(gitName Name) (fetchRefSpec string, worktreeRefSpec string, checkoutRefspec string) {
	if gitName == nil {
		return "HEAD", "FETCH_HEAD", ""
	}
	if gitName.cloneBranch() != "" && gitName.checkout() != "" {
		// When doing branch/tag clones, make sure we use a
		// refspec that creates a local referece in `refs/` even if the ref
		// is remote tracking, so that the checkoutRefs may reference it,
		// for example:
		//   branch=origin/main,ref=origin/main~1
		fetchRefSpec := gitName.cloneBranch() + ":" + gitName.cloneBranch()
		return fetchRefSpec, "FETCH_HEAD", gitName.checkout()
	} else if gitName.cloneBranch() != "" {
		return gitName.cloneBranch(), "FETCH_HEAD", ""
	} else if gitName.checkout() != "" {
		// After fetch we won't have checked out any refs. This
		// will cause `refs=` containing "HEAD" to fail, as HEAD
		// is a special case that is not fetched into a ref but
		// instead refers to the current commit checked out. By
		// checking out "FETCH_HEAD" before checking out the
		// user supplied ref, we behave similarly to git clone.
		return "HEAD", "FETCH_HEAD", gitName.checkout()
	} else {
		return "HEAD", "FETCH_HEAD", ""
	}
}

func newGitCommandError(
	err error,
	buffer *bytes.Buffer,
	tmpDir tmp.Dir,
) error {
	// Suppress printing of temp path
	return fmt.Errorf("%v\n%v", err, strings.TrimSpace(strings.Replace(buffer.String(), tmpDir.AbsPath(), "", -1)))
}
