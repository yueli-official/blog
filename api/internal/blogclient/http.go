package blogclient

import (
	"context"
	"net/url"
	"strings"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"

	"platform/products/blog/api/internal/blogerr"
)

// httpClient is the real asset client, talking to the asset service over HTTP.
// Upload/finalize/delete forward the caller's bearer token (author == asset
// owner). Covers are public, so there is no signed-delivery path.
type httpClient struct {
	base     string
	siteSlug string
	spaceKey string
}

// NewHTTP builds an HTTP-backed asset client rooted at the asset service base URL.
func NewHTTP(baseURL, siteSlug, spaceKey string) Client {
	return &httpClient{base: strings.TrimRight(baseURL, "/"), siteSlug: siteSlug, spaceKey: spaceKey}
}

func (c *httpClient) post(ctx context.Context, bearer, path string, body g.Map) (*gjson.Json, error) {
	cli := g.Client()
	cli.SetHeader("Authorization", "Bearer "+bearer)
	cli.ContentJson()
	resp, err := cli.Post(ctx, c.base+path, body)
	if err != nil {
		return nil, blogerr.UpstreamFailed("asset service unreachable")
	}
	defer resp.Close()
	j := gjson.New(resp.ReadAllString())
	if code := j.Get("code").String(); code != "ok" {
		return nil, blogerr.UpstreamFailed(code)
	}
	return j, nil
}

func (c *httpClient) UploadInit(ctx context.Context, bearer string, in InitInput) (InitOutput, error) {
	j, err := c.post(ctx, bearer, "/api/v1/assets/upload-init", g.Map{
		"filename": in.Filename, "mime": in.Mime, "size": in.Size,
		"category": in.Category, "spaceKey": c.spaceKey, "siteKey": c.siteSlug, "profileKey": in.Category,
		"visibility": in.Visibility,
	})
	if err != nil {
		return InitOutput{}, err
	}
	return InitOutput{
		UploadURL:     j.Get("data.uploadUrl").String(),
		UploadToken:   j.Get("data.uploadToken").String(),
		UploadHeaders: stringMap(j.Get("data.uploadHeaders").Map()),
	}, nil
}

func stringMap(raw map[string]any) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k] = g.NewVar(v).String()
	}
	return out
}

func (c *httpClient) Finalize(ctx context.Context, bearer, uploadToken string) (View, error) {
	j, err := c.post(ctx, bearer, "/api/v1/assets/finalize", g.Map{"uploadToken": uploadToken})
	if err != nil {
		return View{}, err
	}
	return View{
		ID: j.Get("data.asset.id").String(), CdnURL: j.Get("data.asset.cdnUrl").String(),
		Size: j.Get("data.asset.size").Int64(), Mime: j.Get("data.asset.mime").String(),
		Filename: j.Get("data.asset.filename").String(),
	}, nil
}

func (c *httpClient) RegisterReference(ctx context.Context, bearer string, in ReferenceInput) error {
	_, err := c.post(ctx, bearer, "/api/v1/asset-references", g.Map{
		"assetId": in.AssetID, "siteKey": c.siteSlug, "refType": in.RefType, "refId": in.RefID,
		"refLabel": in.RefLabel, "refUrl": in.RefURL,
	})
	return err
}

func (c *httpClient) UnregisterReference(ctx context.Context, bearer string, in ReferenceInput) error {
	cli := g.Client()
	cli.SetHeader("Authorization", "Bearer "+bearer)
	q := url.Values{}
	q.Set("assetId", in.AssetID)
	q.Set("siteKey", c.siteSlug)
	q.Set("refType", in.RefType)
	q.Set("refId", in.RefID)
	resp, err := cli.Delete(ctx, c.base+"/api/v1/asset-references?"+q.Encode())
	if err != nil {
		return blogerr.UpstreamFailed("asset service unreachable")
	}
	defer resp.Close()
	if code := gjson.New(resp.ReadAllString()).Get("code").String(); code != "ok" {
		return blogerr.UpstreamFailed(code)
	}
	return nil
}

func (c *httpClient) Delete(ctx context.Context, bearer, assetID string) error {
	cli := g.Client()
	cli.SetHeader("Authorization", "Bearer "+bearer)
	resp, err := cli.Delete(ctx, c.base+"/api/v1/assets/"+assetID)
	if err != nil {
		return blogerr.UpstreamFailed("asset service unreachable")
	}
	defer resp.Close()
	if code := gjson.New(resp.ReadAllString()).Get("code").String(); code != "ok" {
		return blogerr.UpstreamFailed(code)
	}
	return nil
}
