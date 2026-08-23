package controller

import (
	"context"
	"testing"

	v1 "github.com/yueli-official/blog/api/api/v1"
	"github.com/yueli-official/blog/api/internal/blogauthz"
	"github.com/yueli-official/foundation/go/authorization"
)

func TestAuthorizationSetupStatusIsPublicAndDoesNotExposeOwner(t *testing.T) {
	runtime, err := authorization.NewMemory(
		authorization.MustCompile(blogauthz.Definition()),
		authorization.MemoryOptions{
			RootScopeID:    blogauthz.RootScopeID,
			AllowUnclaimed: true,
			Constraints:    blogauthz.ConstraintEvaluators(),
			Predicates:     blogauthz.PredicateEvaluators(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	controller := NewAuthorizationSetup(blogauthz.New(runtime, nil))
	response, err := controller.GetInitialAdministratorClaimStatus(
		context.Background(), &v1.GetInitialAdministratorClaimStatusReq{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if response.Claimed || !response.CanClaim {
		t.Fatalf("status = %#v", response)
	}
}
