package catalog

import (
	"context"
	"html"
	"strings"

	"github.com/yueli-official/blog/api/internal/blogerr"
	"github.com/yueli-official/blog/api/internal/model"
)

// Subscribe registers (or re-arms) a newsletter subscription with double opt-in:
// a pending row + confirm token and a confirmation email. Idempotent for an
// already-confirmed address (returns pending=false, no email).
func (s *Service) Subscribe(ctx context.Context, email string) (bool, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if !looksLikeEmail(email) {
		return false, blogerr.InvalidInput("subscriber_email_invalid")
	}
	cur, err := s.dao.GetSubscriberByEmail(ctx, email)
	if err != nil {
		return false, err
	}
	if cur != nil && cur.Status == model.SubConfirmed {
		return false, nil // already subscribed
	}
	token := randHex(16)
	if cur == nil {
		if err := s.dao.InsertSubscriber(ctx, email, token); err != nil {
			return false, err
		}
	} else if err := s.dao.RependSubscriber(ctx, email, token); err != nil {
		return false, err
	}
	link := s.siteURL + "/subscribe/confirm?token=" + token
	_ = s.mailer.Send(ctx, email, "确认订阅博客更新",
		"<p>感谢订阅!请点击以下链接确认你的订阅:</p><p><a href=\""+link+"\">"+link+"</a></p><p>若非本人操作请忽略本邮件。</p>")
	return true, nil
}

// ConfirmSubscription confirms a pending subscription by token; returns the email.
func (s *Service) ConfirmSubscription(ctx context.Context, token string) (string, error) {
	if s.privacy != nil {
		email, ok, err := s.privacy.ConfirmSubscription(ctx, token)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", blogerr.InvalidInput("subscription_token_invalid")
		}
		return email, nil
	}
	email, ok, err := s.dao.ConfirmSubscriber(ctx, token)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", blogerr.InvalidInput("subscription_token_invalid")
	}
	return email, nil
}

// Unsubscribe removes a subscription by token.
func (s *Service) Unsubscribe(ctx context.Context, token string) error {
	if s.privacy != nil {
		ok, err := s.privacy.Unsubscribe(ctx, token)
		if err != nil {
			return err
		}
		if !ok {
			return blogerr.InvalidInput("subscription_token_invalid")
		}
		return nil
	}
	ok, err := s.dao.UnsubscribeByToken(ctx, token)
	if err != nil {
		return err
	}
	if !ok {
		return blogerr.InvalidInput("subscription_token_invalid")
	}
	return nil
}

// NotifyNewPost emails confirmed subscribers about a newly published post. The
// mailer delivers best-effort/asynchronously; each email carries an unsubscribe
// link. Safe to call from a goroutine on publish.
func (s *Service) NotifyNewPost(ctx context.Context, p *model.Post) {
	if p == nil {
		return
	}
	subs, err := s.dao.ListConfirmed(ctx)
	if err != nil {
		return
	}
	postURL := s.siteURL + "/posts/" + p.Slug
	for _, sub := range subs {
		if s.privacy != nil {
			allowed, err := s.privacy.CanDeliverNewsletter(ctx, sub.Email)
			if err != nil || !allowed {
				continue
			}
		}
		unsub := s.siteURL + "/unsubscribe?token=" + sub.ConfirmToken
		body := "<p>新文章发布:<a href=\"" + postURL + "\">" + html.EscapeString(p.Title) + "</a></p>"
		if p.Excerpt != "" {
			body += "<p>" + html.EscapeString(p.Excerpt) + "</p>"
		}
		body += "<p style=\"color:#888;font-size:12px\">不想再收到?<a href=\"" + unsub + "\">退订</a></p>"
		_ = s.mailer.Send(ctx, sub.Email, "新文章:"+p.Title, body)
	}
}
