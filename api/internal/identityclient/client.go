// Package identityclient resolves public user display data (name / avatar /
// cover / bio / social) from the identity service's public profiles API. The
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

// SocialLink mirrors the identity public profile's social link.
type SocialLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// Profile is the public display subset of an identity (never email/roles/status).
type Profile struct {
	ID          string       `json:"id"`
	DisplayName string       `json:"displayName"`
	AvatarURL   string       `json:"avatarUrl"`
	CoverURL    string       `json:"coverUrl"`
	Bio         string       `json:"bio"`
	SocialLinks []SocialLink `json:"socialLinks"`
}

// Client resolves identity display profiles. A nil/zero result is valid (the UI
// falls back to the bare id), so lookups never surface an error to callers.
type Client interface {
	Get(ctx context.Context, id string) Profile
	GetMany(ctx context.Context, ids []string) map[string]Profile
}

type httpClient struct{ base string }

// NewHTTP builds a client rooted at the identity service base URL (e.g. http://localhost:8081).
func NewHTTP(baseURL string) Client { return &httpClient{base: strings.TrimRight(baseURL, "/")} }

func (c *httpClient) Get(ctx context.Context, id string) Profile {
	if id == "" {
		return Profile{}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/v1/profiles/"+url.PathEscape(id), nil)
	if err != nil {
		return Profile{ID: id}
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return Profile{ID: id}
	}
	defer response.Body.Close()
	out, err := foundationhttpclient.DecodeJSON[struct {
		Profile Profile `json:"profile"`
	}](response, foundationhttpclient.Limits{})
	if err != nil || out.Profile.ID == "" {
		return Profile{ID: id}
	}
	return out.Profile
}

func (c *httpClient) GetMany(ctx context.Context, ids []string) map[string]Profile {
	out := make(map[string]Profile, len(ids))
	uniq := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if id != "" && !seen[id] {
			seen[id] = true
			uniq = append(uniq, id)
		}
	}
	if len(uniq) == 0 {
		return out
	}
	query := url.Values{"ids": {strings.Join(uniq, ",")}}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/v1/profiles?"+query.Encode(), nil)
	if err != nil {
		return out
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return out
	}
	defer response.Body.Close()
	result, err := foundationhttpclient.DecodeJSON[struct {
		Profiles []Profile `json:"profiles"`
	}](response, foundationhttpclient.Limits{})
	if err != nil {
		return out
	}
	for _, profile := range result.Profiles {
		if profile.ID != "" {
			out[profile.ID] = profile
		}
	}
	return out
}
