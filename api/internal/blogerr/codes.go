// Package blogerr declares Blog's immutable public Problem contract.
package blogerr

import (
	"fmt"
	"net/http"

	"github.com/yueli-official/foundation/go/problem"
)

var (
	DescriptorRateLimited = descriptor("common.rate_limited", http.StatusTooManyRequests)
	DescriptorValidation  = descriptor("common.validation_failed", http.StatusBadRequest)
	DescriptorInternal    = descriptor("common.internal", http.StatusInternalServerError)
)

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

func InitialAdministratorAlreadyClaimed() error {
	return mapped(CodeInitialAdministratorClaimed, nil)
}

func SlugTaken(slug string) error {
	return mapped(CodeSlugTaken, map[string]any{"slug": slug})
}

func InvalidState(detail string) error {
	return mapped(CodeInvalidState, map[string]any{"reason": detail})
}

func InvalidInput(detail string) error {
	return mapped(CodeInvalidInput, map[string]any{"reason": detail})
}

func UpstreamFailed(dependency string) error {
	return mapped(CodeUpstreamFailed, map[string]any{"dependency": dependency})
}

func AssetTooLarge(maxBytes int64) error {
	params := map[string]any{}
	if maxBytes > 0 {
		params["maxBytes"] = maxBytes
	}
	return mapped(CodeAssetTooLarge, params)
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
