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

package appname

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/app"
)

type container struct {
	envContainer app.EnvContainer
	appName      string
}

func newContainer(envContainer app.EnvContainer, appName string) (*container, error) {
	if err := validateAppName(appName); err != nil {
		return nil, err
	}
	return &container{
		envContainer: envContainer,
		appName:      appName,
	}, nil
}

func (c *container) AppName() string {
	return c.appName
}

func (c *container) ConfigDirPath() string {
	return c.getDirPath("CONFIG_DIR", app.ConfigDirPath)
}

func (c *container) CacheDirPath() string {
	return c.getDirPath("CACHE_DIR", app.CacheDirPath)
}

func (c *container) DataDirPath() string {
	return c.getDirPath("DATA_DIR", app.DataDirPath)
}

func (c *container) Port() (uint16, error) {
	portString := c.envContainer.Env(getEnvPrefix(c.appName) + "PORT")
	if portString == "" {
		portString = c.envContainer.Env("PORT")
		if portString == "" {
			return 0, nil
		}
	}
	port, err := strconv.ParseUint(portString, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("could not parse port %q to uint16: %v", portString, err)
	}
	return uint16(port), nil
}

func (c *container) getDirPath(envSuffix string, getBaseDirPath func(app.EnvContainer) (string, error)) string {
	dirPath := c.envContainer.Env(getEnvPrefix(c.appName) + envSuffix)
	if dirPath == "" {
		baseDirPath, err := getBaseDirPath(c.envContainer)
		if err == nil {
			dirPath = filepath.Join(baseDirPath, c.appName)
		}
	}
	return dirPath
}

func getEnvPrefix(appName string) string {
	return strings.ToUpper(strings.ReplaceAll(appName, "-", "_")) + "_"
}
func validateAppName(appName string) error {
	if appName == "" {
		return errors.New("empty application name")
	}
	for _, c := range appName {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return fmt.Errorf("invalid application name: %s", appName)
		}
	}
	return nil
}
