package coverurl

import "testing"

func TestFromMediaKey(t *testing.T) {
	if got := FromMediaKey("34bWyYVg9lhrqru6RsNny"); got != "/media/34bWyYVg9lhrqru6RsNny?format=webp&preset=home&v=1" {
		t.Fatalf("media URL = %q", got)
	}
}

func TestNormalizeManagedBackendURL(t *testing.T) {
	got := NormalizeManaged("asset-1", "https://bucket.cos.ap-shanghai.myqcloud.com/public/default/blog/blog-cover/cover.webp")
	if got != "/asset-api/assets/public/default/blog/blog-cover/cover.webp" {
		t.Fatalf("normalized URL = %q", got)
	}
}

func TestNormalizeManagedRejectsArbitraryExternalURL(t *testing.T) {
	if got := NormalizeManaged("asset-1", "https://example.com/avatar.webp"); got != "" {
		t.Fatalf("arbitrary URL was proxied: %q", got)
	}
}
