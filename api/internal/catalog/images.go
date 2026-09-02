package catalog

import (
	"context"
	"errors"
	"net/url"

	"github.com/yueli-official/blog/api/internal/blogclient"
)

// blogContentCategory is the asset category for inline content images. Public
// delivery (same as covers) but a distinct bucket from the cover category.
const blogContentCategory = "blog-post"

// InitImage opens an upload for a standalone inline content image (public asset),
// not tied to any post. Auth is enforced by the controller; this just brokers the
// asset upload with the caller's bearer.
func (s *Service) InitImage(ctx context.Context, bearer, filename, mime string, size int64) (blogclient.InitOutput, error) {
	if mime == "" {
		mime = "application/octet-stream"
	}
	return s.asset.UploadInit(ctx, bearer, blogclient.InitInput{
		Filename: filename, Mime: mime, Category: blogContentCategory, Visibility: "public", Size: size,
	})
}

// FinalizeImage finalizes the uploaded image and returns the lightweight named
// inline rendition used in article Markdown. The reading surface can retarget the same
// stable media key to the larger content rendition without exposing originals.
func (s *Service) FinalizeImage(ctx context.Context, bearer, uploadToken string) (string, error) {
	view, err := s.asset.Finalize(ctx, bearer, uploadToken)
	if err != nil {
		return "", err
	}
	if view.MediaKey == "" {
		return "", errors.New("asset finalize did not return mediaKey")
	}
	return "/media/" + url.PathEscape(view.MediaKey) + "?format=webp&name=inline&v=1", nil
}
