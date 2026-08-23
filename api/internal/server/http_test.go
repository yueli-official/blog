package server_test

// Full HTTP path against a loopback server + live PG (database `blog`). Skipped
// unless BLOG_PG_HOST is set:
//
//	BLOG_PG_HOST=192.168.5.5 BLOG_PG_USER=postgres BLOG_PG_PASS=postgres \
//	  go test -run TestBlogHTTPRoundTrip ./products/blog/api/internal/server/...

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/gogf/gf/contrib/drivers/pgsql/v2"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/gclient"
	"github.com/gogf/gf/v2/os/gcfg"
	"github.com/gogf/gf/v2/test/gtest"
	_ "github.com/lib/pq"
	"github.com/yueli-official/foundation/go/traffic"

	"github.com/yueli-official/blog/api/internal/blogclient"
	"github.com/yueli-official/blog/api/internal/blogtraffic"
	"github.com/yueli-official/blog/api/internal/catalog"
	"github.com/yueli-official/blog/api/internal/dao"
	"github.com/yueli-official/blog/api/internal/server"
)

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// capMailer is a thread-safe in-memory mail.Sender for the test (the newsletter
// digest fires from a goroutine, hence the mutex).
type capMailer struct {
	mu   sync.Mutex
	sent []struct{ to, subject, body string }
}

func (c *capMailer) Send(_ context.Context, to, subject, htmlBody string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, struct{ to, subject, body string }{to, subject, htmlBody})
	return nil
}

func (c *capMailer) count() int { c.mu.Lock(); defer c.mu.Unlock(); return len(c.sent) }

func (c *capMailer) sentTo(to string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, m := range c.sent {
		if m.to == to {
			return true
		}
	}
	return false
}

