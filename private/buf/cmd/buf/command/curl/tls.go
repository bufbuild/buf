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

package curl

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"strings"
	"time"

	"github.com/bufbuild/buf/private/pkg/verbose"
)

func makeTLSConfig(f *flags, authority string, printer verbose.Printer) (*tls.Config, error) {
	var conf tls.Config
	// we verify manually so that we can emit verbose output while doing so
	conf.InsecureSkipVerify = true
	conf.VerifyConnection = func(state tls.ConnectionState) error {
		printer.Printf("* TLS connection using %s / %s", versionName(state.Version), tls.CipherSuiteName(state.CipherSuite))
		if state.DidResume {
			printer.Printf("* (TLS session resumed)")
		}
		if state.NegotiatedProtocol != "" {
			printer.Printf("* ALPN, server accepted protocol %s", state.NegotiatedProtocol)
		}
		printer.Printf("* Server certificate:")
		printer.Printf("*   subject: %s", state.PeerCertificates[0].Subject.String())
		printer.Printf("*   start date: %s", state.PeerCertificates[0].NotBefore)
		printer.Printf("*   end date: %s", state.PeerCertificates[0].NotAfter)
		var subjectAlternatives []string
		subjectAlternatives = append(subjectAlternatives, state.PeerCertificates[0].DNSNames...)
		for _, ip := range state.PeerCertificates[0].IPAddresses {
			subjectAlternatives = append(subjectAlternatives, ip.String())
		}
		subjectAlternatives = append(subjectAlternatives, state.PeerCertificates[0].EmailAddresses...)
		for _, uri := range state.PeerCertificates[0].URIs {
			subjectAlternatives = append(subjectAlternatives, uri.String())
		}
		printer.Printf("*   subjectAltNames: [%s]", strings.Join(subjectAlternatives, ", "))
		printer.Printf("*   issuer: %s", state.PeerCertificates[0].Issuer.String())

		// now we do verification
		if !f.Insecure {
			opts := x509.VerifyOptions{
				Roots:         conf.RootCAs,
				CurrentTime:   time.Now(),
				Intermediates: x509.NewCertPool(),
			}
			for _, cert := range state.PeerCertificates[1:] {
				opts.Intermediates.AddCert(cert)
			}
			if _, err := state.PeerCertificates[0].Verify(opts); err != nil {
				printer.Printf("* Server certificate chain could not be verified: %v", err)
				return err
			}
			printer.Printf("* Server certificate chain verified")
			if err := state.PeerCertificates[0].VerifyHostname(conf.ServerName); err != nil {
				printer.Printf("* Server certificate is not valid for %s: %v", conf.ServerName, err)
				return err
			}
			printer.Printf("* Server certificate is valid for %s", conf.ServerName)
		}
		return nil
	}
	if f.ServerName != "" {
		conf.ServerName = f.ServerName
	} else if authority != "" {
		// strip port if present
		host, _, err := net.SplitHostPort(authority)
		if err == nil {
			authority = host
		}
		conf.ServerName = authority
	}

	if f.CACert != "" {
		caCert, err := os.ReadFile(f.CACert)
		if err != nil {
			return nil, errorHasFilename(err, f.CACert)
		}
		conf.RootCAs = x509.NewCertPool()
		conf.RootCAs.AppendCertsFromPEM(caCert)
	}

	if f.Key != "" && f.Cert != "" {
		cert, err := os.ReadFile(f.Cert)
		if err != nil {
			return nil, errorHasFilename(err, f.Cert)
		}
		key, err := os.ReadFile(f.Key)
		if err != nil {
			return nil, errorHasFilename(err, f.Key)
		}
		certPair, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return nil, err
		}
		conf.Certificates = []tls.Certificate{certPair}
	}

	return &conf, nil
}

func versionName(tlsVersion uint16) string {
	// TODO: once we can use Go 1.20, it will provide tls.VersionName that we can use
	//       https://github.com/golang/go/issues/46308
	switch tlsVersion {
	case tls.VersionTLS10:
		return "TLSv1.0"
	case tls.VersionTLS11:
		return "TLSv1.1"
	case tls.VersionTLS12:
		return "TLSv1.2"
	case tls.VersionTLS13:
		return "TLSv1.3"
	default:
		return "(unrecognized TLS version)"
	}
}
