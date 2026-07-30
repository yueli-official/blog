// Package blogerr declares Blog's immutable public Problem contract.
package blogerr

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/yueli-official/foundation/go/problem"
)

const (
	CodeNotFound                 = "blog.not_found"
	CodeForbidden                = "blog.forbidden"
	CodeSlugTaken                = "blog.slug_taken"
	CodeInvalidState             = "blog.invalid_state"
	CodeInvalidInput             = "blog.invalid_input"
	CodeUpstreamFailed           = "blog.upstream_failed"
	CodeAuthorizationUnavailable = "blog.authorization_unavailable"
	CodeCommentNotFound          = "blog.comment_not_found"
	CodeCommentsClosed           = "blog.comments_closed"
	CodeCommentRejected          = "blog.comment_rejected"
	CodeRateLimited              = "blog.rate_limited"
	CodeChallengeRequired        = "blog.challenge_required"
	CodeAbuseUnavailable         = "blog.abuse_unavailable"
	CodeAbuseReplay              = "blog.abuse_attempt_replayed"
)

var (
	DescriptorRateLimited = descriptor("common.rate_limited", http.StatusTooManyRequests)
	DescriptorValidation  = descriptor("common.validation_failed", http.StatusBadRequest)
	DescriptorInternal    = descriptor("common.internal", http.StatusInternalServerError)

	descriptors = map[string]problem.Descriptor{
		CodeNotFound:                 descriptor(CodeNotFound, http.StatusNotFound),
		CodeForbidden:                descriptor(CodeForbidden, http.StatusForbidden),
		CodeSlugTaken:                descriptor(CodeSlugTaken, http.StatusConflict),
		CodeInvalidState:             descriptor(CodeInvalidState, http.StatusBadRequest),
		CodeInvalidInput:             descriptor(CodeInvalidInput, http.StatusBadRequest),
		CodeUpstreamFailed:           descriptor(CodeUpstreamFailed, http.StatusBadGateway),
		CodeAuthorizationUnavailable: descriptor(CodeAuthorizationUnavailable, http.StatusServiceUnavailable),
		CodeCommentNotFound:          descriptor(CodeCommentNotFound, http.StatusNotFound),
		CodeCommentsClosed:           descriptor(CodeCommentsClosed, http.StatusConflict),
		CodeCommentRejected:          descriptor(CodeCommentRejected, http.StatusUnprocessableEntity),
		CodeRateLimited:              descriptor(CodeRateLimited, http.StatusTooManyRequests),
		CodeChallengeRequired:        descriptor(CodeChallengeRequired, http.StatusForbidden),
		CodeAbuseUnavailable:         descriptor(CodeAbuseUnavailable, http.StatusServiceUnavailable),
		CodeAbuseReplay:              descriptor(CodeAbuseReplay, http.StatusConflict),
	}
)

func descriptor(code string, status int) problem.Descriptor {
	return problem.MustDescriptor(
		problem.MustKind(code, status),
		"https://errors.yueli.dev/problems/"+code,
	)
}

func DescriptorForCode(code string) (problem.Descriptor, bool) {
	value, ok := descriptors[code]
	return value, ok
}

type CatalogEntry struct {
	Code   string `json:"code"`
	Status int    `json:"status"`
}

func Catalog() []CatalogEntry {
	result := make([]CatalogEntry, 0, len(descriptors))
	for code, value := range descriptors {
		result = append(result, CatalogEntry{Code: code, Status: value.Kind().Status()})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return result
}

func mapped(code string, params problem.Parameters) error {
	value, ok := DescriptorForCode(code)
	if !ok {
		return fmt.Errorf("blog public error code is not declared: %s", code)
	}
	result, err := problem.NewError(value, params)
	if err != nil {
		return fmt.Errorf("blog public error %s: %w", code, err)
	}
	return result
}

func NotFound(id string) error {
	return mapped(CodeNotFound, map[string]any{"id": id})
}

func Forbidden() error { return mapped(CodeForbidden, nil) }

func AuthorizationUnavailable() error {
	return mapped(CodeAuthorizationUnavailable, nil)
}

func SlugTaken(slug string) error {
	return mapped(CodeSlugTaken, map[string]any{"slug": slug})
}

func InvalidState(detail string) error {
	return mapped(CodeInvalidState, map[string]any{"detail": detail})
}

func InvalidInput(detail string) error {
	return mapped(CodeInvalidInput, map[string]any{"detail": detail})
}

func UpstreamFailed(summary string) error {
	return mapped(CodeUpstreamFailed, map[string]any{"detail": summary})
}

func CommentNotFound(id string) error {
	return mapped(CodeCommentNotFound, map[string]any{"id": id})
}

func CommentsClosed() error {
	return mapped(CodeCommentsClosed, nil)
}

func CommentRejected() error {
	return mapped(CodeCommentRejected, nil)
}

func RateLimited() error {
	return mapped(CodeRateLimited, nil)
}

func ChallengeRequired(attemptID string) error {
	return mapped(CodeChallengeRequired, map[string]any{
		"attemptId": attemptID, "challenge": "turnstile",
	})
}

func AbuseUnavailable() error {
	return mapped(CodeAbuseUnavailable, nil)
}

func AbuseAttemptReplayed() error {
	return mapped(CodeAbuseReplay, nil)
}
