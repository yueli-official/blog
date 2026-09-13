package catalog

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"github.com/yueli-official/blog/api/internal/blogclient"
	"github.com/yueli-official/blog/api/internal/blogerr"
	"github.com/yueli-official/blog/api/internal/coverurl"
	"github.com/yueli-official/blog/api/internal/model"
)

// AddCover opens an upload for a post's cover image. A cover is always a public
// asset in the configured cover category, so browse/detail can render it from
// the asset service's public URL. Returns the blob link + upload token.
func (s *Service) AddCover(ctx context.Context, author, bearer, postID, filename, mime string, size int64, preprocessed bool) (blogclient.InitOutput, error) {
	if _, err := s.ownedPost(ctx, author, postID); err != nil {
		return blogclient.InitOutput{}, err
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	return s.asset.UploadInit(ctx, bearer, blogclient.InitInput{
		Filename: filename, Mime: mime, Category: s.coverCategory, Visibility: "public", Size: size,
		Target: "home", Preprocessed: preprocessed,
	})
}

// FinalizeCover finalizes the uploaded cover, points the post at its stable
// same-origin media URL. Reference reconciliation owns the old/new relationships. Backend CDN
// URLs are transport details and must never enter Blog content or Discovery.
func (s *Service) FinalizeCover(ctx context.Context, author, bearer, postID, uploadToken string) (assetID, coverURL string, err error) {
	_, err = s.ownedPost(ctx, author, postID)
	if err != nil {
		return "", "", err
	}
	view, err := s.asset.Finalize(ctx, bearer, uploadToken)
	if err != nil {
		return "", "", err
	}
	coverURL, err = publicCoverURL(view)
	if err != nil {
		return "", "", err
	}
	if _, err := s.dao.Patch(ctx, author, postID, g.Map{"cover_asset_id": view.ID, "cover_url": coverURL}); err != nil {
		return "", "", err
	}
	return view.ID, coverURL, nil
}

func publicCoverURL(view blogclient.View) (string, error) {
	value := coverurl.FromMediaKey(view.MediaKey)
	if value == "" {
		return "", blogerr.UpstreamFailed("asset")
	}
	return value, nil
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
