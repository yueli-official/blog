package blogauthz

import (
	"context"
	"testing"

	foundationauth "github.com/yueli-official/foundation/go/auth"
	"github.com/yueli-official/foundation/go/authorization"
)

func TestAuthenticatedUserClaimsInitialAdministrator(t *testing.T) {
	runtime, err := authorization.NewMemory(
		authorization.MustCompile(Definition()),
		authorization.MemoryOptions{
			RootScopeID:    RootScopeID,
			AllowUnclaimed: true,
			Constraints:    ConstraintEvaluators(),
			Predicates:     PredicateEvaluators(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	service := New(runtime, nil)
	ctx := foundationauth.NewContext(context.Background(), &foundationauth.Principal{
		Subject: "first-user", SubjectKind: foundationauth.SubjectUser,
	})

	status, err := service.AdministratorClaimStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.Claimed {
		t.Fatal("new instance should be unclaimed")
	}
	result, err := service.ClaimInitialAdministrator(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Created || !result.Status.Claimed || result.Grant.Target.ID != "first-user" {
		t.Fatalf("unexpected claim result: %#v", result)
	}
	if !service.IsAdministrator(ctx) {
		t.Fatal("claimant should immediately receive administrator access")
	}
}

func TestAnonymousCannotClaimInitialAdministrator(t *testing.T) {
	runtime, err := authorization.NewMemory(
		authorization.MustCompile(Definition()),
		authorization.MemoryOptions{
			RootScopeID:    RootScopeID,
			AllowUnclaimed: true,
			Constraints:    ConstraintEvaluators(),
			Predicates:     PredicateEvaluators(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = New(runtime, nil).ClaimInitialAdministrator(context.Background())
	if !authorization.Is(err, authorization.ErrorInvalidInput) {
		t.Fatalf("anonymous claim error = %v", err)
	}
}
