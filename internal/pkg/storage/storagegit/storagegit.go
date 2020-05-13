// Copyright 2020 Buf Technologies Inc.
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

// Package storagegit implements git utilities.
//
// This uses https://github.com/go-git/go-git.
package storagegit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/bufbuild/buf/internal/pkg/instrument"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagegit/storagegitplumbing"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/storage/storageutil"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	srcdssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/gofrs/uuid"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh/knownhosts"
)

var gitURLSSHRegex = regexp.MustCompile("^(ssh://)?([^/:]*?)@[^@]+$")

// ExperimentalClone clones the url into the bucket.
//
// This calls git clone --branch [branch/tag] --single-branch --depth 1 [--recurse-submodules]
// Only regular files are added to the bucket.
//
// Branch is required.
//
// Only use for local CLI checking.
func ExperimentalClone(
	ctx context.Context,
	logger *zap.Logger,
	environ []string,
	path string,
	branch string,
	tag string,
	recurseSubmodules bool,
	readWriteBucket storage.ReadWriteBucket,
	options ...normalpath.TransformerOption,
) (retErr error) {
	defer instrument.Start(logger, "git_experimental_clone").End()
	var err error
	switch {
	case strings.HasPrefix(path, "http://"),
		strings.HasPrefix(path, "https://"),
		strings.HasPrefix(path, "ssh://"),
		strings.HasPrefix(path, "file://"):
	case !strings.Contains(path, "//"):
		path, err = filepath.Abs(filepath.Clean(path))
		if err != nil {
			return err
		}
		path = "file://" + path
	default:
		return fmt.Errorf("invalid git path: %q", path)
	}
	args := []string{"clone", "--depth", "1"}
	if branch == "" {
		branch = tag
	}
	if branch == "" {
		return errors.New("must set branch or tag")
	}
	args = append(args, "--branch", branch, "--single-branch")
	if recurseSubmodules {
		args = append(args, "--recurse-submodules")
	}
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	tempDirPath, err := ioutil.TempDir("", id.String())
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, os.RemoveAll(tempDirPath))
	}()
	// just in case
	absTempDirPath, err := filepath.Abs(filepath.Clean(tempDirPath))
	if err != nil {
		return err
	}
	args = append(args, path, absTempDirPath)
	buffer := bytes.NewBuffer(nil)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = environ
	cmd.Stderr = buffer
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v\n%v", err, strings.Replace(buffer.String(), absTempDirPath, "", -1))
	}
	tempReadBucketCloser, err := storageos.NewReadBucketCloser(absTempDirPath)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, tempReadBucketCloser.Close())
	}()
	defer instrument.Start(logger, "git_experimental_clone_copy").End()
	_, err = storageutil.Copy(ctx, tempReadBucketCloser, readWriteBucket, "", options...)
	return err
}

// Clone clones the url into the bucket.
//
// This is roughly equivalent to git clone --branch gitBranch --single-branch --depth 1 gitUrl.
// Only regular files are added to the bucket.
//
// Branch is required.
//
// If the gitURL begins with https:// and there is an HTTPS username and password, basic auth will be used.
// If the gitURL begins with ssh:// and there is a valid SSH configuration, ssh will be used.
//
// This really needs more testing and cleanup.
// Only use for local CLI checking.
func Clone(
	ctx context.Context,
	logger *zap.Logger,
	getenv func(string) string,
	homeDirPath string,
	gitURL string,
	refName storagegitplumbing.RefName,
	recurseSubmodules bool,
	httpsUsernameEnvKey string,
	httpsPasswordEnvKey string,
	sshKeyFileEnvKey string,
	sshKeyPassphraseEnvKey string,
	sshKnownHostsFilesEnvKey string,
	readWriteBucket storage.ReadWriteBucket,
	options ...normalpath.TransformerOption,
) error {
	defer instrument.Start(logger, "git_clone").End()

	if refName == nil {
		// we detect this outside of this function so this is a system error
		return errors.New("refName is nil")
	}
	gitURL, err := normalizeGitURL(gitURL)
	if err != nil {
		return err
	}
	authMethod, err := getAuthMethod(
		logger,
		getenv,
		homeDirPath,
		gitURL,
		httpsUsernameEnvKey,
		httpsPasswordEnvKey,
		sshKeyFileEnvKey,
		sshKeyPassphraseEnvKey,
		sshKnownHostsFilesEnvKey,
	)
	if err != nil {
		return err
	}
	submoduleRecursivity := git.NoRecurseSubmodules
	if recurseSubmodules {
		// TODO: this only does 10 depth, compare to git
		submoduleRecursivity = git.DefaultSubmoduleRecursionDepth
	}
	cloneOptions := &git.CloneOptions{
		URL:               gitURL,
		Auth:              authMethod,
		ReferenceName:     refName.ReferenceName(),
		RecurseSubmodules: submoduleRecursivity,
		SingleBranch:      true,
		Depth:             1,
	}
	filesystem := memfs.New()
	if _, err := git.CloneContext(ctx, memory.NewStorage(), filesystem, cloneOptions); err != nil {
		return err
	}
	return copyBillyFilesystemToBucket(ctx, logger, filesystem, readWriteBucket, options...)
}

