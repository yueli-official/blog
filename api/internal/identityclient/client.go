// Package identityclient resolves public user display data (name / media /
// bio / social) from the identity service's public users API. The
// identity profile is the single source of truth for "who a person is"; the blog
// keeps only domain-specific author state (role / status) locally and overlays
// the display data fetched here when rendering author pages and bylines.
package identityclient

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	foundationhttpclient "github.com/yueli-official/foundation/go/httpclient"
)

// SocialLink mirrors the Identity public user's social link.
type SocialLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type MediaRef struct {
	MediaKey string `json:"mediaKey"`
}

// PublicUser is the public display subset of a user (never email/roles/status).
type PublicUser struct {
	UserKey     string       `json:"userKey"`
	Handle      string       `json:"handle"`
	DisplayName string       `json:"displayName"`
	Avatar      *MediaRef    `json:"avatar"`
	Cover       *MediaRef    `json:"cover"`
	Bio         string       `json:"bio"`
	SocialLinks []SocialLink `json:"socialLinks"`
}

// Client resolves Identity public users. A zero result is valid (the UI falls
// back to the public user key), so lookups never surface an error to callers.
type Client interface {
	Get(ctx context.Context, userKey string) PublicUser
	GetMany(ctx context.Context, userKeys []string) map[string]PublicUser
}

type httpClient struct{ base string }

// NewHTTP builds a client rooted at the identity service base URL (e.g. http://localhost:8081).
func NewHTTP(baseURL string) Client { return &httpClient{base: strings.TrimRight(baseURL, "/")} }

func (c *httpClient) Get(ctx context.Context, userKey string) PublicUser {
	if userKey == "" {
		return PublicUser{}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/v1/users/"+url.PathEscape(userKey), nil)
	if err != nil {
		return PublicUser{UserKey: userKey}
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return PublicUser{UserKey: userKey}
	}
	defer response.Body.Close()
	out, err := foundationhttpclient.DecodeJSON[struct {
		User PublicUser `json:"user"`
	}](response, foundationhttpclient.Limits{})
	if err != nil || out.User.UserKey == "" {
		return PublicUser{UserKey: userKey}
	}
	return out.User
}

func (c *httpClient) GetMany(ctx context.Context, userKeys []string) map[string]PublicUser {
	out := make(map[string]PublicUser, len(userKeys))
	uniq := make([]string, 0, len(userKeys))
	seen := map[string]bool{}
	for _, userKey := range userKeys {
		if userKey != "" && !seen[userKey] {
			seen[userKey] = true
			uniq = append(uniq, userKey)
		}
	}
	if len(uniq) == 0 {
		return out
	}
	query := url.Values{"ids": {strings.Join(uniq, ",")}}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/v1/users?"+query.Encode(), nil)
	if err != nil {
		return out
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return out
	}
	defer response.Body.Close()
	result, err := foundationhttpclient.DecodeJSON[struct {
		Items []PublicUser `json:"items"`
	}](response, foundationhttpclient.Limits{})
	if err != nil {
		return out
	}
	for _, user := range result.Items {
		if user.UserKey != "" {
			out[user.UserKey] = user
		}
	}
	return out
}
