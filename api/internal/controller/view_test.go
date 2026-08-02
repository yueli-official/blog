package controller

import (
	"context"
	"strings"
	"testing"

	"github.com/yueli-official/blog/api/internal/blogauthz"
	"github.com/yueli-official/blog/api/internal/testidentity"
	foundationauth "github.com/yueli-official/foundation/go/auth"
	"github.com/yueli-official/foundation/go/authorization"
)

func TestDeriveExcerpt(t *testing.T) {
	// plain short text is returned as-is
	if got := deriveExcerpt("just a short note", 150); got != "just a short note" {
		t.Errorf("plain: got %q", got)
	}
	// markdown is stripped: heading/emphasis tokens gone, link → text, image dropped
	got := deriveExcerpt("# Title\n\nSome **bold** and a [link](https://x.com) ![pic](p.png) end", 150)
	for _, bad := range []string{"#", "**", "](", "![", "https://x.com", "p.png"} {
		if strings.Contains(got, bad) {
			t.Errorf("markdown leaked %q in %q", bad, got)
		}
	}
	if !strings.Contains(got, "Title") || !strings.Contains(got, "link") || !strings.Contains(got, "end") {
		t.Errorf("text dropped: %q", got)
	}
	// list markers at line start are removed, whitespace collapsed
	if got := deriveExcerpt("- one\n- two", 150); got != "one two" {
		t.Errorf("list: got %q", got)
	}
	// over-length is truncated with an ellipsis
	long := strings.Repeat("a", 200)
	out := deriveExcerpt(long, 150)
	if r := []rune(out); len(r) != 151 || !strings.HasSuffix(out, "…") {
		t.Errorf("truncate: len=%d suffix-ok=%v", len([]rune(out)), strings.HasSuffix(out, "…"))
	}
}

func TestRequireAdmin(t *testing.T) {
	module, err := authorization.NewMemory(
		authorization.MustCompile(blogauthz.Definition()),
		authorization.MemoryOptions{
			RootScopeID: blogauthz.RootScopeID,
			ProtectedSubjects: []authorization.SubjectRef{{
				Kind: authorization.SubjectUser, ID: "u-owner",
			}},
			Constraints: blogauthz.ConstraintEvaluators(),
			Predicates:  blogauthz.PredicateEvaluators(),
		},
	)
	if err != nil {
		t.Fatalf("authorization module: %v", err)
	}
	service := blogauthz.New(module, nil)
	withAuthorization := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, authorizationContextKey{}, service)
	}

	ownerCtx := withAuthorization(foundationauth.NewContext(context.Background(),
		testidentity.User(t, "u-owner", []string{"user"}, nil)))
	if !isAdmin(ownerCtx) {
		t.Fatal("configured owner should be admin")
	}
	if err := requireAdmin(ownerCtx); err != nil {
		t.Fatalf("owner should pass requireAdmin, got %v", err)
	}

	globalAdminCtx := withAuthorization(foundationauth.NewContext(context.Background(),
		testidentity.User(t, "u-admin", []string{"user", "admin"}, nil)))
	if isAdmin(globalAdminCtx) {
		t.Fatal("global admin role should not grant blog owner privileges")
	}
	if err := requireAdmin(globalAdminCtx); err == nil {
		t.Fatal("global admin role should be forbidden unless configured as blog owner")
	}

	userCtx := withAuthorization(foundationauth.NewContext(context.Background(),
		testidentity.User(t, "u-plain", []string{"user"}, nil)))
	if isAdmin(userCtx) {
		t.Fatal("principal without admin role should not be admin")
	}
	if err := requireAdmin(userCtx); err == nil {
		t.Fatal("non-admin should be forbidden")
	}

	if err := requireAdmin(context.Background()); err == nil {
		t.Fatal("missing principal should be forbidden")
	}
}
