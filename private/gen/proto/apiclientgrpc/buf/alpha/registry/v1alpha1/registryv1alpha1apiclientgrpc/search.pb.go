// Code generated by protoc-gen-go-apiclientgrpc. DO NOT EDIT.

package registryv1alpha1apiclientgrpc

import (
	context "context"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	zap "go.uber.org/zap"
)

type searchService struct {
	logger          *zap.Logger
	client          v1alpha1.SearchServiceClient
	contextModifier func(context.Context) context.Context
}

// Search searches the BSR.
func (s *searchService) Search(
	ctx context.Context,
	query string,
	pageSize uint32,
	pageToken uint32,
	filters []v1alpha1.SearchFilter,
) (searchResults []*v1alpha1.SearchResult, nextPageToken uint32, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.Search(
		ctx,
		&v1alpha1.SearchRequest{
			Query:     query,
			PageSize:  pageSize,
			PageToken: pageToken,
			Filters:   filters,
		},
	)
	if err != nil {
		return nil, 0, err
	}
	return response.SearchResults, response.NextPageToken, nil
}
