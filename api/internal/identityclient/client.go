// Package identityclient resolves public user display data (name / avatar /
// cover / bio / social) from the identity service's public profiles API. The
// identity profile is the single source of truth for "who a person is"; the blog
// keeps only domain-specific author state (role / status) locally and overlays
// the display data fetched here when rendering author pages and bylines.
package identityclient

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
)

// SocialLink mirrors the identity public profile's social link.
type SocialLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// Profile is the public display subset of an identity (never email/roles/status).
type Profile struct {
	ID          string
	DisplayName string
	AvatarURL   string
	CoverURL    string
	Bio         string
	SocialLinks []SocialLink
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

func parseProfile(j *gjson.Json) Profile {
	p := Profile{
		ID:          j.Get("id").String(),
		DisplayName: j.Get("displayName").String(),
		AvatarURL:   j.Get("avatarUrl").String(),
		CoverURL:    j.Get("coverUrl").String(),
		Bio:         j.Get("bio").String(),
	}
	for _, l := range j.Get("socialLinks").Array() {
		lj := gjson.New(l)
		p.SocialLinks = append(p.SocialLinks, SocialLink{Label: lj.Get("label").String(), URL: lj.Get("url").String()})
	}
	return p
}

func (c *httpClient) Get(ctx context.Context, id string) Profile {
	if id == "" {
		return Profile{}
	}
	resp, err := g.Client().Get(ctx, c.base+"/api/v1/profiles/"+id)
	if err != nil {
		return Profile{ID: id}
	}
	defer resp.Close()
	j := gjson.New(resp.ReadAllString())
	if j.Get("code").String() != "ok" {
		return Profile{ID: id}
	}
	return parseProfile(j.GetJson("data.profile"))
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
	resp, err := g.Client().Get(ctx, c.base+"/api/v1/profiles", g.Map{"ids": strings.Join(uniq, ",")})
	if err != nil {
		return out
	}
	defer resp.Close()
	j := gjson.New(resp.ReadAllString())
	if j.Get("code").String() != "ok" {
		return out
	}
	for _, item := range j.Get("data.profiles").Array() {
		p := parseProfile(gjson.New(item))
		if p.ID != "" {
			out[p.ID] = p
		}
	}
	return out
}
