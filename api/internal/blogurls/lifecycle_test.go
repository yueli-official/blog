package blogurls

import (
	"context"
	"testing"

	"github.com/yueli-official/foundation/go/urllifecycle"
)

func TestPostRenameAndDelete(t *testing.T) {
	ctx := context.Background()
	module, err := NewMemory("https://blog.example")
	if err != nil {
		t.Fatal(err)
	}
	before := State{ID: "post-1", Kind: PostKind, Slug: "first", Published: true}
	after := State{ID: "post-1", Kind: PostKind, Slug: "second", Published: true}
	if err := module.Change(ctx, nil, State{ID: before.ID, Kind: before.Kind}, before); err != nil {
		t.Fatal(err)
	}
	if err := module.Change(ctx, nil, before, after); err != nil {
		t.Fatal(err)
	}
	resolved, err := module.Resolver().Resolve(ctx, urllifecycle.Lookup{EscapedPath: "/posts/first"})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Kind != urllifecycle.ResolutionRedirect || resolved.Location != "/posts/second" {
		t.Fatalf("old path resolution = %#v", resolved)
	}
	if err := module.Delete(ctx, nil, after); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/posts/first", "/posts/second"} {
		resolved, err = module.Resolver().Resolve(ctx, urllifecycle.Lookup{EscapedPath: path})
		if err != nil {
			t.Fatal(err)
		}
		if resolved.Kind != urllifecycle.ResolutionGone {
			t.Fatalf("%s resolution = %#v, want gone", path, resolved)
		}
	}
}

func TestTaxonomyMergeTargetsStableIdentity(t *testing.T) {
	ctx := context.Background()
	module, err := NewMemory("https://blog.example")
	if err != nil {
		t.Fatal(err)
	}
	source := State{ID: "tag-1", Kind: TagKind, Slug: "old", Published: true}
	target := State{ID: "tag-2", Kind: TagKind, Slug: "new", Published: true}
	if err := module.Merge(ctx, nil, source, target); err != nil {
		t.Fatal(err)
	}
	resolved, err := module.Resolver().Resolve(ctx, urllifecycle.Lookup{EscapedPath: "/tags/old"})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Kind != urllifecycle.ResolutionRedirect || resolved.Location != "/tags/new" {
		t.Fatalf("source resolution = %#v", resolved)
	}
}
