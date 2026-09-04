package blogclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	foundationhttpclient "github.com/yueli-official/foundation/go/httpclient"

	"github.com/yueli-official/blog/api/internal/blogerr"
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
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(raw))
	if err != nil {
		return nil, blogerr.UpstreamFailed("asset")
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, blogerr.UpstreamFailed("asset")
	}
	defer resp.Body.Close()
	out, err := foundationhttpclient.DecodeJSON[map[string]any](resp, foundationhttpclient.Limits{})
	if err != nil {
		return nil, mapRemoteError(err)
	}
	return gjson.New(out), nil
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
		UploadURL:     j.Get("uploadUrl").String(),
		UploadToken:   j.Get("uploadToken").String(),
		UploadHeaders: stringMap(j.Get("uploadHeaders").Map()),
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
		ID: j.Get("asset.id").String(), MediaKey: j.Get("asset.mediaKey").String(),
		Size: j.Get("asset.size").Int64(), Mime: j.Get("asset.mime").String(),
		Filename: j.Get("asset.filename").String(),
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
	q := url.Values{}
	q.Set("assetId", in.AssetID)
	q.Set("siteKey", c.siteSlug)
	q.Set("refType", in.RefType)
	q.Set("refId", in.RefID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.base+"/api/v1/asset-references?"+q.Encode(), nil)
	if err != nil {
		return blogerr.UpstreamFailed("asset")
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return blogerr.UpstreamFailed("asset")
	}
	defer resp.Body.Close()
	if _, err := foundationhttpclient.DecodeJSON[any](resp, foundationhttpclient.Limits{}); err != nil {
		return blogerr.UpstreamFailed("asset")
	}
	return nil
}

func (c *httpClient) Delete(ctx context.Context, bearer, assetID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.base+"/api/v1/assets/"+assetID, nil)
	if err != nil {
		return blogerr.UpstreamFailed("asset")
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return blogerr.UpstreamFailed("asset")
	}
	defer resp.Body.Close()
	if _, err := foundationhttpclient.DecodeJSON[any](resp, foundationhttpclient.Limits{}); err != nil {
		return blogerr.UpstreamFailed("asset")
	}
	return nil
}

func remoteCode(err error) string {
	var remote *foundationhttpclient.RemoteError
	if errors.As(err, &remote) {
		return remote.Problem.Code
	}
	return "foundation.response.invalid"
}

func mapRemoteError(err error) error {
	var remote *foundationhttpclient.RemoteError
	if errors.As(err, &remote) && remote.Problem.Code == "asset.upload.too_large" {
		return blogerr.AssetTooLarge(g.NewVar(remote.Problem.Params["maxBytes"]).Int64())
	}
	return blogerr.UpstreamFailed("asset")
}
