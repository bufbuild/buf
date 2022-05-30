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

package httpserver

import (
	"fmt"
	"net/http"
	"path"

	"github.com/go-chi/chi/v5"
)

type prefixedHTTPHandlerMapper struct {
	handlers     []PrefixedHTTPHandler
	globalPrefix string
}

func newPrefixedHTTPHandlerMapper(
	handlers []PrefixedHTTPHandler,
	options ...PrefixedHTTPHandlerMapperOption,
) *prefixedHTTPHandlerMapper {
	prefixedHTTPHandlerMapper := &prefixedHTTPHandlerMapper{
		handlers: handlers,
	}
	for _, option := range options {
		option(prefixedHTTPHandlerMapper)
	}
	return prefixedHTTPHandlerMapper
}

func (m *prefixedHTTPHandlerMapper) Map(router chi.Router) error {
	if err := validatePrefixedHTTPHandlers(m.handlers); err != nil {
		return err
	}
	for _, handler := range m.handlers {
		prefix := path.Join(m.globalPrefix, handler.PathPrefix())
		var strippedHandler http.Handler = handler
		if m.globalPrefix != "" {
			strippedHandler = http.StripPrefix(m.globalPrefix, handler)
		}
		router.Mount(prefix, strippedHandler)
	}
	return nil
}

func validatePrefixedHTTPHandlers(prefixedHTTPHandlers []PrefixedHTTPHandler) error {
	allPathPrefixes := make(map[string]struct{}, len(prefixedHTTPHandlers))
	for _, prefixedHTTPHandler := range prefixedHTTPHandlers {
		pathPrefix := prefixedHTTPHandler.PathPrefix()
		if _, ok := allPathPrefixes[pathPrefix]; ok {
			return fmt.Errorf("duplicate path prefix: %v", pathPrefix)
		}
		allPathPrefixes[pathPrefix] = struct{}{}
	}
	return nil
}
