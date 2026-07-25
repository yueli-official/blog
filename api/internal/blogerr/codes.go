// Package blogerr declares the blog-site error codes (namespace blog.*) and
// their HTTP status, registered with the shared gokit/errs catalog.
package blogerr

import (
	"net/http"

	"platform/gokit/errs"
)

var (
	CodeNotFound                 = errs.Register("blog.not_found", http.StatusNotFound)
	CodeForbidden                = errs.Register("blog.forbidden", http.StatusForbidden)
	CodeSlugTaken                = errs.Register("blog.slug_taken", http.StatusConflict)
	CodeInvalidState             = errs.Register("blog.invalid_state", http.StatusBadRequest)
	CodeInvalidInput             = errs.Register("blog.invalid_input", http.StatusBadRequest)
	CodeUpstreamFailed           = errs.Register("blog.upstream_failed", http.StatusBadGateway)
	CodeAuthorizationUnavailable = errs.Register("blog.authorization_unavailable", http.StatusServiceUnavailable)

	CodeCommentNotFound   = errs.Register("blog.comment_not_found", http.StatusNotFound)
	CodeCommentsClosed    = errs.Register("blog.comments_closed", http.StatusConflict)
	CodeCommentRejected   = errs.Register("blog.comment_rejected", http.StatusUnprocessableEntity)
	CodeRateLimited       = errs.Register("blog.rate_limited", http.StatusTooManyRequests)
	CodeChallengeRequired = errs.Register("blog.challenge_required", http.StatusForbidden)
	CodeAbuseUnavailable  = errs.Register("blog.abuse_unavailable", http.StatusServiceUnavailable)
	CodeAbuseReplay       = errs.Register("blog.abuse_attempt_replayed", http.StatusConflict)
)

// NotFound is returned when a post id/slug does not exist or is not visible.
func NotFound(id string) *errs.Coded {
	return errs.New(CodeNotFound, "post not found", map[string]any{"id": id})
}

// Forbidden is returned when the caller is not the post author.
func Forbidden() *errs.Coded { return errs.New(CodeForbidden, "not the post author", nil) }

func AuthorizationUnavailable() *errs.Coded {
	return errs.New(CodeAuthorizationUnavailable, "authorization is temporarily unavailable", nil)
}

// SlugTaken is returned when a generated/explicit slug collides (after retries).
func SlugTaken(slug string) *errs.Coded {
	return errs.New(CodeSlugTaken, "slug already taken", map[string]any{"slug": slug})
}

// InvalidState is returned when a publish constraint is unmet (empty title/content).
func InvalidState(detail string) *errs.Coded {
	return errs.New(CodeInvalidState, "invalid state: "+detail, nil)
}

// InvalidInput is returned for malformed request input not caught by binding.
func InvalidInput(detail string) *errs.Coded {
	return errs.New(CodeInvalidInput, "invalid input: "+detail, nil)
}

// UpstreamFailed wraps a downstream (asset) failure with a trimmed summary —
// never the raw downstream body.
func UpstreamFailed(summary string) *errs.Coded {
	return errs.New(CodeUpstreamFailed, "upstream service failed: "+summary, nil)
}

// CommentNotFound is returned when a comment id does not exist or is deleted.
func CommentNotFound(id string) *errs.Coded {
	return errs.New(CodeCommentNotFound, "comment not found", map[string]any{"id": id})
}

// CommentsClosed is returned when a post has comments turned off (comment_status=0).
func CommentsClosed() *errs.Coded {
	return errs.New(CodeCommentsClosed, "comments are closed on this post", nil)
}

// CommentRejected is returned when a comment trips the keyword blacklist
// (anti-spam guard). The reason is intentionally vague so spammers can't probe
// the word list.
func CommentRejected() *errs.Coded {
	return errs.New(CodeCommentRejected, "comment rejected by the content filter", nil)
}

// RateLimited is returned when an IP exceeds the comment rate window (anti-spam).
func RateLimited() *errs.Coded {
	return errs.New(CodeRateLimited, "too many comments — please slow down", nil)
}

func ChallengeRequired(attemptID string) *errs.Coded {
	return errs.New(CodeChallengeRequired, "additional verification required", map[string]any{
		"attemptId": attemptID, "challenge": "turnstile",
	})
}

func AbuseUnavailable() *errs.Coded {
	return errs.New(CodeAbuseUnavailable, "comment admission is temporarily unavailable", nil)
}

func AbuseAttemptReplayed() *errs.Coded {
	return errs.New(CodeAbuseReplay, "comment attempt was already admitted", nil)
}
