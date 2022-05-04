// Copyright 2020-2022 Buf Technologies, Inc.
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

package certserver

import (
	"crypto/tls"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app/appname"
)

// ExternalServerTLSConfig allows users to configure TLS on the server side.
type ExternalServerTLSConfig struct {
	Use                string `json:"use,omitempty" yaml:"use,omitempty"`
	ServerCertFilePath string `json:"server_cert_file_path,omitempty" yaml:"server_cert_file_path,omitempty"`
	ServerKeyFilePath  string `json:"server_key_file_path,omitempty" yaml:"server_key_file_path,omitempty"`
}

// NewServerTLSConfig creates a new *tls.Config from the ExternalTLSConfig
//
// The default is to use a local TLS config at ${configDirPath}/tls/{cert,key}.pem.
func NewServerTLSConfig(
	container appname.Container,
	externalServerTLSConfig ExternalServerTLSConfig,
) (*tls.Config, error) {
	switch t := strings.ToLower(strings.TrimSpace(externalServerTLSConfig.Use)); t {
	case "local", "":
		serverCertFilePath := externalServerTLSConfig.ServerCertFilePath
		if serverCertFilePath == "" {
			serverCertFilePath = filepath.Join(container.ConfigDirPath(), "tls", "cert.pem")
		}
		serverKeyFilePath := externalServerTLSConfig.ServerKeyFilePath
		if serverKeyFilePath == "" {
			serverKeyFilePath = filepath.Join(container.ConfigDirPath(), "tls", "key.pem")
		}
		return newServerTLSConfigFromFiles(serverCertFilePath, serverKeyFilePath)
	case "system":
		return newServerSystemTLSConfig(), nil
	case "false":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown tls.use: %q", t)
	}
}
