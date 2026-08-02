package blogauthz

import (
	"context"
	"database/sql"
	"fmt"

	foundationauth "github.com/yueli-official/foundation/go/auth"
	"github.com/yueli-official/foundation/go/authorization"
)

type Runtime interface {
	authorization.Authorizer
	authorization.QueryPlanner
	authorization.AccessReader
	authorization.ResourceScopeRegistry
	authorization.RoleManager
	authorization.RoleReader
	authorization.GrantManager
	authorization.GrantReader
	authorization.WorkflowManager
	authorization.WorkflowReader
	authorization.PolicyManager
	authorization.PolicyReader
	authorization.Reconciler
}

type Service struct {
	runtime Runtime
	db      *sql.DB
}

func New(runtime Runtime, db *sql.DB) *Service {
	return &Service{runtime: runtime, db: db}
}

func (service *Service) Runtime() Runtime {
	if service == nil {
		return nil
	}
	return service.runtime
}

func (service *Service) Subject(ctx context.Context) authorization.SubjectRef {
	principal, ok := foundationauth.FromContext(ctx)
	if !ok {
		return authorization.SubjectRef{Kind: authorization.SubjectAnonymous}
	}
	subjectKind, _ := principal.Claim("subject_kind")
	if subjectKind == "user" && principal.Subject != "" {
		return authorization.SubjectRef{Kind: authorization.SubjectUser, ID: principal.Subject}
	}
	if subjectKind == "client" && principal.ClientID != "" {
		return authorization.SubjectRef{Kind: authorization.SubjectService, ID: principal.ClientID}
	}
	return authorization.SubjectRef{Kind: authorization.SubjectAnonymous}
}

func (service *Service) ReconcileSubject(ctx context.Context) error {
	if service == nil || service.runtime == nil {
		return unavailable("runtime")
	}
	subject := service.Subject(ctx)
	if subject.Kind != authorization.SubjectUser || subject.ID == "" {
		return nil
	}
	preview, err := service.runtime.PreviewReconcileSubject(ctx, authorization.ReconcileSubjectCommand{
		Subject: subject,
	})
	if err != nil || preview.Created == 0 {
		return err
	}
	_, err = service.runtime.ReconcileSubject(ctx, authorization.ReconcileSubjectCommand{Subject: subject})
	return err
}

func (service *Service) Decide(
	ctx context.Context,
	capability authorization.CapabilityKey,
	scopeID authorization.ScopeID,
	resource authorization.ResourceFacts,
) (authorization.Decision, error) {
	if service == nil || service.runtime == nil {
		return authorization.Decision{}, unavailable("runtime")
	}
	if err := service.ReconcileSubject(ctx); err != nil {
		return authorization.Decision{}, err
	}
	return service.runtime.Decide(ctx, authorization.DecisionRequest{
		Subject: service.Subject(ctx), Capability: capability, ScopeID: scopeID, Resource: resource,
	})
}

func (service *Service) EffectiveAccess(ctx context.Context) (authorization.EffectiveAccess, error) {
	if service == nil || service.runtime == nil {
		return authorization.EffectiveAccess{}, unavailable("runtime")
	}
	if err := service.ReconcileSubject(ctx); err != nil {
		return authorization.EffectiveAccess{}, err
	}
	return service.runtime.EffectiveAccess(ctx, authorization.EffectiveAccessQuery{
		Subject: service.Subject(ctx), ScopeID: RootScopeID,
	})
}

func (service *Service) IsAdministrator(ctx context.Context) bool {
	decision, err := service.Decide(ctx, authorization.CapabilityManage, RootScopeID, authorization.ResourceFacts{})
	return err == nil && decision.Allowed
}

func (service *Service) EnsurePostScope(ctx context.Context, id string) error {
	return ensureScope(ctx, service.runtime, authorization.RegisterScopeCommand{
		ID: PostScopeID(id), Type: ScopePost, ParentID: RootScopeID,
	})
}

func (service *Service) EnsureSeriesScope(ctx context.Context, id string) error {
	return ensureScope(ctx, service.runtime, authorization.RegisterScopeCommand{
		ID: SeriesScopeID(id), Type: ScopeSeries, ParentID: RootScopeID,
	})
}

func (service *Service) EnsureCommentScope(ctx context.Context, id, postID string) error {
	if err := service.EnsurePostScope(ctx, postID); err != nil {
		return err
	}
	return ensureScope(ctx, service.runtime, authorization.RegisterScopeCommand{
		ID: CommentScopeID(id), Type: ScopeComment, ParentID: PostScopeID(postID),
	})
}

