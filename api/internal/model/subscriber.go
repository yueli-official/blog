package model

import "github.com/gogf/gf/v2/os/gtime"

// Subscriber status values (newsletter double opt-in).
const (
	SubPending      = "pending"
	SubConfirmed    = "confirmed"
	SubUnsubscribed = "unsubscribed"
)

// Subscriber is a newsletter subscription. ConfirmToken is the per-subscriber
// secret used in both the confirm and the unsubscribe links (never serialized).
type Subscriber struct {
	ID           string      `json:"id" orm:"id"`
	Email        string      `json:"email" orm:"email"`
	Status       string      `json:"status" orm:"status"`
	ConfirmToken string      `json:"-" orm:"confirm_token"`
	CreatedAt    *gtime.Time `json:"createdAt" orm:"created_at"`
	ConfirmedAt  *gtime.Time `json:"confirmedAt" orm:"confirmed_at"`
}
