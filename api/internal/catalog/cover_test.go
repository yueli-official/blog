package catalog

import (
	"testing"

	"github.com/yueli-official/blog/api/internal/blogclient"
)

func TestPublicCoverURLIgnoresBackendCDNOrigin(t *testing.T) {
	got, err := publicCoverURL(blogclient.View{
		MediaKey: "34bWyYVg9lhrqru6RsNny",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/media/34bWyYVg9lhrqru6RsNny?format=webp&name=home&v=1" {
		t.Fatalf("cover URL = %q", got)
	}
}

func TestPublicCoverURLRequiresMediaKey(t *testing.T) {
	if _, err := publicCoverURL(blogclient.View{}); err == nil {
		t.Fatal("cover URL accepted a backend URL without media key")
	}
}
