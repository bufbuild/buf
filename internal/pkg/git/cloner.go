// Copyright 2020 Buf Technologies, Inc.
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
	"os/exec"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/instrument"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/tmp"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type cloner struct {
	logger  *zap.Logger
	options ClonerOptions
}

func newCloner(logger *zap.Logger, options ClonerOptions) *cloner {
	return &cloner{
		logger:  logger,
		options: options,
	}
}

func (c *cloner) CloneToBucket(
	ctx context.Context,
	envContainer app.EnvContainer,
	url string,
	refName RefName,
	readWriteBucket storage.ReadWriteBucket,
	options CloneToBucketOptions,
) (retErr error) {
	defer instrument.Start(c.logger, "git_clone_to_bucket").End()

	var err error
	switch {
	case strings.HasPrefix(url, "http://"),
		strings.HasPrefix(url, "https://"),
		strings.HasPrefix(url, "ssh://"),
		strings.HasPrefix(url, "file://"):
	default:
		return fmt.Errorf("invalid git url: %q", url)
	}
	args := []string{"clone", "--depth", "1"}
	branch := refName.branch()
	if branch == "" {
		return errors.New("must set git repository branch or tag")
	}
	args = append(args, "--branch", branch, "--single-branch")
	if options.RecurseSubmodules {
		args = append(args, "--recurse-submodules")
	}
	tmpDir, err := tmp.NewDir()
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, tmpDir.Close())
	}()
	args = append(args, url, tmpDir.AbsPath())

	if strings.HasPrefix(url, "https://") {
		extraArgs, err := c.getArgsForHTTPSCommand(envContainer)
		if err != nil {
			return err
		}
		args = append(args, extraArgs...)
	}
	if strings.HasPrefix(url, "ssh://") {
		envContainer, err = c.getEnvContainerWithGitSSHCommand(envContainer)
		if err != nil {
			return err
		}
	}

	buffer := bytes.NewBuffer(nil)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = app.Environ(envContainer)
	cmd.Stderr = buffer
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v\n%v", err, strings.Replace(buffer.String(), tmpDir.AbsPath(), "", -1))
	}
	tmpReadBucketCloser, err := storageos.NewReadBucketCloser(tmpDir.AbsPath())
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, tmpReadBucketCloser.Close())
	}()

	defer instrument.Start(c.logger, "git_clone_to_bucket_copy").End()
	_, err = storage.Copy(ctx, tmpReadBucketCloser, readWriteBucket, options.TransformerOptions...)
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
		"--config",
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
