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

package certclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

type tlsOptions struct {
	useSystemCerts    bool
	rootCertFilePaths []string
}

// TLSOption is an option for a new TLS Config.
type TLSOption func(*tlsOptions)

// WithSystemCertPool returns a new TLSOption to use the system
// certificates. By default, no system certificates are used.
func WithSystemCertPool() TLSOption {
	return func(opts *tlsOptions) {
		opts.useSystemCerts = true
	}
}

// WithRootCertFilePaths returns a new TLSOption to trust the
// specified root CAs at the given paths.
func WithRootCertFilePaths(rootCertFilePaths ...string) TLSOption {
	return func(opts *tlsOptions) {
		opts.rootCertFilePaths = append(opts.rootCertFilePaths, rootCertFilePaths...)
	}
}

// NewClientTLScreates a new tls.Config from a root certificate files.
func NewClientTLS(options ...TLSOption) (*tls.Config, error) {
	opts := &tlsOptions{}
	for _, opt := range options {
		opt(opts)
	}
	rootCertDatas := make([][]byte, len(opts.rootCertFilePaths))
	for i, rootCertFilePath := range opts.rootCertFilePaths {
		rootCertData, err := os.ReadFile(rootCertFilePath)
		if err != nil {
			return nil, err
		}
		rootCertDatas[i] = rootCertData
	}
	return newClientTLSConfigFromRootCertDatas(opts.useSystemCerts, rootCertDatas...)
}

// newClientTLSConfigFromRootCertDatas creates a new tls.Config from root certificate datas.
func newClientTLSConfigFromRootCertDatas(useSystemCerts bool, rootCertDatas ...[]byte) (*tls.Config, error) {
	var certPool *x509.CertPool
	if useSystemCerts {
		var err error
		certPool, err = x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("failed to acquire system cert pool: %w", err)
		}
	} else {
		certPool = x509.NewCertPool()
	}
	for _, rootCertData := range rootCertDatas {
		if !certPool.AppendCertsFromPEM(rootCertData) {
			return nil, errors.New("failed to append root certificate")
		}
	}
	return newClientTLSConfigFromRootCertPool(certPool), nil
}

// newClientTLSConfigFromRootCertPool creates a new tls.Config from a root certificate pool.
func newClientTLSConfigFromRootCertPool(certPool *x509.CertPool) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    certPool,
	}
}
