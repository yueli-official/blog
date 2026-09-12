package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	v1 "github.com/yueli-official/blog/api/api/v1"
	"github.com/yueli-official/blog/api/internal/blogauthz"
	foundationauth "github.com/yueli-official/foundation/go/auth"
	"github.com/yueli-official/foundation/go/authorization"
)

func TestPersonalTokenPublishingRoutes(t *testing.T) {
	allowed := []string{
		"GET /api/v1/posts/mine", "POST /api/v1/posts", "PATCH /api/v1/posts/post-id",
		"DELETE /api/v1/posts/post-id", "POST /api/v1/posts/post-id/restore",
		"POST /api/v1/images", "POST /api/v1/images/finalize", "POST /api/v1/personal-token/media-authorization",
		"POST /api/v1/posts/post-id/cover", "POST /api/v1/posts/post-id/cover/finalize",
		"POST /api/v1/taxonomies", "PATCH /api/v1/taxonomies/category-id",
		"DELETE /api/v1/taxonomies/tag-id", "POST /api/v1/taxonomies/tag-id/merge",
		"PUT /api/v1/posts/post-id/taxonomies", "PUT /api/v1/posts/post-id/series",
		"PUT /api/v1/posts/post-id/flags", "POST /api/v1/series",
		"PATCH /api/v1/series/series-id", "DELETE /api/v1/series/series-id",
	}
	denied := []string{
		"POST /api/v1/posts/batch", "POST /api/v1/authorization/grants",
		"GET /api/v1/authorization", "PUT /api/v1/home", "POST /api/v1/comments",
		"GET /api/v1/internal/personal-token/permissions", "DELETE /api/v1/posts/post-id/permanent",
		"POST /api/v1/taxonomies/category-id/delete", "PATCH /api/v1/posts/post-id/flags",
		"POST /api/v1/posts/post-id/taxonomies", "PUT /api/v1/posts/post-id/cover",
		"PATCH /api/v1/series/series-id/settings", "POST /api/v1/posts//cover",
		"POST /api/v1/posts/post-id/cover/finalize/extra", "PATCH /api/v1/taxonomies/",
		"POST /api/v1/posts/post-id/unknown", "GET /api/v1/posts/post-id/cover",
	}
	for _, group := range []struct {
		requests []string
		want     bool
	}{{allowed, true}, {denied, false}} {
		for _, request := range group.requests {
			t.Run(request, func(t *testing.T) {
				method, path, _ := strings.Cut(request, " ")
				if got := personalTokenRouteAllowed(method, path); got != group.want {
					t.Fatalf("allowed = %v, want %v", got, group.want)
				}
			})
		}
	}
}

func personalTestContext(t *testing.T, user string, permissions ...string) context.Context {
	t.Helper()
	scopes := make([]string, len(permissions))
	for i, permission := range permissions {
		scope, err := foundationauth.PersonalScope("blog-main-web", permission)
		if err != nil {
			t.Fatal(err)
		}
		scopes[i] = scope
	}
	identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"userKey": user, "scopes": scopes})
	}))
	t.Cleanup(identity.Close)
	verifier, err := foundationauth.NewPersonalTokenVerifier(identity.URL, "blog-main-web", nil)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := verifier.Verify(context.Background(), "pat_test")
	if err != nil {
		t.Fatal(err)
	}
	return foundationauth.NewContext(context.Background(), principal)
}

