package blogdiscovery

import (
	"strings"
	"testing"
	"time"

	"github.com/gogf/gf/v2/os/gtime"
	"github.com/yueli-official/foundation/go/discovery"

	"github.com/yueli-official/blog/api/internal/model"
)

func TestProjectPostUsesPersistedSEOThroughOneProjection(t *testing.T) {
	module := discovery.MustCompile(discovery.Definition{
		ContractVersion: discovery.ContractVersion,
		Site: discovery.SiteProfile{
			Origin: "https://blog.example", Name: "Blog", DefaultLocale: "zh-CN",
		},
	})
	published := gtime.NewFromTime(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	updated := gtime.NewFromTime(time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC))
	projection, err := ProjectPost(module, &model.Post{
		ID: "one", AuthorID: "author", Title: "Title", Slug: "one",
		Content: "Body", Status: model.StatusPublished, Locale: "zh-CN",
		PublishedAt: published, UpdatedAt: updated,
	}, &model.SEO{
		MetaTitle: "SEO Title", MetaDesc: "SEO description",
		CanonicalURL: "https://blog.example/posts/canonical",
		Robots:       "noindex,nofollow",
	}, "Author")
	if err != nil {
		t.Fatal(err)
	}
	if projection.Head.Title != "SEO Title" ||
		projection.CanonicalURL != "https://blog.example/posts/canonical" ||
		projection.Headers.XRobotsTag != "noindex,nofollow" ||
		projection.Sitemap != nil {
		t.Fatalf("SEO facts drifted: %#v", projection)
	}
	if !strings.Contains(string(projection.Head.StructuredData[0].JSON), `"Article"`) {
		t.Fatalf("article schema is missing: %s", projection.Head.StructuredData[0].JSON)
	}
}
