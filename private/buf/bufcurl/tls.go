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

package bufcurl

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"strings"
	"time"

	"github.com/bufbuild/buf/private/pkg/verbose"
)

// TLSSettings contains settings related to creating a TLS client.
type TLSSettings struct {
	// Filenames for a private key, certificate, and CA certificate pool.
	KeyFile, CertFile, CACertFile string
	// Override server name, for SNI.
	ServerName string
	// If true, the server's certificate is not verified.
	Insecure bool
	// If true, the server is known to support HTTP/2. When set, the
	// ALPN protocols sent during the TLS handshake will include only
	// "h2", and exclude "http/1.1". If the server does not pick a
	// protocol, "h2" is assumed as the default.
	HTTP2PriorKnowledge bool
}

// MakeVerboseTLSConfig constructs a *tls.Config that logs information to the
// given printer as a TLS connection is negotiated.
func MakeVerboseTLSConfig(settings *TLSSettings, authority string, printer verbose.Printer) (*tls.Config, error) {
	var conf tls.Config
	if settings.HTTP2PriorKnowledge {
		conf.NextProtos = []string{"h2"}
	} else {
		conf.NextProtos = []string{"h2", "http/1.1"}
	}
	// we verify manually so that we can emit verbose output while doing so
	conf.InsecureSkipVerify = true
	conf.VerifyConnection = func(state tls.ConnectionState) error {
		printer.Printf("* TLS connection using %s / %s", tlsVersionName(state.Version), tls.CipherSuiteName(state.CipherSuite))
		if state.DidResume {
			printer.Printf("* (TLS session resumed)")
		}
		if state.NegotiatedProtocol == "" {
			if settings.HTTP2PriorKnowledge {
				printer.Printf("* ALPN: server did not agree on a protocol. Using default h2")
			} else {
				printer.Printf("* ALPN: server did not agree on a protocol. Using default http/1.1.")
			}
		} else {
			printer.Printf("* ALPN: server accepted %s", state.NegotiatedProtocol)
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
		if !settings.Insecure {
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
	if settings.ServerName != "" {
		conf.ServerName = settings.ServerName
	} else if authority != "" {
		// strip port if present
		host, _, err := net.SplitHostPort(authority)
		if err == nil {
			authority = host
		}
		conf.ServerName = authority
	}

	if settings.CACertFile != "" {
		caCert, err := os.ReadFile(settings.CACertFile)
		if err != nil {
			return nil, ErrorHasFilename(err, settings.CACertFile)
		}
		conf.RootCAs = x509.NewCertPool()
		conf.RootCAs.AppendCertsFromPEM(caCert)
	}

	if settings.KeyFile != "" && settings.CertFile != "" {
		cert, err := os.ReadFile(settings.CertFile)
		if err != nil {
			return nil, ErrorHasFilename(err, settings.CertFile)
		}
		key, err := os.ReadFile(settings.KeyFile)
		if err != nil {
			return nil, ErrorHasFilename(err, settings.KeyFile)
		}
		certPair, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return nil, err
		}
		certPair.Leaf, err = x509.ParseCertificate(certPair.Certificate[0])
		if err != nil {
			return nil, err
		}
		conf.GetClientCertificate = func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			printer.Printf("* Offering client cert:")
			printer.Printf("*   subject: %s", certPair.Leaf.Subject.String())
			printer.Printf("*   start date: %s", certPair.Leaf.NotBefore)
			printer.Printf("*   end date: %s", certPair.Leaf.NotAfter)
			printer.Printf("*   issuer: %s", certPair.Leaf.Issuer.String())
			return &certPair, nil
		}
	}

	return &conf, nil
}
