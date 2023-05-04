package bufcurl

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

func TestGetAuthority(t *testing.T) {
	type args struct {
		url     *url.URL
		headers http.Header
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test with Host header",
			args: args{
				url: &url.URL{
					Host: "example.com",
				},
				headers: http.Header{
					"Host": []string{"example.com:8080"},
				},
			},
			want: "example.com:8080",
		},
		{
			name: "Test without Host header",
			args: args{
				url: &url.URL{
					Host: "example.com",
				},
				headers: http.Header{},
			},
			want: "example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAuthority(tt.args.url, tt.args.headers); got != tt.want {
				t.Errorf("GetAuthority() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadHeaders(t *testing.T) {
	headerFlags := []string{"X-Test-Header: test", "*"}
	others := http.Header{"Authorization": []string{"Bearer token"}}
	expectedHeaders := http.Header{
		"X-Test-Header": []string{"test"},
		"Authorization": []string{"Bearer token"},
	}

	headers, _, err := LoadHeaders(headerFlags, "", others)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(headers, expectedHeaders) {
		t.Errorf("unexpected headers: got %v, want %v", headers, expectedHeaders)
	}
}
