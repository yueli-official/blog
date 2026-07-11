package catalog

import (
	"context"
	"strings"

	"platform/products/blog/api/internal/blogerr"
	"platform/products/blog/api/internal/model"
)

// CommentThread is a top-level comment with its flattened approved replies.
type CommentThread struct {
	Comment *model.Comment
	Replies []*model.Comment
}

// AdminComment pairs a comment with its post's head, for the moderation list.
type AdminComment struct {
	Comment   *model.Comment
	PostTitle string
	PostSlug  string
}

// ListComments returns a published post's approved comment threads (two levels
// deep), paginated by top-level comment.
func (s *Service) ListComments(ctx context.Context, slug string, page, size int) ([]*CommentThread, int, int, int, error) {
	page, size = norm(page, size)
	p, err := s.dao.GetBySlug(ctx, slug)
	if err != nil {
		return nil, 0, page, size, err
	}
	if p == nil || p.Status != model.StatusPublished {
		return nil, 0, page, size, blogerr.NotFound(slug)
	}
	tops, total, err := s.dao.ListApproved(ctx, p.ID, size, (page-1)*size)
	if err != nil {
		return nil, 0, page, size, err
	}
	ids := make([]string, 0, len(tops))
	for _, c := range tops {
		ids = append(ids, c.ID)
	}
	repliesMap, err := s.dao.RepliesByParents(ctx, ids)
	if err != nil {
		return nil, 0, page, size, err
	}
	threads := make([]*CommentThread, 0, len(tops))
	for _, c := range tops {
		threads = append(threads, &CommentThread{Comment: c, Replies: repliesMap[c.ID]})
	}
	return threads, total, page, size, nil
}

// CreateComment validates and inserts a comment. A logged-in commenter (sub != "")
// is auto-approved; an anonymous one goes to pending moderation. parentID, when
// set, attaches the comment to that thread's top-level ancestor (max depth 2).
func (s *Service) CreateComment(ctx context.Context, sub, slug, content, parentID, authorName, authorEmail, ip, ua string) (*model.Comment, error) {
	content = strings.TrimSpace(content)
	if n := len([]rune(content)); n < 1 || n > 5000 {
		return nil, blogerr.InvalidInput("comment content length out of range")
	}
	// Anti-spam: blacklisted content is rejected outright (before any DB work).
	if containsBlacklisted(content, s.spam.Blacklist) {
		return nil, blogerr.CommentRejected()
	}
	p, err := s.dao.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if p == nil || p.Status != model.StatusPublished {
		return nil, blogerr.NotFound(slug)
	}
	if p.CommentStatus != 1 {
		return nil, blogerr.CommentsClosed()
	}
	// Anti-spam: per-IP rate limit (flood guard). Counts all of this IP's recent
	// comments regardless of status, so spam that got moderated still counts.
	if s.spam.RatePerWindow > 0 && ip != "" {
		n, err := s.dao.CountRecentCommentsByIP(ctx, ip, s.spam.RateWindowSeconds)
		if err != nil {
			return nil, err
		}
		if n >= s.spam.RatePerWindow {
			return nil, blogerr.RateLimited()
		}
	}

	c := &model.Comment{PostID: p.ID, Content: content, IP: ip, UserAgent: ua}
	if sub != "" {
		c.UserID = sub
		c.Status = model.CommentApproved
	} else {
		authorName = strings.TrimSpace(authorName)
		if authorName == "" {
			return nil, blogerr.InvalidInput("anonymous comment requires a name")
		}
		authorEmail = strings.TrimSpace(authorEmail)
		if authorEmail != "" && !looksLikeEmail(authorEmail) {
			return nil, blogerr.InvalidInput("invalid email")
		}
		c.AuthorName = authorName
		c.AuthorEmail = authorEmail
		c.Status = model.CommentPending
	}

	if parentID != "" {
		parent, err := s.dao.GetComment(ctx, parentID)
		if err != nil {
			return nil, err
		}
		if parent == nil || parent.PostID != p.ID {
			return nil, blogerr.InvalidInput("parent comment not found")
		}
		if parent.ParentID != "" { // replying to a reply → re-point to the ancestor
			c.ParentID = parent.ParentID
		} else {
			c.ParentID = parent.ID
		}
	}

	// Anti-spam: a comment with too many links is forced to moderation rather
	// than rejected — even a logged-in member's link-heavy comment goes pending.
	if s.spam.MaxLinks > 0 && countLinks(content) > s.spam.MaxLinks {
		c.Status = model.CommentPending
	}

	if err := s.dao.InsertComment(ctx, c); err != nil {
		return nil, err
	}
	return s.dao.GetComment(ctx, c.ID)
}

