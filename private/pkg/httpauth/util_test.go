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

package httpauth

import (
	"net/http"
	"testing"
)

func TestSetBasicAuth(t *testing.T) {
	// Valid HTTPS request with username and password
	req1, _ := http.NewRequest("GET", "https://example.com", nil)
	_, err := setBasicAuth(req1, "username", "password", "", "")
	if err != nil {
		t.Errorf("setBasicAuth() returned error for valid request: %v", err)
	}
	if req1.Header.Get("Authorization") == "" {
		t.Errorf("setBasicAuth() didn't set Authorization header for valid request")
	}

	// Valid HTTPS request with no username and password
	req2, _ := http.NewRequest("GET", "https://example.com", nil)
	ok, err := setBasicAuth(req2, "", "", "", "")
	if err != nil {
		t.Errorf("setBasicAuth() returned error for valid request: %v", err)
	}
	if ok || req2.Header.Get("Authorization") != "" {
		t.Errorf("setBasicAuth() didn't handle empty username and password correctly")
	}

	// Invalid request with no URL
	req3 := &http.Request{}
	_, err = setBasicAuth(req3, "username", "password", "", "")
	if err == nil {
		t.Errorf("setBasicAuth() didn't return error for malformed request")
	}

	// Invalid request with no URL scheme
	req4, _ := http.NewRequest("GET", "example.com", nil)
	_, err = setBasicAuth(req4, "username", "password", "", "")
	if err == nil {
		t.Errorf("setBasicAuth() didn't return error for malformed request")
	}

	// Invalid request with non-HTTPS scheme
	req5, _ := http.NewRequest("GET", "http://example.com", nil)
	ok, err = setBasicAuth(req5, "username", "password", "", "")
	if err != nil {
		t.Errorf("setBasicAuth() returned error for non-HTTPS request: %v", err)
	}
	if ok || req5.Header.Get("Authorization") != "" {
		t.Errorf("setBasicAuth() didn't handle non-HTTPS scheme correctly")
	}
}
