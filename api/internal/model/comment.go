package model

import "github.com/gogf/gf/v2/os/gtime"

// CommentStatus is the moderation state of a comment. Per the blog spec the
// numeric mapping is 1 approved / 2 pending / 3 spam / 4 trash (note: this is
// the inverse of the legacy donor plugin, which used 1 pending / 2 approved).
type CommentStatus int

const (
	CommentApproved CommentStatus = 1
	CommentPending  CommentStatus = 2
	CommentSpam     CommentStatus = 3
	CommentTrash    CommentStatus = 4
)

// Comment is one comment on a post. UserID is the JWT sub for a logged-in
// commenter, or "" for an anonymous one (who supplies AuthorName/AuthorEmail).
// ParentID is "" for a top-level comment, else the id of its top-level ancestor
// — the thread is kept two levels deep (replies never nest beyond depth 1).
type Comment struct {
	ID          string        `json:"id" orm:"id"`
	PostID      string        `json:"postId" orm:"post_id"`
	ParentID    string        `json:"parentId" orm:"parent_id"`
	UserID      string        `json:"userId" orm:"user_id"`
	AuthorName  string        `json:"authorName" orm:"author_name"`
	AuthorEmail string        `json:"authorEmail" orm:"author_email"`
	Content     string        `json:"content" orm:"content"`
	Status      CommentStatus `json:"status" orm:"status"`
	IP          string        `json:"ip" orm:"ip"`
	UserAgent   string        `json:"userAgent" orm:"user_agent"`
	CreatedAt   *gtime.Time   `json:"createdAt" orm:"created_at"`
	DeletedAt   *gtime.Time   `json:"deletedAt" orm:"deleted_at"`
}

// PostHead is a post's identity (title/slug) for labelling moderation rows
// without loading the whole post.
type PostHead struct {
	ID    string `json:"id" orm:"id"`
	Title string `json:"title" orm:"title"`
	Slug  string `json:"slug" orm:"slug"`
}
