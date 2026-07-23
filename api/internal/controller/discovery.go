package controller

import (
	"context"

	"github.com/yueli-official/foundation/go/discovery"

	"platform/gokit/discoveryapi"
	v1 "platform/products/blog/api/api/v1"
)

type PublicDiscovery struct {
	cache *discovery.Cache
}

func NewPublicDiscovery(cache *discovery.Cache) *PublicDiscovery {
	return &PublicDiscovery{cache: cache}
}

func (controller *PublicDiscovery) GetDiscoveryArtifact(
	ctx context.Context,
	req *v1.GetDiscoveryArtifactReq,
) (*v1.GetDiscoveryArtifactRes, error) {
	snapshot, err := controller.cache.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetDiscoveryArtifactRes{
		Result: discoveryapi.FindArtifact(snapshot, req.Name),
	}, nil
}