func (service *Service) PostResource(ctx context.Context, id string) (authorization.ResourceFacts, error) {
	if service == nil || service.db == nil {
		return authorization.ResourceFacts{}, unavailable("database")
	}
	var owner string
	if err := service.db.QueryRowContext(ctx,
		`SELECT author_id FROM posts WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&owner); err != nil {
		if err == sql.ErrNoRows {
			return authorization.ResourceFacts{}, &authorization.Error{
				Kind: authorization.ErrorNotFound, Field: "post", Message: "not found",
			}
		}
		return authorization.ResourceFacts{}, fmt.Errorf("load blog post authorization facts: %w", err)
	}
	if err := service.EnsurePostScope(ctx, id); err != nil {
		return authorization.ResourceFacts{}, err
	}
	return ownedResource("post", id, PostScopeID(id), owner), nil
}

func (service *Service) SeriesResource(ctx context.Context, id string) (authorization.ResourceFacts, error) {
	if service == nil || service.db == nil {
		return authorization.ResourceFacts{}, unavailable("database")
	}
	var owner string
	if err := service.db.QueryRowContext(ctx,
		`SELECT author_id FROM series WHERE id = $1`, id,
	).Scan(&owner); err != nil {
		if err == sql.ErrNoRows {
			return authorization.ResourceFacts{}, &authorization.Error{
				Kind: authorization.ErrorNotFound, Field: "series", Message: "not found",
			}
		}
		return authorization.ResourceFacts{}, fmt.Errorf("load blog series authorization facts: %w", err)
	}
	if err := service.EnsureSeriesScope(ctx, id); err != nil {
		return authorization.ResourceFacts{}, err
	}
	return ownedResource("series", id, SeriesScopeID(id), owner), nil
}

func (service *Service) CommentResource(ctx context.Context, id string) (authorization.ResourceFacts, error) {
	if service == nil || service.db == nil {
		return authorization.ResourceFacts{}, unavailable("database")
	}
	var postID, owner string
	if err := service.db.QueryRowContext(ctx, `
		SELECT c.post_id::text, p.author_id
		FROM comments c
		JOIN posts p ON p.id = c.post_id
		WHERE c.id = $1 AND p.deleted_at IS NULL
	`, id).Scan(&postID, &owner); err != nil {
		if err == sql.ErrNoRows {
			return authorization.ResourceFacts{}, &authorization.Error{
				Kind: authorization.ErrorNotFound, Field: "comment", Message: "not found",
			}
		}
		return authorization.ResourceFacts{}, fmt.Errorf("load blog comment authorization facts: %w", err)
	}
	if err := service.EnsureCommentScope(ctx, id, postID); err != nil {
		return authorization.ResourceFacts{}, err
	}
	return ownedResource("comment", id, CommentScopeID(id), owner), nil
}

// ManagePostOwner returns an empty owner for unrestricted access or the caller
// subject for the author relation constraint.
func (service *Service) ManagePostOwner(ctx context.Context) (string, error) {
	if service == nil || service.runtime == nil {
		return "", unavailable("runtime")
	}
	if err := service.ReconcileSubject(ctx); err != nil {
		return "", err
	}
	constraint, err := service.runtime.Plan(ctx, authorization.QueryRequest{
		Subject: service.Subject(ctx), Capability: CapabilityPostRead, ScopeID: RootScopeID,
	})
	if err != nil {
		return "", err
	}
	switch constraint.Kind {
	case authorization.QueryAll:
		return "", nil
	case authorization.QueryRelation:
		if constraint.Relation != RelationOwner {
			return "", unavailable("query relation")
		}
		return constraint.Subject.ID, nil
	case authorization.QueryNone:
		return "", &authorization.Error{Kind: authorization.ErrorDenied, Message: "post collection is not accessible"}
	default:
		return "", unavailable("query constraint")
	}
}

func ownedResource(
	resourceType authorization.ResourceType,
	id string,
	scopeID authorization.ScopeID,
	owner string,
) authorization.ResourceFacts {
	resource := authorization.ResourceFacts{
		Type: resourceType, ID: authorization.ResourceID(id), ScopeID: scopeID,
	}
	if owner != "" {
		resource.Relations = map[authorization.RelationKind][]authorization.SubjectRef{
			RelationOwner: {{Kind: authorization.SubjectUser, ID: owner}},
		}
	}
	return resource
}

func unavailable(field string) *authorization.Error {
	return &authorization.Error{
		Kind: authorization.ErrorUnavailable, Field: field, Message: "is not configured",
	}
}
