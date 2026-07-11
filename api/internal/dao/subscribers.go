package dao

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"

	"platform/products/blog/api/internal/model"
)

const tSubscribers = "subscribers"

// GetSubscriberByEmail returns the subscriber for an email, or (nil, nil).
func (p *PG) GetSubscriberByEmail(ctx context.Context, email string) (*model.Subscriber, error) {
	var s *model.Subscriber
	if err := p.db.Model(tSubscribers).Ctx(ctx).Where("email", email).Limit(1).Scan(&s); err != nil {
		return nil, err
	}
	return s, nil
}

// InsertSubscriber creates a new pending subscriber with a confirm token.
func (p *PG) InsertSubscriber(ctx context.Context, email, token string) error {
	_, err := p.db.Model(tSubscribers).Ctx(ctx).Data(g.Map{
		"id": uuid.NewString(), "email": email, "status": model.SubPending, "confirm_token": token,
	}).Insert()
	return err
}

// RependSubscriber re-arms an existing (pending/unsubscribed) subscriber back to
// pending with a fresh token (status governs; confirmed_at is left as-is).
func (p *PG) RependSubscriber(ctx context.Context, email, token string) error {
	_, err := p.db.Model(tSubscribers).Ctx(ctx).Where("email", email).Data(g.Map{
		"status": model.SubPending, "confirm_token": token,
	}).Update()
	return err
}

// ConfirmSubscriber confirms a pending subscriber by token, returning its email.
// Returns ok=false for an unknown token or one not in the pending state.
func (p *PG) ConfirmSubscriber(ctx context.Context, token string) (string, bool, error) {
	if token == "" {
		return "", false, nil
	}
	var s *model.Subscriber
	if err := p.db.Model(tSubscribers).Ctx(ctx).
		Where("confirm_token", token).Where("status", model.SubPending).Limit(1).Scan(&s); err != nil {
		return "", false, err
	}
	if s == nil {
		return "", false, nil
	}
	if _, err := p.db.Model(tSubscribers).Ctx(ctx).Where("id", s.ID).Data(g.Map{
		"status": model.SubConfirmed, "confirmed_at": gtime.Now(),
	}).Update(); err != nil {
		return "", false, err
	}
	return s.Email, true, nil
}

// UnsubscribeByToken flips any subscriber with this token to unsubscribed.
func (p *PG) UnsubscribeByToken(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, nil
	}
	r, err := p.db.Model(tSubscribers).Ctx(ctx).Where("confirm_token", token).
		Data(g.Map{"status": model.SubUnsubscribed}).Update()
	if err != nil {
		return false, err
	}
	n, _ := r.RowsAffected()
	return n > 0, nil
}

// ListConfirmed returns all confirmed subscribers (email + token for digest +
// unsubscribe links).
func (p *PG) ListConfirmed(ctx context.Context) ([]*model.Subscriber, error) {
	var out []*model.Subscriber
	err := p.db.Model(tSubscribers).Ctx(ctx).Where("status", model.SubConfirmed).OrderAsc("created_at").Scan(&out)
	return out, err
}
