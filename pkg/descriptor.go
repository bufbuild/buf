package descriptor

import (
	"context"
	"fmt"
	"github.com/bufbuild/buf/pkg/internal/cache"

	"buf.build/gen/go/bufbuild/buf/bufbuild/connect-go/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "buf.build/gen/go/bufbuild/buf/protocolbuffers/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/connect-go"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Client ..
type Client struct {
	client registryv1alpha1connect.SchemaServiceClient
	cache  *cache.Cache
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
		cache: cache.NewCache(),
	}
}

// GetSchema ..
func (c *Client) GetSchema(
	ctx context.Context,
	owner, repository, reference, ifNotCommit string,
	excludeCustomOptions, excludeKnownExtensions bool,
	includeTypes ...string,
) (string, *descriptorpb.FileDescriptorSet, error) {
	if out, ok := c.checkCache(owner, repository, reference); ok {
		return out.Commit, out.SchemaFiles, nil
	}
	request := &registryv1alpha1.GetSchemaRequest{
		Owner:                  owner,
		Repository:             repository,
		Version:                reference,
		Types:                  includeTypes,
		IfNotCommit:            ifNotCommit,
		ExcludeCustomOptions:   excludeCustomOptions,
		ExcludeKnownExtensions: excludeKnownExtensions,
	}
	resp, err := c.client.GetSchema(ctx, connect.NewRequest(request))
	if err != nil {
		return "", nil, err
	}
	out := resp.Msg
	go c.cache.Write(fmt.Sprintf("%s/%s/%s", owner, repository, out.Commit), out)
	return out.Commit, out.SchemaFiles, nil
}

// TODO: get this to "work" with the elementNames attribute
func (c *Client) checkCache(owner string, repository string, reference string) (*registryv1alpha1.GetSchemaResponse, bool) {
	module := fmt.Sprintf("%s/%s/%s", owner, repository, reference)
	cached, ok := c.cache.Read(module)
	if ok {
		if schema, ok := cached.(*registryv1alpha1.GetSchemaResponse); ok {
			return schema, ok
		}
	}
	return nil, ok
}
