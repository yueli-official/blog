package blogtraffic

import (
	"context"
	"testing"
	"time"

	"github.com/yueli-official/foundation/go/traffic"
)

func TestSeedLocalNeverWritesFutureObservationsDuringMorningStartup(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 21, 5, 13, 0, 0, location)
	catalog, err := traffic.Compile(Definition("Asia/Shanghai"))
	if err != nil {
		t.Fatal(err)
	}
	module, err := traffic.NewMemory(catalog, traffic.MemoryOptions{
		Clock:  func() time.Time { return now },
		Secret: []byte("blog-local-seed-test-secret-32-bytes"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := SeedLocal(context.Background(), module, now); err != nil {
		t.Fatalf("SeedLocal() during morning startup: %v", err)
	}
}

func TestSeedLocalIsStableAcrossDayRollover(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 20, 15, 0, 0, 0, location)
	catalog, err := traffic.Compile(Definition("Asia/Shanghai"))
	if err != nil {
		t.Fatal(err)
	}
	module, err := traffic.NewMemory(catalog, traffic.MemoryOptions{
		Clock:  func() time.Time { return now },
		Secret: []byte("blog-local-seed-test-secret-32-bytes"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := SeedLocal(context.Background(), module, now); err != nil {
		t.Fatalf("first SeedLocal(): %v", err)
	}

	now = time.Date(2026, 8, 21, 5, 13, 0, 0, location)
	if err := SeedLocal(context.Background(), module, now); err != nil {
		t.Fatalf("SeedLocal() after day rollover: %v", err)
	}
}
