package controller

import (
	"context"
	"strings"
	"testing"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcfg"

	foundationauth "github.com/yueli-official/foundation/go/auth"
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
	adapter, err := gcfg.NewAdapterContent("blog:\n  operatorSubs:\n    - \"u-owner\"\n")
	if err != nil {
		t.Fatalf("config adapter: %v", err)
	}
	g.Cfg().SetAdapter(adapter)

	ownerCtx := foundationauth.NewContext(context.Background(),
		&foundationauth.Principal{Subject: "u-owner", Roles: []string{"user"}})
	if !isAdmin(ownerCtx) {
		t.Fatal("configured owner should be admin")
	}
	if err := requireAdmin(ownerCtx); err != nil {
		t.Fatalf("owner should pass requireAdmin, got %v", err)
	}

	globalAdminCtx := foundationauth.NewContext(context.Background(),
		&foundationauth.Principal{Subject: "u-admin", Roles: []string{"user", "admin"}})
	if isAdmin(globalAdminCtx) {
		t.Fatal("global admin role should not grant blog owner privileges")
	}
	if err := requireAdmin(globalAdminCtx); err == nil {
		t.Fatal("global admin role should be forbidden unless configured as blog owner")
	}

	userCtx := foundationauth.NewContext(context.Background(),
		&foundationauth.Principal{Subject: "u-plain", Roles: []string{"user"}})
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
