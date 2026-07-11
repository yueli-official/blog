package v1

import "github.com/gogf/gf/v2/frame/g"

// SubscribeReq subscribes an email to the newsletter (double opt-in: a
// confirmation email is sent; the subscription is inactive until confirmed).
type SubscribeReq struct {
	g.Meta `path:"/api/v1/subscribe" method:"post" tags:"blog" summary:"Subscribe to the newsletter (double opt-in)"`
	Email  string `json:"email" v:"required|email"`
}

type SubscribeRes struct {
	Pending bool `json:"pending"` // true = confirmation email sent; false = already confirmed
}

// ConfirmSubscribeReq confirms a pending subscription via its token (the link in
// the confirmation email).
type ConfirmSubscribeReq struct {
	g.Meta `path:"/api/v1/subscribe/confirm" method:"get" tags:"blog" summary:"Confirm a newsletter subscription"`
	Token  string `json:"token" v:"required"`
}

type ConfirmSubscribeRes struct {
	Email string `json:"email"`
}

// UnsubscribeReq removes a subscription via its token (the link in every email).
type UnsubscribeReq struct {
	g.Meta `path:"/api/v1/unsubscribe" method:"post" tags:"blog" summary:"Unsubscribe from the newsletter"`
	Token  string `json:"token" v:"required"`
}

type UnsubscribeRes struct {
	Ok bool `json:"ok"`
}