func TestBlogHTTPRoundTrip(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		host := os.Getenv("BLOG_PG_HOST")
		if host == "" {
			t.Skip("set BLOG_PG_HOST to run the blog HTTP integration test")
		}
		port, user, pass := envOr("BLOG_PG_PORT", "5432"), envOr("BLOG_PG_USER", "postgres"), os.Getenv("BLOG_PG_PASS")
		ctx := context.Background()
		config, err := gcfg.NewAdapterContent("blog:\n  operatorSubs:\n    - \"" + testSub + "\"\n")
		t.AssertNil(err)
		previousConfig := g.Cfg().GetAdapter()
		g.Cfg().SetAdapter(config)
		defer g.Cfg().SetAdapter(previousConfig)

		sdb, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=blog sslmode=disable", host, port, user, pass))
		t.AssertNil(err)
		_, err = sdb.Exec(`
DROP TABLE IF EXISTS
  traffic_event_receipts, traffic_visitor_markers, traffic_daily,
  traffic_totals, traffic_baselines, traffic_instances,
  comments, subscribers, post_likes, post_bookmarks,
  blog_post_tag_assignments, blog_post_category_assignments,
  blog_classification_policy_profiles, blog_tag_lookup_entries,
  blog_tags, blog_categories, blog_classification_catalogs,
  post_revisions, post_metas, post_seo, post_stats, posts,
  series, author_profiles, home_config
CASCADE`)
		t.AssertNil(err)
		migrations, err := os.ReadDir("../../manifest/sql/migrations")
		t.AssertNil(err)
		for _, migration := range migrations {
			if migration.IsDir() || !strings.HasSuffix(migration.Name(), ".up.sql") {
				continue
			}
			up, readErr := os.ReadFile("../../manifest/sql/migrations/" + migration.Name())
			t.AssertNil(readErr)
			_, execErr := sdb.Exec(string(up))
			t.AssertNil(execErr)
		}
		_, err = sdb.Exec(`
INSERT INTO blog_classification_catalogs (id, catalog_key) VALUES ('01990000-0000-7000-8000-000000000001', 'blog');
INSERT INTO blog_classification_policy_profiles
    (catalog_id, policy_key, schema_version, policy_revision, category_policy, facet_policies, tag_policy, discovery_policy)
SELECT id, 'blog.post.default', 1, 1,
       '{"minAssignments":0,"maxAssignments":0,"requirePrimary":false,"leafOnly":false,"maxDepth":0}',
       '[]',
       '{"unknown":"create","minAssignments":0,"maxAssignments":0}',
       '{"defaultSort":"name_asc"}'
FROM blog_classification_catalogs WHERE catalog_key = 'blog';`)
		t.AssertNil(err)
		_, err = sdb.Exec(`
INSERT INTO author_profiles (author_id, role, status)
VALUES
  ($1, 'author', 'active'),
  ($2, 'contributor', 'active')`, testSub, testSub2)
		t.AssertNil(err)
		sdb.Close()

		db, err := gdb.New(gdb.ConfigNode{Type: "pgsql", Host: host, Port: port, User: user, Pass: pass, Name: "blog"})
		t.AssertNil(err)
		fake := blogclient.NewFake()
		cm := &capMailer{}
		cat := catalog.New(dao.NewPG(db), fake, "blog-cover", cm, "http://blog.test", catalog.SpamPolicy{})
		trafficCatalog := traffic.MustCompile(blogtraffic.Definition("UTC"))
		trafficModule, err := traffic.NewMemory(trafficCatalog, traffic.MemoryOptions{
			Secret: []byte("blog-http-test-visitor-secret-32-bytes"),
		})
		t.AssertNil(err)
		cat.SetTraffic(trafficModule)

		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		t.AssertNil(err)
		s := g.Server(t.Name())
		s.SetAddr("127.0.0.1:0")
		server.Configure(s, server.Deps{Verifier: mustVerifier(t, priv), Catalog: cat})
		s.SetDumpRouterMap(false)
		s.Start()
		defer s.Shutdown()
		base := prefix(s)
		exp := time.Now().UTC().Add(10 * time.Minute)
		jwt := signToken(t, priv, testSub, exp)
		jwtAdmin := signTokenRoles(t, priv, testSub, []string{"admin"}, exp)

		op := func() *gclient.Client {
			c := g.Client()
			c.SetPrefix(base)
			c.ContentJson()
			c.SetHeader("Authorization", "Bearer "+jwt)
			return c
		}
		adminOp := func() *gclient.Client {
			c := g.Client()
			c.SetPrefix(base)
			c.ContentJson()
			c.SetHeader("Authorization", "Bearer "+jwtAdmin)
			return c
		}
		anon := func() *gclient.Client { c := g.Client(); c.SetPrefix(base); return c }

		// 1. create draft
		rc, err := op().Post(ctx, "/api/v1/posts", g.Map{"title": "Hello World", "content": "body"})
		t.AssertNil(err)
		t.Assert(rc.StatusCode, 200)
		jc := gjson.New(rc.ReadAllString())
		rc.Close()
		id := jc.Get("post.id").String()
		t.AssertNE(id, "")
		t.Assert(jc.Get("post.status").String(), "draft")
		t.Assert(jc.Get("post.slug").String(), "hello-world")

		// 2. anon GET by slug → 404 (draft not public)
		r2, err := anon().Get(ctx, "/api/v1/posts/hello-world")
		t.AssertNil(err)
		t.Assert(r2.StatusCode, 404)
		r2.Close()

		// 3. publish
		rp, err := op().Patch(ctx, "/api/v1/posts/"+id, g.Map{"status": "published"})
		t.AssertNil(err)
		t.Assert(rp.StatusCode, 200)
		rp.Close()

		// 4. anon GET by slug → 200, title round-trips
		r4, err := anon().Get(ctx, "/api/v1/posts/hello-world")
		t.AssertNil(err)
		j4 := gjson.New(r4.ReadAllString())
		r4.Close()
		t.Assert(j4.Get("post.title").String(), "Hello World")
		t.AssertNE(j4.Get("post.publishedAt").String(), "")
		t.Assert(j4.Get("post.excerpt").String(), "body") // auto-derived (no author-set excerpt)

		// 4b. cover: init → finalize → detail shows coverUrl (asset public delivery via fake)
		rci, err := op().Post(ctx, "/api/v1/posts/"+id+"/cover", g.Map{"filename": "c.png", "size": 100})
		t.AssertNil(err)
		t.Assert(rci.StatusCode, 200)
		ctok := gjson.New(rci.ReadAllString()).Get("uploadToken").String()
		rci.Close()
		t.AssertNE(ctok, "")
		rcf, err := op().Post(ctx, "/api/v1/posts/"+id+"/cover/finalize", g.Map{"uploadToken": ctok})
		t.AssertNil(err)
		t.Assert(rcf.StatusCode, 200)
		coverURL := gjson.New(rcf.ReadAllString()).Get("coverUrl").String()
		rcf.Close()
		t.Assert(strings.HasPrefix(coverURL, "/media/"), true)
		t.Assert(strings.Contains(coverURL, "format=webp&name=home"), true)
		rcd, err := anon().Get(ctx, "/api/v1/posts/hello-world")
		t.AssertNil(err)
		t.AssertNE(gjson.New(rcd.ReadAllString()).Get("post.coverUrl").String(), "")
		rcd.Close()
		// upstream failure → 502 blog.upstream_failed
		fake.FailNext()
		rcfail, err := op().Post(ctx, "/api/v1/posts/"+id+"/cover", g.Map{"filename": "c.png", "size": 100})
		t.AssertNil(err)
		t.Assert(rcfail.StatusCode, 502)
		t.Assert(gjson.New(rcfail.ReadAllString()).Get("code").String(), "blog.upstream_failed")
		rcfail.Close()

		// 4b2. inline image upload (editor E2): init → finalize → public url
		rii, err := op().Post(ctx, "/api/v1/images", g.Map{"filename": "inline.png", "mime": "image/png", "size": 2048})
		t.AssertNil(err)
		t.Assert(rii.StatusCode, 200)
		itok := gjson.New(rii.ReadAllString()).Get("uploadToken").String()
		rii.Close()
		t.AssertNE(itok, "")
		rif, err := op().Post(ctx, "/api/v1/images/finalize", g.Map{"uploadToken": itok})
		t.AssertNil(err)
		t.Assert(rif.StatusCode, 200)
		inlineURL := gjson.New(rif.ReadAllString()).Get("url").String()
		t.Assert(strings.HasPrefix(inlineURL, "/media/"), true)
		t.Assert(strings.Contains(inlineURL, "format=webp&name=inline"), true)
		rif.Close()
		// anonymous cannot upload an inline image (JWT-gated)
		ria, err := anon().Post(ctx, "/api/v1/images", g.Map{"filename": "x.png", "size": 1})
		t.AssertNil(err)
		t.Assert(ria.StatusCode >= 400, true)
		ria.Close()

		// 4c. view counter: a replay is idempotent; two events → viewCount == 2.
		viewAt := time.Now().UTC().Format(time.RFC3339Nano)
		for _, eventID := range []string{
			"019c0000-0000-7000-8000-000000000001",
			"019c0000-0000-7000-8000-000000000001",
			"019c0000-0000-7000-8000-000000000002",
		} {
			rv, e := anon().Post(ctx, "/api/v1/posts/hello-world/view", g.Map{
				"eventId": eventID, "occurredAt": viewAt,
			})
			t.AssertNil(e)
			rv.Close()
		}
		rvd, err := anon().Get(ctx, "/api/v1/posts/hello-world")
		t.AssertNil(err)
		t.Assert(gjson.New(rvd.ReadAllString()).Get("post.viewCount").Int(), 2)
		rvd.Close()

		// 4d. revisions: two content edits → 2 revisions; restore oldest rolls back
		re2, err := op().Patch(ctx, "/api/v1/posts/"+id, g.Map{"content": "v2"})
		t.AssertNil(err)
		re2.Close()
		re3, err := op().Patch(ctx, "/api/v1/posts/"+id, g.Map{"content": "v3"})
		t.AssertNil(err)
		re3.Close()
		rrl, err := op().Get(ctx, "/api/v1/posts/"+id+"/revisions")
		t.AssertNil(err)
		jrl := gjson.New(rrl.ReadAllString())
		rrl.Close()
		revs := jrl.Get("items").Array()
		t.Assert(len(revs), 2)
		oldestRev := jrl.Get("items.1.id").String() // newest-first → index 1 is the "body" snapshot
		t.AssertNE(oldestRev, "")
		rrs, err := op().Post(ctx, "/api/v1/posts/"+id+"/revisions/"+oldestRev+"/restore", g.Map{})
		t.AssertNil(err)
		t.Assert(rrs.StatusCode, 200)
		t.Assert(gjson.New(rrs.ReadAllString()).Get("post.content").String(), "body")
		rrs.Close()

		// 5. anon list → total >= 1
		rl, err := anon().Get(ctx, "/api/v1/posts")
		t.AssertNil(err)
		t.Assert(gjson.New(rl.ReadAllString()).Get("total").Int() >= 1, true)
		rl.Close()

		// 6. publish a post with empty content → 400 blog.invalid_state
		re, err := op().Post(ctx, "/api/v1/posts", g.Map{"title": "Empty"})
		t.AssertNil(err)
		emptyID := gjson.New(re.ReadAllString()).Get("post.id").String()
		re.Close()
		rep, err := op().Patch(ctx, "/api/v1/posts/"+emptyID, g.Map{"status": "published"})
		t.AssertNil(err)
		t.Assert(rep.StatusCode, 400)
		t.Assert(gjson.New(rep.ReadAllString()).Get("code").String(), "blog.invalid_state")
		rep.Close()

		// 6b. batch publish applies the same constraint and reports partial failures.
		readyRes, err := op().Post(ctx, "/api/v1/posts", g.Map{"title": "Batch Ready", "content": "ready body"})
		t.AssertNil(err)
		readyID := gjson.New(readyRes.ReadAllString()).Get("post.id").String()
		readyRes.Close()
		batchRes, err := op().Post(ctx, "/api/v1/posts/batch", g.Map{"ids": []string{readyID, emptyID}, "action": "publish"})
		t.AssertNil(err)
		t.Assert(batchRes.StatusCode, 200)
		batchJSON := gjson.New(batchRes.ReadAllString())
		batchRes.Close()
		t.Assert(batchJSON.Get("changed").Int(), 1)
		t.Assert(len(batchJSON.Get("failures").Array()), 1)
		t.Assert(batchJSON.Get("failures.0.id").String(), emptyID)
		t.Assert(batchJSON.Get("failures.0.code").String(), "incomplete")
		batchReadyPublic, err := anon().Get(ctx, "/api/v1/posts/batch-ready")
		t.AssertNil(err)
		t.Assert(batchReadyPublic.StatusCode, 200)
		batchReadyPublic.Close()
		emptyPublic, err := anon().Get(ctx, "/api/v1/posts/empty")
		t.AssertNil(err)
		t.Assert(emptyPublic.StatusCode, 404)
		emptyPublic.Close()

		// 7. owner isolation: testSub2 cannot patch testSub's post
		jwt2 := signToken(t, priv, testSub2, time.Now().UTC().Add(10*time.Minute))
		c2 := g.Client()
		c2.SetPrefix(base)
		c2.ContentJson()
		c2.SetHeader("Authorization", "Bearer "+jwt2)
		r7, err := c2.Patch(ctx, "/api/v1/posts/"+id, g.Map{"title": "hijack"})
		t.AssertNil(err)
		t.Assert(r7.StatusCode, 404)
		r7.Close()

		// 8. ListMine includes the draft "Empty"
		rm, err := op().Get(ctx, "/api/v1/posts/mine")
		t.AssertNil(err)
		jm := gjson.New(rm.ReadAllString())
		t.Assert(jm.Get("total").Int() >= 2, true)
		t.Assert(jm.Get("counts.issues").Int() >= 1, true)
		rm.Close()

		// 8b. taxonomy: create category (admin-only) → assign to published post → archive filter
		rtc, err := adminOp().Post(ctx, "/api/v1/taxonomies", g.Map{"name": "Tech", "taxonomy": "category"})
		t.AssertNil(err)
		t.Assert(rtc.StatusCode, 200)
		jtc := gjson.New(rtc.ReadAllString())
		rtc.Close()
		taxID := jtc.Get("taxonomy.id").String()
		t.AssertNE(taxID, "")
		t.Assert(jtc.Get("taxonomy.slug").String(), "tech")

		ra, err := op().Put(ctx, "/api/v1/posts/"+id+"/taxonomies", g.Map{"taxonomyIds": []string{taxID}})
		t.AssertNil(err)
		t.Assert(ra.StatusCode, 200)
		ra.Close()

		// management rows batch-hydrate taxonomy chips (no per-post fetch).
		rmt, err := op().Get(ctx, "/api/v1/posts/mine", g.Map{"q": "Hello"})
		t.AssertNil(err)
		jmt := gjson.New(rmt.ReadAllString())
		rmt.Close()
		t.Assert(jmt.Get("items.0.taxonomies.0.id").String(), taxID)
		t.Assert(jmt.Get("items.0.taxonomies.0.name").String(), "Tech")

		// unknown taxonomy id → 400 blog.invalid_input (not a 500 FK violation)
		rab, err := op().Put(ctx, "/api/v1/posts/"+id+"/taxonomies", g.Map{"taxonomyIds": []string{"00000000-0000-0000-0000-000000000000"}})
		t.AssertNil(err)
		t.Assert(rab.StatusCode, 400)
		t.Assert(gjson.New(rab.ReadAllString()).Get("code").String(), "blog.invalid_input")
		rab.Close()

		// filter published list by the category slug → contains the post
		rtf, err := anon().Get(ctx, "/api/v1/posts", g.Map{"taxonomy": "tech"})
		t.AssertNil(err)
		t.Assert(gjson.New(rtf.ReadAllString()).Get("total").Int(), 1)
		rtf.Close()

		// empty filter for an unknown taxonomy → zero
		rtz, err := anon().Get(ctx, "/api/v1/posts", g.Map{"taxonomy": "nope"})
		t.AssertNil(err)
		t.Assert(gjson.New(rtz.ReadAllString()).Get("total").Int(), 0)
		rtz.Close()

		// 8b2. site search (ILIKE q): title match → found; gibberish → none
		rsq, err := anon().Get(ctx, "/api/v1/posts", g.Map{"q": "Hello"})
		t.AssertNil(err)
		t.Assert(gjson.New(rsq.ReadAllString()).Get("total").Int() >= 1, true)
		rsq.Close()
		rsq0, err := anon().Get(ctx, "/api/v1/posts", g.Map{"q": "zzznomatchxyz"})
		t.AssertNil(err)
		t.Assert(gjson.New(rsq0.ReadAllString()).Get("total").Int(), 0)
		rsq0.Close()

		// list taxonomies of kind category → contains Tech
		rtl, err := anon().Get(ctx, "/api/v1/taxonomies", g.Map{"taxonomy": "category"})
		t.AssertNil(err)
		t.Assert(len(gjson.New(rtl.ReadAllString()).Get("items").Array()) >= 1, true)
		rtl.Close()

		// 8b3. related posts: a second published post sharing the Tech category
		// surfaces in hello-world's /related; the source post excludes itself.
		rr2, err := op().Post(ctx, "/api/v1/posts", g.Map{"title": "Related Two", "content": "more tech"})
		t.AssertNil(err)
		relID := gjson.New(rr2.ReadAllString()).Get("post.id").String()
		rr2.Close()
		t.AssertNE(relID, "")
		rr2p, err := op().Patch(ctx, "/api/v1/posts/"+relID, g.Map{"status": "published"})
		t.AssertNil(err)
		rr2p.Close()
		rr2a, err := op().Put(ctx, "/api/v1/posts/"+relID+"/taxonomies", g.Map{"taxonomyIds": []string{taxID}})
		t.AssertNil(err)
		t.Assert(rr2a.StatusCode, 200)
		rr2a.Close()
		rrel, err := anon().Get(ctx, "/api/v1/posts/hello-world/related")
		t.AssertNil(err)
		jrel := gjson.New(rrel.ReadAllString())
		rrel.Close()
		t.Assert(len(jrel.Get("items").Array()), 1)
		t.Assert(jrel.Get("items.0.slug").String(), "related-two") // self excluded

		// a published post with no taxonomy → empty related (early return path)
		rln, err := op().Post(ctx, "/api/v1/posts", g.Map{"title": "Lonely", "content": "no tags"})
		t.AssertNil(err)
		lonelyID := gjson.New(rln.ReadAllString()).Get("post.id").String()
		rln.Close()
		rlnp, err := op().Patch(ctx, "/api/v1/posts/"+lonelyID, g.Map{"status": "published"})
		t.AssertNil(err)
		rlnp.Close()
		rlnr, err := anon().Get(ctx, "/api/v1/posts/lonely/related")
		t.AssertNil(err)
		t.Assert(len(gjson.New(rlnr.ReadAllString()).Get("items").Array()), 0)
		rlnr.Close()

		// 8c. SEO: put → detail carries it
		rse, err := op().Put(ctx, "/api/v1/posts/"+id+"/seo", g.Map{"metaTitle": "MT", "metaDesc": "MD"})
		t.AssertNil(err)
		t.Assert(rse.StatusCode, 200)
		rse.Close()
		rsd, err := anon().Get(ctx, "/api/v1/posts/hello-world")
		t.AssertNil(err)
		t.Assert(gjson.New(rsd.ReadAllString()).Get("seo.metaTitle").String(), "MT")
		rsd.Close()

		// 8d. like toggle: like → likeCount 1 + liked true; unlike → 0 + false
		rlk, err := op().Post(ctx, "/api/v1/posts/hello-world/like", g.Map{})
		t.AssertNil(err)
		t.Assert(gjson.New(rlk.ReadAllString()).Get("liked").Bool(), true)
		rlk.Close()
		rld, err := op().Get(ctx, "/api/v1/posts/hello-world")
		t.AssertNil(err)
		jld := gjson.New(rld.ReadAllString())
		rld.Close()
		t.Assert(jld.Get("liked").Bool(), true)
		t.Assert(jld.Get("post.viewCount").Int(), 2) // unchanged by likes
		rlk2, err := op().Post(ctx, "/api/v1/posts/hello-world/like", g.Map{})
		t.AssertNil(err)
		t.Assert(gjson.New(rlk2.ReadAllString()).Get("liked").Bool(), false)
		rlk2.Close()

		// 8e. bookmark toggle
		rbk, err := op().Post(ctx, "/api/v1/posts/hello-world/bookmark", g.Map{})
		t.AssertNil(err)
		t.Assert(gjson.New(rbk.ReadAllString()).Get("bookmarked").Bool(), true)
		rbk.Close()
		rbd, err := op().Get(ctx, "/api/v1/posts/hello-world")
		t.AssertNil(err)
		t.Assert(gjson.New(rbd.ReadAllString()).Get("bookmarked").Bool(), true)
		rbd.Close()

		// 8f. comments: anon → pending (not public); author → auto-approved
		anonJSON := func() *gclient.Client { c := g.Client(); c.SetPrefix(base); c.ContentJson(); return c }
		ra1, err := anonJSON().Post(ctx, "/api/v1/posts/hello-world/comments", g.Map{"content": "anon hi", "authorName": "Guest"})
		t.AssertNil(err)
		t.Assert(ra1.StatusCode, 200)
		t.Assert(gjson.New(ra1.ReadAllString()).Get("pending").Bool(), true)
		ra1.Close()
		rcl0, err := anon().Get(ctx, "/api/v1/posts/hello-world/comments")
		t.AssertNil(err)
		t.Assert(gjson.New(rcl0.ReadAllString()).Get("total").Int(), 0) // anon is pending
		rcl0.Close()
		rco, err := op().Post(ctx, "/api/v1/posts/hello-world/comments", g.Map{"content": "author hi"})
		t.AssertNil(err)
		jco := gjson.New(rco.ReadAllString())
		rco.Close()
		t.Assert(jco.Get("pending").Bool(), false)
		topID := jco.Get("comment.id").String()
		t.AssertNE(topID, "")
		// author reply → approved, nested under the top comment
		rcr, err := op().Post(ctx, "/api/v1/posts/hello-world/comments", g.Map{"content": "a reply", "parentId": topID})
		t.AssertNil(err)
		t.Assert(gjson.New(rcr.ReadAllString()).Get("pending").Bool(), false)
		rcr.Close()
		rcl1, err := anon().Get(ctx, "/api/v1/posts/hello-world/comments")
		t.AssertNil(err)
		jcl1 := gjson.New(rcl1.ReadAllString())
		rcl1.Close()
		t.Assert(jcl1.Get("total").Int(), 1)
		t.Assert(jcl1.Get("items.0.content").String(), "author hi")
		t.Assert(jcl1.Get("items.0.isMember").Bool(), true)
		t.Assert(len(jcl1.Get("items.0.replies").Array()), 1)
		t.Assert(jcl1.Get("items.0.replies.0.content").String(), "a reply")
		// moderation: list mine filtered to pending → the anon comment
		rmp, err := op().Get(ctx, "/api/v1/comments/mine", g.Map{"status": 2})
		t.AssertNil(err)
		jmp := gjson.New(rmp.ReadAllString())
		rmp.Close()
		t.Assert(jmp.Get("total").Int(), 1)
		anonID := jmp.Get("items.0.id").String()
		t.AssertNE(anonID, "")
		t.Assert(jmp.Get("items.0.authorName").String(), "Guest")
		t.Assert(jmp.Get("items.0.postSlug").String(), "hello-world")
		// moderation search covers commenter, content and post identity.
		rsearch, err := op().Get(ctx, "/api/v1/comments/mine", g.Map{"status": 2, "keyword": "Guest"})
		t.AssertNil(err)
		t.Assert(gjson.New(rsearch.ReadAllString()).Get("total").Int(), 1)
		rsearch.Close()
		rmiss, err := op().Get(ctx, "/api/v1/comments/mine", g.Map{"status": 2, "keyword": "no-such-comment"})
		t.AssertNil(err)
		t.Assert(gjson.New(rmiss.ReadAllString()).Get("total").Int(), 0)
		rmiss.Close()
		// owner isolation: testSub2 cannot moderate testSub's comment
		rmod, err := c2.Patch(ctx, "/api/v1/comments/"+topID, g.Map{"status": 4})
		t.AssertNil(err)
		t.Assert(rmod.StatusCode, 403)
		rmod.Close()
		// approve the anon comment → a second approved top-level
		rap, err := op().Patch(ctx, "/api/v1/comments/"+anonID, g.Map{"status": 1})
		t.AssertNil(err)
		t.Assert(rap.StatusCode, 200)
		rap.Close()
		rcl2, err := anon().Get(ctx, "/api/v1/posts/hello-world/comments")
		t.AssertNil(err)
		t.Assert(gjson.New(rcl2.ReadAllString()).Get("total").Int(), 2)
		rcl2.Close()
		// comment_count synced (approved: author top + reply + anon = 3)
		cc, err := db.Model("post_stats").Ctx(ctx).Where("post_id", id).Value("comment_count")
		t.AssertNil(err)
		t.Assert(cc.Int(), 3)
		// delete author's top comment → cascade soft-deletes its reply
		rdc, err := op().Delete(ctx, "/api/v1/comments/"+topID)
		t.AssertNil(err)
		t.Assert(rdc.StatusCode, 200)
		rdc.Close()
		rcl3, err := anon().Get(ctx, "/api/v1/posts/hello-world/comments")
		t.AssertNil(err)
		t.Assert(gjson.New(rcl3.ReadAllString()).Get("total").Int(), 1) // only anon top remains
		rcl3.Close()
		cc2, err := db.Model("post_stats").Ctx(ctx).Where("post_id", id).Value("comment_count")
		t.AssertNil(err)
		t.Assert(cc2.Int(), 1)

		// 8g. newsletter double opt-in: subscribe → confirm → digest on publish → unsubscribe
		subEmail := "reader@example.com"
		rsub, err := anonJSON().Post(ctx, "/api/v1/subscribe", g.Map{"email": subEmail})
		t.AssertNil(err)
		t.Assert(rsub.StatusCode, 200)
		t.Assert(gjson.New(rsub.ReadAllString()).Get("pending").Bool(), true)
		rsub.Close()
		t.Assert(cm.sentTo(subEmail), true) // confirmation email sent
		subTokVar, err := db.Model("subscribers").Ctx(ctx).Where("email", subEmail).Fields("confirm_token").Value()
		t.AssertNil(err)
		subTok := subTokVar.String()
		t.AssertNE(subTok, "")
		// confirm via the token from the email link
		rcfm, err := anon().Get(ctx, "/api/v1/subscribe/confirm", g.Map{"token": subTok})
		t.AssertNil(err)
		t.Assert(rcfm.StatusCode, 200)
		t.Assert(gjson.New(rcfm.ReadAllString()).Get("email").String(), subEmail)
		rcfm.Close()
		stVar, _ := db.Model("subscribers").Ctx(ctx).Where("email", subEmail).Fields("status").Value()
		t.Assert(stVar.String(), "confirmed")
		// bad token → 400
		rbadt, err := anon().Get(ctx, "/api/v1/subscribe/confirm", g.Map{"token": "nope"})
		t.AssertNil(err)
		t.Assert(rbadt.StatusCode, 400)
		rbadt.Close()
		// digest: publishing a fresh post mails the confirmed subscriber (async hook)
		before := cm.count()
		rdp, err := op().Post(ctx, "/api/v1/posts", g.Map{"title": "Digest Post", "content": "hello subscribers"})
		t.AssertNil(err)
		digestID := gjson.New(rdp.ReadAllString()).Get("post.id").String()
		rdp.Close()
		rdpub, err := op().Patch(ctx, "/api/v1/posts/"+digestID, g.Map{"status": "published"})
		t.AssertNil(err)
		rdpub.Close()
		digested := false
		for i := 0; i < 20; i++ {
			if cm.count() > before {
				digested = true
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Assert(digested, true)
		// unsubscribe → status unsubscribed
		runsub, err := anonJSON().Post(ctx, "/api/v1/unsubscribe", g.Map{"token": subTok})
		t.AssertNil(err)
		t.Assert(runsub.StatusCode, 200)
		runsub.Close()
		stVar2, _ := db.Model("subscribers").Ctx(ctx).Where("email", subEmail).Fields("status").Value()
		t.Assert(stVar2.String(), "unsubscribed")

		// 9. trash → hidden publicly, listed in trash, restorable, then purgeable
		rd, err := op().Delete(ctx, "/api/v1/posts/"+id)
		t.AssertNil(err)
		t.Assert(rd.StatusCode, 200)
		rd.Close()
		r9, err := anon().Get(ctx, "/api/v1/posts/hello-world")
		t.AssertNil(err)
		t.Assert(r9.StatusCode, 404)
		r9.Close()

		trashList, err := op().Get(ctx, "/api/v1/posts/mine?status=trash")
		t.AssertNil(err)
		t.Assert(trashList.StatusCode, 200)
		trashJSON := gjson.New(trashList.ReadAllString())
		t.Assert(trashJSON.Get("total").Int(), 1)
		t.Assert(trashJSON.Get("items.0.id").String(), id)
		trashList.Close()

		restored, err := op().Post(ctx, "/api/v1/posts/"+id+"/restore", g.Map{})
		t.AssertNil(err)
		t.Assert(restored.StatusCode, 200)
		restored.Close()
		r9Restored, err := anon().Get(ctx, "/api/v1/posts/hello-world")
		t.AssertNil(err)
		t.Assert(r9Restored.StatusCode, 200)
		r9Restored.Close()

		rdAgain, err := op().Delete(ctx, "/api/v1/posts/"+id)
		t.AssertNil(err)
		t.Assert(rdAgain.StatusCode, 200)
		rdAgain.Close()
		purged, err := op().Delete(ctx, "/api/v1/posts/"+id+"/permanent")
		t.AssertNil(err)
		t.Assert(purged.StatusCode, 200)
		purged.Close()

		_, _ = db.Exec(ctx, "TRUNCATE posts CASCADE")
	})
}

