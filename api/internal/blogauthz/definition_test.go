package blogauthz_test

import (
	"context"
	"testing"

	foundationauth "github.com/yueli-official/foundation/go/auth"
	"github.com/yueli-official/foundation/go/authorization"
	"platform/products/blog/api/internal/blogauthz"
)

func TestDefinitionEnforcesAuthorOwnershipAndAutomaticReconcile(t *testing.T) {
	admin := authorization.SubjectRef{Kind: authorization.SubjectUser, ID: "admin"}
	module, err := authorization.NewMemory(
		authorization.MustCompile(blogauthz.Definition()),
		authorization.MemoryOptions{
			RootScopeID: blogauthz.RootScopeID, ProtectedSubjects: []authorization.SubjectRef{admin},
			Constraints: blogauthz.ConstraintEvaluators(), Predicates: blogauthz.PredicateEvaluators(),
		},
	)
	if err != nil {
		t.Fatalf("NewMemory() error = %v", err)
	}
	ctx := context.Background()
	draft, err := module.CreatePolicyDraft(ctx, authorization.CreatePolicyDraftCommand{
		Actor: admin, ScopeID: blogauthz.RootScopeID, ExpectedActiveRevision: 1,
	})
	if err != nil {
		t.Fatalf("CreatePolicyDraft() error = %v", err)
	}
	if _, err := module.SetAutomaticRuleEnabled(ctx, authorization.SetAutomaticRuleEnabledCommand{
		Actor: admin, Revision: draft.Number,
		Rule: blogauthz.AutomaticRegistrationAuthorKey, Enabled: true,
	}); err != nil {
		t.Fatalf("SetAutomaticRuleEnabled() error = %v", err)
	}
	if _, err := module.ActivatePolicy(ctx, authorization.ActivatePolicyCommand{
		Actor: admin, Revision: draft.Number, ExpectedActiveRevision: 1,
	}); err != nil {
		t.Fatalf("ActivatePolicy() error = %v", err)
	}
	user := authorization.SubjectRef{Kind: authorization.SubjectUser, ID: "author"}
	result, err := module.ReconcileSubject(ctx, authorization.ReconcileSubjectCommand{Subject: user})
	if err != nil || result.Created != 1 || result.Grants[0].Role != blogauthz.RoleAuthor {
		t.Fatalf("ReconcileSubject() = %#v, %v", result, err)
	}
	if _, err := module.RegisterScope(ctx, authorization.RegisterScopeCommand{
		ID: blogauthz.PostScopeID("post-1"), Type: blogauthz.ScopePost, ParentID: blogauthz.RootScopeID,
	}); err != nil {
		t.Fatalf("RegisterScope() error = %v", err)
	}
	own := authorization.ResourceFacts{
		Type: "post", ID: "post-1", ScopeID: blogauthz.PostScopeID("post-1"),
		Relations: map[authorization.RelationKind][]authorization.SubjectRef{
			blogauthz.RelationOwner: {user},
		},
	}
	decision, err := module.Decide(ctx, authorization.DecisionRequest{
		Subject: user, Capability: blogauthz.CapabilityPostPublish,
		ScopeID: blogauthz.PostScopeID("post-1"), Resource: own,
	})
	if err != nil || !decision.Allowed {
		t.Fatalf("Decide(own publish) = %#v, %v; want allow", decision, err)
	}
	own.Relations[blogauthz.RelationOwner] = []authorization.SubjectRef{{Kind: authorization.SubjectUser, ID: "other"}}
	decision, err = module.Decide(ctx, authorization.DecisionRequest{
		Subject: user, Capability: blogauthz.CapabilityPostPublish,
		ScopeID: blogauthz.PostScopeID("post-1"), Resource: own,
	})
	if err != nil || decision.Allowed {
		t.Fatalf("Decide(other publish) = %#v, %v; want deny", decision, err)
	}

	service := blogauthz.New(module, nil)
	userContext := foundationauth.NewContext(ctx, &foundationauth.Principal{Subject: "author"})
	if access, err := service.EffectiveAccess(userContext); err != nil || len(access.Grants) == 0 {
		t.Fatalf("EffectiveAccess() = %#v, %v", access, err)
	}
}
