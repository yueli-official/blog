package catalog

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/frame/g"

	"platform/products/blog/api/internal/blogclient"
	"platform/products/blog/api/internal/blogerr"
	"platform/products/blog/api/internal/model"
)

// AddCover opens an upload for a post's cover image. A cover is always a public
// asset in the configured cover category, so browse/detail can render it from
// the asset service's public URL. Returns the blob link + upload token.
func (s *Service) AddCover(ctx context.Context, author, bearer, postID, filename, mime string, size int64) (blogclient.InitOutput, error) {
	if _, err := s.ownedPost(ctx, author, postID); err != nil {
		return blogclient.InitOutput{}, err
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	return s.asset.UploadInit(ctx, bearer, blogclient.InitInput{
		Filename: filename, Mime: mime, Category: s.coverCategory, Visibility: "public", Size: size,
	})
}

// FinalizeCover finalizes the uploaded cover, points the post at it
// (cover_asset_id + cover_url snapshot), and best-effort removes the old cover.
func (s *Service) FinalizeCover(ctx context.Context, author, bearer, postID, uploadToken string) (assetID, coverURL string, err error) {
	p, err := s.ownedPost(ctx, author, postID)
	if err != nil {
		return "", "", err
	}
	old := p.CoverAssetID
	view, err := s.asset.Finalize(ctx, bearer, uploadToken)
	if err != nil {
		return "", "", err
	}
	if _, err := s.dao.Patch(ctx, author, postID, g.Map{"cover_asset_id": view.ID, "cover_url": view.CdnURL}); err != nil {
		return "", "", err
	}
	if err := s.asset.RegisterReference(ctx, bearer, blogclient.ReferenceInput{
		AssetID: view.ID, RefType: "post-cover", RefID: postID,
		RefLabel: p.Title, RefURL: strings.TrimRight(s.siteURL, "/") + "/posts/" + p.Slug,
	}); err != nil {
		return "", "", err
	}
	if old != "" && old != view.ID {
		_ = s.asset.UnregisterReference(ctx, bearer, blogclient.ReferenceInput{
			AssetID: old, RefType: "post-cover", RefID: postID,
		})
		_ = s.asset.Delete(ctx, bearer, old) // best-effort
	}
	return view.ID, view.CdnURL, nil
}

func (s *Service) ownedPost(ctx context.Context, author, id string) (*model.Post, error) {
	p, err := s.dao.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil || p.AuthorID != author {
		return nil, blogerr.NotFound(id)
	}
	return p, nil
}
