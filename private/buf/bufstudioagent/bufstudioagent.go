// Copyright 2020-2026 Buf Technologies, Inc.
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

package bufstudioagent

import (
	"crypto/tls"
	"log/slog"
	"net/http"

	"github.com/jub0bs/cors"
)

// NewHandler creates a new handler that serves the invoke endpoints for the
// agent.
func NewHandler(
	logger *slog.Logger,
	origin string,
	tlsClientConfig *tls.Config,
	disallowedHeaders map[string]struct{},
	forwardHeaders map[string]string,
) (http.Handler, error) {
	corsMiddleware, err := cors.NewMiddleware(cors.Config{
		Origins:      []string{origin},
		Methods:      []string{http.MethodPost},
		Credentialed: true,
	})
	if err != nil {
		return nil, err
	}
	plainHandler := corsMiddleware.Wrap(newPlainPostHandler(logger, disallowedHeaders, forwardHeaders, tlsClientConfig))
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// In the future we could check for an upgrade header here.
			_, _ = w.Write([]byte("OK"))
		case http.MethodPost, http.MethodOptions:
			plainHandler.ServeHTTP(w, r)
		default:
			http.Error(w, "", http.StatusMethodNotAllowed)
		}
	})
	return mux, nil
}