// ListMineComments returns comments on the caller's posts (status 0 = all),
// newest first, each paired with its post head.
// When isAdmin is true, all posts across the site are included (site-wide moderation queue).
func (s *Service) ListMineComments(ctx context.Context, author string, isAdmin bool, status, page, size int) ([]*AdminComment, int, int, int, error) {
	page, size = norm(page, size)
	daoAuthor := author
	if isAdmin {
		daoAuthor = "" // site-wide moderation queue
	}
	rows, total, err := s.dao.ListMineComments(ctx, daoAuthor, status, size, (page-1)*size)
	if err != nil {
		return nil, 0, page, size, err
	}
	ids := make([]string, 0, len(rows))
	for _, c := range rows {
		ids = append(ids, c.PostID)
	}
	heads, err := s.dao.PostHeadsByIDs(ctx, ids)
	if err != nil {
		return nil, 0, page, size, err
	}
	out := make([]*AdminComment, 0, len(rows))
	for _, c := range rows {
		h := heads[c.PostID]
		out = append(out, &AdminComment{Comment: c, PostTitle: h.Title, PostSlug: h.Slug})
	}
	return out, total, page, size, nil
}

// SetCommentStatus moves a comment to approved/spam/trash. The post author or a
// superadmin may moderate; plain users who are not the post author get 403.
func (s *Service) SetCommentStatus(ctx context.Context, author string, isAdmin bool, id string, status model.CommentStatus) (*model.Comment, error) {
	if status != model.CommentApproved && status != model.CommentSpam && status != model.CommentTrash {
		return nil, blogerr.InvalidInput("invalid target status")
	}
	if !isAdmin {
		if err := s.assertCommentOwner(ctx, author, id); err != nil {
			return nil, err
		}
	}
	n, err := s.dao.SetCommentStatus(ctx, id, int(status))
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, blogerr.CommentNotFound(id)
	}
	return s.dao.GetComment(ctx, id)
}

// DeleteComment soft-deletes a comment (and its replies). The post author or a
// superadmin may delete; plain users who are not the post author get 403.
func (s *Service) DeleteComment(ctx context.Context, author string, isAdmin bool, id string) error {
	if !isAdmin {
		if err := s.assertCommentOwner(ctx, author, id); err != nil {
			return err
		}
	}
	n, err := s.dao.SoftDeleteComment(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return blogerr.CommentNotFound(id)
	}
	return nil
}

// assertCommentOwner loads the comment + its post and checks `author` owns the post.
func (s *Service) assertCommentOwner(ctx context.Context, author, id string) error {
	c, err := s.dao.GetComment(ctx, id)
	if err != nil {
		return err
	}
	if c == nil {
		return blogerr.CommentNotFound(id)
	}
	p, err := s.dao.GetByID(ctx, c.PostID)
	if err != nil {
		return err
	}
	if p == nil || p.AuthorID != author {
		return blogerr.Forbidden()
	}
	return nil
}

// looksLikeEmail is a minimal sanity check (has an '@' with a dotted domain) —
// not RFC validation; the anon→pending gate is the real spam defense.
func looksLikeEmail(s string) bool {
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return false
	}
	return strings.IndexByte(s[at+1:], '.') >= 0
}