// TestAdminModeratesAnyComment verifies that a superadmin can moderate comments
// on any post (bypassing the per-post ownership check), while a plain user who
// is not the post author still gets a 403.
func TestAdminModeratesAnyComment(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		host := os.Getenv("BLOG_PG_HOST")
		if host == "" {
			t.Skip("set BLOG_PG_HOST to run the blog HTTP integration test")
		}
		port, user, pass := envOr("BLOG_PG_PORT", "5432"), envOr("BLOG_PG_USER", "postgres"), os.Getenv("BLOG_PG_PASS")
		ctx := context.Background()

		sdb, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=blog sslmode=disable", host, port, user, pass))
		t.AssertNil(err)
		down, _ := os.ReadFile("../../manifest/sql/migrations/0001_init.down.sql")
		up, _ := os.ReadFile("../../manifest/sql/migrations/0001_init.up.sql")
		up2, _ := os.ReadFile("../../manifest/sql/migrations/0002_comments.up.sql")
		down2, _ := os.ReadFile("../../manifest/sql/migrations/0002_comments.down.sql")
		up3, _ := os.ReadFile("../../manifest/sql/migrations/0003_subscribers.up.sql")
		down3, _ := os.ReadFile("../../manifest/sql/migrations/0003_subscribers.down.sql")
		up4, _ := os.ReadFile("../../manifest/sql/migrations/0004_search_tsvector.up.sql")
		down4, _ := os.ReadFile("../../manifest/sql/migrations/0004_search_tsvector.down.sql")
		_, _ = sdb.Exec(string(down4))
		_, _ = sdb.Exec(string(down3))
		_, _ = sdb.Exec(string(down2))
		_, _ = sdb.Exec(string(down))
		_, err = sdb.Exec(string(up))
		t.AssertNil(err)
		_, err = sdb.Exec(string(up2))
		t.AssertNil(err)
		_, err = sdb.Exec(string(up3))
		t.AssertNil(err)
		_, err = sdb.Exec(string(up4))
		t.AssertNil(err)
		sdb.Close()

		db, err := gdb.New(gdb.ConfigNode{Type: "pgsql", Host: host, Port: port, User: user, Pass: pass, Name: "blog"})
		t.AssertNil(err)
		cat := catalog.New(dao.NewPG(db), blogclient.NewFake(), "blog-cover", &capMailer{}, "http://blog.test", catalog.SpamPolicy{})

		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		t.AssertNil(err)
		s := g.Server(t.Name())
		s.SetAddr("127.0.0.1:0")
		server.Configure(s, server.Deps{Verifier: mustVerifier(t, priv), Catalog: cat})
		s.SetDumpRouterMap(false)
		s.Start()
		defer s.Shutdown()
		base := prefix(s)

		exp := time.Now().UTC().Add(10 * time.Minute)
		// author A creates the post and comment
		jwtA := signToken(t, priv, testSub, exp)
		opA := func() *gclient.Client {
			c := g.Client()
			c.SetPrefix(base)
			c.ContentJson()
			c.SetHeader("Authorization", "Bearer "+jwtA)
			return c
		}
		// plain user B (not the post author)
		jwtB := signTokenRoles(t, priv, testSub2, []string{"user"}, exp)
		opB := func() *gclient.Client {
			c := g.Client()
			c.SetPrefix(base)
			c.ContentJson()
			c.SetHeader("Authorization", "Bearer "+jwtB)
			return c
		}
		// superadmin C
		jwtC := signTokenRoles(t, priv, "admin-C", []string{"admin"}, exp)
		opC := func() *gclient.Client {
			c := g.Client()
			c.SetPrefix(base)
			c.ContentJson()
			c.SetHeader("Authorization", "Bearer "+jwtC)
			return c
		}

		// author A creates and publishes a post
		rc, err := opA().Post(ctx, "/api/v1/posts", g.Map{"title": "Admin Test Post", "content": "some content"})
		t.AssertNil(err)
		postID := gjson.New(rc.ReadAllString()).Get("post.id").String()
		rc.Close()
		t.AssertNE(postID, "")
		rp, err := opA().Patch(ctx, "/api/v1/posts/"+postID, g.Map{"status": "published"})
		t.AssertNil(err)
		rp.Close()

		// author A posts a comment (auto-approved since they're the post author)
		rco, err := opA().Post(ctx, "/api/v1/posts/admin-test-post/comments", g.Map{"content": "A comment by author"})
		t.AssertNil(err)
		jco := gjson.New(rco.ReadAllString())
		rco.Close()
		commentID := jco.Get("comment.id").String()
		t.AssertNE(commentID, "")

		// plain user B (non-author, non-admin) tries to moderate → 403
		rmod, err := opB().Patch(ctx, "/api/v1/comments/"+commentID, g.Map{"status": 4})
		t.AssertNil(err)
		t.Assert(rmod.StatusCode, 403)
		rmod.Close()

		// superadmin C moderates the same comment → 200
		radm, err := opC().Patch(ctx, "/api/v1/comments/"+commentID, g.Map{"status": 4})
		t.AssertNil(err)
		t.Assert(radm.StatusCode, 200)
		radm.Close()

		_, _ = db.Exec(ctx, "TRUNCATE posts CASCADE")
	})
}