func personalTestService(t *testing.T) (*PersonalPermissions, authorization.SubjectRef, authorization.SubjectRef) {
	t.Helper()
	admin := authorization.SubjectRef{Kind: authorization.SubjectUser, ID: "owner"}
	author := authorization.SubjectRef{Kind: authorization.SubjectUser, ID: "writer"}
	module, err := authorization.NewMemory(authorization.MustCompile(blogauthz.Definition()), authorization.MemoryOptions{
		RootScopeID: blogauthz.RootScopeID, ProtectedSubjects: []authorization.SubjectRef{admin},
		Constraints: blogauthz.ConstraintEvaluators(), Predicates: blogauthz.PredicateEvaluators(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := module.Grant(context.Background(), authorization.GrantCommand{
		Actor: admin, Target: author, Role: blogauthz.RoleAuthor, ScopeID: blogauthz.RootScopeID, Source: authorization.GrantSourceDirect,
	}); err != nil {
		t.Fatal(err)
	}
	return NewPersonalPermissions("blog-main-web", blogauthz.New(module, nil)), admin, author
}

func TestPersonalPermissionDirectoryFollowsCurrentProductRights(t *testing.T) {
	c, admin, author := personalTestService(t)
	caller := foundationauth.NewContext(context.Background(), &foundationauth.Principal{
		SubjectKind: foundationauth.SubjectClient, ClientID: "identity-svc", Scopes: []string{foundationauth.PersonalPermissionsScope},
	})
	for _, test := range []struct {
		user  string
		admin bool
	}{{admin.ID, true}, {author.ID, false}} {
		response, err := c.GetPersonalPermissions(caller, &v1.PersonalPermissionsReq{UserKey: test.user})
		if err != nil {
			t.Fatal(err)
		}
		keys := []string{}
		for _, permission := range response.Items {
			if slices.Contains(keys, permission.Key) {
				t.Fatalf("duplicate permission: %s", permission.Key)
			}
			keys = append(keys, permission.Key)
		}
		for _, key := range []string{"media.upload", "blog.post.read", "blog.post.create", "blog.post.update", "blog.post.publish", "blog.post.archive", "blog.post.delete", "blog.tag.create", "blog.series.create", "blog.series.update", "blog.series.delete"} {
			if !slices.Contains(keys, key) {
				t.Errorf("%s is missing %s", test.user, key)
			}
		}
		for _, key := range []string{"blog.taxonomy.manage", "blog.post_flags.manage"} {
			if slices.Contains(keys, key) != test.admin {
				t.Errorf("%s: governance permission %s must follow current role", test.user, key)
			}
		}
		if slices.Contains(keys, string(authorization.CapabilityManage)) || slices.Contains(keys, "blog.site_settings.manage") {
			t.Fatal("publishing directory must not grant account or site administration")
		}
	}
	if _, err := c.GetPersonalPermissions(personalTestContext(t, admin.ID, "blog.taxonomy.manage"), &v1.PersonalPermissionsReq{UserKey: admin.ID}); err == nil {
		t.Fatal("PAT must not read another user's delegable directory")
	}
}

func TestPersonalMediaProfilesAndCoverScope(t *testing.T) {
	c, _, author := personalTestService(t)
	for _, test := range []struct {
		name   string
		user   string
		scopes []string
		want   []string
	}{
		{"body-only", author.ID, []string{"media.upload"}, []string{"asset.profile.blog-post.upload"}},
		{"body-and-cover", author.ID, []string{"media.upload", "blog.post.update"}, []string{"asset.profile.blog-post.upload", "asset.profile.blog-cover.upload"}},
		{"missing-upload", author.ID, []string{"blog.post.update"}, nil},
		{"missing-current-role", "outsider", []string{"media.upload", "blog.post.update"}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := personalTestContext(t, test.user, test.scopes...)
			response, err := c.AuthorizePersonalMedia(ctx, &v1.PersonalMediaAuthorizationReq{})
			if test.want == nil {
				if err == nil {
					t.Fatal("unauthorized media grant")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got := []string{}
			for _, scope := range response.Scopes {
				site, permission, ok := foundationauth.ParsePersonalScope(scope)
				if !ok || site != "blog-main-web" {
					t.Fatalf("media escaped site: %s", scope)
				}
				got = append(got, permission)
			}
			if !slices.Equal(got, test.want) {
				t.Fatalf("profiles = %v, want %v", got, test.want)
			}
		})
	}
	ctx := personalTestContext(t, author.ID, "blog.post.update")
	cover := NewCover(nil)
	if _, err := cover.CoverInit(ctx, &v1.CoverInitReq{ID: "post"}); err == nil {
		t.Fatal("cover init accepted a token without media.upload")
	}
	if _, err := cover.CoverFinalize(ctx, &v1.CoverFinalizeReq{ID: "post"}); err == nil {
		t.Fatal("cover finalize accepted a token without media.upload")
	}
}

func TestPersonalTaxonomyScopeDoesNotCreateAnAdministrator(t *testing.T) {
	c, admin, author := personalTestService(t)
	for _, test := range []struct {
		user   string
		scopes []string
		allow  bool
	}{
		{admin.ID, []string{"blog.taxonomy.manage"}, true},
		{admin.ID, []string{"blog.tag.create"}, false},
		{author.ID, []string{"blog.taxonomy.manage"}, false},
	} {
		ctx := personalTestContext(t, test.user, test.scopes...)
		decision, err := c.service.Decide(ctx, blogauthz.CapabilityTaxonomyManage, blogauthz.RootScopeID, authorization.ResourceFacts{})
		if err != nil || decision.Allowed != test.allow {
			t.Fatalf("user %s scopes %v: %v, %v", test.user, test.scopes, decision, err)
		}
	}
}
