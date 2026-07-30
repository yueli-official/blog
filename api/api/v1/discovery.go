package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"github.com/yueli-official/blog/api/internal/discoveryview"
)

type GetDiscoveryArtifactReq struct {
	g.Meta `path:"/api/v1/discovery/artifact" method:"get" tags:"discovery" summary:"Get one artifact from the current atomic discovery publication"`
	Name   string `json:"name" v:"required"`
}

type GetDiscoveryArtifactRes struct {
	Result discoveryview.ArtifactResult `json:"result"`
}
