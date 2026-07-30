package controller

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/yueli-official/foundation/go/authorization"

	"github.com/yueli-official/blog/api/internal/blogauthz"
	"github.com/yueli-official/blog/api/internal/blogerr"
)

type authorizationContextKey struct{}

func AuthorizationMiddleware(service *blogauthz.Service) ghttp.HandlerFunc {
	return func(request *ghttp.Request) {
		ctx := context.WithValue(request.Context(), authorizationContextKey{}, service)
		correlationID := strings.TrimSpace(request.Header.Get("X-Trace-Id"))
		if correlationID == "" {
			correlationID = strings.TrimSpace(request.Header.Get("X-Request-Id"))
		}
		ctx = authorization.WithRequestMetadata(ctx, authorization.RequestMetadata{CorrelationID: correlationID})
		request.SetCtx(ctx)
		request.Middleware.Next()
	}
}

func authorizationService(ctx context.Context) *blogauthz.Service {
	service, _ := ctx.Value(authorizationContextKey{}).(*blogauthz.Service)
	return service
}

func isAdmin(ctx context.Context) bool {
	service := authorizationService(ctx)
	return service != nil && service.IsAdministrator(ctx)
}

func requireAdmin(ctx context.Context) error {
	return requireCapability(ctx, authorization.CapabilityManage, blogauthz.RootScopeID, authorization.ResourceFacts{})
}

func requireCapability(
	ctx context.Context,
	capability authorization.CapabilityKey,
	scopeID authorization.ScopeID,
	resource authorization.ResourceFacts,
) error {
	service := authorizationService(ctx)
	if service == nil {
		return blogerr.AuthorizationUnavailable()
	}
	decision, err := service.Decide(ctx, capability, scopeID, resource)
	if err != nil {
		if authorization.Is(err, authorization.ErrorUnavailable) {
			return blogerr.AuthorizationUnavailable()
		}
		return blogerr.Forbidden()
	}
	if !decision.Allowed {
		return blogerr.Forbidden()
	}
	return nil
}

func postResource(ctx context.Context, id string) (authorization.ResourceFacts, error) {
	service := authorizationService(ctx)
	if service == nil {
		return authorization.ResourceFacts{}, blogerr.AuthorizationUnavailable()
	}
	resource, err := service.PostResource(ctx, id)
	if err != nil {
		return authorization.ResourceFacts{}, mapAuthorizationError(err)
	}
	return resource, nil
}

func seriesResource(ctx context.Context, id string) (authorization.ResourceFacts, error) {
	service := authorizationService(ctx)
	if service == nil {
		return authorization.ResourceFacts{}, blogerr.AuthorizationUnavailable()
	}
	resource, err := service.SeriesResource(ctx, id)
	if err != nil {
		return authorization.ResourceFacts{}, mapAuthorizationError(err)
	}
	return resource, nil
}

func commentResource(ctx context.Context, id string) (authorization.ResourceFacts, error) {
	service := authorizationService(ctx)
	if service == nil {
		return authorization.ResourceFacts{}, blogerr.AuthorizationUnavailable()
	}
	resource, err := service.CommentResource(ctx, id)
	if err != nil {
		return authorization.ResourceFacts{}, mapAuthorizationError(err)
	}
	return resource, nil
}

func mapAuthorizationError(err error) error {
	switch {
	case authorization.Is(err, authorization.ErrorDenied):
		return blogerr.Forbidden()
	case authorization.Is(err, authorization.ErrorUnavailable):
		return blogerr.AuthorizationUnavailable()
	case authorization.Is(err, authorization.ErrorNotFound):
		return blogerr.NotFound("authorization")
	case authorization.Is(err, authorization.ErrorInvalidInput),
		authorization.Is(err, authorization.ErrorConflict),
		authorization.Is(err, authorization.ErrorExpired):
		return blogerr.InvalidInput(err.Error())
	default:
		return blogerr.AuthorizationUnavailable()
	}
}

func resourceOwner(resource authorization.ResourceFacts) string {
	owners := resource.Relations[blogauthz.RelationOwner]
	if len(owners) == 0 {
		return ""
	}
	return owners[0].ID
}
