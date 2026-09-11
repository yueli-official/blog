package blogauthz_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yueli-official/blog/api/internal/blogauthz"
	foundationauth "github.com/yueli-official/foundation/go/auth"
	"github.com/yueli-official/foundation/go/authorization"
)

func TestPersonalTokenCannotRetainRevokedAdministratorRights(t *testing.T) {
	ctx := context.Background()
	admin := authorization.SubjectRef{Kind: authorization.SubjectUser, ID: "owner"}
	user := authorization.SubjectRef{Kind: authorization.SubjectUser, ID: "writer"}
	module, err := authorization.NewMemory(authorization.MustCompile(blogauthz.Definition()), authorization.MemoryOptions{
		RootScopeID: blogauthz.RootScopeID, ProtectedSubjects: []authorization.SubjectRef{admin}, Constraints: blogauthz.ConstraintEvaluators(), Predicates: blogauthz.PredicateEvaluators(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, descendants := range []bool{false, true} {
		access, err := module.EffectiveAccess(ctx, authorization.EffectiveAccessQuery{Subject: admin, ScopeID: blogauthz.RootScopeID, IncludeDescendants: descendants})
		if err != nil {
			t.Fatal(err)
		}
		hasUpdate := false
		for _, key := range access.Capabilities {
			if key == blogauthz.CapabilityPostUpdate {
				hasUpdate = true
			}
		}
		if hasUpdate != descendants {
			t.Fatalf("descendant permission discovery=%v, update=%v", descendants, hasUpdate)
		}
	}
	grant, err := module.Grant(ctx, authorization.GrantCommand{Actor: admin, Target: user, Role: blogauthz.RoleAdministrator, ScopeID: blogauthz.RootScopeID, Source: authorization.GrantSourceDirect})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := module.RegisterScope(ctx, authorization.RegisterScopeCommand{ID: "post:other", Type: blogauthz.ScopePost, ParentID: blogauthz.RootScopeID}); err != nil {
		t.Fatal(err)
	}
	scope, _ := foundationauth.PersonalScope("blog-main-web", string(blogauthz.CapabilityPostUpdate))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"userKey": user.ID, "scopes": []string{scope}})
	}))
	defer server.Close()
	verifier, _ := foundationauth.NewPersonalTokenVerifier(server.URL, "blog-main-web", nil)
	p, err := verifier.Verify(ctx, "pat_test")
	if err != nil {
		t.Fatal(err)
	}
	ctx = foundationauth.NewContext(ctx, p)
	service := blogauthz.New(module, nil)
	resource := authorization.ResourceFacts{Type: "post", ID: "other", ScopeID: "post:other", Relations: map[authorization.RelationKind][]authorization.SubjectRef{blogauthz.RelationOwner: {admin}}}
	decision, err := service.Decide(ctx, blogauthz.CapabilityPostUpdate, "post:other", resource)
	if err != nil || !decision.Allowed {
		t.Fatalf("admin denied: %v %v", decision, err)
	}
	if decision, err := service.Decide(ctx, blogauthz.CapabilityPostPublish, "post:other", resource); err != nil || decision.Allowed {
		t.Fatal("token gained unselected publish permission")
	}
	if _, err := module.Revoke(ctx, authorization.RevokeCommand{Actor: admin, GrantID: grant.ID}); err != nil {
		t.Fatal(err)
	}
	// The same already-authenticated principal must immediately lose admin rights.
	decision, err = service.Decide(ctx, blogauthz.CapabilityPostUpdate, "post:other", resource)
	if err != nil || decision.Allowed {
		t.Fatalf("revoked admin retained access: %v %v", decision, err)
	}
}