func normalizeGitURL(gitURL string) (string, error) {
	switch {
	case isHTTPGitURL(gitURL), isHTTPSGitURL(gitURL), isSSHGitURL(gitURL), isFileGitURL(gitURL):
		return gitURL, nil
	case isLocalFileGitURL(gitURL):
		absGitPath, err := filepath.Abs(gitURL)
		if err != nil {
			return "", err
		}
		return "file://" + absGitPath, nil
	default:
		return "", fmt.Errorf("invalid git url: %v", gitURL)
	}
}

func isHTTPGitURL(gitURL string) bool {
	return strings.HasPrefix(gitURL, "http://")
}
func isHTTPSGitURL(gitURL string) bool {
	return strings.HasPrefix(gitURL, "https://")
}

func isSSHGitURL(gitURL string) bool {
	_, ok := getSSHGitUser(gitURL)
	return ok
}

func isFileGitURL(gitURL string) bool {
	return strings.HasPrefix(gitURL, "file://")
}

func isLocalFileGitURL(gitURL string) bool {
	return !strings.Contains(gitURL, "://")
}

func getSSHGitUser(gitURL string) (string, bool) {
	if matches := gitURLSSHRegex.FindStringSubmatch(gitURL); len(matches) > 2 {
		return matches[2], true
	}
	return "", false
}

func getAuthMethod(
	logger *zap.Logger,
	getenv func(string) string,
	homeDirPath string,
	gitURL string,
	httpsUsernameEnvKey string,
	httpsPasswordEnvKey string,
	sshKeyFileEnvKey string,
	sshKeyPassphraseEnvKey string,
	sshKnownHostsFilesEnvKey string,
) (transport.AuthMethod, error) {
	if isHTTPSGitURL(gitURL) {
		if getenv == nil || httpsUsernameEnvKey == "" || httpsPasswordEnvKey == "" {
			return nil, nil
		}
		httpsUsername := getenv(httpsUsernameEnvKey)
		httpsPassword := getenv(httpsPasswordEnvKey)
		if httpsUsername != "" && httpsPassword != "" {
			logger.Debug("git_https_basic_auth_enabled")
			return &http.BasicAuth{
				Username: httpsUsername,
				Password: httpsPassword,
			}, nil
		}
		return nil, nil
	}
	if sshUser, ok := getSSHGitUser(gitURL); ok {
		var sshKeyFile string
		if getenv != nil && sshKeyFileEnvKey != "" {
			sshKeyFile = getenv(sshKeyFileEnvKey)
		}
		if sshKeyFile == "" && homeDirPath != "" {
			sshKeyFile = filepath.Join(homeDirPath, ".ssh", "id_rsa")
		}
		if sshKeyFile == "" {
			return nil, errors.New("cannot set up ssh auth")
		}
		sshKeyData, err := ioutil.ReadFile(sshKeyFile)
		if err != nil {
			return nil, err
		}
		var sshKeyPassphrase string
		if getenv != nil && sshKeyPassphraseEnvKey != "" {
			sshKeyPassphrase = getenv(sshKeyPassphraseEnvKey)
		}
		publicKeys, err := srcdssh.NewPublicKeys(sshUser, sshKeyData, sshKeyPassphrase)
		if err != nil {
			return nil, err
		}
		var knownHostsFilePaths []string
		if getenv != nil && sshKnownHostsFilesEnvKey != "" {
			knownHostsFilePaths = filepath.SplitList(getenv(sshKnownHostsFilesEnvKey))
		}
		if len(knownHostsFilePaths) == 0 && homeDirPath != "" {
			knownHostsFilePaths = []string{
				filepath.Join(homeDirPath, ".ssh", "known_hosts"),
				filepath.Join(string(os.PathSeparator), "etc", "ssh", "ssh_known_hosts"),
			}
		}
		knownHostsFilePaths, err = filterKnownHostsFilePaths(knownHostsFilePaths)
		if err != nil {
			return nil, err
		}
		if len(knownHostsFilePaths) == 0 {
			return nil, fmt.Errorf("cannot find ssh known_hosts at $%s, ~/.ssh/known_hosts, or /etc/ssh/ssh_known_hosts", sshKnownHostsFilesEnvKey)
		}
		hostKeyCallback, err := knownhosts.New(knownHostsFilePaths...)
		if err != nil {
			return nil, err
		}
		publicKeys.HostKeyCallback = hostKeyCallback
		logger.Debug("git_ssh_public_key_auth_enabled")
		return publicKeys, nil
	}
	return nil, nil
}

