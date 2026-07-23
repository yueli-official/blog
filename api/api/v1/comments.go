package v1

import "github.com/gogf/gf/v2/frame/g"

// CommentView is the public projection of a comment. AuthorName is a display
// label (anonymous name, or a short member tag); the raw sub is never exposed.
type CommentView struct {
	ID         string         `json:"id"`
	ParentID   string         `json:"parentId,omitempty"`
	AuthorName string         `json:"authorName"`
	IsMember   bool           `json:"isMember"`
	Content    string         `json:"content"`
	CreatedAt  string         `json:"createdAt"`
	Replies    []*CommentView `json:"replies,omitempty"`
}

// CommentAdminView is the moderation projection (carries status + contact + post).
type CommentAdminView struct {
	ID          string `json:"id"`
	PostID      string `json:"postId"`
	PostTitle   string `json:"postTitle,omitempty"`
	PostSlug    string `json:"postSlug,omitempty"`
	ParentID    string `json:"parentId,omitempty"`
	AuthorName  string `json:"authorName"`
	AuthorEmail string `json:"authorEmail,omitempty"`
	UserID      string `json:"userId,omitempty"`
	Content     string `json:"content"`
	Status      int    `json:"status"`
	IP          string `json:"ip,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

// ── comments (public, optional auth) ─────────────────────────────────────────

type ListCommentsReq struct {
	g.Meta `path:"/api/v1/posts/{slug}/comments" method:"get" tags:"blog" summary:"List approved comments for a post"`
	Slug   string `json:"slug" in:"path" v:"required"`
	Page   int    `json:"page"`
	Size   int    `json:"size"`
}

type ListCommentsRes struct {
	Items []*CommentView `json:"items"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
}

// CreateCommentReq posts a comment. A logged-in caller (bearer present) comments
// under their identity and is auto-approved; otherwise AuthorName is required and
// the comment is held for moderation.
type CreateCommentReq struct {
	g.Meta         `path:"/api/v1/posts/{slug}/comments" method:"post" tags:"blog" summary:"Post a comment (login or anonymous)"`
	Slug           string `json:"slug" in:"path" v:"required"`
	Content        string `json:"content" v:"required"`
	ParentID       string `json:"parentId"`
	AuthorName     string `json:"authorName"`
	AuthorEmail    string `json:"authorEmail"`
	AbuseAttemptID string `json:"abuseAttemptId,omitempty"`
	ChallengeProof string `json:"challengeProof,omitempty"`
}

type CreateCommentRes struct {
	Comment *CommentView `json:"comment"`
	Pending bool         `json:"pending"` // true → awaiting moderation, not yet public
}

// ── comments moderation (author JWT) ─────────────────────────────────────────

type ListMyCommentsReq struct {
	g.Meta  `path:"/api/v1/comments/mine" method:"get" tags:"blog" summary:"List comments on my posts (moderation)"`
	Status  int    `json:"status"` // 0 all | 1 approved | 2 pending | 3 spam | 4 trash
	Keyword string `json:"keyword"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
}

type ListMyCommentsRes struct {
	Items []*CommentAdminView `json:"items"`
	Total int                 `json:"total"`
	Page  int                 `json:"page"`
	Size  int                 `json:"size"`
}

// SetCommentStatusReq moderates a comment (Status: 1 approve / 3 spam / 4 trash).
type SetCommentStatusReq struct {
	g.Meta `path:"/api/v1/comments/{id}" method:"patch" tags:"blog" summary:"Moderate a comment (approve/spam/trash)"`
	ID     string `json:"id" in:"path" v:"required"`
	Status int    `json:"status"`
}

type SetCommentStatusRes struct {
	Comment *CommentAdminView `json:"comment"`
}

type DeleteCommentReq struct {
	g.Meta `path:"/api/v1/comments/{id}" method:"delete" tags:"blog" summary:"Delete a comment"`
	ID     string `json:"id" in:"path" v:"required"`
}

type DeleteCommentRes struct {
	Deleted bool `json:"deleted"`
}
