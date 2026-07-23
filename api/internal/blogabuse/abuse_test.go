package blogabuse

import (
	"context"
	"testing"
	"time"

	"github.com/yueli-official/foundation/go/abuse"
)

func TestAnonymousAndMemberBudgetsAreIndependent(t *testing.T) {
	module, err := abuse.NewMemory(
		abuse.MustCompile(Definition(Policy{AnonymousCapacity: 2, Window: time.Hour})),
		abuse.MemoryOptions{Secret: []byte("blog-abuse-test-secret-at-least-32-bytes")},
	)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := Bind(module)
	if err != nil {
		t.Fatal(err)
	}
	network, err := NetworkPrefix("192.0.2.10")
	if err != nil {
		t.Fatal(err)
	}
	for index, want := range []abuse.Disposition{
		abuse.DispositionAllow,
		abuse.DispositionAllow,
		abuse.DispositionReject,
	} {
		got, err := actions.Anonymous.Admit(context.Background(), abuse.Input{
			ID:      abuse.AttemptID("anonymous-" + string(rune('a'+index))),
			Signals: abuse.Signals{Network: network},
		})
		if err != nil {
			t.Fatal(err)
		}
		if got.Disposition != want {
			t.Fatalf("anonymous attempt %d: got %q, want %q", index+1, got.Disposition, want)
		}
	}
	member, err := actions.Member.Admit(context.Background(), abuse.Input{
		ID:      "member-a",
		Signals: abuse.Signals{Network: network, Actor: "user-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if member.Disposition != abuse.DispositionAllow {
		t.Fatalf("member action must have an independent budget, got %q", member.Disposition)
	}
}