func filterKnownHostsFilePaths(knownHostsFilePaths []string) ([]string, error) {
	var out []string
	for _, knownHostsFilePath := range knownHostsFilePaths {
		_, err := os.Stat(knownHostsFilePath)
		if err == nil {
			out = append(out, knownHostsFilePath)
			continue
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return out, nil
}

func copyBillyFilesystemToBucket(
	ctx context.Context,
	logger *zap.Logger,
	filesystem billy.Filesystem,
	readWriteBucket storage.ReadWriteBucket,
	options ...normalpath.TransformerOption,
) error {
	defer instrument.Start(logger, "git_clone_copy").End()

	transformer := normalpath.NewTransformer(options...)
	semaphoreC := make(chan struct{}, runtime.NumCPU())
	var retErr error
	var wg sync.WaitGroup
	var lock sync.Mutex
	if walkErr := walkBillyFilesystemDir(
		ctx,
		filesystem,
		func(regularFilePath string, regularFileSize uint32) error {
			if regularFilePath == "" || regularFilePath[0] != '/' {
				return fmt.Errorf("invalid regularFilePath: %q", regularFilePath)
			}
			// just to make sure
			path, err := normalpath.NormalizeAndValidate(regularFilePath[1:])
			if err != nil {
				return err
			}
			path, ok := transformer.Transform(path)
			if !ok {
				return nil
			}
			wg.Add(1)
			semaphoreC <- struct{}{}
			go func() {
				err := copyBillyPath(ctx, filesystem, readWriteBucket, regularFilePath, regularFileSize, path)
				lock.Lock()
				retErr = multierr.Append(retErr, err)
				lock.Unlock()
				<-semaphoreC
				wg.Done()
			}()
			return nil
		},
		"/",
	); walkErr != nil {
		return walkErr
	}
	wg.Wait()
	return retErr
}

func copyBillyPath(
	ctx context.Context,
	from billy.Filesystem,
	to storage.ReadWriteBucket,
	fromPath string,
	fromSize uint32,
	toPath string,
) error {
	file, err := from.Open(fromPath)
	if err != nil {
		return err
	}
	writeObject, err := to.Put(ctx, toPath, fromSize)
	if err != nil {
		return multierr.Append(err, file.Close())
	}
	_, err = io.Copy(writeObject, file)
	return multierr.Append(err, multierr.Append(writeObject.Close(), file.Close()))
}

func walkBillyFilesystemDir(
	ctx context.Context,
	filesystem billy.Filesystem,
	// regularFilePath will be the billy filesystem path
	f func(regularFilePath string, regularFileSize uint32) error,
	dirPath string,
) error {
	if dirPath == "" || dirPath[0] != '/' {
		return fmt.Errorf("invalid dirPath: %q", dirPath)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	fileInfos, err := filesystem.ReadDir(dirPath)
	if err != nil {
		return err
	}
	for _, fileInfo := range fileInfos {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		name := fileInfo.Name()
		if name == "" || name[0] == '/' {
			return fmt.Errorf("invalid name: %q", name)
		}
		if fileInfo.Mode().IsRegular() {
			size := fileInfo.Size()
			if size > math.MaxUint32 {
				return fmt.Errorf("size %d is greater than uint32", size)
			}
			// TODO: check to make sure normalization matches up with billy package
			if err := f(normalpath.Join(dirPath, name), uint32(size)); err != nil {
				return err
			}
		}
		if fileInfo.Mode().IsDir() {
			// TODO: check to make sure normalization matches up with billy package
			if err := walkBillyFilesystemDir(ctx, filesystem, f, normalpath.Join(dirPath, name)); err != nil {
				return err
			}
		}
	}
	return nil
}
