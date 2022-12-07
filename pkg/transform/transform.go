package transform

import (
	"buf.build/gen/go/bufbuild/buf/bufbuild/connect-go/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	"github.com/bufbuild/connect-go"
)

// Client ..
type Client struct {
	client registryv1alpha1connect.SchemaServiceClient
}

// NewClient ..
func NewClient(
	httpClient connect.HTTPClient,
	baseURL string,
	opts ...connect.ClientOption,
) *Client {
	return &Client{
		client: registryv1alpha1connect.NewSchemaServiceClient(
			httpClient,
			baseURL,
			opts...,
		),
	}
}

// Transform ..
func (c *Client) Transform() ([]byte, error) {
	return nil, nil
}
