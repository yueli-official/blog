// Package coverurl owns Blog's narrow adapter from Asset cover facts to
// browser-safe same-origin URLs. It never decodes media keys or storage keys.
package coverurl

import (
	"net/url"
	"strings"
)

func FromMediaKey(mediaKey string) string {
	mediaKey = strings.TrimSpace(mediaKey)
	if mediaKey == "" {
		return ""
	}
	return "/media/" + url.PathEscape(mediaKey) + "?format=webp&name=home&v=1"
}

// NormalizeManaged repairs the short-lived pre-media contract that persisted a
// backend public URL. Only managed covers and Asset's public object prefix are
// eligible; arbitrary external images are not proxied.
func NormalizeManaged(assetID, raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.TrimSpace(assetID) == "" || raw == "" || strings.HasPrefix(raw, "/media/") {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" || !strings.HasPrefix(parsed.EscapedPath(), "/public/") {
		return ""
	}
	return "/asset-api/assets" + parsed.EscapedPath()
}
