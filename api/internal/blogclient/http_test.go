package blogclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/yueli-official/foundation/go/problem"
)

func TestHTTPClientUsesConfiguredSiteContextForAssetWrites(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			bodies = append(bodies, map[string]any{"siteKey": r.URL.Query().Get("siteKey")})
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
			return
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			values, parseErr := url.ParseQuery(string(raw))
			if parseErr != nil {
				t.Errorf("decode request: json=%v form=%v", err, parseErr)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			body = make(map[string]any, len(values))
			for key := range values {
				body[key] = values.Get(key)
			}
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/assets/upload-init" {
			_, _ = w.Write([]byte(`{"uploadUrl":"http://upload.test","uploadToken":"token"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewHTTP(server.URL, "blog-ai", "yueli")
	if _, err := client.UploadInit(context.Background(), "token", InitInput{
		Filename: "cover.png", Mime: "image/png", Category: "blog-cover", Visibility: "public", Size: 12,
	}); err != nil {
		t.Fatalf("UploadInit() error = %v", err)
	}
	if err := client.RegisterReference(context.Background(), "token", ReferenceInput{
		AssetID: "asset-1", RefType: "post-cover", RefID: "post-1",
	}); err != nil {
		t.Fatalf("RegisterReference() error = %v", err)
	}
	if err := client.UnregisterReference(context.Background(), "token", ReferenceInput{
		AssetID: "asset-1", RefType: "post-cover", RefID: "post-1",
	}); err != nil {
		t.Fatalf("UnregisterReference() error = %v", err)
	}

	if len(bodies) != 3 {
		t.Fatalf("request count = %d, want 3", len(bodies))
	}
	for i, body := range bodies {
		if body["siteKey"] != "blog-ai" {
			t.Fatalf("request %d siteKey = %#v, want blog-ai", i, body["siteKey"])
		}
	}
	if bodies[0]["spaceKey"] != "yueli" {
		t.Fatalf("upload spaceKey = %#v, want yueli", bodies[0]["spaceKey"])
	}
}

func TestHTTPClientPreservesAssetUploadLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.Header().Set("X-Trace-Id", "asset-limit")
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_, _ = w.Write([]byte(`{"type":"https://errors.yueli.dev/problems/asset.upload.too_large","status":413,"code":"asset.upload.too_large","params":{"maxBytes":10485760},"traceId":"asset-limit"}`))
	}))
	defer server.Close()

	client := NewHTTP(server.URL, "blog-main", "yueli")
	_, err := client.UploadInit(context.Background(), "token", InitInput{
		Filename: "large.png", Mime: "image/png", Category: "blog-cover", Size: 11 << 20,
	})
	if err == nil {
		t.Fatal("UploadInit() error = nil")
	}
	value, ok, resolveErr := problem.FromError(err, "blog-limit")
	if !ok || resolveErr != nil {
		t.Fatalf("mapped error did not resolve: ok=%v err=%v", ok, resolveErr)
	}
	if value.Code != "blog.asset_too_large" || value.Status != http.StatusRequestEntityTooLarge {
		t.Fatalf("problem = %#v", value)
	}
	if got := value.Params["maxBytes"]; got != int64(10<<20) {
		t.Fatalf("maxBytes = %#v, want %d", got, 10<<20)
	}
}
