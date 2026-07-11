package controller

import (
	"context"

	v1 "platform/products/blog/api/api/v1"
	"platform/products/blog/api/internal/catalog"
)

// Subscribers handles the public newsletter endpoints (no auth — anyone may
// subscribe; confirm/unsubscribe are token-bearer).
type Subscribers struct{ svc *catalog.Service }

func NewSubscribers(svc *catalog.Service) *Subscribers { return &Subscribers{svc: svc} }

func (c *Subscribers) Subscribe(ctx context.Context, req *v1.SubscribeReq) (*v1.SubscribeRes, error) {
	pending, err := c.svc.Subscribe(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	return &v1.SubscribeRes{Pending: pending}, nil
}

func (c *Subscribers) Confirm(ctx context.Context, req *v1.ConfirmSubscribeReq) (*v1.ConfirmSubscribeRes, error) {
	email, err := c.svc.ConfirmSubscription(ctx, req.Token)
	if err != nil {
		return nil, err
	}
	return &v1.ConfirmSubscribeRes{Email: email}, nil
}

func (c *Subscribers) Unsubscribe(ctx context.Context, req *v1.UnsubscribeReq) (*v1.UnsubscribeRes, error) {
	if err := c.svc.Unsubscribe(ctx, req.Token); err != nil {
		return nil, err
	}
	return &v1.UnsubscribeRes{Ok: true}, nil
}
