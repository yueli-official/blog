package blogclient

import (
	"context"
	"strings"
	"sync"

	"github.com/yueli-official/blog/api/internal/blogerr"
	"github.com/yueli-official/foundation/go/identifier"
)

// Fake is an in-memory asset client for tests. Finalize always yields a public
// cover URL. FailNext arms a one-shot upstream error (resilience tests).
type Fake struct {
	mu         sync.Mutex
	deleted    []string
	refs       []ReferenceInput
	unrefs     []ReferenceInput
	failNext   bool
	PublicBase string
}

func NewFake() *Fake {
	return &Fake{PublicBase: "http://asset.test/public"}
}

// FailNext arms a one-shot upstream failure on the next call.
func (f *Fake) FailNext() { f.mu.Lock(); f.failNext = true; f.mu.Unlock() }

func (f *Fake) tripped() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext {
		f.failNext = false
		return true
	}
	return false
}

func (f *Fake) UploadInit(_ context.Context, _ string, _ InitInput) (InitOutput, error) {
	if f.tripped() {
		return InitOutput{}, blogerr.UpstreamFailed("fake upstream 503")
	}
	tok := "faketok-" + identifier.MustNew().String()
	return InitOutput{UploadURL: "http://asset.test/api/v1/assets/blob/" + tok, UploadToken: tok}, nil
}

func (f *Fake) Finalize(_ context.Context, _, _ string) (View, error) {
	if f.tripped() {
		return View{}, blogerr.UpstreamFailed("fake upstream 503")
	}
	id := identifier.MustNew().String()
	return View{
		ID: id, MediaKey: strings.ReplaceAll(id, "-", ""), CdnURL: f.PublicBase + "/" + id,
		Size: 1234, Mime: "image/png", Filename: "cover.png",
	}, nil
}

func (f *Fake) Delete(_ context.Context, _, assetID string) error {
	if f.tripped() {
		return blogerr.UpstreamFailed("fake upstream 503")
	}
	f.mu.Lock()
	f.deleted = append(f.deleted, assetID)
	f.mu.Unlock()
	return nil
}

func (f *Fake) RegisterReference(_ context.Context, _ string, in ReferenceInput) error {
	if f.tripped() {
		return blogerr.UpstreamFailed("fake upstream 503")
	}
	f.mu.Lock()
	f.refs = append(f.refs, in)
	f.mu.Unlock()
	return nil
}

func (f *Fake) UnregisterReference(_ context.Context, _ string, in ReferenceInput) error {
	if f.tripped() {
		return blogerr.UpstreamFailed("fake upstream 503")
	}
	f.mu.Lock()
	f.unrefs = append(f.unrefs, in)
	f.mu.Unlock()
	return nil
}

// Deleted returns the asset ids Delete was called with (cleanup assertions).
func (f *Fake) Deleted() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.deleted...)
}