// TestBlogCommentGuard exercises the comment anti-spam policy (blacklist reject,
// link-count → pending, per-IP rate limit) against a live PG. Separate from the
// round-trip test so its strict policy doesn't perturb that flow.
func TestBlogCommentGuard(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		host := os.Getenv("BLOG_PG_HOST")
		if host == "" {
			t.Skip("set BLOG_PG_HOST to run the blog comment-guard integration test")
		}
		port, user, pass := envOr("BLOG_PG_PORT", "5432"), envOr("BLOG_PG_USER", "postgres"), os.Getenv("BLOG_PG_PASS")
		ctx := context.Background()

		sdb, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=blog sslmode=disable", host, port, user, pass))
		t.AssertNil(err)
		down, _ := os.ReadFile("../../manifest/sql/migrations/0001_init.down.sql")
		up, _ := os.ReadFile("../../manifest/sql/migrations/0001_init.up.sql")
		up2, _ := os.ReadFile("../../manifest/sql/migrations/0002_comments.up.sql")
		down2, _ := os.ReadFile("../../manifest/sql/migrations/0002_comments.down.sql")
		up3, _ := os.ReadFile("../../manifest/sql/migrations/0003_subscribers.up.sql")
		down3, _ := os.ReadFile("../../manifest/sql/migrations/0003_subscribers.down.sql")
		_, _ = sdb.Exec(string(down3))
		_, _ = sdb.Exec(string(down2))
		_, _ = sdb.Exec(string(down))
		_, err = sdb.Exec(string(up))
		t.AssertNil(err)
		_, err = sdb.Exec(string(up2))
		t.AssertNil(err)
		_, err = sdb.Exec(string(up3))
		t.AssertNil(err)
		sdb.Close()

		db, err := gdb.New(gdb.ConfigNode{Type: "pgsql", Host: host, Port: port, User: user, Pass: pass, Name: "blog"})
		t.AssertNil(err)
		policy := catalog.SpamPolicy{Blacklist: []string{"casino"}, MaxLinks: 2, RatePerWindow: 3, RateWindowSeconds: 60}
		cat := catalog.New(dao.NewPG(db), blogclient.NewFake(), "blog-cover", &capMailer{}, "http://blog.test", policy)

		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		t.AssertNil(err)
		s := g.Server(t.Name())
		s.SetAddr("127.0.0.1:0")
		server.Configure(s, server.Deps{Verifier: mustVerifier(t, priv), Catalog: cat})
		s.SetDumpRouterMap(false)
		s.Start()
		defer s.Shutdown()
		base := prefix(s)
		jwt := signToken(t, priv, testSub, time.Now().UTC().Add(10*time.Minute))
		op := func() *gclient.Client {
			c := g.Client()
			c.SetPrefix(base)
			c.ContentJson()
			c.SetHeader("Authorization", "Bearer "+jwt)
			return c
		}

		// a published post to comment on
		rc, err := op().Post(ctx, "/api/v1/posts", g.Map{"title": "Guard Post", "content": "body"})
		t.AssertNil(err)
		id := gjson.New(rc.ReadAllString()).Get("post.id").String()
		rc.Close()
		rp, err := op().Patch(ctx, "/api/v1/posts/"+id, g.Map{"status": "published"})
		t.AssertNil(err)
		rp.Close()
		const slug = "guard-post"

		// 1. blacklist hit → 422 blog.comment_rejected, nothing stored
		rb, err := op().Post(ctx, "/api/v1/posts/"+slug+"/comments", g.Map{"content": "join the CASINO now"})
		t.AssertNil(err)
		t.Assert(rb.StatusCode, 422)
		t.Assert(gjson.New(rb.ReadAllString()).Get("code").String(), "blog.comment_rejected")
		rb.Close()

		// 2. too many links → forced to pending even for the (author) auto-approve path
		rl, err := op().Post(ctx, "/api/v1/posts/"+slug+"/comments", g.Map{"content": "see https://a.com https://b.com https://c.com"})
		t.AssertNil(err)
		t.Assert(rl.StatusCode, 200)
		t.Assert(gjson.New(rl.ReadAllString()).Get("pending").Bool(), true)
		rl.Close()

		// reset the rate budget the above accumulated, then test the per-IP limit
		_, _ = db.Exec(ctx, "TRUNCATE comments")
		for i := 0; i < 3; i++ { // RatePerWindow=3 → first 3 succeed
			r, e := op().Post(ctx, "/api/v1/posts/"+slug+"/comments", g.Map{"content": fmt.Sprintf("ok %d", i)})
			t.AssertNil(e)
			t.Assert(r.StatusCode, 200)
			r.Close()
		}
		r4, err := op().Post(ctx, "/api/v1/posts/"+slug+"/comments", g.Map{"content": "one too many"})
		t.AssertNil(err)
		t.Assert(r4.StatusCode, 429)
		t.Assert(gjson.New(r4.ReadAllString()).Get("code").String(), "blog.rate_limited")
		r4.Close()

		_, _ = db.Exec(ctx, "TRUNCATE posts CASCADE")
	})
}
