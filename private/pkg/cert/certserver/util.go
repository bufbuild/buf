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
	"os"
)

// newServerTLSConfigFromFiles creates a new tls.Config from a server certificate file and a server key file.
func newServerTLSConfigFromFiles(serverCertFile string, serverKeyFile string) (*tls.Config, error) {
	certPEMBlock, err := os.ReadFile(serverCertFile)
	if err != nil {
		return nil, err
	}
	keyPEMBlock, err := os.ReadFile(serverKeyFile)
	if err != nil {
		return nil, err
	}
	return newServerTLSConfigFromData(certPEMBlock, keyPEMBlock)
}

// newServerTLSConfigFromData creates a new tls.Config from server certificate data and server key data.
func newServerTLSConfigFromData(serverCertData []byte, serverKeyData []byte) (*tls.Config, error) {
	certificate, err := tls.X509KeyPair(serverCertData, serverKeyData)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{certificate},
	}, nil
}

// newServerSystemTLSConfig creates a new tls.Config that uses the system cert pool for verifying
// server certificates.
func newServerSystemTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		// An empty TLS config will use the system certificate pool
		// when verifying the servers certificate. This is because
		// not setting any RootCAs will set `x509.VerifyOptions.Roots`
		// to nil, which triggers the loading of system certs (including
		// on Windows somehow) within (*x509.Certificate).Verify.
		RootCAs: nil,
	}
}
