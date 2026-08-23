package controller

import (
	"context"

	v1 "github.com/yueli-official/blog/api/api/v1"
	"github.com/yueli-official/blog/api/internal/blogauthz"
	"github.com/yueli-official/blog/api/internal/blogerr"
)

// AuthorizationSetup exposes only the non-sensitive, read-only claim state.
// The mutating claim endpoint remains on the authenticated Authorization API.
type AuthorizationSetup struct {
	service *blogauthz.Service
}

func NewAuthorizationSetup(service *blogauthz.Service) *AuthorizationSetup {
	return &AuthorizationSetup{service: service}
}

func (controller *AuthorizationSetup) GetInitialAdministratorClaimStatus(
	ctx context.Context,
	_ *v1.GetInitialAdministratorClaimStatusReq,
) (*v1.GetInitialAdministratorClaimStatusRes, error) {
	if controller == nil || controller.service == nil {
		return nil, blogerr.AuthorizationUnavailable()
	}
	status, err := controller.service.AdministratorClaimStatus(ctx)
	if err != nil {
		return nil, mapAuthorizationError(err)
	}
	return &v1.GetInitialAdministratorClaimStatusRes{
		Claimed: status.Claimed, CanClaim: !status.Claimed,
	}, nil
}
