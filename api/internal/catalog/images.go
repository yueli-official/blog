package catalog

import (
	"context"

	"platform/products/blog/api/internal/blogclient"
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

// FinalizeImage finalizes the uploaded image and returns its public CDN URL (to
// embed in the post markdown).
func (s *Service) FinalizeImage(ctx context.Context, bearer, uploadToken string) (string, error) {
	view, err := s.asset.Finalize(ctx, bearer, uploadToken)
	if err != nil {
		return "", err
	}
	return view.CdnURL, nil
}
