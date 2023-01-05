// Copyright 2020-2023 Buf Technologies, Inc.
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

package observability

import (
	"net/http"

	"go.opencensus.io/tag"
)

type wrappedRoundTripper struct {
	Base http.RoundTripper
	Tags []tag.Mutator
}

func (w *wrappedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, err := tag.New(req.Context(), w.Tags...)
	if err != nil {
		return nil, err
	}
	wrappedRequest := req.WithContext(ctx)
	return w.Base.RoundTrip(wrappedRequest)
}
