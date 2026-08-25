package controller

import (
	"context"

	v1 "github.com/yueli-official/blog/api/api/v1"
	"github.com/yueli-official/blog/api/internal/catalog"
	"github.com/yueli-official/blog/api/internal/model"
	foundationauth "github.com/yueli-official/foundation/go/auth"
	"github.com/yueli-official/foundation/go/identifier"
)

// PublicComments handles the reader-facing comment endpoints (optional login):
// list approved comments + post a comment (logged-in or anonymous). It verifies
// the bearer token itself when present (not behind Foundation auth middleware).
type PublicComments struct {
	svc      *catalog.Service
	verifier *foundationauth.Verifier
}

func NewPublicComments(svc *catalog.Service, v *foundationauth.Verifier) *PublicComments {
	return &PublicComments{svc: svc, verifier: v}
}

func (c *PublicComments) ListComments(ctx context.Context, req *v1.ListCommentsReq) (*v1.ListCommentsRes, error) {
	threads, total, page, size, err := c.svc.ListComments(ctx, req.Slug, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	profiles := c.svc.ResolveAuthors(ctx, commentThreadUserIDs(threads))
	return &v1.ListCommentsRes{Items: commentThreadViews(threads, profiles), Total: total, Page: page, Size: size}, nil
}

func (c *PublicComments) CreateComment(ctx context.Context, req *v1.CreateCommentReq) (*v1.CreateCommentRes, error) {
	sub := optionalSubject(ctx, c.verifier)
	ip, ua := clientMeta(ctx)
	attemptID := req.AbuseAttemptID
	if attemptID == "" {
		attemptID = identifier.MustNew().String()
	}
	cm, err := c.svc.CreateComment(
		ctx, sub, req.Slug, req.Content, req.ParentID, req.AuthorName,
		req.AuthorEmail, ip, ua, attemptID, req.ChallengeProof,
	)
	if err != nil {
		return nil, err
	}
	profiles := c.svc.ResolveAuthors(ctx, []string{cm.UserID})
	return &v1.CreateCommentRes{Comment: commentViewWithProfiles(cm, profiles), Pending: cm.Status != model.CommentApproved}, nil
}
